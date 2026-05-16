package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPoolProcessesJobs(t *testing.T) {
	processed, err := ProcessWithLimit(context.Background(), 4, 100)
	if err != nil {
		t.Fatalf("ProcessWithLimit returned error: %v", err)
	}
	if processed != 100 {
		t.Fatalf("processed: got %d, want 100", processed)
	}
}

func TestWorkerPoolCancelsOnError(t *testing.T) {
	boom := errors.New("boom")
	jobs := make([]Job, 1000)
	for i := range jobs {
		jobs[i] = Job(i)
	}
	var processed int64
	err := RunWorkerPool(context.Background(), 4, jobs, func(ctx context.Context, job Job) error {
		atomic.AddInt64(&processed, 1)
		if job == 10 {
			return boom
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err: got %v, want boom", err)
	}
	if processed >= int64(len(jobs)) {
		t.Fatalf("expected cancellation before all jobs processed, processed=%d", processed)
	}
}

func TestCancelableSenderExitsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan int)
	done := StartCancelableSender(ctx, ch)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelable sender did not exit")
	}
}

func TestGoroutineCountDoesNotGrowAfterCancelableSenders(t *testing.T) {
	before := NumGoroutine()
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := StartCancelableSender(ctx, make(chan int))
		cancel()
		<-done
	}
	after := NumGoroutine()
	if after > before+5 {
		t.Fatalf("goroutine count grew too much: before=%d after=%d", before, after)
	}
}

func TestBusyWorkProducesValue(t *testing.T) {
	if got := BusyWork(1000); got == 0 {
		t.Fatal("BusyWork returned zero")
	}
}
