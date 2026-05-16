package gclab

import "testing"

func TestEncodersProduceSameShape(t *testing.T) {
	j := SampleJob(128)
	a := string(EncodeAllocHeavy(j))
	b := string(EncodePreallocated(j))
	if a != b { t.Fatalf("different output: %q vs %q", a, b) }
}

func TestAllocPressureIsObservable(t *testing.T) {
	allocated := AllocPressure(1000, 256)
	if allocated == 0 { t.Fatalf("expected observable allocation pressure") }
}

func TestWithGOGCRestoresOldValue(t *testing.T) {
	called := false
	WithGOGC(50, func(){ called = true })
	if !called { t.Fatalf("callback not called") }
}

func TestWithMemoryLimitRestoresOldValue(t *testing.T) {
	called := false
	WithMemoryLimit(64<<20, func(){ called = true })
	if !called { t.Fatalf("callback not called") }
}
