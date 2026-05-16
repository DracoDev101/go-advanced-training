package channelboundaries

import "testing"

var intSink int

func BenchmarkMutexCounter(b *testing.B) {
	var c MutexCounter
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
	intSink = c.Value()
}

func BenchmarkChannelCounter(b *testing.B) {
	c := NewChannelCounter()
	defer c.Close()
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
	intSink = c.Value()
}

func BenchmarkBufferedTrySendDrop(b *testing.B) {
	ch := make(chan int, 1)
	ch <- 1
	var dropped int
	for i := 0; i < b.N; i++ {
		if !TrySend(ch, i) {
			dropped++
		}
	}
	intSink = dropped
}
