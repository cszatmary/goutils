package color_test

import (
	"testing"

	"github.com/TouchBistro/goutils/color"
)

func TestColors(t *testing.T) {
	color.SetEnabled(true)
	tests := []struct {
		name    string
		colorFn func(string) string
		input   string
		want    string
	}{
		{
			"Red() test",
			color.Red,
			"foo bar",
			"\x1b[31mfoo bar\x1b[39m",
		},
		{
			"Green() test",
			color.Green,
			"foo bar",
			"\x1b[32mfoo bar\x1b[39m",
		},
		{
			"Yellow() test",
			color.Yellow,
			"foo bar",
			"\x1b[33mfoo bar\x1b[39m",
		},
		{
			"Blue() test",
			color.Blue,
			"foo bar",
			"\x1b[34mfoo bar\x1b[39m",
		},
		{
			"Magenta() test",
			color.Magenta,
			"foo bar",
			"\x1b[35mfoo bar\x1b[39m",
		},
		{
			"Cyan() test",
			color.Cyan,
			"foo bar",
			"\x1b[36mfoo bar\x1b[39m",
		},
		{
			"White() test",
			color.White,
			"foo bar",
			"\x1b[37mfoo bar\x1b[39m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.colorFn(tt.input)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestStripReset(t *testing.T) {
	color.SetEnabled(true)
	got := color.Red("foo \x1b[39mbar")
	want := "\x1b[31mfoo bar\x1b[39m"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestColorDisabled(t *testing.T) {
	color.SetEnabled(false)
	got := color.Red("foo bar")
	want := "foo bar"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
