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
			"Black",
			color.Black,
			"foo bar",
			"\x1b[30mfoo bar\x1b[39m",
		},
		{
			"Red",
			color.Red,
			"foo bar",
			"\x1b[31mfoo bar\x1b[39m",
		},
		{
			"Green",
			color.Green,
			"foo bar",
			"\x1b[32mfoo bar\x1b[39m",
		},
		{
			"Yellow",
			color.Yellow,
			"foo bar",
			"\x1b[33mfoo bar\x1b[39m",
		},
		{
			"Blue",
			color.Blue,
			"foo bar",
			"\x1b[34mfoo bar\x1b[39m",
		},
		{
			"Magenta",
			color.Magenta,
			"foo bar",
			"\x1b[35mfoo bar\x1b[39m",
		},
		{
			"Cyan",
			color.Cyan,
			"foo bar",
			"\x1b[36mfoo bar\x1b[39m",
		},
		{
			"White",
			color.White,
			"foo bar",
			"\x1b[37mfoo bar\x1b[39m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.colorFn(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripReset(t *testing.T) {
	color.SetEnabled(true)
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"single reset", "foo \x1b[39mbar", "\x1b[31mfoo bar\x1b[39m"},
		{"multiple resets", "foo \x1b[39m\x1b[39mbar", "\x1b[31mfoo bar\x1b[39m"},
	}
	for _, tt := range tests {
		got := color.Red(tt.in)
		if got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestColorDisabled(t *testing.T) {
	color.SetEnabled(false)
	got := color.Red("foo bar")
	want := "foo bar"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestColorerDisabled(t *testing.T) {
	color.SetEnabled(true)
	var c color.Colorer
	c.SetEnabled(false)
	if got, want := c.Red("foo bar"), "foo bar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Make sure package functions were not affected
	if got, want := color.Red("foo bar"), "\x1b[31mfoo bar\x1b[39m"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func BenchmarkRed(b *testing.B) {
	color.SetEnabled(true)
	b.Run("no strip", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			color.Red("foo bar")
		}
	})
	b.Run("strip", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			color.Red("foo \x1b[39m\x1b[39mbar")
		}
	})
}
