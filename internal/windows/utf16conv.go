package windows

import (
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// This file has no build tag: the conversions between UTF-16, which Windows
// speaks, and Go's UTF-8 are plain Go, and tested on every platform.

// utf16ToString is string(utf16.Decode(u)) in one pass and one allocation:
// utf16.Decode makes a []rune that grows as it goes (1.6 MB for a 72 000-unit
// message), which the string then copies. A lone surrogate becomes U+FFFD.
func utf16ToString(u []uint16) string {
	n := 0
	for i := 0; i < len(u); i++ {
		switch r := rune(u[i]); {
		case r < 0x80:
			n++
		case r < 0x800:
			n += 2
		case isHigh(r) && i+1 < len(u) && isLow(rune(u[i+1])):
			n += 4
			i++
		default: // another BMP rune, or a lone surrogate (U+FFFD)
			n += 3
		}
	}
	if n == 0 {
		return ""
	}
	b := make([]byte, 0, n)
	for i := 0; i < len(u); i++ {
		r := rune(u[i])
		switch {
		case isHigh(r) && i+1 < len(u) && isLow(rune(u[i+1])):
			r = utf16.DecodeRune(r, rune(u[i+1]))
			i++
		case utf16.IsSurrogate(r):
			r = utf8.RuneError
		}
		b = utf8.AppendRune(b, r)
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func isHigh(r rune) bool { return 0xD800 <= r && r < 0xDC00 }
func isLow(r rune) bool  { return 0xDC00 <= r && r < 0xE000 }

// appendUTF16z appends s to dst as a NUL-terminated UTF-16 string; NULs
// inside s are dropped.
func appendUTF16z(dst []uint16, s string) []uint16 {
	for _, r := range s {
		if r == 0 {
			continue
		}
		if r >= 0x10000 {
			r1, r2 := utf16.EncodeRune(r)
			dst = append(dst, uint16(r1), uint16(r2))
		} else {
			dst = append(dst, uint16(r))
		}
	}
	return append(dst, 0)
}
