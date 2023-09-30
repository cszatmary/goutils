package progress_test

import (
	"context"
	"io"
	"testing"

	"github.com/cszatmary/goutils/progress"
)

func TestTrackerFromContext(t *testing.T) {
	tracker := newMockTracker(io.Discard)
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

func TestTrackerFromContextWithKey(t *testing.T) {
	tracker := newMockTracker(io.Discard)
	type customKey struct{}
	key := customKey{}
	ctx := progress.ContextWithTrackerUsingKey(context.Background(), tracker, key)
	got := progress.TrackerFromContextUsingKey(ctx, key)
	if got != tracker {
		t.Errorf("got %+v, want %+v", got, tracker)
	}
}

func TestTrackerFromContextUsingKeyMissing(t *testing.T) {
	type customKey struct{}
	key := customKey{}
	got := progress.TrackerFromContextUsingKey(context.Background(), key)
	want := progress.NoopTracker{}
	if got != want {
		t.Errorf("got %T, want %T", got, want)
	}
}

func TestTrackerFromContextUsingKeyInvalidPanic(t *testing.T) {
	type customKey struct{}
	key := customKey{}
	ctx := context.WithValue(context.Background(), key, "boom")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, did not happen")
		}
	}()
	progress.TrackerFromContextUsingKey(ctx, key)
}
