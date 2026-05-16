package scheduler

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

type Job int

type Result struct {
	Job    Job
	Worker int
}

func RunWorkerPool(ctx context.Context, workers int, jobs []Job, fn func(context.Context, Job) error) error {
	if workers <= 0 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan Job)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobCh:
					if !ok {
						return
					}
					if err := fn(ctx, job); err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

sendLoop:
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobCh <- job:
		}
	}
	close(jobCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return ctx.Err()
	}
}

func ProcessWithLimit(ctx context.Context, workers int, n int) (int64, error) {
	jobs := make([]Job, n)
	for i := range jobs {
		jobs[i] = Job(i)
	}
	var processed int64
	err := RunWorkerPool(ctx, workers, jobs, func(ctx context.Context, job Job) error {
		atomic.AddInt64(&processed, 1)
		return nil
	})
	if err == context.Canceled || err == context.DeadlineExceeded {
		return processed, err
	}
	return processed, nil
}

func StartLeakySender(ch chan<- int) {
	go func() {
		ch <- 1
	}()
}

func StartCancelableSender(ctx context.Context, ch chan<- int) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case ch <- 1:
		case <-ctx.Done():
		}
	}()
	return done
}

func BusyWork(iterations int) uint64 {
	var x uint64
	for i := 0; i < iterations; i++ {
		x += uint64(i*i) ^ (x << 1)
		if i%1024 == 0 {
			runtime.Gosched()
		}
	}
	return x
}

func NumGoroutine() int { return runtime.NumGoroutine() }
func GOMAXPROCS(n int) int { return runtime.GOMAXPROCS(n) }
