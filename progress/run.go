package progress

import (
	"context"
	"runtime"
	"time"

	"github.com/TouchBistro/goutils/errors"
)

const defaultTimeout = 10 * time.Minute

// RunOptions is used to customize how Run behaves.
// All fields are optional and have defaults.
type RunOptions struct {
	// Message is the message that will be passed to Tracker.Start.
	// If omitted no message will be written by the Tracker.
	Message string
	// Count is the count passed to Tracker.Start to track the number of operations.
	// Run will not automatically increment the progress count, instead it is
	// up to the RunFunc to call Tracker.Inc.
	// If omitted it will be 0, i.e. no count.
	Count int
	// Timeout sets a timeout after which the running function will be cancelled.
	// Defaults to 10min if omitted.
	Timeout time.Duration
}

// RunFunc is a function run by Run. ctx should be passed to any operations
// that take a Context to ensure that timeouts and cancellations are propagated.
type RunFunc func(ctx context.Context) error

// Run runs fn. If ctx contains a Tracker, it will be used to display progress.
// fn will be run on a separate goroutine so that timeouts can be enforced.
//
// opts can be used to customize the behaviour of Run. See each option for more details.
func Run(ctx context.Context, opts RunOptions, fn RunFunc) error {
	_, err := RunT(ctx, opts, func(ctx context.Context) (struct{}, error) {
		err := fn(ctx)
		return struct{}{}, err
	})
	return err
}

// RunFuncT is like RunFunc but allows returning a value.
type RunFuncT[T any] func(ctx context.Context) (T, error)

// RunT is like Run but returns a value.
func RunT[T any](ctx context.Context, opts RunOptions, fn RunFuncT[T]) (T, error) {
	if opts.Timeout == 0 {
		// Always provide a timeout to make sure the program doesn't hang and run forever.
		opts.Timeout = defaultTimeout
	}

	tracker := TrackerFromContext(ctx)
	defer tracker.Stop()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	tracker.Start(opts.Message, opts.Count)
	resCh := make(chan result[T], 1)
	go func() {
		t, err := fn(ctx)
		resCh <- result[T]{t, err}
	}()

	var t T
	select {
	case res := <-resCh:
		if res.err != nil {
			return res.t, res.err
		}
		t = res.t
	// Handle the context being cancelled or the deadline being exceeded.
	case <-ctx.Done():
		return t, ctx.Err()
	}
	return t, nil
}

// RunParallelOptions is used to customize how RunParallel behaves.
// All fields are optional and have defaults.
type RunParallelOptions struct {
	// Message is the message that will be passed to Tracker.Start.
	// If omitted no message will be written by the Tracker.
	Message string
	// Count is the number of times the function will be run.
	// It is passed to Tracker.Start to keep track of progress.
	// If omitted or explicitly set to 0, RunParallel will no-op.
	Count int
	// Concurrency controls how many goroutines can run concurrently.
	// Defaults to runtime.NumCPU if omitted.
	Concurrency int
	// CancelOnError determines how Run should behave if a function returns an error.
	// If true, Run will immediately return an error and cancel all other running functions.
	// If false, Run will let the other functions continue and will return an errors.List
	// containing any errors from each function.
	//
	// This option only applies if Count > 1.
	CancelOnError bool
	// Timeout sets a timeout after which any running functions will be cancelled.
	// Defaults to 10min if omitted.
	Timeout time.Duration
}

// RunParallelFunc is a function run by RunParallel. ctx should be passed to any operations
// that take a Context to ensure that timeouts and cancellations are propagated.
//
// i is the the index of this function invocation.
type RunParallelFunc func(ctx context.Context, i int) error

// RunParallel runs fn multiple times concurrently.
// If ctx contains a Tracker, it will be used to display progress.
// Each call to fn will happen in a separate goroutine.
// RunParallel will block until all calls to fn have returned.
//
// opts can be used to customize the behaviour of RunParallel. See each option for more details.
func RunParallel(ctx context.Context, opts RunParallelOptions, fn RunParallelFunc) error {
	_, err := RunParallelT(ctx, opts, func(ctx context.Context, i int) (struct{}, error) {
		err := fn(ctx, i)
		return struct{}{}, err
	})
	return err
}

// RunParallelFuncT is like RunParallelFunc but allows returning a value.
type RunParallelFuncT[T any] func(ctx context.Context, i int) (T, error)

// RunParallelT is like RunParallel but returns a slice containing all the return values
// from each run fn.
func RunParallelT[T any](ctx context.Context, opts RunParallelOptions, fn RunParallelFuncT[T]) ([]T, error) {
	// No-op if count is zero since we have nothing to run.
	if opts.Count < 1 {
		return nil, nil
	}
	if opts.Timeout == 0 {
		// Always provide a timeout to make sure the program doesn't hang and run forever.
		opts.Timeout = defaultTimeout
	}
	if opts.Concurrency < 1 {
		opts.Concurrency = DefaultConcurrency()
	}

	tracker := TrackerFromContext(ctx)
	defer tracker.Stop()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	tracker.Start(opts.Message, opts.Count)
	resCh := make(chan result[T], opts.Count)
	semCh := make(chan struct{}, opts.Concurrency)
	for i := 0; i < opts.Count; i++ {
		semCh <- struct{}{}
		go func(i int) {
			defer func() {
				<-semCh
			}()
			t, err := fn(ctx, i)
			resCh <- result[T]{t, err}
			tracker.Inc()
		}(i)
	}

	var t []T
	var errs errors.List
	for i := 0; i < opts.Count; i++ {
		select {
		case res := <-resCh:
			// No error occurred, continue
			if res.err == nil {
				t = append(t, res.t)
				continue
			}
			// Handle error based on how RunParallel was configured.
			if opts.CancelOnError {
				// Bail and cancel any runs that are in progress.
				cancel()
				return nil, res.err
			}
			// Continue and keep track of the error.
			errs = append(errs, res.err)
		// Handle the context being cancelled or the deadline being exceeded.
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return t, nil
}

// result is a small helper type to combine the return values from a RunFuncT or RunParallelFuncT
// so it can be sent through a channel.
type result[T any] struct {
	t   T
	err error
}

// DefaultConcurrency returns default concurrency that should be used for parallel operations
// by using runtime.NumCPU.
func DefaultConcurrency() int {
	// Check for negative number just to be safe since the type is int.
	// Better safe than sorry and having an overflow.
	if numCPUs := runtime.NumCPU(); numCPUs > 0 {
		return numCPUs
	}
	// If we get here somehow just execute everything serially.
	return 1
}
