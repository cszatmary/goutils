package log

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/TouchBistro/goutils/color"
)

// Formatter is the interface for a type that can format logs.
//
// The Format method takes an Entry and is expected to return an
// array of bytes which is a serialized version of the log that
// can be written to the logger's output.
//
// The second argument to Format is a buffer provided for convenience.
// This can be used to accumulate the bytes as the log is being serialized.
type Formatter interface {
	Format(e *Entry, buf *bytes.Buffer) ([]byte, error)
}

// Key names for default fields
const (
	fieldMessage = "message"
	fieldLevel   = "level"
	fieldTime    = "time"
)

// replaceFieldClashes replaces any field names that would clash with the default fields.
// This is to prevent silently overwriting the default fields.
func replaceFieldClashes(fields Fields) {
	if v, ok := fields[fieldMessage]; ok {
		fields["fields."+fieldMessage] = v
		delete(fields, fieldMessage)
	}
	if v, ok := fields[fieldLevel]; ok {
		fields["fields."+fieldLevel] = v
		delete(fields, fieldLevel)
	}
	if v, ok := fields[fieldTime]; ok {
		fields["fields."+fieldTime] = v
		delete(fields, fieldTime)
	}
}

// TextFormatter formats logs into text.
type TextFormatter struct {
	// Pretty controls whether or not logs should be written in pretty format.
	// Pretty format causes the level and message to be written at the start without keys like so:
	//
	// DEBUG some log message foo=bar
	Pretty bool
	// ForceQuote forces quoting of all values. If disabled quoting will only be applied if required.
	ForceQuote bool
	// DisableTimestamp prevents the timestamp from being added to the log.
	DisableTimestamp bool
	// TimestampFormat os the format to use for the log timestamp.
	// This format is the same as the ones used for time.Format or time.Parse
	// from the standard library.
	TimestampFormat string
	// SortingFunc is the sorting function to use for the keys of the log fields.
	// If empty, sort.Strings will be used.
	SortingFunc func([]string)
}

// Format formats a log entry.
func (f *TextFormatter) Format(e *Entry, buf *bytes.Buffer) ([]byte, error) {
	// Create a copy of fields since we need to mutate it if we replace field clashes.
	fields := make(Fields, len(e.Fields))
	for k, v := range e.Fields {
		fields[k] = v
	}
	replaceFieldClashes(fields)

	// First figure out the keys and sort them. If not pretty we want to have
	// the default keys at the start and be excluded from sorting.
	var keys []string
	sortStart := 0
	if f.Pretty {
		keys = make([]string, 0, len(fields))
	} else {
		keys = make([]string, 0, len(fields)+3)
		keys = append(keys, fieldTime, fieldLevel, fieldMessage)
		sortStart = 3
	}
	for k := range fields {
		keys = append(keys, k)
	}

	sortFn := f.SortingFunc
	if sortFn == nil {
		sortFn = sort.Strings
	}
	sortFn(keys[sortStart:])

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339
	}

	if f.Pretty {
		// Assume info by default
		colorFn := color.Cyan
		switch e.Level {
		case LevelDebug:
			colorFn = color.White
		case LevelWarn:
			colorFn = color.Yellow
		case LevelError:
			colorFn = color.Red
		}

		// Pad level so that it is the same length for every line, i.e.
		// "INFO "
		// "DEBUG"
		// The max level length is always 5.
		levelText := fmt.Sprintf("%-5s", strings.ToUpper(e.Level.String()))
		// Remove a single newline if it already exists since a newline is added at the end of Format.
		e.Message = strings.TrimSuffix(e.Message, "\n")

		if f.DisableTimestamp {
			fmt.Fprintf(buf, "%s %-44s", colorFn(levelText), e.Message)
		} else {
			fmt.Fprintf(buf, "%s %s %-44s", e.Time.Format(timestampFormat), colorFn(levelText), e.Message)
		}
		for _, k := range keys {
			buf.WriteByte(' ')
			buf.WriteString(colorFn(k))
			buf.WriteByte('=')
			f.writeVal(buf, fields[k])
		}
	} else {
		for _, k := range keys {
			var v interface{}
			switch k {
			case fieldMessage:
				if e.Message == "" {
					continue
				}
				v = e.Message
			case fieldLevel:
				v = e.Level.String()
			case fieldTime:
				if f.DisableTimestamp {
					continue
				}
				v = e.Time.Format(timestampFormat)
			default:
				v = fields[k]
			}

			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(k)
			buf.WriteByte('=')
			f.writeVal(buf, v)
		}
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func (f *TextFormatter) writeVal(buf *bytes.Buffer, v interface{}) {
	s := stringify(v)
	if f.needsQuoting(s) {
		fmt.Fprintf(buf, "%q", s)
	} else {
		buf.WriteString(s)
	}
}

func stringify(v interface{}) string {
	if s, ok := v.(string); ok {
		// Already a string, easy
		return s
	}
	// Check to see if a we have a function or channel.
	// These can't be printed properly and fmt.Sprint will generate an
	// ugly pointer representation, ex: %!s(func()=0x10bf380)
	if t := reflect.TypeOf(v); t != nil {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Chan:
			return "chan"
		case reflect.Func:
			return "func()"
		}
	}
	return fmt.Sprint(v)
}

func (f *TextFormatter) needsQuoting(s string) bool {
	if f.ForceQuote {
		return true
	}
	for _, c := range s {
		// Needs to be quoted if it's not alphanumeric and not one of the special chars below.
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '/' || c == '@' || c == '^' || c == '+') {
			return true
		}
	}
	return false
}
