package progress_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/progress"
)

func TestTrackerFromContext(t *testing.T) {
	tracker := &progress.PlainTracker{}
	ctx := progress.ContextWithTracker(context.Background(), tracker)
	got := progress.TrackerFromContext(ctx)
	if got != tracker {
		t.Errorf("got %+v, want %+v", got, tracker)
	}
}

func TestTrackerFromContextMissing(t *testing.T) {
	got := progress.TrackerFromContext(context.Background())
	want := progress.NoopTracker{}
	if got != want {
		t.Errorf("got %T, want %T", got, want)
	}
}

func TestPlainTracker(t *testing.T) {
	var buf bytes.Buffer
	tracker := &progress.PlainTracker{Logger: &logger{out: &buf}}
	tracker.Info("hello world")
	tracker.Start("doing stuff", 4)
	tracker.WithFields(progress.Fields{"id": "foo"}).Debug("processing...")
	tracker.UpdateMessage("cleaning up")
	tracker.Stop()

	got := buf.String()
	want := `info hello world
info doing stuff count=4
debug processing... id=foo
info cleaning up
`
	if got != want {
		t.Errorf("got logs\n\t%s\nwant\n\t%s", got, want)
	}
}

func TestSpinnerTracker(t *testing.T) {
	var buf bytes.Buffer
	tracker := &progress.SpinnerTracker{
		OutputLogger: &logger{out: &buf},
		Interval:     10 * time.Millisecond,
	}
	tracker.Info("hello world")
	tracker.Start("doing stuff", 2)
	time.Sleep(15 * time.Millisecond)
	tracker.WithFields(progress.Fields{"id": "foo"}).Debug("processing...")
	tracker.Inc()
	tracker.UpdateMessage("cleaning up")
	time.Sleep(15 * time.Millisecond)
	tracker.Stop()

	// wait a bit because the spinner still has to erase before stopping
	time.Sleep(25 * time.Millisecond)
	got := buf.String()

	// Should be at least 3 frames
	wantFrames := "⠋⠙⠹"
	if !containsAll(got, wantFrames) {
		t.Errorf("got %q, want to contain all %q", got, wantFrames)
	}

	wantMsgs := []string{
		"info hello world\n",
		"doing stuff (0/2)",
		"debug processing... id=foo\n",
		"cleaning up (1/2)",
	}
	for _, wantMsg := range wantMsgs {
		if !strings.Contains(got, wantMsg) {
			t.Errorf("got %q, want to contain %q", got, wantMsg)
		}
	}
}

func TestLogWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := &logger{out: &buf}
	w := progress.LogWriter(logger, logger.WithFields(progress.Fields{"id": "foo"}).Debug)
	t.Cleanup(func() {
		w.Close()
	})

	if _, err := io.WriteString(w, "hello world\n"); err != nil {
		t.Fatalf("failed to write to log writer: %v", err)
	}
	if _, err := io.WriteString(w, "adding some more stuff\n"); err != nil {
		t.Fatalf("failed to write to log writer: %v", err)
	}

	// Sleep to make sure the logs have time to be written since it is running
	// on a separate goroutine
	time.Sleep(100 * time.Millisecond)
	got := buf.String()
	want := "debug hello world id=foo\ndebug adding some more stuff id=foo\n"
	if got != want {
		t.Errorf("got logs\n\t%s\nwant\n\t%s", got, want)
	}
}

func containsAll(s string, chars string) bool {
	for _, r := range chars {
		if !strings.ContainsRune(s, r) {
			return false
		}
	}
	return true
}

// Very basic logger implemenation for tests

type logger struct {
	mu  sync.Mutex
	out io.Writer
	buf bytes.Buffer
}

func (l *logger) newEntry() *entry {
	return &entry{l, nil}
}

func (l *logger) WithFields(fields progress.Fields) progress.Logger {
	return l.newEntry().WithFields(fields)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.newEntry().Debugf(format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.newEntry().Infof(format, args...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.newEntry().Warnf(format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.newEntry().Errorf(format, args...)
}

func (l *logger) Debug(args ...interface{}) {
	l.newEntry().Debug(args...)
}

func (l *logger) Info(args ...interface{}) {
	l.newEntry().Info(args...)
}

func (l *logger) Warn(args ...interface{}) {
	l.newEntry().Warn(args...)
}

func (l *logger) Error(args ...interface{}) {
	l.newEntry().Error(args...)
}

func (l *logger) Output() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out
}

func (l *logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

type entry struct {
	logger *logger
	data   progress.Fields
}

func (e *entry) WithFields(fields progress.Fields) progress.Logger {
	data := make(progress.Fields, len(e.data)+len(fields))
	for k, v := range e.data {
		data[k] = v
	}
	for k, v := range fields {
		data[k] = v
	}
	return &entry{e.logger, data}
}

func (e *entry) Debugf(format string, args ...interface{}) {
	e.log("debug", fmt.Sprintf(format, args...))
}

func (e *entry) Infof(format string, args ...interface{}) {
	e.log("info", fmt.Sprintf(format, args...))
}

func (e *entry) Warnf(format string, args ...interface{}) {
	e.log("warn", fmt.Sprintf(format, args...))
}

func (e *entry) Errorf(format string, args ...interface{}) {
	e.log("error", fmt.Sprintf(format, args...))
}

func (e *entry) Debug(args ...interface{}) {
	e.log("debug", fmt.Sprint(args...))
}

func (e *entry) Info(args ...interface{}) {
	e.log("info", fmt.Sprint(args...))
}

func (e *entry) Warn(args ...interface{}) {
	e.log("warn", fmt.Sprint(args...))
}

func (e *entry) Error(args ...interface{}) {
	e.log("error", fmt.Sprint(args...))
}

func (e *entry) log(level, msg string) {
	e.logger.mu.Lock()
	defer e.logger.mu.Unlock()
	b := &e.logger.buf
	b.Reset()
	b.WriteString(level)
	b.WriteByte(' ')
	b.WriteString(msg)

	var fields []string
	for k, v := range e.data {
		fields = append(fields, fmt.Sprintf("%s=%v", k, v))
	}
	// Sort fields to make sure they are always in the same order
	// since map iterations are not guaranteed to be in the same order
	sort.Strings(fields)
	for _, f := range fields {
		b.WriteByte(' ')
		b.WriteString(f)
	}
	b.WriteByte('\n')
	_, _ = b.WriteTo(e.logger.out)
}
