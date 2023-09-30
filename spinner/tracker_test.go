package spinner_test

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cszatmary/goutils/logutil"
	"github.com/cszatmary/goutils/spinner"
)

func TestSpinnerTracker(t *testing.T) {
	var b bytes.Buffer
	tracker := spinner.NewTracker(spinner.TrackerOptions{
		Writer:   &b,
		Interval: 10 * time.Millisecond,
		NewHandler: func(w io.Writer) slog.Handler {
			return slog.NewTextHandler(w, &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: logutil.RemoveKeys(slog.TimeKey),
			})
		},
	})
	tracker.Info("hello world")
	tracker.Start("doing stuff", 2)
	time.Sleep(15 * time.Millisecond)
	tracker.WithAttrs("id", "foo").Debug("processing...")
	tracker.Inc()
	tracker.UpdateMessage("cleaning up")
	time.Sleep(15 * time.Millisecond)
	tracker.Stop()

	// wait a bit because the spinner still has to erase before stopping
	time.Sleep(25 * time.Millisecond)
	got := b.String()

	// Should be at least 3 frames
	wantFrames := "⠋⠙⠹"
	if !containsAll(got, wantFrames) {
		t.Errorf("got %q, want to contain all %q", got, wantFrames)
	}

	wantMsgs := []string{
		"level=INFO msg=\"hello world\"\n",
		"doing stuff (0/2)",
		"level=DEBUG msg=processing... id=foo\n",
		"cleaning up (1/2)",
	}
	for _, wantMsg := range wantMsgs {
		if !strings.Contains(got, wantMsg) {
			t.Errorf("got %q, want to contain %q", got, wantMsg)
		}
	}
}

func TestTrackerDisableSpinner(t *testing.T) {
	var b bytes.Buffer
	tracker := spinner.NewTracker(spinner.TrackerOptions{
		Writer: &b,
		NewHandler: func(w io.Writer) slog.Handler {
			return slog.NewTextHandler(w, &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: logutil.RemoveKeys(slog.TimeKey),
			})
		},
		DisableSpinner: true,
	})
	tracker.Info("hello world")
	tracker.Start("doing stuff", 4)
	tracker.WithAttrs("id", "foo").Debug("processing...")
	tracker.UpdateMessage("cleaning up")
	tracker.Stop()

	want := `level=INFO msg="hello world"
level=INFO msg="doing stuff" count=4
level=DEBUG msg=processing... id=foo
level=INFO msg="cleaning up"
`
	if got := b.String(); got != want {
		t.Errorf("\ngot logs\n\t%s\nwant\n\t%s", got, want)
	}
}
