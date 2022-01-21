package spinner_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/log"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/goutils/spinner"
)

func TestSpinnerTracker(t *testing.T) {
	var buf bytes.Buffer
	tracker := &spinner.Tracker{
		OutputLogger: log.New(
			log.WithOutput(&buf),
			log.WithFormatter(&log.TextFormatter{DisableTimestamp: true}),
			log.WithLevel(log.LevelDebug),
		),
		Interval: 10 * time.Millisecond,
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
		"level=info message=\"hello world\"\n",
		"doing stuff (0/2)",
		"level=debug message=processing... id=foo\n",
		"cleaning up (1/2)",
	}
	for _, wantMsg := range wantMsgs {
		if !strings.Contains(got, wantMsg) {
			t.Errorf("got %q, want to contain %q", got, wantMsg)
		}
	}
}
