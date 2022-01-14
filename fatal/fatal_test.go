package fatal_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/fatal"
)

func TestExiterExit(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "not a ExitCoder",
			err:      fmt.Errorf("oops error"),
			wantCode: 1,
		},
		{
			name:     "ExitCoder",
			err:      coder(2),
			wantCode: 2,
		},
		{
			name: "fatal.Error",
			err: &fatal.Error{
				Code: 130,
				Msg:  "Operation cancelled",
			},
			wantCode: 130,
		},
		{
			name:     "handle zero",
			err:      coder(0),
			wantCode: 1,
		},
		{
			name:     "handle negative",
			err:      coder(-1),
			wantCode: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var me mockExit
			exiter := fatal.Exiter{ExitFunc: me.Exit}
			exiter.Exit(tt.err)
			if me.code != tt.wantCode {
				t.Errorf("got exit code %d, want %d", me.code, tt.wantCode)
			}
		})
	}
}

func TestExiterPrintAndExit(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		printDetailed bool
		wantCode      int
		wantOutput    string
	}{
		{
			name:       "any error",
			err:        fmt.Errorf("oops error"),
			wantCode:   1,
			wantOutput: "oops error\n",
		},
		{
			name: "fatal.Error",
			err: &fatal.Error{
				Code: 2,
				Msg:  "Something broke",
				Err:  errors.New(nil, "err everything broke", errors.Op("test.Foo")),
			},
			wantCode:   2,
			wantOutput: "Error: err everything broke\n\nSomething broke\n",
		},
		{
			name: "fatal.Error with detail",
			err: &fatal.Error{
				Code: 2,
				Msg:  "Something broke",
				Err:  errors.New(nil, "err everything broke", errors.Op("test.Foo")),
			},
			printDetailed: true,
			wantCode:      2,
			wantOutput:    "Error: test.Foo: err everything broke\n\nSomething broke\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var me mockExit
			var buf bytes.Buffer
			exiter := fatal.Exiter{
				Out:           &buf,
				PrintDetailed: tt.printDetailed,
				ExitFunc:      me.Exit,
			}
			exiter.PrintAndExit(tt.err)
			if me.code != tt.wantCode {
				t.Errorf("got exit code %d, want %d", me.code, tt.wantCode)
			}
			if buf.String() != tt.wantOutput {
				t.Errorf("got output:\n%s\nwant:\n%s", buf.String(), tt.wantOutput)
			}
		})
	}
}

type mockExit struct {
	code int
}

func (me *mockExit) Exit(code int) {
	me.code = code
}

type coder int

func (c coder) ExitCode() int {
	return int(c)
}

func (c coder) Error() string {
	return fmt.Sprintf("Code: %d", c)
}
