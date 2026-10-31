package ui

import (
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
