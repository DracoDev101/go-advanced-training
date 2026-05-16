package escape

import "testing"

var smallSink Small
var smallPointerSink *Small
var largeIntSink int64
var intSink int
var anySink any

func BenchmarkReturnValue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		smallSink = ReturnValue()
	}
}

func BenchmarkReturnPointer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		smallPointerSink = ReturnPointer()
	}
}

func BenchmarkReturnAny(b *testing.B) {
	for i := 0; i < b.N; i++ {
		anySink = ReturnAny()
	}
}

func BenchmarkSmallByValue(b *testing.B) {
	s := Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		intSink = SumSmallByValue(s)
	}
}

func BenchmarkSmallByPointer(b *testing.B) {
	s := Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		intSink = SumSmallByPointer(&s)
	}
}

func BenchmarkLargeByValue(b *testing.B) {
	var l Large
	l.Data[0] = 1
	l.Data[len(l.Data)-1] = 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		largeIntSink = SumLargeByValue(l)
	}
}

func BenchmarkLargeByPointer(b *testing.B) {
	var l Large
	l.Data[0] = 1
	l.Data[len(l.Data)-1] = 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		largeIntSink = SumLargeByPointer(&l)
	}
}

func BenchmarkConsumeConcreteAsAny(b *testing.B) {
	s := Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		anySink = ConsumeAny(s)
	}
}
