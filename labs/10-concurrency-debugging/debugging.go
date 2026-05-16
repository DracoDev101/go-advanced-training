package debugging

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// UnsafeCounter intentionally has a data race when Inc is called concurrently.
// It is used for manual race-detector demonstration, not in normal tests.
type UnsafeCounter struct{ n int }

func (c *UnsafeCounter) Inc() { c.n++ }
func (c *UnsafeCounter) Value() int { return c.n }

type SafeCounter struct {
	mu sync.Mutex
	n  int
}

func (c *SafeCounter) Inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *SafeCounter) Value() int { c.mu.Lock(); defer c.mu.Unlock(); return c.n }

// BlockedSend leaks until the receiver is ready because the send has no cancel path.
func BlockedSend(out chan<- int, started chan<- struct{}) {
	go func() {
		close(started)
		out <- 1
	}()
}

// CancellableSend is the production version: every potentially blocking send has ctx.Done.
func CancellableSend(ctx context.Context, out chan<- int, started chan<- struct{}) {
	go func() {
		close(started)
		select {
		case out <- 1:
		case <-ctx.Done():
			return
		}
	}()
}

// RunWorkers exits when jobs is closed or ctx is canceled. This is the worker lifecycle shape
// expected in Production Job Runner.
func RunWorkers(ctx context.Context, workers int, jobs <-chan int, handle func(context.Context, int) error) <-chan error {
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				case job, ok := <-jobs:
					if !ok { return }
					if err := handle(ctx, job); err != nil { errs <- err; return }
				}
			}
		}()
	}
	go func(){ wg.Wait(); close(errs) }()
	return errs
}

func ContendedMutex(iterations, goroutines int) int {
	var mu sync.Mutex
	n := 0
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				mu.Lock()
				n++
				// Keep the critical section non-trivial so mutex profile has something to see.
				runtime.Gosched()
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return n
}

func WaitUntilGoroutinesBelow(limit int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= limit { return true }
		runtime.GC()
		time.Sleep(5 * time.Millisecond)
	}
	return runtime.NumGoroutine() <= limit
}
