package ui

import (
	"math"
	"reflect"
	"unsafe"
)

// valueRead is a value a control showed in a row as it was built, and where it
// came from: the *bool of a Checkbox or Toggle, the *string of a Select, the
// *float64 of a Slider. An app
// may change it without input (a "select all" outside the list) and without
// saying so in RowVersion: a row is kept only while each holds what the row was
// built with, so the control versions itself. (By address: see the doc of
// RowVersion for data that moves.)
//
// A read holds one pointer, the kind of what it points to, and what it held:
// in bits (a bool, a number) or in str (a string).
type valueRead struct {
	ptr  unsafe.Pointer
	bits uint64
	str  string
	kind valueKind
}

type valueKind uint8

const (
	readBool valueKind = iota + 1
	readString
	readFloat
	// readBits1 to readBits8 are an integer of that many bytes, of any signedness,
	// compared as the bits it holds.
	readBits1
	readBits2
	readBits4
	readBits8
)

// bitsAt returns the integer of the width of kind at ptr, as the bits it holds.
func bitsAt(kind valueKind, ptr unsafe.Pointer) uint64 {
	switch kind {
	case readBits1:
		return uint64(*(*uint8)(ptr))
	case readBits2:
		return uint64(*(*uint16)(ptr))
	case readBits4:
		return uint64(*(*uint32)(ptr))
	}
	return *(*uint64)(ptr)
}

// same reports whether what r points to holds what the row was built with.
// A NaN is never equal to itself.
func (r *valueRead) same() bool {
	switch r.kind {
	case readBool:
		return *(*bool)(r.ptr) == (r.bits != 0)
	case readString:
		return *(*string)(r.ptr) == r.str
	case readFloat:
		return *(*float64)(r.ptr) == math.Float64frombits(r.bits)
	case readBits1, readBits2, readBits4, readBits8:
		return bitsAt(r.kind, r.ptr) == r.bits
	}
	return true
}

// note records a read of kind at ptr once: a read again with another value, or
// more controls than the limit, make the row (and the rows around it) built in
// every frame.
func (rb *rowBuild) note(r valueRead) {
	for i := range rb.values {
		if b := &rb.values[i]; b.ptr == r.ptr && b.kind == r.kind {
			if r.kind == readFloat && math.Float64frombits(b.bits) == math.Float64frombits(r.bits) || r.kind != readFloat && b.bits == r.bits && b.str == r.str {
				return
			}
			rb.neverKept()
			return
		}
	}
	if len(rb.values) >= maxValueReads {
		rb.neverKept()
		return
	}
	if rb.values == nil && rb.f != nil {
		rb.values = rb.f.valueBuf[:0]
	}
	rb.values = append(rb.values, r)
}

// maxValueReads is how many different controls a row records; more, and the row
// is built in every frame.
const maxValueReads = 8

// readsBool notes the *bool of a control the row being built made.
func (c *context) readsBool(p *bool, v bool) {
	if c != nil && c.row != nil {
		c.row.noteBool(p, v)
	}
}

// noteBool records p.
func (rb *rowBuild) noteBool(p *bool, v bool) {
	rb.note(valueRead{ptr: unsafe.Pointer(p), bits: uint64(b2i(v)), kind: readBool})
}

// neverKept marks the row, and the rows around it, built in every frame.
func (rb *rowBuild) neverKept() {
	for r := rb; r != nil; r = r.outer {
		r.outside = true
	}
}

// sameValues reports whether every control holds what the row was built with.
func sameValues(bs []valueRead) bool {
	for i := range bs {
		if !bs[i].same() {
			return false
		}
	}
	return true
}

// readsString notes the *string a control (Select) made the row show.
func (c *context) readsString(s *string, v string) {
	if c != nil && c.row != nil {
		c.row.noteString(s, v)
	}
}

// noteString is noteBool for a string.
func (rb *rowBuild) noteString(s *string, v string) {
	rb.note(valueRead{ptr: unsafe.Pointer(s), str: v, kind: readString})
}

// readsFloat notes the *float64 a control (Slider) made the row show.
func (c *context) readsFloat(f *float64, v float64) {
	if c != nil && c.row != nil {
		c.row.noteFloat(f, v)
	}
}

// noteFloat is noteBool for a number. A NaN is never equal to itself, so a row
// that shows one is built in every frame.
func (rb *rowBuild) noteFloat(f *float64, v float64) {
	rb.note(valueRead{ptr: unsafe.Pointer(f), bits: math.Float64bits(v), kind: readFloat})
}

// readsChoice notes the *T a radio button compares with its value, which the
// app and the buttons of other rows may change: a row records it when T is a
// bool, a string or an integer (read through the kind it has, once, as the row is
// built; the check in each frame uses the kind it stored). Any other T cannot be
// compared without calling its ==, so the row is built in every frame.
func readsChoice[T comparable](c *context, p *T) {
	if c == nil || c.row == nil {
		return
	}
	ptr := unsafe.Pointer(p)
	switch reflect.TypeFor[T]().Kind() {
	case reflect.Bool:
		c.row.noteBool((*bool)(ptr), *(*bool)(ptr))
	case reflect.String:
		c.row.noteString((*string)(ptr), *(*string)(ptr))
	case reflect.Int8, reflect.Uint8:
		c.row.note(valueRead{ptr: ptr, bits: bitsAt(readBits1, ptr), kind: readBits1})
	case reflect.Int16, reflect.Uint16:
		c.row.note(valueRead{ptr: ptr, bits: bitsAt(readBits2, ptr), kind: readBits2})
	case reflect.Int32, reflect.Uint32:
		c.row.note(valueRead{ptr: ptr, bits: bitsAt(readBits4, ptr), kind: readBits4})
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		c.row.note(valueRead{ptr: ptr, bits: bitsAt(readBits8, ptr), kind: readBits8})
	default:
		c.row.neverKept()
	}
}
