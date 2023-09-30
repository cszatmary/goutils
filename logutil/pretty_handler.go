package logutil

import (
	"bytes"
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/cszatmary/goutils/color"
)

// PrettyHandler is a Handler that writes Records to an io.Writer in a pretty format that looks like so:
//
// DEBUG some log message foo=bar
type PrettyHandler struct {
	opts        PrettyHandlerOptions
	w           io.Writer
	mu          sync.Mutex
	c           color.Colorer
	attrsList   []attrsNode
	groupPrefix string
	groups      []string
}

// PrettyHandlerOptions are options for a PrettyHandler.
// A zero value consists entirely of default values.
type PrettyHandlerOptions struct {
	// AddSource adds source code position information to the log using
	// the SourceKey attribute.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// See the Level field of [slog.HandlerOptions].
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See the ReplaceAttr field of [slog.HandlerOptions].
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr

	// ForceQuote forces quoting of all values.
	// By default, quoting will only be applied if required.
	ForceQuote bool

	// Disables using colours in logs.
	DisableColor bool
}

// NewPrettyHandler creates a new PrettyHandler that writes to the given writer,
// using the given options. If opts is nil, the default options are used.
func NewPrettyHandler(w io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	var o PrettyHandlerOptions
	if opts != nil {
		o = *opts
	}
	if o.Level == nil {
		o.Level = slog.LevelInfo
	}
	var c color.Colorer
	c.SetEnabled(!o.DisableColor)
	return &PrettyHandler{opts: o, w: w, c: c}
}

func (h *PrettyHandler) clone() *PrettyHandler {
	return &PrettyHandler{
		opts:        h.opts,
		w:           h.w,
		c:           h.c,
		attrsList:   slices.Clip(h.attrsList),
		groupPrefix: h.groupPrefix,
		groups:      slices.Clip(h.groups),
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()
	h2.attrsList = append(h2.attrsList, attrsNode{groupPrefix: h2.groupPrefix, groups: h.groups, attrs: attrs})
	return h2
}

type attrsNode struct {
	groupPrefix string
	groups      []string
	attrs       []slog.Attr
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	b := newBuffer()
	defer freeBuffer(b)

	var colorFunc func(string) string
	switch r.Level {
	case slog.LevelDebug:
		colorFunc = h.c.White
	case slog.LevelInfo:
		colorFunc = h.c.Cyan
	case slog.LevelWarn:
		colorFunc = h.c.Yellow
	case slog.LevelError:
		colorFunc = h.c.Red
	}

	// Treat all built-in fields as Attrs, this simplifies the branching needed here to handle ReplaceAttr.
	// appendAttr will figure out how to handle everything correctly.
	if !r.Time.IsZero() {
		// strip monotonic to match Attr behavior
		h.appendAttr(b, slog.Time(slog.TimeKey, r.Time.Round(0)), state{colorFunc: colorFunc})
	}
	h.appendAttr(b, slog.Any(slog.LevelKey, r.Level), state{colorFunc: colorFunc})
	if h.opts.AddSource {
		src := CallerSource(r.PC)
		h.appendAttr(b, slog.Any(slog.SourceKey, &src), state{colorFunc: colorFunc})
	}
	h.appendAttr(b, slog.String(slog.MessageKey, r.Message), state{colorFunc: colorFunc})

	// attrs
	if len(h.attrsList) > 0 {
		for _, n := range h.attrsList {
			s := state{n.groupPrefix, n.groups, colorFunc}
			for _, a := range n.attrs {
				h.appendAttr(b, a, s)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(b, a, state{h.groupPrefix, h.groups, colorFunc})
		return true
	})
	data := b.Bytes()
	if len(data) > 0 {
		// If there was any data written there must be a trailing space
		// since appendAttr always adds a space at the end.
		// Replace the final space with a newline.
		data[len(data)-1] = '\n'
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(data)
	return err
}

func (h *PrettyHandler) appendAttr(b *bytes.Buffer, a slog.Attr, s state) {
	if rep := h.opts.ReplaceAttr; rep != nil && a.Value.Kind() != slog.KindGroup {
		// Resolve before calling ReplaceAttr so the caller doesn't have to.
		a.Value = a.Value.Resolve()
		a = rep(s.groups, a)
	}
	a.Value = a.Value.Resolve()
	// Skip empty attrs.
	if a.Equal(slog.Attr{}) {
		return
	}
	// Handle group.
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		// Skip empty groups.
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			s.groupPrefix += a.Key + "."
			s.groups = append(s.groups, a.Key)
		}
		for _, aa := range attrs {
			h.appendAttr(b, aa, s)
		}
		return
	}
	// Special case, stringify source nicely.
	if v := a.Value; v.Kind() == slog.KindAny {
		if src, ok := v.Any().(*slog.Source); ok {
			a.Value = slog.StringValue(fmt.Sprintf("%s:%d", src.File, src.Line))
		}
	}

	// Handle built-ins first.
	if a.Key == slog.TimeKey {
		b.WriteString(stringify(a.Value))
	} else if a.Key == slog.LevelKey {
		if l, ok := a.Value.Any().(slog.Level); ok {
			// Pad level so that it is the same length for every line, i.e.
			// "INFO "
			// "DEBUG"
			str := fmt.Sprintf("%-5s", l.String())
			if s.colorFunc != nil {
				str = s.colorFunc(str)
			}
			b.WriteString(str)
		} else {
			// If the level was modified by ReplaceAttrs then it is the caller's
			// responsibility to handle colouring.
			b.WriteString(stringify(a.Value))
		}
	} else if a.Key == slog.SourceKey {
		b.WriteString(h.c.Magenta(stringify(a.Value)))
	} else if a.Key == slog.MessageKey {
		fmt.Fprintf(b, "%-44s", stringify(a.Value))
	} else {
		// Handle remaining attrs.
		h.appendString(b, s.groupPrefix+a.Key, s.colorFunc)
		b.WriteByte('=')
		h.appendString(b, stringify(a.Value), nil)
	}
	b.WriteByte(' ')
}

type state struct {
	groupPrefix string
	groups      []string
	colorFunc   func(string) string
}

func (h *PrettyHandler) appendString(b *bytes.Buffer, s string, colorFunc func(string) string) {
	if h.needsQuoting(s) {
		s = strconv.Quote(s)
	}
	if colorFunc != nil {
		s = colorFunc(s)
	}
	b.WriteString(s)
}

func (h *PrettyHandler) needsQuoting(s string) bool {
	if h.opts.ForceQuote || s == "" {
		return true
	}
	for _, c := range s {
		// Needs to be quoted if it's not alphanumeric and not one of the special chars below.
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '/' || c == '@' || c == '^' || c == '+' || c == ':') {
			return true
		}
	}
	return false
}

func stringify(v slog.Value) string {
	switch v.Kind() {
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'g', -1, 64)
	case slog.KindString:
		return v.String()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	case slog.KindAny:
		vv := v.Any()
		if tm, ok := vv.(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				return fmt.Sprintf("!ERROR:%v", err)
			}
			return string(data)
		}
		// Handle byte slices specially and try and print it nicely.
		if bs, ok := vv.([]byte); ok {
			return string(bs)
		}
		if t := reflect.TypeOf(vv); t != nil {
			// Handle the underlying type being []byte.
			if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
				return string(reflect.ValueOf(vv).Bytes())
			}
			// Check to see if a we have a function or channel.
			// These can't be printed properly and fmt.Sprint will generate an
			// ugly pointer representation, ex: %!s(func()=0x10bf380)
			switch t.Kind() {
			case reflect.Chan:
				name := t.Elem().Name()
				if name == "" {
					name = "unknown"
				}
				return "chan " + name
			case reflect.Func:
				return "func()"
			}
		}

		return fmt.Sprintf("%+v", v)
	default:
		panic(fmt.Errorf("impossible: invalid slog.Value kind: %s", v.Kind()))
	}
}

// Pool of reusable buffers to reduce allocation.
var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func newBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func freeBuffer(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}
