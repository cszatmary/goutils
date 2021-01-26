// Package color provides functions for creating coloured strings.
package color

import (
	"fmt"
	"os"
	"regexp"
)

const (
	ansiFgRed     = 31
	ansiFgGreen   = 32
	ansiFgYellow  = 33
	ansiFgBlue    = 34
	ansiFgMagenta = 35
	ansiFgCyan    = 36
	asnsiFgWhite  = 37
	ansiResetFg   = 39
)

// Support for NO_COLOR env var
// https://no-color.org/
var (
	noColor = false
	enabled bool
)

func init() {
	// The standard says the value doesn't matter, only whether or not it's set
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		noColor = true
	}
	enabled = !noColor
}

func apply(str string, start, end int) string {
	if !enabled {
		return str
	}

	regex := regexp.MustCompile(fmt.Sprintf("\\x1b\\[%dm", end))
	// Remove any occurrences of reset to make sure color isn't messed up
	sanitized := regex.ReplaceAllString(str, "")
	return fmt.Sprintf("\x1b[%dm%s\x1b[%dm", start, sanitized, end)
}

// SetEnabled sets whether color is enabled or disabled.
// If the NO_COLOR environment variable is set, this function will
// do nothing as NO_COLOR takes precedence.
func SetEnabled(e bool) {
	// NO_COLOR overrides this
	if noColor {
		return
	}
	enabled = e
}

// Red creates a red colored string
func Red(str string) string {
	return apply(str, ansiFgRed, ansiResetFg)
}

// Green creates a green colored string
func Green(str string) string {
	return apply(str, ansiFgGreen, ansiResetFg)
}

// Yellow creates a yellow colored string
func Yellow(str string) string {
	return apply(str, ansiFgYellow, ansiResetFg)
}

// Blue creates a blue colored string
func Blue(str string) string {
	return apply(str, ansiFgBlue, ansiResetFg)
}

// Magenta creates a magenta colored string
func Magenta(str string) string {
	return apply(str, ansiFgMagenta, ansiResetFg)
}

// Cyan creates a cyan colored string
func Cyan(str string) string {
	return apply(str, ansiFgCyan, ansiResetFg)
}

// White creates a white colored string
func White(str string) string {
	return apply(str, asnsiFgWhite, ansiResetFg)
}
