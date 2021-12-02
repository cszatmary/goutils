package fatal

import (
	"bytes"
	"testing"

	"github.com/TouchBistro/goutils/errors"
)

type mockExit struct {
	code int
}

func (me *mockExit) Exit(code int) {
	me.code = code
}

type client struct {
	flushed bool
}

func (c *client) Flush() {
	c.flushed = true
}

func resetState() {
	// Before each: need to make sure global state is
	// reset so it doesn't poison tests
	PrintDetailedError(false)
	OnExit(nil)
}

func TestExit(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	Exit("Something broke")

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "Something broke\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
}

func TestExitOnExit(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}
	c := client{}

	errWriter = buf
	exitFunc = me.Exit

	OnExit(func() {
		c.Flush()
	})

	Exit("Something broke")

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "Something broke\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
	if !c.flushed {
		t.Error("want flushed to be true, was false")
	}
}

func TestExitf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	Exitf("%d failures", 3)

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "3 failures\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
}

func TestExitfOnExit(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}
	c := client{}

	errWriter = buf
	exitFunc = me.Exit

	OnExit(func() {
		c.Flush()
	})

	Exitf("%d failures", 3)

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "3 failures\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
	if !c.flushed {
		t.Error("want flushed to be true, was false")
	}
}

func TestExitErr(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New(nil, "err everything broke", errors.Op("test.Foo"))

	ExitErr(err, "Something broke")

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "Something broke\nError: err everything broke\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
}

func TestExitErrStack(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New(nil, "err everything broke", errors.Op("test.Foo"))

	PrintDetailedError(true)

	ExitErr(err, "Something broke")

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "Something broke\nError: test.Foo: err everything broke\n"
	if buf.String() != want {
		t.Errorf("got output\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestExitErrf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New(nil, "err everything broke", errors.Op("test.Foo"))

	ExitErrf(err, "%d failures", 3)

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "3 failures\nError: err everything broke\n"
	if buf.String() != want {
		t.Errorf("got output %q, want %q", buf.String(), want)
	}
}

func TestExitErrStackf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New(nil, "err everything broke", errors.Op("test.Foo"))

	PrintDetailedError(true)

	ExitErrf(err, "%d failures", 3)

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "3 failures\nError: test.Foo: err everything broke\n"
	if buf.String() != want {
		t.Errorf("got output\n%q\nwant\n%q", buf.String(), want)
	}
}
