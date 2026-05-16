package perflab

import "testing"

func TestSummariesMatch(t *testing.T) {
	parts := Parts(5)
	if SlowSummary(parts) != FastSummary(parts) { t.Fatalf("summary mismatch") }
}

func TestEncodeJSON(t *testing.T) {
	b, err := EncodeJSON(SampleResult())
	if err != nil { t.Fatal(err) }
	if len(b) == 0 { t.Fatal("empty json") }
}
