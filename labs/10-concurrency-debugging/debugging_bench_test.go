package debugging

import "testing"

func BenchmarkSafeCounter(b *testing.B) {
	var c SafeCounter
	b.RunParallel(func(pb *testing.PB) { for pb.Next() { c.Inc() } })
}

func BenchmarkContendedMutex(b *testing.B) {
	for i := 0; i < b.N; i++ { _ = ContendedMutex(10, 8) }
}
