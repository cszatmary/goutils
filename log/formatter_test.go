package log_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/log"
)

func TestTextFormatter(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		entry     *log.Entry
		formatter *log.TextFormatter
		want      string
	}{
		{
			name: "default",
			entry: &log.Entry{
				Fields:  log.Fields{"foo": "bar", "quote": "for sure"},
				Time:    now,
				Level:   log.LevelInfo,
				Message: "some msg",
			},
			formatter: &log.TextFormatter{},
			want:      `time="` + now.Format(time.RFC3339) + `" level=info message="some msg" foo=bar quote="for sure"` + "\n",
		},
		{
			name: "pretty",
			entry: &log.Entry{
				Fields:  log.Fields{"foo": "bar"},
				Time:    now,
				Level:   log.LevelInfo,
				Message: "some msg",
			},
			formatter: &log.TextFormatter{Pretty: true},
			// This sucks, but it was the best way I could think of to test pretty without just reimplementing the logic in the test.
			want: now.Format(time.RFC3339) + " " + color.Cyan("INFO ") + " some msg                                     " + color.Cyan("foo") + "=bar\n",
		},
		{
			name: "handles field clashes",
			entry: &log.Entry{
				Fields:  log.Fields{"level": "first", "time": "morning", "message": "hello there"},
				Time:    now,
				Level:   log.LevelWarn,
				Message: "some msg",
			},
			formatter: &log.TextFormatter{},
			want:      `time="` + now.Format(time.RFC3339) + `" level=warn message="some msg" fields.level=first fields.message="hello there" fields.time=morning` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.formatter.Format(tt.entry, &bytes.Buffer{})
			if err != nil {
				t.Fatalf("want no error, got: %v", err)
			}
			if string(b) != tt.want {
				t.Fatalf("\ngot formatted log: %s\nwant: %s", b, tt.want)
			}
		})
	}
}
