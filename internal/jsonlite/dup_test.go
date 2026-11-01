package jsonlite

import (
	"fmt"
	"strings"
	"testing"
)

func objectOf(n int) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"k%07d":0`, i)
	}
	sb.WriteByte('}')
	return sb.String()
}

// Strict's check for duplicate names does a bounded amount of work per member,
// however many an object has: a page sends objects of any size, and the check
// runs before anything is trusted. Counted, not timed.
func TestStrictDuplicateCheckIsLinear(t *testing.T) {
	for _, n := range []int{0, 1, 8, 9, 1000, 5000} {
		r := NewReader([]byte(objectOf(n)))
		r.Strict = true
		if err := r.Skip(); err != nil {
			t.Fatalf("%d members: %v", n, err)
		}
		if limit := 9 * n; r.probes > limit {
			t.Errorf("%d members: %d probes, more than %d", n, r.probes, limit)
		}
	}
	// Many small objects, and a deep chain of them, are linear too.
	small := "[" + strings.Repeat(`{"a":1,"b":2,"c":3},`, 50000) + `{}]`
	r := NewReader([]byte(small))
	r.Strict = true
	if err := r.Skip(); err != nil {
		t.Fatal(err)
	}
	if r.probes > 50000*9 {
		t.Errorf("small objects: %d probes", r.probes)
	}
	deep := strings.Repeat(`{"a":`, 5000) + "0" + strings.Repeat("}", 5000)
	r = NewReader([]byte(deep))
	r.Strict = true
	if err := r.Skip(); err != nil {
		t.Fatal(err)
	}
	if r.probes > 5000*9 {
		t.Errorf("deep objects: %d probes", r.probes)
	}
}

// A name that occurs twice is found wherever it sits: among the first members,
// across the point where the check moves to a set, and far past it.
func TestStrictFindsADuplicateName(t *testing.T) {
	for _, n := range []int{2, 8, 9, 10, 5000} {
		for _, first := range []int{0, 7, 8, n - 2} {
			if first < 0 || first >= n-1 {
				continue
			}
			var sb strings.Builder
			sb.WriteByte('{')
			for i := range n - 1 {
				if i > 0 {
					sb.WriteByte(',')
				}
				fmt.Fprintf(&sb, `"k%d":0`, i)
			}
			fmt.Fprintf(&sb, `,"k%d":1}`, first)
			doc := sb.String()
			r := NewReader([]byte(doc))
			r.Strict = true
			if r.Skip() == nil {
				t.Errorf("%d members, the name of member %d repeated: read without error", n, first)
			}
			if lax := NewReader([]byte(doc)); lax.Skip() != nil {
				t.Errorf("%d members: not Strict, a repeated name is an error", n)
			}
		}
	}
}

// A string with an escape after a byte that is not UTF-8 is as invalid as one
// without: Strict refuses it, as encoding/json/v2 does, and the default reader
// reads the byte as U+FFFD.
func TestStrictInvalidUTF8WithEscape(t *testing.T) {
	for _, doc := range []string{"\"a\xff\\n\"", "\"\\n\xff\"", "\"\xc3\\u0041\"", "\"a\xff\\u0041b\""} {
		r := NewReader([]byte(doc))
		r.Strict = true
		if _, err := r.String(); err == nil {
			t.Errorf("%q: Strict read it", doc)
		}
		lax := NewReader([]byte(doc))
		if s, err := lax.String(); err != nil || !strings.Contains(s, "\ufffd") {
			t.Errorf("%q: read %q, %v", doc, s, err)
		}
	}
}

// The check for duplicate names allocates a bounded amount whatever the number
// of objects: page messages are read on the main thread, and a megabyte of
// small objects must not cost megabytes of garbage. Counted, not timed.
func TestStrictDuplicateCheckDoesNotAllocatePerObject(t *testing.T) {
	// 9 members: one more than the scan covers, so each object needs a set.
	obj := `{"a":0,"b":0,"c":0,"d":0,"e":0,"f":0,"g":0,"h":0,"i":0}`
	doc := []byte("[" + strings.TrimSuffix(strings.Repeat(obj+",", 2000), ",") + "]")
	read := func(strict bool) float64 {
		return testing.AllocsPerRun(5, func() {
			r := NewReader(doc)
			r.Strict = strict
			err := r.Array(func() error {
				return r.Object(func(string) error { return r.Skip() })
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	lax, strict := read(false), read(true)
	if extra := strict - lax; extra > 16 {
		t.Errorf("Strict allocates %.0f times more than lax for 2000 objects, want a constant", extra)
	}
}
