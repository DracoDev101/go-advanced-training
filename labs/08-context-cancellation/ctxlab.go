package ctxlab

import (
	"context"
	"errors"
	"sync"
	"time"
)

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok
}

func WaitForCancel(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func WorkUntilCanceled(ctx context.Context, tick <-chan time.Time) int {
	count := 0
	for {
		select {
		case <-ctx.Done():
			return count
		case <-tick:
			count++
		}
	}
}

func SendWithContext[T any](ctx context.Context, ch chan<- T, v T) error {
	select {
	case ch <- v:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func SlowOperation(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Worker struct {
	jobs <-chan int
	mu sync.Mutex
	processed []int
}

func NewWorker(jobs <-chan int) *Worker { return &Worker{jobs: jobs} }

func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job, ok := <-w.jobs:
			if !ok {
				return nil
			}
			w.mu.Lock()
			w.processed = append(w.processed, job)
			w.mu.Unlock()
		}
	}
}

func (w *Worker) Processed() []int {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]int, len(w.processed))
	copy(out, w.processed)
	return out
}

type ShutdownGroup struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewShutdownGroup(parent context.Context) *ShutdownGroup {
	ctx, cancel := context.WithCancel(parent)
	return &ShutdownGroup{ctx: ctx, cancel: cancel}
}

func (g *ShutdownGroup) Go(fn func(context.Context) error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		_ = fn(g.ctx)
	}()
}

func (g *ShutdownGroup) Shutdown(ctx context.Context) error {
	g.cancel()
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func BackgroundTaskContext(requestCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	id, _ := RequestID(requestCtx)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if id != "" {
		ctx = WithRequestID(ctx, id)
	}
	return ctx, cancel
}

var ErrBadValue = errors.New("business parameter hidden in context")

func BadValueUsage(ctx context.Context) error {
	if ctx.Value("limit") != nil {
		return ErrBadValue
	}
	return nil
}
