package log_test

import (
	"errors"
	"testing"

	"github.com/TouchBistro/goutils/log"
	"github.com/TouchBistro/goutils/progress"
)

// Make sure Logger and Entry satisfies the necessary interfaces.
var _ progress.OutputLogger = &log.Logger{}
var _ progress.Logger = &log.Entry{}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		wantLevel log.Level
		wantErr   error
	}{
		{
			name:      "debug",
			s:         "debug",
			wantLevel: log.LevelDebug,
			wantErr:   nil,
		},
		{
			name:      "info",
			s:         "info",
			wantLevel: log.LevelInfo,
			wantErr:   nil,
		},
		{
			name:      "debug",
			s:         "debug",
			wantLevel: log.LevelDebug,
			wantErr:   nil,
		},
		{
			name:      "warn",
			s:         "warn",
			wantLevel: log.LevelWarn,
			wantErr:   nil,
		},
		{
			name:      "error",
			s:         "error",
			wantLevel: log.LevelError,
			wantErr:   nil,
		},
		{
			name:      "case insensitive",
			s:         "Info",
			wantLevel: log.LevelInfo,
			wantErr:   nil,
		},
		{
			name:      "invalid",
			s:         "unknown",
			wantLevel: -1,
			wantErr:   log.ErrInvalidLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lvl, err := log.ParseLevel(tt.s)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if lvl != tt.wantLevel {
				t.Errorf("got level %d, want %d", lvl, tt.wantLevel)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		name string
		lvl  log.Level
		want string
	}{
		{
			name: "debug",
			lvl:  log.LevelDebug,
			want: "debug",
		},
		{
			name: "info",
			lvl:  log.LevelInfo,
			want: "info",
		},
		{
			name: "warn",
			lvl:  log.LevelWarn,
			want: "warn",
		},
		{
			name: "error",
			lvl:  log.LevelError,
			want: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.lvl.String()
			if s != tt.want {
				t.Errorf("got %s, want %s", s, tt.want)
			}
		})
	}
}

func TestLevelUnmarshalText(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		wantLevel log.Level
		wantErr   error
	}{
		{
			name:      "debug",
			s:         "debug",
			wantLevel: log.LevelDebug,
			wantErr:   nil,
		},
		{
			name:      "info",
			s:         "info",
			wantLevel: log.LevelInfo,
			wantErr:   nil,
		},
		{
			name:      "debug",
			s:         "debug",
			wantLevel: log.LevelDebug,
			wantErr:   nil,
		},
		{
			name:      "warn",
			s:         "warn",
			wantLevel: log.LevelWarn,
			wantErr:   nil,
		},
		{
			name:      "error",
			s:         "error",
			wantLevel: log.LevelError,
			wantErr:   nil,
		},
		{
			name:      "case insensitive",
			s:         "Info",
			wantLevel: log.LevelInfo,
			wantErr:   nil,
		},
		{
			name:      "invalid",
			s:         "unknown",
			wantLevel: -1,
			wantErr:   log.ErrInvalidLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lvl := log.Level(-1)
			err := lvl.UnmarshalText([]byte(tt.s))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if lvl != tt.wantLevel {
				t.Errorf("got level %d, want %d", lvl, tt.wantLevel)
			}
		})
	}
}

func TestLevelMarshalText(t *testing.T) {
	tests := []struct {
		name     string
		lvl      log.Level
		wantText string
		wantErr  error
	}{
		{
			name:     "debug",
			lvl:      log.LevelDebug,
			wantText: "debug",
			wantErr:  nil,
		},
		{
			name:     "info",
			lvl:      log.LevelInfo,
			wantText: "info",
			wantErr:  nil,
		},
		{
			name:     "debug",
			lvl:      log.LevelDebug,
			wantText: "debug",
			wantErr:  nil,
		},
		{
			name:     "warn",
			lvl:      log.LevelWarn,
			wantText: "warn",
			wantErr:  nil,
		},
		{
			name:     "error",
			lvl:      log.LevelError,
			wantText: "error",
			wantErr:  nil,
		},
		{
			name:     "invalid",
			lvl:      -1,
			wantText: "",
			wantErr:  log.ErrInvalidLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := tt.lvl.MarshalText()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if string(text) != tt.wantText {
				t.Errorf("got text %s, want %s", text, tt.wantText)
			}
		})
	}
}
