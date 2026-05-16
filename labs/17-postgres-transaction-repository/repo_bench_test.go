package pglab

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkClaimAtomic(b *testing.B) {
	jobs := make([]Job, b.N)
	for i := range jobs { jobs[i] = Job{ID: fmt.Sprintf("j-%d", i), Status:"pending"} }
	r := NewRepository(jobs...)
	b.ReportAllocs(); b.ResetTimer()
	for i := 0; i < b.N; i++ { _, _ = r.ClaimAtomic(context.Background(), fmt.Sprintf("j-%d", i), "w") }
}
