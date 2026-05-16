package channelboundaries

import (
	"context"
	"sync"
)

func SendWithContext[T any](ctx context.Context, ch chan<- T, v T) error {
	select {
	case ch <- v:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TrySend[T any](ch chan<- T, v T) bool {
	select {
	case ch <- v:
		return true
	default:
		return false
	}
}

func ReceiveUntilClosed[T any](ch <-chan T) []T {
	var out []T
	for v := range ch {
		out = append(out, v)
	}
	return out
}

func SendOnClosedPanics() (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	ch := make(chan int)
	close(ch)
	ch <- 1
	return false
}

func FanIn[T any](ctx context.Context, inputs ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for _, in := range inputs {
		in := in
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-in:
					if !ok {
						return
					}
					select {
					case out <- v:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func Generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func Square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- n * n:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

type MutexCounter struct {
	mu sync.Mutex
	n  int
}

func (c *MutexCounter) Inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

func (c *MutexCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

type ChannelCounter struct {
	ops chan counterOp
}

type counterOp struct {
	kind string
	resp chan int
}

func NewChannelCounter() *ChannelCounter {
	c := &ChannelCounter{ops: make(chan counterOp)}
	go func() {
		var n int
		for op := range c.ops {
			switch op.kind {
			case "inc":
				n++
			case "value":
				op.resp <- n
			}
		}
	}()
	return c
}

func (c *ChannelCounter) Inc() { c.ops <- counterOp{kind: "inc"} }

func (c *ChannelCounter) Value() int {
	resp := make(chan int)
	c.ops <- counterOp{kind: "value", resp: resp}
	return <-resp
}

func (c *ChannelCounter) Close() { close(c.ops) }
