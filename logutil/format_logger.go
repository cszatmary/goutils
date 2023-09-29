package logutil

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/TouchBistro/goutils/progress"
)

// FormatLogger wraps a slog.Logger and gives it Printf-like functions for each log level.
// It also conforms to the progess.Logger interface.
type FormatLogger struct {
	*slog.Logger
}

// NewFormatLogger is a convenience function to create a new FormatLogger using a handler.
func NewFormatLogger(h slog.Handler) *FormatLogger {
	return &FormatLogger{slog.New(h)}
}

func (l *FormatLogger) WithAttrs(args ...any) progress.Logger {
	return l.With(args...)
}

func (l *FormatLogger) With(args ...any) *FormatLogger {
	if len(args) == 0 {
		return l
	}
	return &FormatLogger{l.Logger.With(args...)}
}

func (l *FormatLogger) WithGroup(name string) *FormatLogger {
	if name == "" {
		return l
	}
	return &FormatLogger{l.Logger.WithGroup(name)}
}

func (l *FormatLogger) Debugf(format string, args ...any) {
	l.logf(slog.LevelDebug, format, args...)
}

func (l *FormatLogger) Infof(format string, args ...any) {
	l.logf(slog.LevelInfo, format, args...)
}

func (l *FormatLogger) Warnf(format string, args ...any) {
	l.logf(slog.LevelWarn, format, args...)
}

func (l *FormatLogger) Errorf(format string, args ...any) {
	l.logf(slog.LevelError, format, args...)
}

func (l *FormatLogger) logf(level slog.Level, format string, args ...any) {
	ctx := context.Background()
	if !l.Logger.Enabled(ctx, level) {
		return
	}
	// Calculate source, skip [CallerPC, this function, this function's caller]
	pc := CallerPC(3)
	r := slog.NewRecord(time.Now(), level, fmt.Sprintf(format, args...), pc)
	_ = l.Logger.Handler().Handle(ctx, r)
}
