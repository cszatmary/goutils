// Package errors provides various error handling functionality.
//
// The Error type provides a way to create errors that contain details
// about the error and how it occurred. It is designed to produce both
// simple and clear errors for users as well as detailed errors for developers.
// It is recommended to provide an op to all errors to allowed building up a
// logic trace of where the error occurred for debugging purposes. This package
// provides several convenience functions for creating new errors and wrapping
// existing errors.
//
// The List type allows for keeping track of multiple errors that occurred so
// they can be reported together.
//
// This package also provides all functionality from the standard library errors
// package. As such, it can be used as a complete replacement for it.
// The String type can be used to create constant error values from strings.
//
// Both Error and List implement fmt.Formatter and can be formatted by the fmt package.
// Using the %+v verb will create a detailed description of the error that is suited for debugging.
//
// Note that this package is not a solution for all cases. There is no one size fits all for error
// handling, as errors will depend on the domain of the program and its requirements.
// This package is intended to facilitate building detailed error chains to provide context to
// both users and developers. It should be used in conjunction with other error handling strategies,
// not as a replacement for them.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// Error represents an error that occurred.
// It contains a number of fields that provide details about the error.
//
// When wrapping another Error it is recommended to use Wrap instead of initializing
// an Error directly to ensure a proper error chain is built.
type Error struct {
	// Kind is the category of error. Kind can be used to group errors
	// in order to better identify and action them.
	Kind Kind
	// Reason is a human-readable message containing
	// the details of the error.
	Reason string
	// Op is the operation being performed, usually the
	// name of a function or method being invoked.
	Op Op
	// Err is the underlying error that triggered this one.
	// If no underlying error occurred, it will be nil.
	Err error
}

// Kind represents any type that can categorize errors.
// It is recommended to categorize errors based on how they can be actioned.
//
// Kind requires a single method Kind() which returns a string that
// clearly describes the category of a given error.
//
// Kind should be implemented by types that are comparable, so that '=='
// can be used to check if two errors have the same kind.
type Kind interface {
	Kind() string
}

// Op describes an operation, usually a function or method name.
// It is recommended to have Op be of the form package.function
// or package.type.method to make it easy to identify the operation.
//
//   const op = errors.Op("foo.Bar")
type Op string

// New creates a new error using kind, reason and op.
func New(kind Kind, reason string, op Op) error {
	return newError(kind, reason, op, nil)
}

// Wrap wraps an existing error. It can be used to provide additional context
// to an error and create detailed error chains.
//
// If err is an Error, Wrap will create a copy of it and perform modifications
// to make error chains nicer. If meta.Kind is nil, it will be hoisted from err.
// If meta.Kind == err.Kind, err.Kind will be set to nil, to prevent duplicate kinds.
func Wrap(err error, meta Meta) error {
	return newError(meta.Kind, meta.Reason, meta.Op, err)
}

// Meta allows for specifying the fields for a wrapped error provided to Wrap.
type Meta struct {
	// Kind is the category of error. See Error.Kind
	Kind Kind
	// Reason is the reason for the error. See Error.Reason.
	Reason string
	// Op is the operation being performed. See Error.Op.
	Op Op
}

func newError(kind Kind, reason string, op Op, err error) error {
	e := &Error{Kind: kind, Reason: reason, Op: op}
	if err == nil {
		return e
	}
	prev, ok := err.(*Error)
	if !ok {
		e.Err = err
		return e
	}

	// Make a copy so error chains are immutable.
	copy := *prev
	prev = &copy
	// If the previous error has the same kind, remove it to prevent duplicates
	// in the error string.
	if prev.Kind == e.Kind {
		prev.Kind = nil
	}
	// If this error has no kind, grab it from the inner one.
	if e.Kind == nil {
		e.Kind = prev.Kind
		prev.Kind = nil
	}
	e.Err = prev
	return e
}

func (e *Error) Error() string {
	sb := &strings.Builder{}
	if e.Kind != nil {
		pad(sb, ": ")
		sb.WriteString(e.Kind.Kind())
	}
	if e.Reason != "" {
		pad(sb, ": ")
		sb.WriteString(e.Reason)
	}
	if e.Err != nil {
		pad(sb, ": ")
		sb.WriteString(e.Err.Error())
	}
	return sb.String()
}

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		// If '%+v' print a detailed description for debugging purposes.
		if s.Flag('+') {
			sb := &strings.Builder{}
			if e.Op != "" {
				pad(sb, ": ")
				sb.WriteString(string(e.Op))
			}
			if e.Kind != nil {
				pad(sb, ": ")
				sb.WriteString(e.Kind.Kind())
			}
			if e.Reason != "" {
				pad(sb, ": ")
				sb.WriteString(e.Reason)
			}
			if e.Err != nil {
				if prevErr, ok := e.Err.(*Error); ok {
					pad(sb, ":\n\t")
					fmt.Fprintf(sb, "%+v", prevErr)
				} else {
					pad(sb, ": ")
					sb.WriteString(e.Err.Error())
				}
			}
			fmt.Fprint(s, sb.String())
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// pad appends s to sb if b already has some data.
func pad(sb *strings.Builder, s string) {
	if sb.Len() == 0 {
		return
	}
	sb.WriteString(s)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// List is a list of errors. It allows for operations to keep track of
// multiple errors and return them as a single error value.
type List []error

func (e List) Error() string {
	var sb strings.Builder
	for i, err := range e {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

func (e List) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		// If '%+v' print a detailed description of each error.
		if s.Flag('+') {
			var sb strings.Builder
			for i, err := range e {
				if i > 0 {
					sb.WriteByte('\n')
				}
				fmt.Fprintf(&sb, "%+v", err)
			}
			fmt.Fprint(s, sb.String())
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// The following is all functionality provided by the standard library errors package.
// This is so that this package can be used as a full replacement.

// String is a simple error based on a string.
//
// It provides similar functionality to the errors.New function from the standard library.
// However, unlike with errors.New, String allows defining constant error values.
// This can be useful for creating sentinel errors.
//
//   const EOF errors.String = "end of file"
type String string

func (e String) Error() string {
	return string(e)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
//
// An error type might provide an Is method so it can be treated as equivalent
// to an existing error. For example, if MyError defines
//
//	func (m MyError) Is(target error) bool { return target == fs.ErrExist }
//
// then Is(MyError{}, fs.ErrExist) returns true. See syscall.Errno.Is for
// an example in the standard library.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// An error type might provide an As method so it can be treated as if it were a
// different error type.
//
// As panics if target is not a non-nil pointer to either a type that implements
// error, or to any interface type.
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}
