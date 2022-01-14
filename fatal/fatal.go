// Package fatal provides functionality for dealing with fatal errors
// and handling program termination.
//
// A fatal error is an error from which the program has no reasonable way to
// recover from and therefore the program cannot continue running.
//
// It is important to note that a fatal error is distinct from a panic
// and fatal should be used in different situations. A panic should occur
// when something unexpected happens and the program cannot recover.
// Examples of this are programming errors such as the program ending up in an
// impossible state, or a runtime error such as the program running out of memory.
// In these cases the program should panic in order to abort as quickly and loudly
// as possible to alert users of the issue.
//
// A fatal error on the other hand is an error that can reasonably occur in a program
// and is not unexpected, but is also unrecoverable. Examples of this are a config file
// was unable to be read, or a user provided an invalid argument. These cases are not
// exceptional and should do not deserve panics, however there is likely no way to recover
// from them. Instead the program should exit with a meaningful exit code as well as
// a message informing the user of what went wrong and how to proceed.
//
// The fatal package provides two primary mechanisms to support dealing with these sitations.
//
// The Error type represents a fatal error that occurred. It allows signaling that the program
// should exit with a given exit code. It also allows for adding a message that describes
// the problem along with the underlying error that occurred.
//
// The Exiter type provides the ability to exit the program based on an error. The Exiter.Exit
// method takes an error and determines the exit code from it. The Exiter.PrintAndExit method
// is similar, but it also prints a description of the error before exiting to provide context.
// The top level Exit and PrintAndExit functions are provided for convenience and offer the
// functionality provided by Exiter with defaults.
package fatal

import (
	"fmt"
	"io"
	"os"
)

// ExitCoder defines a type that can provide an exit code.
//
// The value returned by the ExitCode method is up to interpretation
// by the caller. For example, certain APIs might only deal with error
// cases and might treat a value of 0 as meaning an exit code was not
// specified and default it to 1 instead.
type ExitCoder interface {
	ExitCode() int
}

// Error is used to communicate that a program should exit.
// It represents a fatal (but not unexpected) error that cannot be recovered from.
// The fields can be used to control how the program exits.
//
// Error implements the error interface for convenience so it can be returned as
// an error value from functions. However, Error should be treated specially and
// not like a normal error.
//
// There are two rules that should be followed when working with Error:
//
// 1. An Error instance should always be a top level error and should not be wrapped.
//
// 2. The Error method should generally not be used. Instead it should be used with a
// printf-like function using either the '%v' or '%+v' verbs. This will create a nice
// message from the error that can be displayed to users to provide information on what
// went wrong and offer guidance on how to proceed.
type Error struct {
	// Code is the code that the program should exit with.
	Code int
	// Msg is a message to print to provide information on what went wrong
	// and how to proceed.
	Msg string
	// Err is the underlying error that occurred (if any).
	Err error
}

// ExitCode implements the ExitCoder interface and returns the error's code.
func (e *Error) ExitCode() int {
	return e.Code
}

func (e *Error) Error() string {
	return fmt.Sprint(e)
}

func (e *Error) Format(s fmt.State, verb rune) {
	if e.Err != nil {
		_, _ = io.WriteString(s, "Error: ")
		format := "%v\n"
		if s.Flag('+') {
			format = "%+v\n"
		}
		fmt.Fprintf(s, format, e.Err)
	}
	// If an error was just printed and a message is going to be printed,
	// add an extra newline inbetween them.
	if e.Err != nil && e.Msg != "" {
		_, _ = io.WriteString(s, "\n")
	}
	if e.Msg != "" {
		// If e.Msg ends with a newline remove it so that callers can control
		// whether or not to add a newline with a printf-like function.
		// We could use strings.HasSuffix and strings.TrimSuffix but it's just a single
		// byte/rune so lets do it ourselves and avoid a dependency on the strings package.
		msg := e.Msg
		if msg[len(msg)-1] == '\n' {
			msg = msg[:len(msg)-1]
		}
		_, _ = io.WriteString(s, msg)
	}
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Exiter is used to terminate a program.
// The fields can be used to customize how the program exits.
type Exiter struct {
	// Out is where the error should be printed when using PrintAndExit.
	// If nil, it will be defaulted to os.Stderr.
	Out io.Writer
	// PrintDetailed controls how the error is formatted when using PrintAndExit.
	// If true, the error is formatted using '%+v', otherwise '%v' is used.
	PrintDetailed bool
	// ExitFunc is the function that will be called to exit the program.
	// A custom function can be provided to control exit behaviour and perform
	// additional tasks before exiting.
	// If nil, it will be defaulted to os.Exit.
	ExitFunc func(code int)
}

// Exit causes the program to exit. The exit code is determined based on err.
// If err implements ExitCoder and the value of ExitCode is greater than zero,
// it will be used. Otherwise, the exit code will be 1.
func (e *Exiter) Exit(err error) {
	var code int
	if ec, ok := err.(ExitCoder); ok {
		code = ec.ExitCode()
	}
	// If the code couldn't be determined or an invalid code was provided,
	// default to code to 1 since that is the general catch all error code.
	// Exit should not be used to exit successfully so assume 0 means not provided
	// even if it was the actual value.
	if code < 1 {
		code = 1
	}
	if e.ExitFunc == nil {
		e.ExitFunc = os.Exit
	}
	e.ExitFunc(code)
}

// PrintAndExit prints the error and then causes the program to exit.
// The exit code is determined based on err. If err implements ExitCoder
// and the value of ExitCode is greater than zero, it will be used.
// Otherwise, the exit code will be 1.
func (e *Exiter) PrintAndExit(err error) {
	format := "%v\n"
	if e.PrintDetailed {
		format = "%+v\n"
	}
	if e.Out == nil {
		e.Out = os.Stderr
	}
	fmt.Fprintf(e.Out, format, err)
	e.Exit(err)
}

// Exit causes the program to exit. The exit code is determined based on err.
// If err implements ExitCoder and the value of ExitCode is greater than zero,
// it will be used. Otherwise, the exit code will be 1.
func Exit(err error) {
	var e Exiter
	e.Exit(err)
}

// PrintAndExit prints the error and then causes the program to exit.
// The exit code is determined based on err. If err implements ExitCoder
// and the value of ExitCode is greater than zero, it will be used.
// Otherwise, the exit code will be 1.
func PrintAndExit(err error) {
	var e Exiter
	e.PrintAndExit(err)
}
