package ui

import (
	"math"
	"testing"
	"unsafe"
)

// A read of a control's value is one pointer, a kind, 8 bytes and a string:
// every row of a list keeps up to maxValueReads of them.
func TestValueReadSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("sizes are for 64-bit")
	}
	if got := unsafe.Sizeof(valueRead{}); got != 40 {
		t.Errorf("valueRead is %d bytes, want 40 (it was 56 with a field for each type)", got)
	}
}

// What note does with a control read twice in one build, for each kind: the
// same value is one read, another value or a NaN builds the row in every frame,
// and more controls than maxValueReads do the same.
func TestNoteDedupe(t *testing.T) {
	var b bool
	var s string
	var f float64
	nan, negZero := math.NaN(), math.Copysign(0, -1)
	cases := []struct {
		name    string
		reads   func(rb *rowBuild)
		outside bool
		n       int
	}{
		{"bool same", func(rb *rowBuild) { rb.noteBool(&b, true); rb.noteBool(&b, true) }, false, 1},
		{"bool differs", func(rb *rowBuild) { rb.noteBool(&b, true); rb.noteBool(&b, false) }, true, 1},
		{"string same", func(rb *rowBuild) { rb.noteString(&s, "a"); rb.noteString(&s, "a") }, false, 1},
		{"string differs", func(rb *rowBuild) { rb.noteString(&s, "a"); rb.noteString(&s, "b") }, true, 1},
		{"float same", func(rb *rowBuild) { rb.noteFloat(&f, 1); rb.noteFloat(&f, 1) }, false, 1},
		{"float differs", func(rb *rowBuild) { rb.noteFloat(&f, 1); rb.noteFloat(&f, 2) }, true, 1},
		{"float -0 +0", func(rb *rowBuild) { rb.noteFloat(&f, 0); rb.noteFloat(&f, negZero) }, false, 1},
		{"float NaN twice", func(rb *rowBuild) { rb.noteFloat(&f, nan); rb.noteFloat(&f, nan) }, true, 1},
		{"limit", func(rb *rowBuild) {
			bs := make([]bool, maxValueReads+1)
			for i := range bs {
				rb.noteBool(&bs[i], false)
			}
		}, true, maxValueReads},
	}
	for _, tc := range cases {
		outer := &rowBuild{}
		rb := &rowBuild{outer: outer}
		tc.reads(rb)
		if rb.outside != tc.outside || outer.outside != tc.outside || len(rb.values) != tc.n {
			t.Errorf("%s: outside %v (outer %v), %d values; want %v, %d", tc.name, rb.outside, outer.outside, len(rb.values), tc.outside, tc.n)
		}
	}
	// same(): NaN never holds what it was built with; -0 holds +0.
	f = nan
	if sameValues([]valueRead{floatRead(&f, nan)}) {
		t.Error("NaN is the same")
	}
	f = negZero
	if !sameValues([]valueRead{floatRead(&f, 0)}) {
		t.Error("-0 is not +0")
	}
	s = "x"
	sp := new(string)
	*sp = "x"[:1]
	rb := &rowBuild{}
	rb.noteString(sp, s)
	*sp = string([]byte{'x'}) // another backing array, same value
	if !sameValues(rb.values) {
		t.Error("a string compared by its pointer")
	}
}

func floatRead(f *float64, v float64) valueRead {
	rb := &rowBuild{}
	rb.noteFloat(f, v)
	return rb.values[0]
}

type choiceInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// A row records the choice of a radio button in the bytes of its type, not of
// another: a byte after it (a neighbor in the struct, in 8 bytes of the choice
// of a 4-byte int on a 32-bit target) that changes leaves the read the same, and
// every bit of the choice changes it.
func checkChoiceWidth[T choiceInt](t *testing.T) {
	t.Helper()
	var d struct {
		choice T
		next   [8]byte
	}
	rb := &rowBuild{}
	readsChoice(&context{row: rb}, &d.choice)
	if len(rb.values) != 1 {
		t.Fatalf("%T: %d reads, want 1", d.choice, len(rb.values))
	}
	same := func() bool { return sameValues(rb.values) }
	for i := range d.next {
		d.next[i] = 0xff
	}
	if !same() {
		t.Errorf("%T: a byte after the choice changed the read", d.choice)
	}
	for bit := range unsafe.Sizeof(d.choice) * 8 {
		d.choice = T(1) << bit
		if same() {
			t.Errorf("%T: bit %d of the choice is not read", d.choice, bit)
		}
	}
	d.choice = 0
	if !same() {
		t.Errorf("%T: the choice changed back and the read did not", d.choice)
	}
}

func TestChoiceWidth(t *testing.T) {
	checkChoiceWidth[int](t)
	checkChoiceWidth[int8](t)
	checkChoiceWidth[int16](t)
	checkChoiceWidth[int32](t)
	checkChoiceWidth[int64](t)
	checkChoiceWidth[uint](t)
	checkChoiceWidth[uint8](t)
	checkChoiceWidth[uint16](t)
	checkChoiceWidth[uint32](t)
	checkChoiceWidth[uint64](t)
	checkChoiceWidth[uintptr](t)
}
