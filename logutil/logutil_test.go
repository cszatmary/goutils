package logutil_test

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cszatmary/goutils/logutil"
)

func TestWriterVar(t *testing.T) {
	var wv logutil.WriterVar
	data := []byte("hello")
	// Check that a zero value works
	if gotN, gotErr := wv.Write([]byte("hello")); gotN != len(data) || gotErr != nil {
		t.Errorf("got %d, %v; want %d, nil", gotN, gotErr, len(data))
	}
	var b bytes.Buffer
	wv.Set(&b)
	if gotN, gotErr := wv.Write([]byte("hello")); gotN != len(data) || gotErr != nil {
		t.Errorf("got %d, %v; want %d, nil", gotN, gotErr, len(data))
	}
	if got := b.String(); got != "hello" {
		t.Errorf("got %q; want %q", got, "hello")
	}
}

func TestLogWriter(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  string
	}{
		{
			"debug",
			slog.LevelDebug,
			`level=DEBUG msg="hello world" id=foo
level=DEBUG msg="adding some more stuff" id=foo
`,
		},
		{
			"info",
			slog.LevelInfo,
			`level=INFO msg="hello world" id=foo
level=INFO msg="adding some more stuff" id=foo
`,
		},
		{
			"warn",
			slog.LevelWarn,
			`level=WARN msg="hello world" id=foo
level=WARN msg="adding some more stuff" id=foo
`,
		},
		{
			"error",
			slog.LevelError,
			`level=ERROR msg="hello world" id=foo
level=ERROR msg="adding some more stuff" id=foo
`,
		},
		{
			"custom level",
			slog.Level(2),
			`level=INFO+2 msg="hello world" id=foo
level=INFO+2 msg="adding some more stuff" id=foo
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			logger := logutil.NewFormatLogger(slog.NewTextHandler(&b, &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: logutil.RemoveKeys(slog.TimeKey),
			}))
			w := logutil.LogWriter(logger.With("id", "foo"), tt.level)
			t.Cleanup(func() {
				w.Close()
			})

			if _, err := io.WriteString(w, "hello world\n"); err != nil {
				t.Fatalf("failed to write to log writer: %v", err)
			}
			if _, err := io.WriteString(w, "adding some more stuff\n"); err != nil {
				t.Fatalf("failed to write to log writer: %v", err)
			}

			// Sleep to make sure the logs have time to be written since it is running
			// on a separate goroutine
			time.Sleep(100 * time.Millisecond)
			if got := b.String(); got != tt.want {
				t.Errorf("\ngot logs\n\t%s\nwant\n\t%s", got, tt.want)
			}
		})
	}
}
