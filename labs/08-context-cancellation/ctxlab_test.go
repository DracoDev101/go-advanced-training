package ctxlab

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestParentCancelPropagatesToChild(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	child, childCancel := context.WithCancel(parent)
	defer childCancel()
	cancel()
	if err := WaitForCancel(child); !errors.Is(err, context.Canceled) {
		t.Fatalf("child err: %v", err)
	}
}

func TestChildCancelDoesNotCancelParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	child, childCancel := context.WithCancel(parent)
	childCancel()
	if err := child.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("child err: %v", err)
	}
	if err := parent.Err(); err != nil {
		t.Fatalf("parent should not be canceled, got %v", err)
	}
}

func TestTimeoutReturnsDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err := SlowOperation(ctx, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v want deadline exceeded", err)
	}
}

func TestWorkerExitsOnCancel(t *testing.T) {
	jobs := make(chan int, 4)
	jobs <- 1
	jobs <- 2
	ctx, cancel := context.WithCancel(context.Background())
	w := NewWorker(jobs)
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("worker err: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not exit")
	}
	if got := w.Processed(); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("processed %v", got)
	}
}

func TestSendWithContextAvoidsBlockedSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan int)
	cancel()
	if err := SendWithContext(ctx, ch, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("err: %v", err)
	}
}

func TestTypedRequestIDKey(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	id, ok := RequestID(ctx)
	if !ok || id != "req-123" {
		t.Fatalf("request id got %q ok=%v", id, ok)
	}
}

func TestBackgroundTaskCopiesMetadataButNotCancellation(t *testing.T) {
	reqCtx, reqCancel := context.WithCancel(WithRequestID(context.Background(), "req-1"))
	bgCtx, bgCancel := BackgroundTaskContext(reqCtx, time.Second)
	defer bgCancel()
	reqCancel()
	if err := bgCtx.Err(); err != nil {
		t.Fatalf("background ctx should not be canceled by request ctx: %v", err)
	}
	if id, _ := RequestID(bgCtx); id != "req-1" {
		t.Fatalf("request id not copied: %q", id)
	}
}

func TestShutdownGroupStopsWorkers(t *testing.T) {
	g := NewShutdownGroup(context.Background())
	started := make(chan struct{})
	g.Go(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := g.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown err: %v", err)
	}
}

func TestBadValueUsageDetectsHiddenBusinessParam(t *testing.T) {
	ctx := context.WithValue(context.Background(), "limit", 100)
	if !errors.Is(BadValueUsage(ctx), ErrBadValue) {
		t.Fatal("expected bad value usage error")
	}
}
