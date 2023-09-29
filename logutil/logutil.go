// Package logutil provides various logging utilities that are meant to expand the capabilities of [log/slog].
//
// [PrettyHandler] is a [slog.Handler] that outputs logs in a text format similar to [slog.TextHandler] but
// with pretty formatting and colours. It is intended for use in CLIs to make easy to read logs for users.
//
// [MultiHandler] is a [slog.Handler] that allows a single log record to be processed by multiple handlers.
// It is akin to [io.MultiWriter]. Each handler has the ability to customize its behaviour.
package logutil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"

	"github.com/TouchBistro/goutils/progress"
)

// WriterVar is a io.Writer variable, to allow a Handler writer to change dynamically.
// It implements io.Writer as well as a Set method, and is safe for use by multiple goroutines.
//
// A WriterVar must not be copied after first use.
//
// The zero value LevelVar is a no-op writer that discards all data written to it (similar to io.Discard).
type WriterVar struct {
	w  io.Writer
	mu sync.Mutex
}

// NewWriterVar creates a new WriterVar with the given writer.
func NewWriterVar(w io.Writer) *WriterVar {
	return &WriterVar{w: w}
}

// Set sets the underlying writer.
func (v *WriterVar) Set(w io.Writer) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.w = w
}

func (v *WriterVar) Write(p []byte) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	// Handle zero value case, if no writer just no-op.
	if v.w == nil {
		return len(p), nil
	}
	return v.w.Write(p)
}

// LogWriter returns an io.Writer that can be used to write arbitrary text to the logger.
// logger is used to log an error if one occurs.
//
// It is the caller's responsibility to close the returned io.WriteCloser in order
// to free resources.
func LogWriter(logger progress.Logger, level slog.Level) io.WriteCloser {
	pr, pw := io.Pipe()
	var logFunc func(string, ...any)
	switch level {
	case slog.LevelDebug:
		logFunc = logger.Debug
	case slog.LevelInfo:
		logFunc = logger.Info
	case slog.LevelWarn:
		logFunc = logger.Warn
	case slog.LevelError:
		logFunc = logger.Error
	default:
		// See if the logger has a Log method that can be passed a level.
		type withlog interface {
			Log(context.Context, slog.Level, string, ...any)
		}
		wl, ok := logger.(withlog)
		if !ok {
			panic(fmt.Errorf("logutil.LogWriter: unsupported level %s(%d)", level.String(), level))
		}
		logFunc = func(s string, a ...any) {
			wl.Log(context.Background(), level, s, a...)
		}
	}
	go logText(logger, pr, logFunc)
	runtime.SetFinalizer(pw, (*io.PipeWriter).Close)
	return pw
}

func logText(logger progress.Logger, pr *io.PipeReader, logFunc func(string, ...any)) {
	s := bufio.NewScanner(pr)
	for s.Scan() {
		logFunc(s.Text())
	}
	if err := s.Err(); err != nil {
		logger.Error("Error while reading from Writer", "err", err)
	}
	pr.Close()
}

// CallerPC returns the program counter at the given stack depth.
func CallerPC(depth int) uintptr {
	var pcs [1]uintptr
	// Need to add +1 to depth in order to skip this function.
	runtime.Callers(depth+1, pcs[:])
	return pcs[0]
}

// CallerSource returns a slog.Source for the given program counter.
// If the location is unavailable, it returns a slog.Source with zero fields.
func CallerSource(pc uintptr) slog.Source {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

// RemoveKeys returns a function suitable for HandlerOptions.ReplaceAttr
// that removes all Attrs with the given keys.
func RemoveKeys(keys ...string) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		for _, k := range keys {
			if a.Key == k {
				return slog.Attr{}
			}
		}
		return a
	}
}
