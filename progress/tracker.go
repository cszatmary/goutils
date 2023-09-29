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
	"context"
)

// Logger represents a structured logger that can log messages at different levels.
//
// A logger should support the log levels of debug, info, warn, and error.
// These are implemented through the corresponding methods.
//
// The WithAttrs method is used to create structured logs. It must return
// another Logger that will contain the given attributes when a creating logs.
// The arguments to WithAttrs are expected to be a set of key-pair values representing attributes.
//
//	logger.WithAttrs("id", id).Info(...)
type Logger interface {
	WithAttrs(args ...any) Logger

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)

	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Spinner represents a type that can display the progress of an operation
// using an animation along with a message.
//
// The Inc and UpdateMessage methods must be safe to call across multiple goroutines.
type Spinner interface {
	Start(msg string, count int)
	Stop()
	Inc()
	UpdateMessage(msg string)
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

func (t NoopTracker) WithAttrs(...any) Logger { return t }
func (NoopTracker) Debugf(string, ...any)     {}
func (NoopTracker) Infof(string, ...any)      {}
func (NoopTracker) Warnf(string, ...any)      {}
func (NoopTracker) Errorf(string, ...any)     {}
func (NoopTracker) Debug(string, ...any)      {}
func (NoopTracker) Info(string, ...any)       {}
func (NoopTracker) Warn(string, ...any)       {}
func (NoopTracker) Error(string, ...any)      {}
func (NoopTracker) Start(string, int)         {}
func (NoopTracker) Stop()                     {}
func (NoopTracker) Inc()                      {}
func (NoopTracker) UpdateMessage(string)      {}
