package sms

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

func AppendOne(xs []int, v int) []int {
	return append(xs, v)
}

func LimitCapacity(xs []int) []int {
	return xs[:len(xs):len(xs)]
}

func FirstNShared(buf []byte, n int) []byte {
	return buf[:n]
}

func FirstNCopy(buf []byte, n int) []byte {
	out := make([]byte, n)
	copy(out, buf[:n])
	return out
}

func MarshalNilSlice() string {
	var xs []int
	b, _ := json.Marshal(xs)
	return string(b)
}

func MarshalEmptySlice() string {
	xs := []int{}
	b, _ := json.Marshal(xs)
	return string(b)
}

func SortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func FormatMapStable(m map[string]int) string {
	keys := SortedMapKeys(m)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ",")
}

func ByteLen(s string) int {
	return len(s)
}

func RuneLen(s string) int {
	return utf8.RuneCountInString(s)
}

func FirstNRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

func ConcatPlus(parts []string) string {
	var s string
	for _, p := range parts {
		s += p
	}
	return s
}

func ConcatSprintf(parts []string) string {
	var s string
	for _, p := range parts {
		s = fmt.Sprintf("%s%s", s, p)
	}
	return s
}

func ConcatBuilder(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

func ConcatBuilderGrow(parts []string) string {
	var total int
	for _, p := range parts {
		total += len(p)
	}
	var b strings.Builder
	b.Grow(total)
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}
