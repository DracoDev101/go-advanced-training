package perflab

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JobResult struct {
	ID       string            `json:"id"`
	Status   string            `json:"status"`
	Attempts int               `json:"attempts"`
	Meta     map[string]string `json:"meta"`
}

func SampleResult() JobResult {
	return JobResult{ID: "job-123", Status: "succeeded", Attempts: 2, Meta: map[string]string{"kind":"email", "worker":"w1"}}
}

func EncodeJSON(r JobResult) ([]byte, error) { return json.Marshal(r) }

// SlowSummary is intentionally inefficient: fmt.Sprintf in a loop creates avoidable allocations.
func SlowSummary(parts []string) string {
	out := ""
	for i, p := range parts { out = fmt.Sprintf("%s%d=%s;", out, i, p) }
	return out
}

func FastSummary(parts []string) string {
	var b strings.Builder
	b.Grow(len(parts) * 12)
	for i, p := range parts {
		b.WriteString(fmt.Sprint(i))
		b.WriteByte('=')
		b.WriteString(p)
		b.WriteByte(';')
	}
	return b.String()
}

func Parts(n int) []string {
	out := make([]string, n)
	for i := range out { out[i] = "payload" }
	return out
}
