package gclab

import (
	"bytes"
	"runtime"
	"runtime/debug"
	"strconv"
)

type Job struct {
	ID      string
	Payload []byte
	Tags    map[string]string
}

// EncodeAllocHeavy intentionally allocates many short-lived objects: map, strings, buffer growth.
func EncodeAllocHeavy(j Job) []byte {
	fields := map[string]string{
		"id":      j.ID,
		"payload": strconv.Itoa(len(j.Payload)),
		"kind":    j.Tags["kind"],
	}
	var b bytes.Buffer
	b.WriteString(fields["id"])
	b.WriteByte(':')
	b.WriteString(fields["payload"])
	b.WriteByte(':')
	b.WriteString(fields["kind"])
	return b.Bytes()
}

// EncodePreallocated reduces temporary objects. It is still readable and avoids a premature pool.
func EncodePreallocated(j Job) []byte {
	kind := j.Tags["kind"]
	payloadLen := strconv.Itoa(len(j.Payload))
	out := make([]byte, 0, len(j.ID)+len(payloadLen)+len(kind)+2)
	out = append(out, j.ID...)
	out = append(out, ':')
	out = append(out, payloadLen...)
	out = append(out, ':')
	out = append(out, kind...)
	return out
}

func SampleJob(payloadSize int) Job {
	return Job{ID: "job-123", Payload: bytes.Repeat([]byte("x"), payloadSize), Tags: map[string]string{"kind": "email"}}
}

func AllocPressure(iterations, payloadSize int) uint64 {
	j := SampleJob(payloadSize)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < iterations; i++ { _ = EncodeAllocHeavy(j) }
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func WithGOGC(percent int, fn func()) {
	old := debug.SetGCPercent(percent)
	defer debug.SetGCPercent(old)
	fn()
}

func WithMemoryLimit(bytes int64, fn func()) {
	old := debug.SetMemoryLimit(bytes)
	defer debug.SetMemoryLimit(old)
	fn()
}
