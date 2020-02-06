package command

import (
	"bytes"
	"os/exec"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func IsCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "command": command}).Debug("Error looking up command.")
		return false
	}
	return true
}

func Exec(id string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	stdout := log.WithFields(log.Fields{
		"id": id,
	}).WriterLevel(log.DebugLevel)
	defer stdout.Close()

	stderr := log.WithFields(log.Fields{
		"id": id,
	}).WriterLevel(log.DebugLevel)
	defer stderr.Close()

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Exec failed to run %s %s", name, arg)
	}

	return nil
}

func ExecResult(id string, name string, args ...string) (*bytes.Buffer, error) {
	cmd := exec.Command(name, args...)

	stdoutBuf := &bytes.Buffer{}
	stderr := log.WithFields(log.Fields{
		"id": id,
	}).WriterLevel(log.DebugLevel)
	defer stderr.Close()

	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrapf(err, "Exec failed to run %s %s", name, args)
	}

	return stdoutBuf, nil
}
