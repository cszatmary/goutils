// Package log provides a structured logger for creating logs.
// The Logger type is fully compatible with the progress.Logger interface.
package log

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidLevel is returned if a given level is invalid.
var ErrInvalidLevel = errors.New("invalid level")

// Level represents a log level.
type Level int8

const (
	// levelInvalid is a special pseudo-level that is used when the level cannot
	// be determined. It is not exported since it should not be used directly.
	// Any operations that return levelInvalid should also return ErrInvalidLevel for
	// callers to check instead of needing to check for levelInvalid.
	levelInvalid Level = iota - 1
	// LevelDebug is for low level details about what an application is doing.
	// Used to create verbose logs.
	LevelDebug
	// LeveInfo is for general information about what an application is doing.
	LevelInfo
	// LevelWarn is for non-critical errors that should still be
	// noted and investigated.
	LevelWarn
	// LevelError is for critical errors that require attention.
	LevelError
)

// ParseLevel parses a string into a log level.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	}
	return levelInvalid, fmt.Errorf("%w: %q", ErrInvalidLevel, s)
}

// String returns the string name of the level.
func (lvl Level) String() string {
	switch lvl {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		// Just to be safe
		return "invalid"
	}
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (lvl *Level) UnmarshalText(text []byte) error {
	l, err := ParseLevel(string(text))
	if err != nil {
		return err
	}
	*lvl = l
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (lvl Level) MarshalText() ([]byte, error) {
	switch lvl {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return []byte(lvl.String()), nil
	}
	return nil, fmt.Errorf("%w: %d", ErrInvalidLevel, lvl)
}
