package logutil_test

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/TouchBistro/goutils/logutil"
)

func TestFormatLogger(t *testing.T) {
	tests := []struct {
		name string
		do   func(*logutil.FormatLogger)
		want string
	}{
		{
			name: "Debugf",
			do: func(l *logutil.FormatLogger) {
				l.Debugf("hello %s %d", "foo", 20)
			},
			want: `level=DEBUG msg="hello foo 20"` + "\n",
		},
		{
			name: "Infof",
			do: func(l *logutil.FormatLogger) {
				l.Infof("hello %s %d", "foo", 20)
			},
			want: `level=INFO msg="hello foo 20"` + "\n",
		},
		{
			name: "Warnf",
			do: func(l *logutil.FormatLogger) {
				l.Warnf("hello %s %d", "foo", 20)
			},
			want: `level=WARN msg="hello foo 20"` + "\n",
		},
		{
			name: "Errorf",
			do: func(l *logutil.FormatLogger) {
				l.Errorf("hello %s %d", "foo", 20)
			},
			want: `level=ERROR msg="hello foo 20"` + "\n",
		},
		{
			name: "WithAttrs-Infof",
			do: func(l *logutil.FormatLogger) {
				l.WithAttrs("bar", "baz").Infof("hello %s %d", "foo", 20)
			},
			want: `level=INFO msg="hello foo 20" bar=baz` + "\n",
		},
		{
			name: "WithGroup-Infof",
			do: func(l *logutil.FormatLogger) {
				l.WithGroup("std").With("bar", "baz").Infof("hello %s %d", "foo", 20)
			},
			want: `level=INFO msg="hello foo 20" std.bar=baz` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			logger := logutil.NewFormatLogger(slog.NewTextHandler(&b, &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: logutil.RemoveKeys(slog.TimeKey),
			}))
			tt.do(logger)
			src := logutil.CallerSource(logutil.CallerPC(2))
			want := strings.ReplaceAll(tt.want, "$LINE", strconv.Itoa(src.Line))
			if got := b.String(); got != want {
				t.Errorf("\ngot\n\t%s\nwant\n\t%s", got, want)
			}
		})
	}
}

func TestFormatLoggerSource(t *testing.T) {
	var b bytes.Buffer
	logger := logutil.NewFormatLogger(slog.NewTextHandler(&b, &slog.HandlerOptions{
		AddSource: true,
		ReplaceAttr: func(gs []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				s := a.Value.Any().(*slog.Source)
				s.File = filepath.Base(s.File)
				return slog.Any(a.Key, s)
			}
			return logutil.RemoveKeys(slog.TimeKey, slog.LevelKey)(gs, a)
		},
	}))
	logger.Infof("hello %s %d", "foo", 20)
	src := logutil.CallerSource(logutil.CallerPC(1))
	want := `source=format_logger_test.go:$LINE msg="hello foo 20"` + "\n"
	want = strings.ReplaceAll(want, "$LINE", strconv.Itoa(src.Line-1))
	if got := b.String(); got != want {
		t.Errorf("\ngot\n\t%s\nwant\n\t%s", got, want)
	}
}

func TestFormatLoggerLevelDisabled(t *testing.T) {
	var b bytes.Buffer
	logger := logutil.NewFormatLogger(slog.NewTextHandler(&b, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	logger.Infof("hello %s %d", "foo", 20)
	if got := b.String(); got != "" {
		t.Errorf("\ngot\n\t%s\nwant empty string", got)
	}
}
