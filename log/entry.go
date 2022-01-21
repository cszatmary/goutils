package log

import (
	"fmt"
	"time"

	"github.com/TouchBistro/goutils/progress"
)

// Entry is an intermediate representation of a log entry.
// It contains all the fields passed with WithFields.
// It is recommended to reuse and share Entry objects in order to reuse fields.
type Entry struct {
	Logger *Logger
	// Fields is all the fields set by the user.
	Fields Fields
	// Time is the time at which the entry was logged.
	// It is set when the log is finalized.
	Time time.Time
	// Level is the level the entry was logged at.
	Level Level
	// Message is the log message.
	Message string
}

// Copy creates a copy of entry with all fields duplicated.
func (e *Entry) Copy() *Entry {
	fields := make(Fields, len(e.Fields))
	for k, v := range e.Fields {
		fields[k] = v
	}
	return &Entry{Logger: e.Logger, Fields: fields, Time: e.Time}
}

// WithFields adds a set of fields to the log.
// It returns a new entry and does not mutate the existing entry.
func (e *Entry) WithFields(fields Fields) progress.Logger {
	// Allocate space for all existing and new fields
	mergedFields := make(Fields, len(e.Fields)+len(fields))
	for k, v := range e.Fields {
		fields[k] = v
	}
	for k, v := range fields {
		mergedFields[k] = v
	}
	return &Entry{Logger: e.Logger, Fields: mergedFields, Time: e.Time}
}

// Log writes a log at the given level.
func (e *Entry) Log(lvl Level, args ...interface{}) {
	if lvl >= e.Logger.Level() {
		e.Logger.log(e, lvl, fmt.Sprint(args...))
	}
}

func (e *Entry) Debug(args ...interface{}) {
	e.Log(LevelDebug, args...)
}

func (e *Entry) Info(args ...interface{}) {
	e.Log(LevelInfo, args...)
}

func (e *Entry) Warn(args ...interface{}) {
	e.Log(LevelWarn, args...)
}

func (e *Entry) Error(args ...interface{}) {
	e.Log(LevelError, args...)
}

// Log writes a log at the given level. Supports printf-like formatting.
func (e *Entry) Logf(lvl Level, format string, args ...interface{}) {
	if lvl >= e.Logger.Level() {
		e.Logger.log(e, lvl, fmt.Sprintf(format, args...))
	}
}

func (e *Entry) Debugf(format string, args ...interface{}) {
	e.Logf(LevelDebug, format, args...)
}

func (e *Entry) Infof(format string, args ...interface{}) {
	e.Logf(LevelInfo, format, args...)
}

func (e *Entry) Warnf(format string, args ...interface{}) {
	e.Logf(LevelWarn, format, args...)
}

func (e *Entry) Errorf(format string, args ...interface{}) {
	e.Logf(LevelError, format, args...)
}
