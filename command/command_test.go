package command

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCommandAvailable(t *testing.T) {
	exists := IsCommandAvailable("echo")

	assert.True(t, exists)
}

func TestNotIsCommandAvailable(t *testing.T) {
	exists := IsCommandAvailable("asljhasld")

	assert.False(t, exists)
}

func TestExec(t *testing.T) {
	err := Exec("echo", []string{"Hello world"}, "test-exec")

	assert.NoError(t, err)
}

func TestExecOpts(t *testing.T) {
	assert := assert.New(t)
	buf := &bytes.Buffer{}
	err := Exec("echo", []string{"Hello world"}, "test-exec-opts", func(cmd *exec.Cmd) {
		cmd.Stdout = buf
	})

	assert.NoError(err)
	assert.Equal("Hello world\n", buf.String())
}

func TestExecError(t *testing.T) {
	err := Exec("notacmd", []string{"Hello World"}, "test-exec-error")

	assert.Error(t, err)
}
