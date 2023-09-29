// Package color provides functions for creating colored strings.
//
// There are several functions provided to make it easy to set foreground colors.
//
//	// creates a string with a red foreground color
//	color.Red("uh oh")
//
// Colors can be globally enabled or disabled by using SetEnabled.
// If you wish to control colors in a local scope and not affect the global state,
// create a Colorer instance.
//
//	var c color.Colorer
//	// Disable colors only for this Colorer
//	c.SetEnabled(false)
//	s := c.Red("uh oh") // Will not be colored
//
// This package also supports the NO_COLOR environment variable.
// If NO_COLOR is set with any value, colors will be disabled.
// See https://no-color.org for more details.
package color

import (
	"os"
	"strconv"
	"strings"
)

type ansiCode int

const (
	fgBlack ansiCode = iota + 30
	fgRed
	fgGreen
	fgYellow
	fgBlue
	fgMagenta
	fgCyan
	fgWhite
	_ // skip value
	fgReset
)

var (
	noColor = os.Getenv("NO_COLOR") != "" // value doesn't matter, only if it's set
	shared  Colorer
)

// IsNoColorEnvSet returns true if the NO_COLOR environment variable is set, regardless of its value.
// See https://no-color.org for more details.
func IsNoColorEnvSet() bool {
	return noColor
}

// Colorer allows for creating coloured strings. Using a Colorer instance allows
// for modifying certain attributes that affect output locally instead of globally,
// for example, disable colouring in a local context and not globally.
//
// A zero value Colorer is a valid Colorer ready for use.
// Colors are enabled by default, unless NO_COLOR is set.
type Colorer struct {
	disabled bool // disabled so the zero value is enabled
}

// SetEnabled sets whether color is enabled or disabled.
// Note that if NO_COLOR is set this will have no effect.
func (c *Colorer) SetEnabled(e bool) {
	c.disabled = !e
}

// Black creates a black colored string.
func (c *Colorer) Black(s string) string {
	return c.apply(s, fgBlack, fgReset)
}

// Red creates a red colored string.
func (c *Colorer) Red(s string) string {
	return c.apply(s, fgRed, fgReset)
}

// Green creates a green colored string.
func (c *Colorer) Green(s string) string {
	return c.apply(s, fgGreen, fgReset)
}

// Yellow creates a yellow colored string.
func (c *Colorer) Yellow(s string) string {
	return c.apply(s, fgYellow, fgReset)
}

// Blue creates a blue colored string.
func (c *Colorer) Blue(s string) string {
	return c.apply(s, fgBlue, fgReset)
}

// Magenta creates a magenta colored string.
func (c *Colorer) Magenta(s string) string {
	return c.apply(s, fgMagenta, fgReset)
}

// Cyan creates a cyan colored string.
func (c *Colorer) Cyan(s string) string {
	return c.apply(s, fgCyan, fgReset)
}

// White creates a white colored string.
func (c *Colorer) White(s string) string {
	return c.apply(s, fgWhite, fgReset)
}

func (c *Colorer) apply(s string, start, end ansiCode) string {
	// NO_COLOR always takes precedence.
	if noColor || c.disabled {
		return s
	}

	const prefix = "\x1b["
	var sb strings.Builder
	// Build out reset for the end
	sb.WriteString(prefix)
	sb.WriteString(strconv.Itoa(int(end)))
	sb.WriteByte('m')
	reset := sb.String()
	sb.Reset()

	// Build colored string.
	// We also want to check if there are any occurrences of reset
	// in s and remove them so that the color isn't messed up.
	sb.WriteString(prefix)
	sb.WriteString(strconv.Itoa(int(start)))
	sb.WriteByte('m')

	// We are only dealing with ASCII so it's safe to look at individual bytes.
	j := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && strings.HasPrefix(s[i:], reset) {
			sb.WriteString(s[j:i])
			i += len(reset) - 1 // -1 to account for i++
			j = i + 1
		}
	}
	sb.WriteString(s[j:])
	sb.WriteString(reset)
	return sb.String()
}

// SetEnabled sets whether color is enabled or disabled.
// Note that if NO_COLOR is set this will have no effect.
func SetEnabled(e bool) {
	shared.SetEnabled(e)
}

// Black creates a black colored string.
func Black(s string) string {
	return shared.Black(s)
}

// Red creates a red colored string.
func Red(s string) string {
	return shared.Red(s)
}

// Green creates a green colored string.
func Green(s string) string {
	return shared.Green(s)
}

// Yellow creates a yellow colored string.
func Yellow(s string) string {
	return shared.Yellow(s)
}

// Blue creates a blue colored string.
func Blue(s string) string {
	return shared.Blue(s)
}

// Magenta creates a magenta colored string.
func Magenta(s string) string {
	return shared.Magenta(s)
}

// Cyan creates a cyan colored string.
func Cyan(s string) string {
	return shared.Cyan(s)
}

// White creates a white colored string.
func White(s string) string {
	return shared.White(s)
}
