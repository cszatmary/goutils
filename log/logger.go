package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/TouchBistro/goutils/progress"
)

// Fields is a convenience type that is equivalent to progress.Fields.
// It is provided here so that callers do not need to import progress
// as well if they are not using it.
type Fields = progress.Fields

// Hook is the interface for a log hook. Hooks are run before a log
// is written and allow for customizing the behaviour of the logger.
//
// Hooks are executed in the same goroutine as the Logger and do not
// provide synchronization. These features must be implemented by
// the hook if required.
type Hook interface {
	Run(e *Entry) error
}

// Logger is a structured logger that supports logging at different levels.
type Logger struct {
	// mu is a mutex used to ensure the logger is safe to use across goroutines.
	// It protects all subsequent fields.
	mu         sync.RWMutex
	out        io.Writer // where logs are written
	formatter  Formatter
	lvl        Level // min level, all levels greater than or equal to are logged
	hooks      []Hook
	errHandler func(error)
	buf        bytes.Buffer // used for formatting logs; cached for reuse
}

// New creates a new logger. The logger can be configured by providing one or more options.
//
// By default the logger will write to stderr with a TextFormatter at info level.
func New(opts ...Option) *Logger {
	l := &Logger{
		out:       os.Stderr,
		lvl:       LevelInfo,
		formatter: &TextFormatter{},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Option is a function that takes a logger and applies a configuration to it.
type Option func(*Logger)

// WithOuput sets the io.Writer that should be used for outputting logs.
func WithOutput(out io.Writer) Option {
	return func(l *Logger) {
		l.out = out
	}
}

// WithFormatter sets the Formatter that should be used to format logs.
func WithFormatter(f Formatter) Option {
	return func(l *Logger) {
		l.formatter = f
	}
}

// WithLevel sets the minimum level that the logger will log at.
// Anything less than this level will be ignored.
func WithLevel(lvl Level) Option {
	return func(l *Logger) {
		l.lvl = lvl
	}
}

// A pool of reusable entries to save on allocations.
var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{}
	},
}

// newEntry allocates a new Entry or grabs a cached one.
// The returned entry is bound to the Logger l.
func newEntry(l *Logger) *Entry {
	e := entryPool.Get().(*Entry)
	e.Logger = l
	return e
}

// freeEntry puts e back in entryPool to allow it to be reused.
func freeEntry(e *Entry) {
	// Clear fields so it's ready for reuse.
	e.Logger = nil
	e.Fields = nil
	entryPool.Put(e)
}

// Output returns the output where logs are written.
func (l *Logger) Output() io.Writer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.out
}

// SetOutput sets the output where logs should be written.
func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = out
}

// Formatter returns the formatter used to format logs.
func (l *Logger) Formatter() Formatter {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.formatter
}

// SetFormatter sets the formatter that should be used to format logs.
func (l *Logger) SetFormatter(f Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = f
}

// Level returns the logger level.
func (l *Logger) Level() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lvl
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(lvl Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lvl = lvl
}

// AddHook adds one or more hooks to the logger that will be executed for each log.
func (l *Logger) AddHook(hooks ...Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hooks...)
}

// SetErrorHandler sets the error handler that should be used by the Logger.
// h will be called any time an error occurs while logging.
func (l *Logger) SetErrorHandler(h func(error)) {
	l.errHandler = h
}

// WithFields adds a set of fields to the log.
func (l *Logger) WithFields(fields Fields) progress.Logger {
	e := newEntry(l)
	defer freeEntry(e)
	return e.WithFields(fields)
}

// Log writes a log at the given level.
func (l *Logger) Log(lvl Level, args ...interface{}) {
	if lvl >= l.Level() {
		e := newEntry(l)
		l.log(e, lvl, fmt.Sprint(args...))
		freeEntry(e)
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.Log(LevelDebug, args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.Log(LevelInfo, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.Log(LevelWarn, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.Log(LevelError, args...)
}

// Log writes a log at the given level. Supports printf-like formatting.
func (l *Logger) Logf(lvl Level, format string, args ...interface{}) {
	if lvl >= l.Level() {
		e := newEntry(l)
		l.log(e, lvl, fmt.Sprintf(format, args...))
		freeEntry(e)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logf(LevelDebug, format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logf(LevelInfo, format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logf(LevelWarn, format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(LevelError, format, args...)
}

func (l *Logger) log(e *Entry, lvl Level, msg string) {
	// Create a copy since we set fields on it but we don't want to
	// mutate the current entry since it might be reused for logs.
	entryCopy := e.Copy()
	if entryCopy.Time.IsZero() {
		entryCopy.Time = time.Now()
	}
	entryCopy.Level = lvl
	entryCopy.Message = msg

	// Run all the hooks, this needs to happen before the log is serialized and written.
	for _, h := range l.hooks {
		if err := h.Run(entryCopy); err != nil {
			l.handleErr(err, "Failed to run hook")
		}
	}

	// Serialize the log, then write it out.
	// Need to make sure this whole process is protected with the lock.
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf.Reset()
	serialized, err := l.formatter.Format(entryCopy, &l.buf)
	if err != nil {
		l.handleErr(err, "Failed to format log")
		return
	}
	if _, err := l.out.Write(serialized); err != nil {
		l.handleErr(err, "Failed to write log")
	}
}

func (l *Logger) handleErr(err error, msg string) {
	if l.errHandler != nil {
		l.errHandler(err)
		return
	}
	// If no error handler provided all we can really do is write to stderr
	// to alert that something went wrong.
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
}
