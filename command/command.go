package command

import (
	"os/exec"
	"strings"

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

func Exec(cmdName string, args []string, id string, opts ...func(*exec.Cmd)) error {
	cmd := exec.Command(cmdName, args...)

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

	for _, opt := range opts {
		opt(cmd)
	}

	err := cmd.Run()
	if err != nil {
		argsStr := strings.Join(args, " ")
		return errors.Wrapf(err, "Exec failed to run %s %s", cmdName, argsStr)
	}

	return nil
}
