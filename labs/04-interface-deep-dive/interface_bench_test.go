package iface

import "testing"

var intSink int
var anySink any

func BenchmarkConcreteCall(b *testing.B) {
	s := Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		intSink = SumConcrete(s)
	}
}

func BenchmarkInterfaceCall(b *testing.B) {
	var s Summer = Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		intSink = SumInterface(s)
	}
}

func BenchmarkBoxAny(b *testing.B) {
	s := Small{A: 1, B: 2}
	for i := 0; i < b.N; i++ {
		anySink = BoxAny(s)
	}
}
