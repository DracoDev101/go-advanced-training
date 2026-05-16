package escape

import "testing"

func TestChangeIntDoesNotModifyCaller(t *testing.T) {
	n := 1

	ChangeInt(n)

	if n != 1 {
		t.Fatalf("ChangeInt modified caller variable: got %d, want 1", n)
	}
}

func TestChangeIntByPointerModifiesCaller(t *testing.T) {
	n := 1

	ChangeIntByPointer(&n)

	if n != 100 {
		t.Fatalf("ChangeIntByPointer did not modify caller variable: got %d, want 100", n)
	}
}

func TestModifySliceElementSharesBackingArray(t *testing.T) {
	xs := []int{1, 2, 3}

	ModifySliceElement(xs)

	if xs[0] != 100 {
		t.Fatalf("slice element was not modified through shared backing array: got %d, want 100", xs[0])
	}
}

func TestAppendLocalDoesNotUpdateCallerHeader(t *testing.T) {
	xs := make([]int, 3, 4)
	xs[0], xs[1], xs[2] = 1, 2, 3

	AppendLocal(xs)

	if len(xs) != 3 {
		t.Fatalf("AppendLocal changed caller length: got %d, want 3", len(xs))
	}
}

func TestAppendReturnUpdatesCallerWhenAssigned(t *testing.T) {
	xs := make([]int, 3, 4)
	xs[0], xs[1], xs[2] = 1, 2, 3

	xs = AppendReturn(xs)

	if len(xs) != 4 {
		t.Fatalf("AppendReturn length: got %d, want 4", len(xs))
	}
	if xs[3] != 4 {
		t.Fatalf("AppendReturn appended value: got %d, want 4", xs[3])
	}
}

func TestReturnPointerIsSafe(t *testing.T) {
	p := ReturnPointer()

	if p == nil {
		t.Fatal("ReturnPointer returned nil")
	}
	if *p != (Small{A: 1, B: 2}) {
		t.Fatalf("ReturnPointer: got %+v, want {A:1 B:2}", *p)
	}
}

func TestMakeCounterCapturesState(t *testing.T) {
	counter := MakeCounter()

	if got := counter(); got != 1 {
		t.Fatalf("first call: got %d, want 1", got)
	}
	if got := counter(); got != 2 {
		t.Fatalf("second call: got %d, want 2", got)
	}
}
