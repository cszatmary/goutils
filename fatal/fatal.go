// Package fatal provides functionality for terminating the program when a
// fatal condition occurs. It allows for printing messages and errors and
// running a clean up function before the program is terminated.
package fatal

import (
	"fmt"
	"io"
	"os"
)

// Package state
var (
	printDetailedError = false
	onExitHandler      func()
)

// Used for dependency injection in tests
// Normally having tests touch private stuff is bad
// but this is the only way I could figure out to mock os.Exit
var (
	errWriter io.Writer      = os.Stderr
	exitFunc  func(code int) = os.Exit
)

// PrintDetailedError sets whether or not a detailed version of an error
// should be printed when calling ExitErr or ExitErrf.
//
// A detailed error is printed by using the '%+v' format verb.
func PrintDetailedError(show bool) {
	printDetailedError = show
}

// OnExit registers a handler that will run before os.Exit is called.
// This is useful for performing any clean up that would usually be called
// in a defer block since defers are not called when os.Exit is used.
func OnExit(handler func()) {
	onExitHandler = handler
}

// ExitErr prints the given message and error to stderr then exits the program.
func ExitErr(err error, message string) {
	fmt.Fprintln(errWriter, message)
	if err != nil {
		if printDetailedError {
			fmt.Fprintf(errWriter, "Error: %+v\n", err)
		} else {
			fmt.Fprintf(errWriter, "Error: %s\n", err)
		}
	}
	if onExitHandler != nil {
		onExitHandler()
	}
	exitFunc(1)
}

// ExitErrf prints the given message and error to stderr then exits the program.
// Supports printf like formatting.
func ExitErrf(err error, format string, a ...interface{}) {
	ExitErr(err, fmt.Sprintf(format, a...))
}

// Exit prints the given message to stderr then exists the program.
func Exit(message string) {
	ExitErr(nil, message)
}

// Exitf prints the given message to stderr then exits the program.
// Supports printf like formatting.
func Exitf(format string, a ...interface{}) {
	ExitErr(nil, fmt.Sprintf(format, a...))
}
