package progress

import (
	"context"
	"runtime"
	"time"

	"github.com/cszatmary/goutils/async"
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
	// TrackerKey can be used to specify a custom context key for retrieving a Tracker.
	// This should be used if ContextWithTrackerUsingKey was used.
	// If omitted, the default key will be used.
	TrackerKey any
}

// RunFunc is a function run by Run. ctx should be passed to any operations
// that take a Context to ensure that timeouts and cancellations are propagated.
type RunFunc func(ctx context.Context) error

// Run runs fn. If ctx contains a Tracker, it will be used to display progress.
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

	tracker := TrackerFromContextUsingKey(ctx, opts.TrackerKey)
	tracker.Start(opts.Message, opts.Count)
	defer tracker.Stop()
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	return fn(ctx)
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
	// CancelOnError determines how RunParallel should behave if a function returns an error.
	//
	// If true, Run will cancel all other running functions when it receives an error and
	// return that first error. Note that RunParallel will still wait for all running functions
	// to complete. This way you can guarantee that when RunParallel returns all concurrent operations
	// have stopped.
	//
	// If false, RunParallel will let the other functions continue and will return an errors.List
	// containing any errors from each function.
	//
	// This option only applies if Count > 1.
	CancelOnError bool
	// Timeout sets a timeout after which any running functions will be cancelled.
	// Defaults to 10min if omitted.
	Timeout time.Duration
	// TrackerKey can be used to specify a custom context key for retrieving a Tracker.
	// This should be used if ContextWithTrackerUsingKey was used.
	// If omitted, the default key will be used.
	TrackerKey any
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
// from each run fn. The slice will be sorted based on the order the functions were called.
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

	tracker := TrackerFromContextUsingKey(ctx, opts.TrackerKey)
	tracker.Start(opts.Message, opts.Count)
	defer tracker.Stop()

	var group async.Group[T]
	group.SetLocking(false)
	group.SetMaxGoroutines(opts.Concurrency)
	group.SetCancelOnError(opts.CancelOnError)
	group.SetTimeout(opts.Timeout)
	for i := 0; i < opts.Count; i++ {
		i := i // https://go.dev/doc/faq#closures_and_goroutines
		group.Queue(func(ctx context.Context) (T, error) {
			v, err := fn(ctx, i)
			tracker.Inc()
			return v, err
		})
	}
	return group.Wait(ctx)
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
