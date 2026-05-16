package sms

import (
	"strconv"
	"testing"
)

var stringSink string
var bytesSink []byte

func benchmarkParts() []string {
	parts := make([]string, 128)
	for i := range parts {
		parts[i] = "part-" + strconv.Itoa(i) + ";"
	}
	return parts
}

func BenchmarkConcatPlus(b *testing.B) {
	parts := benchmarkParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stringSink = ConcatPlus(parts)
	}
}

func BenchmarkConcatSprintf(b *testing.B) {
	parts := benchmarkParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stringSink = ConcatSprintf(parts)
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	parts := benchmarkParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stringSink = ConcatBuilder(parts)
	}
}

func BenchmarkConcatBuilderGrow(b *testing.B) {
	parts := benchmarkParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stringSink = ConcatBuilderGrow(parts)
	}
}

func BenchmarkFirstNShared(b *testing.B) {
	buf := make([]byte, 1024*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesSink = FirstNShared(buf, 16)
	}
}

func BenchmarkFirstNCopy(b *testing.B) {
	buf := make([]byte, 1024*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesSink = FirstNCopy(buf, 16)
	}
}
