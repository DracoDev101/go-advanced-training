package otellab

import (
	"context"
	"testing"
	"time"
)

var logSink []byte

func BenchmarkStructuredLog(b *testing.B) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, JobIDKey, "job-1")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { logSink = StructuredLog(ctx, "worker", "ExecuteJob", "ok", "", time.Millisecond) }
}
