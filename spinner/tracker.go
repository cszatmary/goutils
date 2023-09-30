package spinner

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/cszatmary/goutils/logutil"
	"github.com/cszatmary/goutils/progress"
)

// tracker is a progress.Tracker that uses a Spinner to display progress.
type tracker struct {
	*logutil.FormatLogger
	w  io.Writer          // saved logger output
	wv *logutil.WriterVar // used to modify the logger's hander's output dynamically
	s  *Spinner           // the running spinner, nil if no spinner is running

	// spinner options

	interval       time.Duration
	maxMsgLen      int
	persistMsgs    bool
	disableSpinner bool
}

// NewTracker creates a progress.Tracker that uses a Spinner to display progress.
func NewTracker(opts TrackerOptions) progress.Tracker {
	if opts.Writer == nil {
		opts.Writer = os.Stderr
	}
	wv := logutil.NewWriterVar(opts.Writer)
	var h slog.Handler
	if opts.NewHandler != nil {
		h = opts.NewHandler(wv)
	} else {
		h = slog.NewTextHandler(wv, nil)
	}
	return &tracker{
		FormatLogger:   logutil.NewFormatLogger(h),
		w:              opts.Writer,
		wv:             wv,
		interval:       opts.Interval,
		maxMsgLen:      opts.MaxMessageLength,
		persistMsgs:    opts.PersistMessages,
		disableSpinner: opts.DisableSpinner,
	}
}

// TrackerOptions allows for customizing a Tracker created with NewTracker.
// See each field for more details.
type TrackerOptions struct {
	// Writer is where logs and spinner output is written.
	// If nil it defaults to os.Stderr.
	Writer io.Writer
	// Interval is how often the spinner updates. See spinner.WithInterval.
	Interval time.Duration
	// MaxMessageLength is the max length a message can be. See spinner.WithMaxMessageLength.
	MaxMessageLength int
	// PersistMessages controls whether or not messages are persisted by the spinner.
	// See spinner.WithPersistMessages.
	PersistMessages bool
	// NewHandler is a function that creates a new slog.Handler to use for logging.
	// If nil a slog.TextHandler will be created with default options.
	NewHandler func(w io.Writer) slog.Handler
	// DisableSpinner disables usage of a spinner and simply logs spinner messages.
	// This is useful if you wish to dynamically control spinner behaviour based on
	// an environment variable or command line flag.
	DisableSpinner bool
}

// Start starts the spinner with the given message and count.
// If the spinner is already it will be restarted.
func (t *tracker) Start(msg string, count int) {
	if t.disableSpinner {
		l := t.FormatLogger
		if count > 1 {
			l = l.With("count", count)
		}
		l.Info(msg)
		return
	}

	// Allow calling Start without having first called Stop.
	if t.s != nil {
		t.s.Stop()
	}

	// Create spinner and apply options
	t.s = New()
	t.s.w = t.w
	t.s.startMsg = msg
	if count > 1 {
		t.s.count = count
	}
	if t.interval > 0 {
		t.s.interval = t.interval
	}
	if t.maxMsgLen > 0 {
		t.s.maxMsgLen = t.maxMsgLen
	}
	if t.persistMsgs {
		t.s.persistMsgs = t.persistMsgs
	}
	t.wv.Set(t.s)
	t.s.Start()
}

// Stop stops the spinner if it is currently running.
// If the spinner is not running, Stop does nothing.
func (t *tracker) Stop() {
	if t.s != nil {
		t.s.Stop()
		t.s = nil
		t.wv.Set(t.w)
	}
}

// Inc increments the progress of the spinner if it is running.
// If the spinner is not running, Inc does nothing.
func (t *tracker) Inc() {
	if t.s != nil {
		t.s.Inc()
	}
}

// UpdateMessage updates the message shown by the spinner if it is running.
// If the spinner is not running, UpdateMessage does nothing.
func (t *tracker) UpdateMessage(msg string) {
	if t.disableSpinner {
		t.Info(msg)
	} else if t.s != nil {
		t.s.UpdateMessage(msg)
	}
}
