package otellab

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestStructuredLogCarriesCorrelationIDs(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, JobIDKey, "job-1")
	b := StructuredLog(ctx, "api", "SubmitJob", "ok", "", 12*time.Millisecond)
	var got LogEntry
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.RequestID != "req-1" || got.TraceID != "trace-1" || got.JobID != "job-1" { t.Fatalf("missing correlation ids: %+v", got) }
}

func TestMetricsAvoidJobIDAsLabel(t *testing.T) {
	m := NewMetrics()
	m.Inc("job_retry_total", "kind", "email", "reason", "timeout")
	if m.Count("job_retry_total", "kind", "email", "reason", "timeout") != 1 { t.Fatal("missing counter") }
}

func TestTracerPropagatesAttributes(t *testing.T) {
	tr := &Tracer{}
	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, JobIDKey, "job-1")
	_, end := tr.Start(ctx, "ExecuteJob", map[string]string{"worker.id":"w1"})
	end()
	spans := tr.Snapshot()
	if len(spans) != 1 || spans[0].Attributes["job.id"] != "job-1" || spans[0].Attributes["trace_id"] != "trace-1" { t.Fatalf("spans=%+v", spans) }
}

func TestJobRunnerRecordsMetricsAndTrace(t *testing.T) {
	m := NewMetrics(); tr := &Tracer{}
	r := JobRunner{Metrics:m, Tracer:tr}
	ctx := context.WithValue(context.Background(), JobIDKey, "job-1")
	err := r.Execute(ctx, "email", func(context.Context) error { return errors.New("downstream") })
	if err == nil { t.Fatal("expected error") }
	if m.Count("job_retry_total", "kind", "email", "reason", "dependency") != 1 { t.Fatal("missing retry metric") }
	if len(tr.Snapshot()) != 1 { t.Fatal("missing span") }
}
