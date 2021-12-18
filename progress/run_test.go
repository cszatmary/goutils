package progress_test

import (
	"bytes"
	"context"
	"sort"
	"testing"
	"time"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/goutils/progress"
)

const errOops errors.String = "oops"

func TestRun(t *testing.T) {
	var buf bytes.Buffer
	tracker := &mockSpinnerTracker{logger: &logger{out: &buf}}
	ctx := progress.ContextWithTracker(context.Background(), tracker)
	err := progress.Run(ctx, progress.RunOptions{
		Message: "performing operation",
	}, func(ctx context.Context) error {
		if !tracker.active {
			t.Error("want tracker to be running, but isn't")
		}

		tracker.Debug("doing stuff")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.active {
		t.Error("want tracker to be stopped, but isn't")
	}
	gotLogs := buf.String()
	wantLogs := "info performing operation\ndebug doing stuff\n"
	if gotLogs != wantLogs {
		t.Errorf("got logs\n\t%s\nwant\n\t%s", gotLogs, wantLogs)
	}
}

func TestRunError(t *testing.T) {
	tests := []struct {
		name    string
		runFn   progress.RunFunc
		wantErr error
	}{
		{
			name: "error from run func",
			runFn: func(ctx context.Context) error {
				return errOops
			},
			wantErr: errOops,
		},
		{
			name: "timeout",
			runFn: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
				case <-time.After(10 * time.Millisecond):
				}
				return nil
			},
			wantErr: context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tracker := &mockSpinnerTracker{logger: &logger{out: &buf}}
			ctx := progress.ContextWithTracker(context.Background(), tracker)
			err := progress.Run(ctx, progress.RunOptions{
				Message: "performing operation",
				Timeout: 5 * time.Millisecond,
			}, tt.runFn)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err\n\t%v\nwant\n\t%v", err, tt.wantErr)
			}
		})
	}
}

func TestRunParallel(t *testing.T) {
	var buf bytes.Buffer
	tracker := &mockSpinnerTracker{logger: &logger{out: &buf}}
	ctx := progress.ContextWithTracker(context.Background(), tracker)
	outCh := make(chan int, 3)
	err := progress.RunParallel(ctx, progress.RunParallelOptions{
		Message: "performing operation",
		Count:   3,
	}, func(ctx context.Context, i int) error {
		outCh <- i
		return nil
	})
	close(outCh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracker.active {
		t.Error("want tracker to be stopped, but isn't")
	}
	var vals []int
	for i := range outCh {
		vals = append(vals, i)
	}
	sort.Ints(vals)
	if len(vals) != 3 {
		t.Errorf("got %d values, want 3", len(vals))
	}
	if vals[0] != 0 || vals[1] != 1 || vals[2] != 2 {
		t.Errorf("got %v, want [0 1 2]", vals)
	}
}

func TestRunParallelError(t *testing.T) {
	tests := []struct {
		name    string
		runFn   progress.RunParallelFunc
		wantErr error
	}{
		{
			name: "error from run func",
			runFn: func(ctx context.Context, i int) error {
				return errOops
			},
			wantErr: errOops,
		},
		{
			name: "timeout",
			runFn: func(ctx context.Context, i int) error {
				select {
				case <-ctx.Done():
				case <-time.After(10 * time.Millisecond):
				}
				return nil
			},
			wantErr: context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tracker := &mockSpinnerTracker{logger: &logger{out: &buf}}
			ctx := progress.ContextWithTracker(context.Background(), tracker)
			err := progress.RunParallel(ctx, progress.RunParallelOptions{
				Message:       "performing operation",
				Count:         3,
				CancelOnError: true,
				Timeout:       5 * time.Millisecond,
			}, tt.runFn)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err\n\t%v\nwant\n\t%v", err, tt.wantErr)
			}
		})
	}
}

func TestRunParallelMultipleErrors(t *testing.T) {
	var buf bytes.Buffer
	tracker := &mockSpinnerTracker{logger: &logger{out: &buf}}
	ctx := progress.ContextWithTracker(context.Background(), tracker)
	err := progress.RunParallel(ctx, progress.RunParallelOptions{
		Message: "performing operation",
		Count:   3,
	}, func(ctx context.Context, i int) error {
		return errors.String("failed")
	})
	var errList errors.List
	if !errors.As(err, &errList) {
		t.Errorf("got err type %T, want %T", err, errList)
	}
	if len(errList) != 3 {
		t.Errorf("got %d errors, want 3", len(errList))
	}
}

type mockSpinnerTracker struct {
	*logger

	count  int
	i      int
	active bool
}

func (t *mockSpinnerTracker) Start(message string, count int) {
	t.count = count
	t.i = 0
	t.active = true
	t.logger.Info(message)
}

func (t *mockSpinnerTracker) Stop() {
	t.active = false
}

func (t *mockSpinnerTracker) Inc() {
	t.i++
}

func (t *mockSpinnerTracker) UpdateMessage(m string) {
	t.logger.Info(m)
}
