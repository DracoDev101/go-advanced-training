package ctxlab

import (
	"context"
	"testing"
	"time"
)

var errSink error
var idSink string
var okSink bool

func BenchmarkWithTimeoutCancel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		errSink = ctx.Err()
		cancel()
	}
}

func BenchmarkContextValueTypedKey(b *testing.B) {
	ctx := WithRequestID(context.Background(), "req")
	for i := 0; i < b.N; i++ {
		idSink, okSink = RequestID(ctx)
	}
}
