package fatal

import (
	"fmt"
	"os"
)

var ShowStackTraces = true

func ExitErr(err error, message string) {
	fmt.Fprintln(os.Stderr, message)

	if ShowStackTraces {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}

	os.Exit(1)
}

func ExitErrf(err error, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)

	if ShowStackTraces {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}

	os.Exit(1)
}

func Exit(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func Exitf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
