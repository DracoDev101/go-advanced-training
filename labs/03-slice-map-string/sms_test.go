package sms

import (
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestAppendWithSpareCapacitySharesBackingArray(t *testing.T) {
	xs := make([]int, 2, 4)
	xs[0], xs[1] = 1, 2

	ys := AppendOne(xs, 3)
	ys[0] = 100

	if xs[0] != 100 {
		t.Fatalf("expected xs and ys to share backing array; xs[0]=%d", xs[0])
	}
}

func TestAppendWithoutCapacityAllocatesNewBackingArray(t *testing.T) {
	xs := make([]int, 2, 2)
	xs[0], xs[1] = 1, 2

	ys := AppendOne(xs, 3)
	ys[0] = 100

	if xs[0] != 1 {
		t.Fatalf("expected xs to keep original backing array; xs[0]=%d", xs[0])
	}
}

func TestFullSliceExpressionForcesNextAppendToAllocate(t *testing.T) {
	xs := make([]int, 2, 4)
	xs[0], xs[1] = 1, 2

	limited := LimitCapacity(xs)
	ys := AppendOne(limited, 3)
	ys[0] = 100

	if xs[0] != 1 {
		t.Fatalf("limited cap append should not modify original backing array; xs[0]=%d", xs[0])
	}
}

func TestSubsliceSharedVsCopy(t *testing.T) {
	buf := []byte("abcdefghijklmnopqrstuvwxyz")
	shared := FirstNShared(buf, 3)
	copied := FirstNCopy(buf, 3)

	buf[0] = 'Z'

	if string(shared) != "Zbc" {
		t.Fatalf("shared subslice should observe backing array mutation: %q", shared)
	}
	if string(copied) != "abc" {
		t.Fatalf("copied subslice should be independent: %q", copied)
	}
}

func TestNilAndEmptySliceJSON(t *testing.T) {
	if got := MarshalNilSlice(); got != "null" {
		t.Fatalf("nil slice JSON: got %q, want null", got)
	}
	if got := MarshalEmptySlice(); got != "[]" {
		t.Fatalf("empty slice JSON: got %q, want []", got)
	}
}

func TestSortedMapKeysAndStableFormat(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1, "c": 3}

	keys := SortedMapKeys(m)
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys: got %v, want %v", keys, want)
	}

	if got := FormatMapStable(m); got != "a=1,b=2,c=3" {
		t.Fatalf("stable format: got %q", got)
	}
}

func TestStringByteAndRuneLength(t *testing.T) {
	s := "你好"

	if got := ByteLen(s); got != 6 {
		t.Fatalf("byte len: got %d, want 6", got)
	}
	if got := RuneLen(s); got != 2 {
		t.Fatalf("rune len: got %d, want 2", got)
	}
}

func TestFirstNRunesDoesNotSplitUTF8(t *testing.T) {
	s := "你好世界"

	got := FirstNRunes(s, 2)
	if got != "你好" {
		t.Fatalf("FirstNRunes: got %q, want 你好", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("FirstNRunes returned invalid UTF-8: %q", got)
	}
}
