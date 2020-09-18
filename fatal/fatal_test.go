package fatal

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "Something broke\n", buf.String())
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

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "Something broke\n", buf.String())
	assert.True(t, c.flushed)
}

func TestExitf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	Exitf("%d failures", 3)

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "3 failures\n", buf.String())
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

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "3 failures\n", buf.String())
	assert.True(t, c.flushed)
}

func TestExitErr(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New("err everything broke")

	ExitErr(err, "Something broke")

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "Something broke\nError: err everything broke\n", buf.String())
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

	assert.Equal(t, 1, me.code)

	expected := "Something broke\n" +
		"Error: err everything broke\n" +
		"github.com/TouchBistro/goutils/fatal.TestExitErrStack\n" +
		"\t.+"
	testFormatRegexp(t, 0, err, buf.String(), expected)
}

func TestExitErrf(t *testing.T) {
	resetState()

	buf := &bytes.Buffer{}
	me := mockExit{}

	errWriter = buf
	exitFunc = me.Exit

	err := errors.New("err everything broke")

	ExitErrf(err, "%d failures", 3)

	assert.Equal(t, 1, me.code)
	assert.Equal(t, "3 failures\nError: err everything broke\n", buf.String())
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

	assert.Equal(t, 1, me.code)

	expected := "3 failures\n" +
		"Error: err everything broke\n" +
		"github.com/TouchBistro/goutils/fatal.TestExitErrStack\n" +
		"\t.+"
	testFormatRegexp(t, 0, err, buf.String(), expected)
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
