package gclab

import "testing"

var sink []byte

func BenchmarkEncodeAllocHeavy(b *testing.B) {
	j := SampleJob(1024)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { sink = EncodeAllocHeavy(j) }
}

func BenchmarkEncodePreallocated(b *testing.B) {
	j := SampleJob(1024)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { sink = EncodePreallocated(j) }
}

func BenchmarkAllocPressureGOGC100(b *testing.B) {
	WithGOGC(100, func(){
		j := SampleJob(2048)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ { sink = EncodeAllocHeavy(j) }
	})
}
