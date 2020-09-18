package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColors(t *testing.T) {
	noColor = false

	tests := []struct {
		name     string
		colorFn  func(string) string
		input    string
		expected string
	}{
		{
			"Red() test",
			Red,
			"foo bar",
			"\x1b[31mfoo bar\x1b[39m",
		},
		{
			"Green() test",
			Green,
			"foo bar",
			"\x1b[32mfoo bar\x1b[39m",
		},
		{
			"Yellow() test",
			Yellow,
			"foo bar",
			"\x1b[33mfoo bar\x1b[39m",
		},
		{
			"Blue() test",
			Blue,
			"foo bar",
			"\x1b[34mfoo bar\x1b[39m",
		},
		{
			"Magenta() test",
			Magenta,
			"foo bar",
			"\x1b[35mfoo bar\x1b[39m",
		},
		{
			"Cyan() test",
			Cyan,
			"foo bar",
			"\x1b[36mfoo bar\x1b[39m",
		},
		{
			"White() test",
			White,
			"foo bar",
			"\x1b[37mfoo bar\x1b[39m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := tt.colorFn(tt.input)
			assert.Equal(t, tt.expected, received)
		})
	}
}

func TestStripReset(t *testing.T) {
	noColor = false

	received := Red("foo \x1b[39mbar")
	assert.Equal(t, "\x1b[31mfoo bar\x1b[39m", received)
}

func TestNoColor(t *testing.T) {
	noColor = true

	received := Red("foo bar")
	assert.Equal(t, "foo bar", received)
}
