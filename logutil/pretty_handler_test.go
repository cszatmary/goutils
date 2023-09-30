package logutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cszatmary/goutils/logutil"
)

// These tests are adapted from src/log/slog/handler_test.go in order
// to ensure that the behaviour is consistent with slog.TextHandler.

func TestPrettyHandler(t *testing.T) {
	ctx := context.Background()

	// remove all Attrs
	removeAll := func(_ []string, a slog.Attr) slog.Attr { return slog.Attr{} }

	attrs := []slog.Attr{slog.String("a", "one"), slog.Int("b", 2), slog.Any("", nil)}
	preAttrs := []slog.Attr{slog.Int("pre", 3), slog.String("x", "y")}

	for _, test := range []struct {
		name      string
		replace   func([]string, slog.Attr) slog.Attr
		addSource bool
		with      func(slog.Handler) slog.Handler
		preAttrs  []slog.Attr
		attrs     []slog.Attr
		want      string
	}{
		{
			name:  "basic",
			attrs: attrs,
			want:  "2000-01-02T03:04:05Z INFO  message                                      a=one b=2",
		},
		{
			name:  "empty key",
			attrs: append(slices.Clip(attrs), slog.Any("", "v")),
			want:  `2000-01-02T03:04:05Z INFO  message                                      a=one b=2 ""=v`,
		},
		{
			name:    "cap keys",
			replace: upperCaseKey,
			attrs:   attrs,
			want:    "TIME=2000-01-02T03:04:05Z LEVEL=INFO MSG=message A=one B=2",
		},
		{
			name:    "remove all",
			replace: removeAll,
			attrs:   attrs,
			want:    "",
		},
		{
			name:     "preformatted",
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			want:     "2000-01-02T03:04:05Z INFO  message                                      pre=3 x=y a=one b=2",
		},
		{
			name:     "preformatted cap keys",
			replace:  upperCaseKey,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			want:     "TIME=2000-01-02T03:04:05Z LEVEL=INFO MSG=message PRE=3 X=y A=one B=2",
		},
		{
			name:     "preformatted remove all",
			replace:  removeAll,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			want:     "",
		},
		{
			name:    "remove built-in",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   attrs,
			want:    "a=one b=2",
		},
		{
			name:    "preformatted remove built-in",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with:    func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			attrs:   attrs,
			want:    "pre=3 x=y a=one b=2",
		},
		{
			name:    "groups",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey), // to simplify the result
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("g",
					slog.Int("b", 2),
					slog.Group("h", slog.Int("c", 3)),
					slog.Int("d", 4),
				),
				slog.Int("e", 5),
			},
			want: "a=1 g.b=2 g.h.c=3 g.d=4 e=5",
		},
		{
			name:    "empty group",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Group("g"), slog.Group("h", slog.Int("a", 1))},
			want:    "h.a=1",
		},
		{
			name:    "escapes",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs: []slog.Attr{
				slog.String("a b", "x\t\n\000y"),
				slog.Group(" b.c=\"\\x2E\t",
					slog.String("d=e", "f.g\""),
					slog.Int("m.d", 1),
				), // dot is not escaped
			},
			want: `"a b"="x\t\n\x00y" " b.c=\"\\x2E\t.d=e"="f.g\"" " b.c=\"\\x2E\t.m.d"=1`,
		},
		{
			name:    "LogValuer",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Any("name", logValueName{"Ren", "Hoek"}),
				slog.Int("b", 2),
			},
			want: "a=1 name.first=Ren name.last=Hoek b=2",
		},
		{
			// Test resolution when there is no ReplaceAttr function.
			name: "resolve",
			attrs: []slog.Attr{
				slog.Any("", slog.Value{}), // should be elided
				slog.Any("name", logValueName{"Ren", "Hoek"}),
			},
			want: "2000-01-02T03:04:05Z INFO  message                                      name.first=Ren name.last=Hoek",
		},
		{
			name:    "with-group",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with:    func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs:   attrs,
			want:    "pre=3 x=y s.a=one s.b=2",
		},
		{
			name:    "preformatted with-groups",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]slog.Attr{slog.Int("p2", 2)}).
					WithGroup("s2").
					WithAttrs([]slog.Attr{slog.Int("p3", 3)})
			},
			attrs: attrs,
			want:  "p1=1 s1.p2=2 s1.s2.p3=3 s1.s2.a=one s1.s2.b=2",
		},
		{
			name:    "two with-groups",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs: attrs,
			want:  "p1=1 s1.s2.a=one s1.s2.b=2",
		},
		{
			name:    "GroupValue as Attr value",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{{Key: "v", Value: slog.AnyValue(slog.IntValue(3))}},
			want:    "v=3",
		},
		{
			name:    "byte slice",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Any("bs", []byte{1, 2, 3, 4})},
			want:    `bs="\x01\x02\x03\x04"`,
		},
		{
			name:    "json.RawMessage",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Any("bs", json.RawMessage([]byte("1234")))},
			want:    `bs=1234`,
		},
		{
			name:    "inline group",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("", slog.Int("b", 2), slog.Int("c", 3)),
				slog.Int("d", 4),
			},
			want: `a=1 b=2 c=3 d=4`,
		},
		{
			name: "Source",
			replace: func(gs []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					s := a.Value.Any().(*slog.Source)
					s.File = filepath.Base(s.File)
					return slog.Any(a.Key, s)
				}
				return logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey)(gs, a)
			},
			addSource: true,
			want:      `pretty_handler_test.go:$LINE`,
		},
		{
			name: "replace built-in with group",
			replace: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Group(slog.TimeKey, "mins", 3, "secs", 2)
				}
				if a.Key == slog.LevelKey || a.Key == slog.MessageKey {
					return slog.Attr{}
				}
				return a
			},
			want: `time.mins=3 time.secs=2`,
		},
		{
			name:    "custom byte slice type",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Any("bs", myByteSlice{1, 2, 3, 4})},
			want:    `bs="\x01\x02\x03\x04"`,
		},
		{
			name:    "channel",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Any("c", make(chan int))},
			want:    `c="chan int"`,
		},
		{
			name:    "function",
			replace: logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:   []slog.Attr{slog.Any("f", func() {})},
			want:    `f="func()"`,
		},
	} {
		r := slog.NewRecord(testTime, slog.LevelInfo, "message", logutil.CallerPC(1))
		src := logutil.CallerSource(r.PC)
		line := strconv.Itoa(src.Line)
		r.AddAttrs(test.attrs...)
		var buf bytes.Buffer
		t.Run(test.name, func(t *testing.T) {
			h := slog.Handler(logutil.NewPrettyHandler(&buf, &logutil.PrettyHandlerOptions{
				AddSource:    test.addSource,
				ReplaceAttr:  test.replace,
				DisableColor: true, // Disable colours to make want eaiser
			}))
			if test.with != nil {
				h = test.with(h)
			}
			if err := h.Handle(ctx, r); err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(test.want, "$LINE", line)
			got := strings.TrimSuffix(buf.String(), "\n")
			if got != want {
				t.Errorf("\ngot  %s\nwant %s\n", got, want)
			}
		})
	}
}

func upperCaseKey(_ []string, a slog.Attr) slog.Attr {
	a.Key = strings.ToUpper(a.Key)
	return a
}

type logValueName struct {
	first, last string
}

func (n logValueName) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("first", n.first),
		slog.String("last", n.last),
	)
}

type myByteSlice []byte

func TestHandlerEnabled(t *testing.T) {
	levelVar := func(l slog.Level) *slog.LevelVar {
		var al slog.LevelVar
		al.Set(l)
		return &al
	}

	for _, test := range []struct {
		leveler slog.Leveler
		want    bool
	}{
		{nil, true},
		{slog.LevelWarn, false},
		{&slog.LevelVar{}, true}, // defaults to Info
		{levelVar(slog.LevelWarn), false},
		{slog.LevelDebug, true},
		{levelVar(slog.LevelDebug), true},
	} {
		h := logutil.NewPrettyHandler(io.Discard, &logutil.PrettyHandlerOptions{Level: test.leveler})
		got := h.Enabled(context.Background(), slog.LevelInfo)
		if got != test.want {
			t.Errorf("%v: got %t, want %t", test.leveler, got, test.want)
		}
	}
}

func TestSecondWith(t *testing.T) {
	// Verify that a second call to Logger.With does not corrupt
	// the original.
	var buf bytes.Buffer
	h := logutil.NewPrettyHandler(&buf, &logutil.PrettyHandlerOptions{
		ReplaceAttr:  logutil.RemoveKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
		DisableColor: true,
	})
	logger := slog.New(h).With(
		slog.String("app", "playground"),
		slog.String("role", "tester"),
		slog.Int("data_version", 2),
	)
	appLogger := logger.With("type", "log") // this becomes type=met
	_ = logger.With("type", "metric")
	appLogger.Info("foo")
	got := strings.TrimSpace(buf.String())
	want := `app=playground role=tester data_version=2 type=log`
	if got != want {
		t.Errorf("\ngot  %s\nwant %s", got, want)
	}
}

func TestReplaceAttrGroups(t *testing.T) {
	// Verify that ReplaceAttr is called with the correct groups.
	type ga struct {
		groups string
		key    string
		val    string
	}

	var got []ga

	h := logutil.NewPrettyHandler(io.Discard, &logutil.PrettyHandlerOptions{ReplaceAttr: func(gs []string, a slog.Attr) slog.Attr {
		v := a.Value.String()
		if a.Key == slog.TimeKey {
			v = "<now>"
		}
		got = append(got, ga{strings.Join(gs, ","), a.Key, v})
		return a
	}})
	slog.New(h).
		With(slog.Int("a", 1)).
		WithGroup("g1").
		With(slog.Int("b", 2)).
		WithGroup("g2").
		With(
			slog.Int("c", 3),
			slog.Group("g3", slog.Int("d", 4)),
			slog.Int("e", 5),
		).
		Info("m",
			slog.Int("f", 6),
			slog.Group("g4", slog.Int("h", 7)),
			slog.Int("i", 8),
		)

	want := []ga{
		{"", "time", "<now>"},
		{"", "level", "INFO"},
		{"", "msg", "m"},
		{"", "a", "1"},
		{"g1", "b", "2"},
		{"g1,g2", "c", "3"},
		{"g1,g2,g3", "d", "4"},
		{"g1,g2", "e", "5"},
		{"g1,g2", "f", "6"},
		{"g1,g2,g4", "h", "7"},
		{"g1,g2", "i", "8"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("\ngot  %v\nwant %v", got, want)
	}
}
