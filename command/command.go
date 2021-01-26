// Package command provides functionality for working with programs the host OS.
// It provides a high level API over os/exec for running commands, which is
// easier to use for common cases.
package command

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// IsAvailable checks if command is available on the system. This is done by
// checking if command exists within the user's PATH.
func IsAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Command manages the configuration of a command
// that will be run in a child process.
type Command struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	env    map[string]string
	dir    string
}

// New creates a command instance from the given options.
func New(opts ...Option) *Command {
	c := &Command{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Option is a function that takes a command and applies
// a configuration to it.
type Option func(*Command)

// WithStdin sets the reader the the command's stdin should read from.
func WithStdin(stdin io.Reader) Option {
	return func(c *Command) {
		c.stdin = stdin
	}
}

// WithStdout sets the writer that the command's stdout
// should be written to.
func WithStdout(stdout io.Writer) Option {
	return func(c *Command) {
		c.stdout = stdout
	}
}

// WithStderr sets the writer that the command's stderr
// should be written to.
func WithStderr(stderr io.Writer) Option {
	return func(c *Command) {
		c.stderr = stderr
	}
}

// WithEnv sets the environment variables for the process
// the command will be run in.
func WithEnv(env map[string]string) Option {
	return func(c *Command) {
		c.env = env
	}
}

// WithDir sets the directory the command should be run in.
func WithDir(dir string) Option {
	return func(c *Command) {
		c.dir = dir
	}
}

// Exec executes the named program with the given arguments.
func (c *Command) Exec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}
	if c.stdout != nil {
		cmd.Stdout = c.stdout
	}
	if c.stderr != nil {
		cmd.Stderr = c.stderr
	}
	if c.env != nil {
		for k, v := range c.env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	if err := cmd.Run(); err != nil {
		argsStr := strings.Join(args, " ")
		return fmt.Errorf("command: failed to run '%s %s': %w", name, argsStr, err)
	}
	return nil
}

// Exec executes the named program with the given arguments.
// This is a shorthand for when the default command options wish to be used.
func Exec(name string, args ...string) error {
	return New().Exec(name, args...)
}
