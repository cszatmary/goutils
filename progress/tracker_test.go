package progress_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/log"
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
	tracker := &progress.PlainTracker{Logger: log.New(
		log.WithOutput(&buf),
		log.WithFormatter(formatter{}),
		log.WithLevel(log.LevelDebug),
	)}
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

func TestLogWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(
		log.WithOutput(&buf),
		log.WithFormatter(formatter{}),
		log.WithLevel(log.LevelDebug),
	)
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

// formatter is a very basic log.Formatter used in tests.
type formatter struct{}

func (f formatter) Format(e *log.Entry, buf *bytes.Buffer) ([]byte, error) {
	buf.WriteString(e.Level.String())
	buf.WriteByte(' ')
	buf.WriteString(e.Message)

	var fields []string
	for k, v := range e.Fields {
		fields = append(fields, fmt.Sprintf("%s=%v", k, v))
	}
	// Sort fields to make sure they are always in the same order
	// since map iterations are not guaranteed to be in the same order
	sort.Strings(fields)
	for _, f := range fields {
		buf.WriteByte(' ')
		buf.WriteString(f)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
