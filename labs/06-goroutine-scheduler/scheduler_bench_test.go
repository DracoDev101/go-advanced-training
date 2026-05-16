package scheduler

import (
	"context"
	"testing"
)

var int64Sink int64
var uint64Sink uint64

func BenchmarkWorkerPool1(b *testing.B) { benchmarkWorkerPool(b, 1) }
func BenchmarkWorkerPool2(b *testing.B) { benchmarkWorkerPool(b, 2) }
func BenchmarkWorkerPool4(b *testing.B) { benchmarkWorkerPool(b, 4) }
func BenchmarkWorkerPool8(b *testing.B) { benchmarkWorkerPool(b, 8) }

func benchmarkWorkerPool(b *testing.B, workers int) {
	for i := 0; i < b.N; i++ {
		processed, err := ProcessWithLimit(context.Background(), workers, 1000)
		if err != nil {
			b.Fatal(err)
		}
		int64Sink = processed
	}
}

func BenchmarkBusyWork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uint64Sink = BusyWork(10000)
	}
}
