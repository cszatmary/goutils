// Package spinner provides a spinner that can be used to display progress to a user
// in a command line application.
package spinner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var frames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner represents the state of the spinner. A spinner can be created
// using the spinner.New function.
//
// Spinner can keep track of and display progress through a list of items
// that need to be completed.
//
// It is safe to use a Spinner across multiple goroutines.
// The spinner will ensure only one goroutine at a time can modify it.
//
// The Spinner runs on a separate goroutine so that blocking operations can
// be run on the current goroutine and the Spinner will continue displaying progress.
//
// Spinner implements the io.Writer interface. It can be written to in order
// to print messages while the spinner is running. It is not recommended to
// write directly to the writer the spinner is writing to (by default stderr),
// as it can cause issues with the spinner animation. Writing to the spinner
// directly provides a way to get around this limitation, as the spinner will
// ensure that the text will be written properly without interfering with the animation.
type Spinner struct {
	interval time.Duration
	w        io.Writer
	mu       *sync.Mutex
	// stopChan is used to stop the spinner
	stopChan chan struct{}
	active   bool
	// last string written to out
	lastOutput string
	startMsg   string
	stopMsg    string
	// msg written on each frame
	msg string
	// total number of items
	count int
	// number of items completed
	completed int
	maxMsgLen int
	// buffer to keep track of message to write to w
	// these will be written on each call of erase
	msgBuf      *bytes.Buffer
	persistMsgs bool
}

// New creates a new spinner instance using the given options.
func New(opts ...Option) *Spinner {
	s := &Spinner{
		interval: 100 * time.Millisecond,
		w:        os.Stderr,
		mu:       &sync.Mutex{},
		stopChan: make(chan struct{}, 1),
		active:   false,
		// default to 1 since we don't show progress on 1 anyway
		count:     1,
		maxMsgLen: 80,
		msgBuf:    &bytes.Buffer{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option is a function that takes a spinner and applies
// a configuration to it.
type Option func(*Spinner)

// WithInterval sets how often the spinner updates.
// This controls the speed of the spinner.
// By default the interval is 100ms.
func WithInterval(d time.Duration) Option {
	return func(s *Spinner) {
		s.interval = d
	}
}

// WithWriter sets the writer that should be used for writing the spinner to.
func WithWriter(w io.Writer) Option {
	return func(s *Spinner) {
		s.w = w
	}
}

// WithStartMessage sets a string that should be written after the spinner
// when the spinnner is started.
func WithStartMessage(m string) Option {
	return func(s *Spinner) {
		s.startMsg = m
	}
}

// WithStopMessage sets a string that should be written when the spinner is stopped.
// This message will replace the spinner.
func WithStopMessage(m string) Option {
	return func(s *Spinner) {
		s.stopMsg = m
	}
}

// WithCount sets the total number of items to track the progress of.
func WithCount(c int) Option {
	return func(s *Spinner) {
		s.count = c
	}
}

// WithMaxMessageLength sets the maximum length of the message that is written
// by the spinner. If the message is longer then this length it will be truncated.
// The default max length is 80.
func WithMaxMessageLength(l int) Option {
	return func(s *Spinner) {
		s.maxMsgLen = l
	}
}

// WithPersistMessages sets whether or not messages should be persisted to the writter
// when the message is updated. By default messages are not persisted and are replaced.
func WithPersistMessages(b bool) Option {
	return func(s *Spinner) {
		s.persistMsgs = b
	}
}

// Start will start the spinner.
// If the spinner is already running, Start will do nothing.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.setMsg(s.startMsg)
	s.mu.Unlock()
	go s.run()
}

// Stop stops the spinner if it is currently running.
// If the spinner is not running, Stop will do nothing.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}

	s.active = false
	s.stopChan <- struct{}{}
	// Persist last msg before we do the final erase.
	// Need to do this manually since we aren't using setMsg
	s.persistMsg()
	s.erase()
	if s.stopMsg != "" {
		// Make sure there's a trailing newline
		if s.stopMsg[len(s.stopMsg)-1] != '\n' {
			s.stopMsg += "\n"
		}
		fmt.Fprint(s.w, s.stopMsg)
	}
}

// Inc increments the progress of the spinner. If the spinner
// has already reached full progress, Inc does nothing.
func (s *Spinner) Inc() {
	s.IncWithMessage("")
}

// IncWithMessage increments the progress of the spinner and updates
// the spinner message to m. If the spinner has already reached
// full progress, IncWithMessage does nothing.
func (s *Spinner) IncWithMessage(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.completed >= s.count {
		return
	}
	s.completed++
	s.setMsg(m)
}

// IncWithMessagef increments the progress of the spinner and updates
// the spinner message to the format specifier. If the spinner has already
// reached full progress, IncWithMessagef does nothing.
func (s *Spinner) IncWithMessagef(format string, args ...interface{}) {
	s.IncWithMessage(fmt.Sprintf(format, args...))
}

// UpdateMessage changes the current message being shown by the spinner.
func (s *Spinner) UpdateMessage(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setMsg(m)
}

// setMsg sets the spinner message to m. If m is longer then s.maxMsgLen it will
// be truncated. If m is empty, setMsg will do nothing.
// The caller must already hold s.lock.
func (s *Spinner) setMsg(m string) {
	if m == "" {
		return
	}
	// Make sure there is no trailing newline or it will mess up the spinner
	if m[len(m)-1] == '\n' {
		m = m[:len(m)-1]
	}
	// Truncate msg if it's too long
	const ellipses = "..."
	if len(m)-len(ellipses) > s.maxMsgLen {
		m = m[:s.maxMsgLen-len(ellipses)] + ellipses
	}
	// Make sure message has a leading space to pad between it and the spinner icon
	if m[0] != ' ' {
		m = " " + m
	}
	s.persistMsg()
	s.msg = m
}

// persistMsg will handle persisting msg if required. The caller must already hold s.lock.
func (s *Spinner) persistMsg() {
	if !s.persistMsgs || s.msg == "" {
		return
	}
	// The message should always be written on it's own line so make sure there is a newline before
	if s.msgBuf.Len() > 0 && s.msgBuf.Bytes()[s.msgBuf.Len()-1] != '\n' {
		s.msgBuf.WriteByte('\n')
	}
	// Drop first char since it's a space
	s.msgBuf.WriteString(s.msg[1:])
	s.msgBuf.WriteByte('\n')
}

// Write writes p to the spinner's writer after the current frame has been erased.
// Write will always immediately return successfully as p is first written to an internal buffer.
// The actual writing of the data to the spinner's writer will not occur until the appropriate time
// during the spinner animation.
//
// Write will add a newline to the end of p in order to ensure that it does not interfere with
// the spinner animation.
func (s *Spinner) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.msgBuf.Write(p)
}

// run runs the spinner. It should be called in a separate goroutine because
// it will run forever until it receives a value on s.stopChan.
func (s *Spinner) run() {
	for {
		for i := 0; i < len(frames); i++ {
			select {
			case <-s.stopChan:
				return
			default:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				s.erase()

				line := fmt.Sprintf("\r%s%s ", frames[i], s.msg)
				if s.count > 1 {
					line += fmt.Sprintf("(%d/%d) ", s.completed, s.count)
				}
				fmt.Fprint(s.w, line)
				s.lastOutput = line
				// Store interval in a var because we unlock the mutex
				// before sleeping so we can't read properties from s
				d := s.interval

				s.mu.Unlock()
				time.Sleep(d)
			}
		}
	}
}

// erase deletes written characters. The caller must already hold s.lock.
func (s *Spinner) erase() {
	n := utf8.RuneCountInString(s.lastOutput)
	if runtime.GOOS == "windows" {
		clearString := "\r" + strings.Repeat(" ", n) + "\r"
		fmt.Fprint(s.w, clearString)
	} else {
		// "\033[K" for macOS Terminal
		for _, c := range []string{"\b", "\127", "\b", "\033[K"} {
			fmt.Fprint(s.w, strings.Repeat(c, n))
		}
		// erases to end of line
		fmt.Fprint(s.w, "\r\033[K")
	}

	if s.msgBuf.Len() > 0 {
		if s.msgBuf.Bytes()[s.msgBuf.Len()-1] != '\n' {
			s.msgBuf.WriteByte('\n')
		}
		// Ignore error because there's nothing we can really do about it
		_, _ = s.msgBuf.WriteTo(s.w)
	}
	s.lastOutput = ""
}
