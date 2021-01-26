package fatal

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
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
	ShowStackTraces(false)
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

	err := errors.New("err everything broke")

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

	err := errors.New("err everything broke")

	ShowStackTraces(true)

	ExitErr(err, "Something broke")

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "Something broke\n" +
		"Error: err everything broke\n" +
		"github.com/TouchBistro/goutils/fatal.TestExitErrStack\n" +
		"\t.+"
	testFormatRegexp(t, 0, err, buf.String(), want)
}

func TestExitErrf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New("err everything broke")

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

	err := errors.New("err everything broke")

	ShowStackTraces(true)

	ExitErrf(err, "%d failures", 3)

	if me.code != 1 {
		t.Errorf("got error code %d, expected 1", me.code)
	}
	want := "3 failures\n" +
		"Error: err everything broke\n" +
		"github.com/TouchBistro/goutils/fatal.TestExitErrStack\n" +
		"\t.+"
	testFormatRegexp(t, 0, err, buf.String(), want)
}

// Taken from https://github.com/pkg/errors/blob/614d223910a179a466c1767a985424175c39b465/format_test.go#L387
// Helper to test string with regexp
func testFormatRegexp(t *testing.T, n int, arg interface{}, format, want string) {
	t.Helper()
	got := fmt.Sprintf(format, arg)
	gotLines := strings.SplitN(got, "\n", -1)
	wantLines := strings.SplitN(want, "\n", -1)

	if len(wantLines) > len(gotLines) {
		t.Errorf("test %d: wantLines(%d) > gotLines(%d):\n got: %q\nwant: %q", n+1, len(wantLines), len(gotLines), got, want)
		return
	}

	for i, w := range wantLines {
		match, err := regexp.MatchString(w, gotLines[i])
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			t.Errorf("test %d: line %d: fmt.Sprintf(%q, err):\n got: %q\nwant: %q", n+1, i+1, format, got, want)
		}
	}
}
