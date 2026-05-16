package perflab

import "testing"

var stringSink string
var bytesSink []byte

func BenchmarkSlowSummary(b *testing.B) {
	parts := Parts(20)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { stringSink = SlowSummary(parts) }
}

func BenchmarkFastSummary(b *testing.B) {
	parts := Parts(20)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { stringSink = FastSummary(parts) }
}

func BenchmarkEncodeJSON(b *testing.B) {
	r := SampleResult()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var err error
		bytesSink, err = EncodeJSON(r)
		if err != nil { b.Fatal(err) }
	}
}

// Bad example: setup work is inside the timed loop, so it measures Parts + Summary.
func BenchmarkBadSummaryIncludesSetup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { stringSink = FastSummary(Parts(20)) }
}

// Good example: setup is outside the timed loop.
func BenchmarkGoodSummaryExcludesSetup(b *testing.B) {
	parts := Parts(20)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ { stringSink = FastSummary(parts) }
}
