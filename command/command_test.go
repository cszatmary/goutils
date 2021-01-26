package command_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TouchBistro/goutils/command"
)

func TestIsAvailable(t *testing.T) {
	exists := command.IsAvailable("echo")
	if !exists {
		t.Error("want command to exist but it does not")
	}
}

func TestNotIsAvailable(t *testing.T) {
	exists := command.IsAvailable("asljhasld")
	if exists {
		t.Error("want command to not exist but it does")
	}
}

func TestExec(t *testing.T) {
	err := command.Exec("echo", "Hello world")
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
}

func TestExecOpts(t *testing.T) {
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	cmd := command.New(
		command.WithStdout(stdoutBuf),
		command.WithStderr(stderrBuf),
		command.WithEnv(map[string]string{
			"FOO": "BAR",
		}),
	)
	err := cmd.Exec("sh", "-c", "echo $FOO")
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	wantStdout := "BAR\n"
	if stdoutBuf.String() != wantStdout {
		t.Errorf("got stdout %s, want %s", stdoutBuf.String(), wantStdout)
	}
	if stderrBuf.String() != "" {
		t.Errorf("got stderr %s, want it to be empty", stderrBuf.String())
	}
}

func TestExecWithDir(t *testing.T) {
	tmpdir := t.TempDir()
	buf := &bytes.Buffer{}
	cmd := command.New(
		command.WithStdout(buf),
		command.WithDir(tmpdir),
	)
	err := cmd.Exec("pwd")
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, tmpdir) {
		t.Errorf("got stdout %s, want %s", got, tmpdir)
	}
}

func TestExecError(t *testing.T) {
	err := command.Exec("notacmd", "Hello World")
	if err == nil {
		t.Error("want non-nil error, got nil")
	}
}
