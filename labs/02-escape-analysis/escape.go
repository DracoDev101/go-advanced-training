package escape

// Small is intentionally tiny so value passing is usually cheap.
type Small struct {
	A int
	B int
}

// Large is intentionally large enough to make copies visible in benchmarks.
type Large struct {
	Data [1024]int64
}

// ChangeInt receives a copy of x. The caller's variable is unchanged.
func ChangeInt(x int) {
	x = 100
}

// ChangeIntByPointer receives a copy of an address. Dereferencing it mutates
// the caller-visible variable.
func ChangeIntByPointer(p *int) {
	*p = 100
}

// ModifySliceElement receives a copy of the slice header, but that header still
// points at the same backing array as the caller's slice.
func ModifySliceElement(xs []int) {
	xs[0] = 100
}

// AppendLocal modifies only the local slice header. The caller does not see the
// new length unless the new header is returned.
func AppendLocal(xs []int) {
	xs = append(xs, 4)
}

// AppendReturn returns the new slice header after append.
func AppendReturn(xs []int) []int {
	return append(xs, 4)
}

// ReturnPointer returns a pointer to a local variable. This is safe in Go; the
// compiler moves x to the heap if needed.
func ReturnPointer() *Small {
	x := Small{A: 1, B: 2}
	return &x
}

// ReturnValue returns a Small by value.
func ReturnValue() Small {
	return Small{A: 1, B: 2}
}

// ReturnAny boxes Small into an interface value. Depending on context and
// compiler optimizations, this can force allocation.
func ReturnAny() any {
	x := Small{A: 1, B: 2}
	return x
}

// MakeCounter returns a closure that captures n. Because n must live after
// MakeCounter returns, it commonly escapes.
func MakeCounter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

// SumSmallByValue demonstrates value passing for small structs.
func SumSmallByValue(s Small) int {
	return s.A + s.B
}

// SumSmallByPointer demonstrates pointer passing for small structs.
func SumSmallByPointer(s *Small) int {
	return s.A + s.B
}

// SumLargeByValue copies a large struct.
func SumLargeByValue(l Large) int64 {
	return l.Data[0] + l.Data[len(l.Data)-1]
}

// SumLargeByPointer avoids copying the large struct.
func SumLargeByPointer(l *Large) int64 {
	return l.Data[0] + l.Data[len(l.Data)-1]
}

// ConsumeAny accepts an interface value to make boxing effects visible in
// benchmarks and escape-analysis output.
func ConsumeAny(v any) any {
	return v
}
