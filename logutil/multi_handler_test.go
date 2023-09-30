package logutil_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cszatmary/goutils/logutil"
)

var testTime = time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)

func TestMultiHandlerEnabled(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  slog.Leveler
		testLevel slog.Level
		want      bool
	}{
		{
			"no min level present",
			nil,
			slog.LevelDebug,
			true,
		},
		{
			"min level present, enabled",
			slog.LevelInfo,
			slog.LevelWarn,
			true,
		},
		{
			"min level present, disabled",
			slog.LevelWarn,
			slog.LevelInfo,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := logutil.NewMultiHandler(nil, &logutil.MultiHandlerOptions{
				Level: tt.minLevel,
			})
			if got := h.Enabled(context.Background(), tt.testLevel); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiHandler(t *testing.T) {
	tests := []struct {
		name     string
		with     func(slog.Handler) slog.Handler
		opts     [2]slog.HandlerOptions
		wantText string
		wantJSON string
	}{
		{
			name:     "basic usage",
			wantText: `time=2000-01-02T03:04:05.000Z level=INFO msg="a message" foo=bar` + "\n",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"a message","foo":"bar"}` + "\n",
		},
		{
			name:     "different levels",
			opts:     [2]slog.HandlerOptions{{}, {Level: slog.LevelWarn}},
			wantText: `time=2000-01-02T03:04:05.000Z level=INFO msg="a message" foo=bar` + "\n",
			wantJSON: "", // Shouldn't be handled
		},
		{
			name:     "WithAttrs",
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs([]slog.Attr{slog.String("baz", "qux")}) },
			wantText: `time=2000-01-02T03:04:05.000Z level=INFO msg="a message" baz=qux foo=bar` + "\n",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"a message","baz":"qux","foo":"bar"}` + "\n",
		},
		{
			name:     "WithGroup",
			with:     func(h slog.Handler) slog.Handler { return h.WithGroup("group") },
			wantText: `time=2000-01-02T03:04:05.000Z level=INFO msg="a message" group.foo=bar` + "\n",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"INFO","msg":"a message","group":{"foo":"bar"}}` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := slog.NewRecord(testTime, slog.LevelInfo, "a message", 0)
			r.AddAttrs(slog.String("foo", "bar"))
			var b1, b2 bytes.Buffer
			h := slog.Handler(logutil.NewMultiHandler([]slog.Handler{
				slog.NewTextHandler(&b1, &tt.opts[0]),
				slog.NewJSONHandler(&b2, &tt.opts[1]),
			}, nil))
			if tt.with != nil {
				h = tt.with(h)
			}
			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			if gotText := b1.String(); gotText != tt.wantText {
				t.Errorf("got\n\t%q\nwant\n\t%q", gotText, tt.wantText)
			}
			if gotJSON := b2.String(); gotJSON != tt.wantJSON {
				t.Errorf("got\n\t%q\nwant\n\t%q", gotJSON, tt.wantJSON)
			}
		})
	}
}
