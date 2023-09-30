// Package async provides functionality for working with async operations.
package async

import (
	"context"
	"sync"
	"time"

	"github.com/cszatmary/goutils/errors"
)

// Group is used to manage a group of goroutines that are concurrently running sub-operations
// that are part of the same overall operation.
//
// A zero value Group is a valid Group that has no max goroutines, does not cancel on error,
// and has no timeout.
//
// A Group can be reused after a call to Wait.
//
// A Group must not be copied after first use.
type Group[T any] struct {
	cancelOnErr bool
	timeout     time.Duration

	semCh chan struct{}                      // max goroutines
	funcs []func(context.Context) (T, error) // queued operations
	mu    toggleableMutex
}

// SetLocking controls if a lock should be used on Group methods.
//
// By default Group uses locking to ensure that it is safe to use across multiple goroutines.
// However, if the Group is only be used on a single goroutine this can be unnecessary overhead.
// By passing false the locking can be disabled.
func (g *Group[T]) SetLocking(enabled bool) {
	g.mu.disabled = !enabled
}

// SetMaxGoroutines sets the max number of active goroutines that are allowed.
// If the value is zero or negative, there will be no limit on the number of active goroutines.
func (g *Group[T]) SetMaxGoroutines(n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n > 0 {
		g.semCh = make(chan struct{}, n)
		return
	}
	g.semCh = nil
}

// SetCancelOnError determines how the Group should behave if a goroutine results in an error.
//
// If the value is true, all running goroutines will be cancelled and the first error
// will be returned by Wait.
//
// If the value is false, all other running goroutines will continue and will return an
// errors.List containing any errors from each function.
func (g *Group[T]) SetCancelOnError(b bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cancelOnErr = b
}

// Timeout sets a timeout after which any running goroutines will be cancelled.
// If the value is zero or negative, no timeout will be set.
func (g *Group[T]) SetTimeout(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.timeout = d
}

// Queue queues a function to be run in a goroutine.
// Once all desired functions have been queued, execute them by calling Wait.
func (g *Group[T]) Queue(f func(context.Context) (T, error)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.funcs = append(g.funcs, f)
}

// Wait executes all the queued functions, each of them in their own goroutines, and waits
// for them to complete. It then returns a list of results and any errors that occurred.
//
// The returned results will be in the same order that calls to Queue were made.
// If an error occurred, the result slice will be nil.
//
// If the Group was configured to cancel on the first error, if a goroutine errors all others
// will be cancelled and the returned error will be the first error that occurred.
// Otherwise, all goroutines will run to completion, and the returned error will be an
// errors.List containing each error. The errors will not be in any particular order.
func (g *Group[T]) Wait(ctx context.Context) ([]T, error) {
	// Ensure that the Group is not modified while running.
	// If anything tries to modify the Group it will be blocked until Wait completes.
	g.mu.Lock()
	defer g.mu.Unlock()
	rs, firstErr := g.wait(ctx, false)
	if firstErr != nil {
		if g.cancelOnErr {
			return nil, firstErr
		}

		var errs errors.List
		for _, r := range rs {
			if r.Err != nil {
				errs = append(errs, r.Err)
			}
		}
		return nil, errs
	}

	vs := make([]T, len(rs))
	for i, r := range rs {
		vs[i] = r.Value
	}
	return vs, nil
}

// WaitLax is similar to Wait but returns a slice of Result values containing the returned
// value and error, if any, from each goroutine. This can be useful if you wish to get a list
// of partial results and errors associated with each goroutine.
//
// The CancelOnError option does not apply to WaitLax, since it will always wait for all
// goroutines and return all results.
func (g *Group[T]) WaitLax(ctx context.Context) []Result[T] {
	// Ensure that the Group is not modified while running.
	// If anything tries to modify the Group it will be blocked until Wait completes.
	g.mu.Lock()
	defer g.mu.Unlock()
	rs, _ := g.wait(ctx, true)
	return rs
}

// Result contains the result of a goroutine that was ran. It is returned by Group.WaitLax.
type Result[T any] struct {
	// Value is the value returned from the goroutine.
	Value T
	// Err is the error returned from the goroutine. If no error occurred it will be nil.
	Err error

	i int // used to order the results
}

// wait is the actual implementation of Wait and WaitLax. It runs all the queued operations in separate
// goroutines and collects the results.
// The caller must already hold the lock.
func (g *Group[T]) wait(ctx context.Context, lax bool) (results []Result[T], firstErr error) {
	// See if we need to create a custom context with a timeout or cancellation.
	runCtx := ctx
	var cancel context.CancelFunc
	if g.timeout > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, g.timeout)
	} else if g.cancelOnErr {
		// Create a cancel context if no timeout.
		// If a timeout was provided there will already be a cancellable context.
		runCtx, cancel = context.WithCancel(runCtx)
	}
	if cancel != nil {
		defer cancel()
	}

	// Need a buffered channel to collect the results since we might be blocked on starting
	// some goroutines if we hit the defined limit.
	resCh := make(chan Result[T], len(g.funcs))
	for i, f := range g.funcs {
		if g.semCh != nil {
			g.semCh <- struct{}{}
		}
		go func(i int, f func(context.Context) (T, error)) {
			defer func() {
				if g.semCh != nil {
					<-g.semCh
				}
			}()
			v, err := f(runCtx)
			resCh <- Result[T]{v, err, i}
		}(i, f)
	}

	results = make([]Result[T], len(g.funcs))
	for i := 0; i < len(g.funcs); i++ {
		res := <-resCh
		results[res.i] = res
		if res.Err != nil && firstErr == nil {
			firstErr = res.Err
			if g.cancelOnErr && !lax {
				cancel()
				// Continue because we still want to wait for all running goroutines to finish.
			}
		}
	}

	// Clear the queue so the Group can be reused.
	g.funcs = nil
	return
}

// toggleableMutex is a small wrapper over a sync.Mutex that allows it to be disabled.
// If disabled, calls to Lock and Unlock will no-op.
type toggleableMutex struct {
	mu       sync.Mutex
	disabled bool // disabled so the zero value (false) means the mutex is enabled
}

func (tm *toggleableMutex) Lock() {
	if !tm.disabled {
		tm.mu.Lock()
	}
}

func (tm *toggleableMutex) Unlock() {
	if !tm.disabled {
		tm.mu.Unlock()
	}
}
