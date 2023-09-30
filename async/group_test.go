package async_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cszatmary/goutils/async"
	"github.com/cszatmary/goutils/errors"
)

// Parallel illustrates the use of a Group for synchronizing a simple parallel operation.
func ExampleGroup_parallel() {
	updateService := func(_ context.Context, service string) (string, error) {
		return fmt.Sprintf("service %s updated", service), nil
	}

	services := []string{"A", "B", "C"}
	var g async.Group[string]
	for _, s := range services {
		s := s // https://golang.org/doc/faq#closures_and_goroutines
		g.Queue(func(ctx context.Context) (string, error) {
			return updateService(ctx, s)
		})
	}
	results, err := g.Wait(context.TODO())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, result := range results {
		fmt.Println(result)
	}

	// Output:
	// service A updated
	// service B updated
	// service C updated
}

func TestGroupOrdered(t *testing.T) {
	var g async.Group[int]
	for i := 0; i < 5; i++ {
		i := i
		g.Queue(func(ctx context.Context) (int, error) {
			// Sleep for a bit with the sleep decreasing each iteration to ensure that later
			// queued functions finish first so we can test the returned order is correct.
			millis := time.Duration(50 / (i + 1))
			time.Sleep(millis * time.Millisecond)
			return i, nil
		})
	}
	results, err := g.Wait(context.Background())
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	want := []int{0, 1, 2, 3, 4}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("got %v, want %v", results, want)
	}
}

func TestGroupMultipleErrors(t *testing.T) {
	var g async.Group[int]
	for i := 0; i < 5; i++ {
		i := i
		g.Queue(func(ctx context.Context) (int, error) {
			if i%2 == 1 {
				return -1, fmt.Errorf("error %d", i)
			}
			return i, nil
		})
	}
	results, err := g.Wait(context.Background())
	if results != nil {
		t.Errorf("want nil slice, got %v", results)
	}
	var errList errors.List
	if !errors.As(err, &errList) {
		t.Fatalf("got err type %T, want %T", err, errList)
	}
	if len(errList) != 2 {
		t.Errorf("got %d errors, want 2", len(errList))
	}
	want := []string{"error 1", "error 3"}
	for i, err := range errList {
		if err.Error() != want[i] {
			t.Errorf("got err %v, want %s", err.Error(), want[i])
		}
	}
}

func TestGroupCancelOnError(t *testing.T) {
	var g async.Group[int]
	g.SetCancelOnError(true)
	// First one will error
	firstErr := fmt.Errorf("boom")
	g.Queue(func(ctx context.Context) (int, error) {
		return -1, firstErr
	})
	ch := make(chan int) // unbuffered so it blocks
	errCh := make(chan error, 4)
	for i := 1; i < 5; i++ {
		i := i
		g.Queue(func(ctx context.Context) (int, error) {
			select {
			case ch <- i:
				return i, nil
			case <-ctx.Done():
				errCh <- ctx.Err()
				return -1, ctx.Err()
			}
		})
	}
	results, err := g.Wait(context.Background())
	if results != nil {
		t.Errorf("want nil slice, got %v", results)
	}
	if err != firstErr {
		t.Errorf("got %v, want %v", err, firstErr)
	}
	// Make sure the other goroutines received the cancellation
	close(errCh)
	if len(errCh) != 4 {
		t.Errorf("got channel length of %d, want 4", len(errCh))
	}
	for err := range errCh {
		if err != context.Canceled {
			t.Errorf("got %v, want %v", err, context.Canceled)
		}
	}
}

func TestGroupTimeout(t *testing.T) {
	var group async.Group[int]
	group.SetCancelOnError(true)
	group.SetTimeout(5 * time.Millisecond)
	group.Queue(func(ctx context.Context) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return 10, nil
		}
	})
	results, err := group.Wait(context.Background())
	if results != nil {
		t.Errorf("want nil slice, got %v", results)
	}
	if err != context.DeadlineExceeded {
		t.Errorf("got %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestGroupMaxGoroutines(t *testing.T) {
	const limit = 10
	var g async.Group[int]
	g.SetMaxGoroutines(limit)
	var active int32
	for i := 0; i <= 1<<10; i++ {
		g.Queue(func(_ context.Context) (int, error) {
			n := atomic.AddInt32(&active, 1)
			if n > limit {
				return 0, fmt.Errorf("saw %d active goroutines; want <= %d", n, limit)
			}
			time.Sleep(1 * time.Microsecond) // Give other goroutines a chance to increment active.
			atomic.AddInt32(&active, -1)
			return 0, nil
		})
	}
	if _, err := g.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestGroupWaitLax(t *testing.T) {
	var g async.Group[int]
	g.SetLocking(false)
	for i := 0; i < 5; i++ {
		i := i
		g.Queue(func(ctx context.Context) (int, error) {
			if i%2 == 1 {
				return -1, errors.String(fmt.Sprintf("error %d", i))
			}
			return i, nil
		})
	}
	results := g.WaitLax(context.Background())
	if len(results) != 5 {
		t.Errorf("got %d results, want 5", len(results))
	}

	wantResults := []async.Result[int]{
		{Value: 0, Err: nil},
		{Value: -1, Err: errors.String("error 1")},
		{Value: 2, Err: nil},
		{Value: -1, Err: errors.String("error 3")},
		{Value: 4, Err: nil},
	}
	for i, res := range results {
		want := wantResults[i]
		if res.Value != want.Value {
			t.Errorf("got value %d, want %d", res.Value, want.Value)
		}
		if res.Err != want.Err {
			t.Errorf("got err %v, want %v", res.Err, want.Err)
		}
	}
}
