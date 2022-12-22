// Package progress provides support for displaying the progress of one or
// more operations. It also provides logging capabilities.
//
// The core part of this package is the Tracker interface which is a combination of
// the Logger and Spinner interfaces. A Tracker allows for display progress and logging
// messages while one or more operations are being performed. Some convenience Tracker types
// are provided to make it easier to create Trackers. This package does not provide a
// Logger or Spinner implementation directly. Instead types implementing these interfaces
// can be provided by other packages and composed as necessary.
//
// This package also provides the Run and RunParallel functions with allow running a single
// operation or multiple operations respectively while displaying progress and handling
// errors, cancellation, and timeouts.
package progress

import (
	"bufio"
	"context"
	"io"
	"runtime"
)

// Fields is a collection of fields provided to Logger.WithFields.
type Fields map[string]interface{}

// Logger represents a structured logger that can log messages at different levels.
//
// A logger should support the log levels of debug, info, warn, and error.
// These are implemented through the corresponding methods.
//
// The WithFields method is used to create structured logs. It must return
// another Logger that will contain the given fields when a creating logs.
type Logger interface {
	WithFields(fields Fields) Logger

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

// OutputLogger is a Logger that allows accessing and updating the underlying
// io.Writer that logs are written to.
type OutputLogger interface {
	Logger
	Output() io.Writer
	SetOutput(w io.Writer)
}

// Spinner represents a type that can display the progress of an operation
// using an animation along with a message.
//
// The Inc and UpdateMessage methods must be safe to call across multiple goroutines.
type Spinner interface {
	Start(message string, count int)
	Stop()
	Inc()
	UpdateMessage(m string)
}

// Tracker combines the Logger and Spinner interfaces.
// It provides the necessary functionality for tracking the progress of operations
// by displaying a spinner animation, as well as providing log messages.
// A Tracker should allow logging messages while the spinner animation is running.
type Tracker interface {
	Logger
	Spinner
}

// Custom type so that context key is globally unique.
// As a bonus use empty struct so the key takes up no memory.
type trackerKey struct{}

// ContextWithTracker returns a new context with t added to it.
// The tracker can be retrieved later using TrackerFromContext.
func ContextWithTracker(ctx context.Context, t Tracker) context.Context {
	return ContextWithTrackerUsingKey(ctx, t, nil)
}

// ContextWithTrackerUsingKey is like ContextWithTracker but allows for using a custom key.
// This can be useful if you want to avoid using the default key to prevent clashes.
// The tracker can be retrieved later using TrackerFromContextUsingKey.
func ContextWithTrackerUsingKey(ctx context.Context, t Tracker, key any) context.Context {
	if key == nil {
		key = trackerKey{}
	}
	return context.WithValue(ctx, key, t)
}

// TrackerFromContext returns the Tracker from ctx.
//
// If no Tracker exists in ctx, a no-op Tracker will be returned.
// Thus, the returned Tracker will never be nil, and it is always safe to call methods on it.
func TrackerFromContext(ctx context.Context) Tracker {
	return TrackerFromContextUsingKey(ctx, nil)
}

// TrackerFromContextUsingKey is like TrackerFromContext but allows for using a custom key.
// It should be used if ContextWithTrackerUsingKey was used to create a context with a custom key.
//
// If a value exists in the context for the given key but is not a Tracker, the function will panic.
func TrackerFromContextUsingKey(ctx context.Context, key any) Tracker {
	if key == nil {
		key = trackerKey{}
	}
	v := ctx.Value(key)
	if v == nil {
		return NoopTracker{}
	}
	t, ok := v.(Tracker)
	if !ok {
		// If the value is not a Tracker this is an invariant violation and it should explode loudly.
		panic("impossible: progress.TrackerFromContextUsingKey: value is not of type Tracker")
	}
	return t
}

// NoopTracker is a Tracker that no-ops on every method.
type NoopTracker struct{}

func (t NoopTracker) WithFields(fields Fields) Logger         { return t }
func (NoopTracker) Debugf(format string, args ...interface{}) {}
func (NoopTracker) Infof(format string, args ...interface{})  {}
func (NoopTracker) Warnf(format string, args ...interface{})  {}
func (NoopTracker) Errorf(format string, args ...interface{}) {}
func (NoopTracker) Debug(args ...interface{})                 {}
func (NoopTracker) Info(args ...interface{})                  {}
func (NoopTracker) Warn(args ...interface{})                  {}
func (NoopTracker) Error(args ...interface{})                 {}
func (NoopTracker) Start(message string, count int)           {}
func (NoopTracker) Stop()                                     {}
func (NoopTracker) Inc()                                      {}
func (NoopTracker) UpdateMessage(m string)                    {}

// PlainTracker is a tracker that does not display a spinner.
// It is effectively a no-op Spinner that wraps a Logger.
type PlainTracker struct {
	Logger
}

func (t *PlainTracker) Start(message string, count int) {
	l := t.Logger
	if count > 1 {
		l = l.WithFields(Fields{"count": count})
	}
	l.Info(message)
}

func (*PlainTracker) Stop() {}
func (*PlainTracker) Inc()  {}

func (t *PlainTracker) UpdateMessage(m string) {
	t.Logger.Info(m)
}

// LogWriter returns an io.Writer that can be used to write arbitrary text to the logger.
// logFn should be a logging method such as Logger.Info. logger is used to log an error
// if one occurs.
//
// It is the caller's responsibility to close the returned io.WriteCloser in order
// to free resources.
func LogWriter(logger Logger, logFn func(args ...interface{})) io.WriteCloser {
	pr, pw := io.Pipe()
	go logText(logger, pr, logFn)
	runtime.SetFinalizer(pw, (*io.PipeWriter).Close)
	return pw
}

func logText(logger Logger, pr *io.PipeReader, logFn func(args ...interface{})) {
	s := bufio.NewScanner(pr)
	for s.Scan() {
		logFn(s.Text())
	}
	if err := s.Err(); err != nil {
		logger.Errorf("Error while reading from Writer: %v", err)
	}
	pr.Close()
}
