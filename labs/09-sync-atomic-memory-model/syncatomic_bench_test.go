package syncatomic

import "testing"

var int64Sink int64

func BenchmarkMutexCounter(b *testing.B) {
	var c MutexCounter
	for i:=0; i<b.N; i++ { c.Inc() }
	int64Sink = c.Value()
}

func BenchmarkAtomicCounter(b *testing.B) {
	var c AtomicCounter
	for i:=0; i<b.N; i++ { c.Inc() }
	int64Sink = c.Value()
}

func BenchmarkChannelCounter(b *testing.B) {
	c := NewChannelCounter()
	defer c.Close()
	for i:=0; i<b.N; i++ { c.Inc() }
	int64Sink = c.Value()
}

func BenchmarkCacheRWMutexRead(b *testing.B) {
	c := NewCache(); c.Set("k", "v")
	for i:=0; i<b.N; i++ { _, _ = c.Get("k") }
}
