package debugging

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestSafeCounterConcurrent(t *testing.T) {
	var c SafeCounter
	done := make(chan struct{})
	for g := 0; g < 20; g++ {
		go func() { for i := 0; i < 1000; i++ { c.Inc() }; done <- struct{}{} }()
	}
	for g := 0; g < 20; g++ { <-done }
	if got := c.Value(); got != 20_000 { t.Fatalf("counter=%d", got) }
}

func TestCancellableSendDoesNotLeak(t *testing.T) {
	baseline := runtime.NumGoroutine()
	out := make(chan int)
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	CancellableSend(ctx, out, started)
	<-started
	cancel()
	if !WaitUntilGoroutinesBelow(baseline+1, time.Second) {
		t.Fatalf("goroutine likely leaked: baseline=%d current=%d", baseline, runtime.NumGoroutine())
	}
}

func TestRunWorkersExitOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	jobs := make(chan int)
	errs := RunWorkers(ctx, 3, jobs, func(ctx context.Context, job int) error {
		<-ctx.Done()
		return ctx.Err()
	})
	jobs <- 1
	cancel()
	seenCancel := false
	for err := range errs {
		if errors.Is(err, context.Canceled) { seenCancel = true }
	}
	if !seenCancel { t.Fatalf("expected context.Canceled from at least one worker") }
}

func TestRunWorkersExitWhenJobsClosed(t *testing.T) {
	ctx := context.Background()
	jobs := make(chan int)
	errs := RunWorkers(ctx, 2, jobs, func(context.Context, int) error { return nil })
	close(jobs)
	for err := range errs { t.Fatalf("unexpected error: %v", err) }
}

// Manual demonstration. Run explicitly with:
// go test -race -run TestUnsafeCounterRaceManual
func TestUnsafeCounterRaceManual(t *testing.T) {
	if testing.Short() { t.Skip("manual race detector demo") }
	t.Skip("remove this Skip to see race detector report")
	var c UnsafeCounter
	done := make(chan struct{})
	for g := 0; g < 2; g++ { go func(){ for i:=0;i<1000;i++{ c.Inc() }; done<-struct{}{} }() }
	<-done; <-done
}
