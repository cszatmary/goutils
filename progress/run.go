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
	if opts.Timeout == 0 {
		// Always provide a timeout to make sure the program doesn't hang and run forever.
		opts.Timeout = defaultTimeout
	}

	t := TrackerFromContext(ctx)
	defer t.Stop()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	t.Start(opts.Message, 0)
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	// Handle the context being cancelled or the deadline being exceeded.
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
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
	Concurrency uint
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
	// No-op if count is zero since we have nothing to run.
	if opts.Count < 1 {
		return nil
	}
	if opts.Timeout == 0 {
		// Always provide a timeout to make sure the program doesn't hang and run forever.
		opts.Timeout = defaultTimeout
	}
	if opts.Concurrency == 0 {
		opts.Concurrency = DefaultConcurrency()
	}

	t := TrackerFromContext(ctx)
	defer t.Stop()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	t.Start(opts.Message, opts.Count)
	errCh := make(chan error, opts.Count)
	semCh := make(chan struct{}, opts.Concurrency)
	for i := 0; i < opts.Count; i++ {
		semCh <- struct{}{}
		go func(i int) {
			defer func() {
				<-semCh
			}()
			errCh <- fn(ctx, i)
			t.Inc()
		}(i)
	}

	var errs errors.List
	for i := 0; i < opts.Count; i++ {
		select {
		case err := <-errCh:
			// No error occurred, continue
			if err == nil {
				continue
			}
			// Handle error based on how RunParallel was configured.
			if opts.CancelOnError {
				// Bail and cancel any runs that are in progress.
				cancel()
				return err
			}
			// Continue and keep track of the error.
			errs = append(errs, err)
		// Handle the context being cancelled or the deadline being exceeded.
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// DefaultConcurrency returns default concurrency that should be used for parallel operations
// by using runtime.NumCPU.
func DefaultConcurrency() uint {
	// Check for negative number just to be safe since the type is int.
	// Better safe than sorry and having an overflow.
	if numCPUs := runtime.NumCPU(); numCPUs > 0 {
		return uint(numCPUs)
	}
	// If we get here somehow just execute everything serially.
	return 1
}
