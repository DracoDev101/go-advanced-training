package channelboundaries

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestUnbufferedChannelSynchronizes(t *testing.T) {
	ch := make(chan int)
	done := make(chan struct{})
	go func() {
		ch <- 42
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("send should block until receive")
	case <-time.After(10 * time.Millisecond):
	}

	if got := <-ch; got != 42 {
		t.Fatalf("got %d", got)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("send did not complete after receive")
	}
}

func TestBufferedChannelBlocksWhenFull(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 1
	if TrySend(ch, 2) {
		t.Fatal("try send should fail when buffer is full")
	}
	<-ch
	if !TrySend(ch, 2) {
		t.Fatal("try send should succeed after space is available")
	}
}

func TestCloseReceiveZeroValueAndOKFalse(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 7
	close(ch)
	if v, ok := <-ch; !ok || v != 7 {
		t.Fatalf("first receive got v=%d ok=%v", v, ok)
	}
	if v, ok := <-ch; ok || v != 0 {
		t.Fatalf("closed drained receive got v=%d ok=%v", v, ok)
	}
}

func TestSendOnClosedPanics(t *testing.T) {
	if !SendOnClosedPanics() {
		t.Fatal("send on closed channel should panic")
	}
}

func TestSendWithContextReturnsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan int)
	cancel()
	if err := SendWithContext(ctx, ch, 1); err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestFanIn(t *testing.T) {
	ctx := context.Background()
	a := make(chan int, 2)
	b := make(chan int, 2)
	a <- 1; a <- 2; close(a)
	b <- 3; b <- 4; close(b)
	got := ReceiveUntilClosed(FanIn(ctx, a, b))
	m := map[int]bool{}
	for _, v := range got { m[v] = true }
	for _, want := range []int{1,2,3,4} {
		if !m[want] { t.Fatalf("missing %d from fan-in result %v", want, got) }
	}
}

func TestPipelineSquares(t *testing.T) {
	ctx := context.Background()
	got := ReceiveUntilClosed(Square(ctx, Generate(ctx, 1, 2, 3)))
	if want := []int{1, 4, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPipelineCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out := Square(ctx, Generate(ctx, 1, 2, 3, 4, 5))
	cancel()
	select {
	case <-out:
	case <-time.After(time.Second):
		t.Fatal("pipeline did not exit after cancel")
	}
}
