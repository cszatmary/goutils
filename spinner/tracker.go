package spinner

import (
	"io"
	"time"

	"github.com/TouchBistro/goutils/progress"
)

// Tracker is a progress.Tracker that uses a Spinner to display progress.
//
// Tracker will modify the logger's output to allow logging while the spinner is running.
// As a result, it is not safe to call OutputLogger.SetOutput while the spinner is running.
type Tracker struct {
	progress.OutputLogger

	// Options to use when creating a spinner.

	// Interval is how often the spinner updates. See spinner.WithInterval.
	Interval time.Duration
	// MaxMessageLength is the max length a message can be. See spinner.WithMaxMessageLength.
	MaxMessageLength int
	// PersistMessages controls whether or not messages are persisted by the spinner.
	// See spinner.WithPersistMessages.
	PersistMessages bool

	s   *Spinner  // the running spinner, nil if no spinner is running
	out io.Writer // saved logger.Output()
}

// Start starts the spinner with the given message and count.
// If the spinner is already it will be restarted.
func (t *Tracker) Start(message string, count int) {
	// Make sure we save the logger output since we modify the logger.
	if t.out == nil {
		t.out = t.OutputLogger.Output()
	}
	// Allow calling Start without having first called Stop.
	if t.s != nil {
		t.s.Stop()
	}

	// Create spinner and apply options
	t.s = New()
	t.s.w = t.out
	t.s.startMsg = message
	if count > 1 {
		t.s.count = count
	}
	if t.Interval > 0 {
		t.s.interval = t.Interval
	}
	if t.MaxMessageLength > 0 {
		t.s.maxMsgLen = t.MaxMessageLength
	}
	if t.PersistMessages {
		t.s.persistMsgs = t.PersistMessages
	}
	t.OutputLogger.SetOutput(t.s)
	t.s.Start()
}

// Stop stops the spinner if it is currently running.
// If the spinner is not running, Stop does nothing.
func (t *Tracker) Stop() {
	if t.s != nil {
		t.s.Stop()
		t.s = nil
		t.OutputLogger.SetOutput(t.out)
	}
}

// Inc increments the progress of the spinner if it is running.
// If the spinner is not running, Inc does nothing.
func (t *Tracker) Inc() {
	if t.s != nil {
		t.s.Inc()
	}
}

// UpdateMessage updates the message shown by the spinner if it is running.
// If the spinner is not running, UpdateMessage does nothing.
func (t *Tracker) UpdateMessage(m string) {
	if t.s != nil {
		t.s.UpdateMessage(m)
	}
}
