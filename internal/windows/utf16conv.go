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

// maxKeptPost is the largest UTF-16 buffer, in units, a window keeps for the
// next reply it posts.
const maxKeptPost = 1 << 20

// postBuffer holds the UTF-16 copy of the reply a window is posting, for the
// next one. The COM call that posts can run the message loop, and a post from a
// handler it runs would write over the copy in flight: while one holds the
// buffer, another gets a buffer of its own.
type postBuffer struct {
	buf  []uint16
	busy bool
}

// take returns s in UTF-16, NUL-terminated, in the buffer when it is free
// (owned is true: give it back), or in a new one.
func (p *postBuffer) take(s string) (units []uint16, owned bool) {
	need := len(s) + 1 // a unit takes at least a byte of s, a pair four
	if p.busy {
		return appendUTF16z(make([]uint16, 0, need), s), false
	}
	p.busy = true
	if cap(p.buf) < need {
		p.buf = make([]uint16, 0, need) // appending to nil would grow it by doubling
	}
	return appendUTF16z(p.buf[:0], s), true
}

// give returns what take lent, kept for the next one unless it is huge.
func (p *postBuffer) give(units []uint16, owned bool) {
	if !owned {
		return
	}
	if cap(units) <= maxKeptPost {
		p.buf = units[:0]
	} else {
		p.buf = nil // a huge reply does not pin its buffer
	}
	p.busy = false
}
