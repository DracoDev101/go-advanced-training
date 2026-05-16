package otellab

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ctxKey string
const (
	RequestIDKey ctxKey = "request_id"
	TraceIDKey ctxKey = "trace_id"
	JobIDKey ctxKey = "job_id"
)

type LogEntry struct {
	Level string `json:"level"`
	Component string `json:"component"`
	Operation string `json:"operation"`
	RequestID string `json:"request_id,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	JobID string `json:"job_id,omitempty"`
	DurationMS int64 `json:"duration_ms"`
	Status string `json:"status"`
	ErrorKind string `json:"error_kind,omitempty"`
}

func StructuredLog(ctx context.Context, component, operation, status, errorKind string, d time.Duration) []byte {
	e := LogEntry{Level:"info", Component:component, Operation:operation, Status:status, ErrorKind:errorKind, DurationMS:d.Milliseconds()}
	if v, _ := ctx.Value(RequestIDKey).(string); v != "" { e.RequestID = v }
	if v, _ := ctx.Value(TraceIDKey).(string); v != "" { e.TraceID = v }
	if v, _ := ctx.Value(JobIDKey).(string); v != "" { e.JobID = v }
	b, _ := json.Marshal(e)
	return b
}

type Metrics struct {
	mu sync.Mutex
	Counters map[string]int64
	Gauges map[string]int64
	Durations map[string][]time.Duration
}

func NewMetrics() *Metrics { return &Metrics{Counters:map[string]int64{}, Gauges:map[string]int64{}, Durations:map[string][]time.Duration{}} }
func (m *Metrics) Inc(name string, labels ...string) { m.mu.Lock(); defer m.mu.Unlock(); m.Counters[key(name, labels...)]++ }
func (m *Metrics) SetGauge(name string, value int64, labels ...string) { m.mu.Lock(); defer m.mu.Unlock(); m.Gauges[key(name, labels...)] = value }
func (m *Metrics) Observe(name string, d time.Duration, labels ...string) { m.mu.Lock(); defer m.mu.Unlock(); k:=key(name, labels...); m.Durations[k]=append(m.Durations[k], d) }
func (m *Metrics) Count(name string, labels ...string) int64 { m.mu.Lock(); defer m.mu.Unlock(); return m.Counters[key(name, labels...)] }
func key(name string, labels ...string) string { return fmt.Sprintf("%s{%v}", name, labels) }

type Span struct { Name string; Attributes map[string]string; Started, Ended time.Time }

type Tracer struct { mu sync.Mutex; Spans []Span }

func (t *Tracer) Start(ctx context.Context, name string, attrs map[string]string) (context.Context, func()) {
	if attrs == nil { attrs = map[string]string{} }
	if traceID, _ := ctx.Value(TraceIDKey).(string); traceID != "" { attrs["trace_id"] = traceID }
	if jobID, _ := ctx.Value(JobIDKey).(string); jobID != "" { attrs["job.id"] = jobID }
	span := Span{Name:name, Attributes:attrs, Started:time.Now()}
	return ctx, func(){ t.mu.Lock(); span.Ended=time.Now(); t.Spans=append(t.Spans, span); t.mu.Unlock() }
}

func (t *Tracer) Snapshot() []Span { t.mu.Lock(); defer t.mu.Unlock(); out:=make([]Span,len(t.Spans)); copy(out,t.Spans); return out }

type JobRunner struct { Metrics *Metrics; Tracer *Tracer }

func (r JobRunner) Execute(ctx context.Context, kind string, fn func(context.Context) error) error {
	ctx, end := r.Tracer.Start(ctx, "ExecuteJob", map[string]string{"job.kind":kind})
	defer end()
	start := time.Now()
	err := fn(ctx)
	status := "succeeded"; errorKind := ""
	if err != nil { status="failed"; errorKind="dependency"; r.Metrics.Inc("job_retry_total", "kind", kind, "reason", errorKind) }
	r.Metrics.Observe("job_execution_duration_seconds", time.Since(start), "kind", kind, "status", status)
	_ = StructuredLog(ctx, "worker", "ExecuteJob", status, errorKind, time.Since(start))
	return err
}
