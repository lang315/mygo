package windows

import (
	"math/rand/v2"
	"testing"
	"unicode/utf16"
)

func TestUTF16ToStringIsDecode(t *testing.T) {
	cases := [][]uint16{
		nil, {}, {0}, {'a', 0, 'b'}, {'h', 'i'}, {0x7F, 0x80, 0x7FF, 0x800, 0xFFFF},
		{0xD83D, 0xDE00},                   // a pair
		{'x', 0xD83D, 0xDE00, 'y'},         // a pair inside
		{0xD83D},                           // a high surrogate alone
		{'a', 0xD83D},                      // ... at the end
		{0xDE00},                           // a low surrogate alone
		{0xD83D, 0xD83D, 0xDE00},           // high, then a pair
		{0xDE00, 0xD83D},                   // reversed
		{0xD83D, 'a', 0xDE00},              // split
		{0xDBFF, 0xDFFF},                   // the last pair
		{0xD800, 0xDC00},                   // the first pair
		{'a', 'b', 0xD83D, 0xDE00, 0xD83D}, // a pair, then a high at the end
	}
	for _, c := range cases {
		if got, want := utf16ToString(c), string(utf16.Decode(c)); got != want {
			t.Errorf("utf16ToString(%#x) = %q, want %q", c, got, want)
		}
	}
	rng := rand.New(rand.NewPCG(1, 2))
	pick := []uint16{0, 'a', 'z', 0x7F, 0x80, 0x7FF, 0x800, 0xFFFF, 0xD7FF, 0xD800, 0xDBFF, 0xDC00, 0xDFFF, 0xE000, 0x4E2D}
	for range 20000 {
		u := make([]uint16, rng.IntN(12))
		for i := range u {
			if rng.IntN(3) == 0 {
				u[i] = uint16(rng.IntN(0x10000))
			} else {
				u[i] = pick[rng.IntN(len(pick))]
			}
		}
		if got, want := utf16ToString(u), string(utf16.Decode(u)); got != want {
			t.Fatalf("utf16ToString(%#x) = %q, want %q", u, got, want)
		}
	}
}

func TestUTF16ToStringAllocatesOnce(t *testing.T) {
	u := make([]uint16, 72000)
	for i := range u {
		u[i] = uint16('a' + i%26)
	}
	if n := testing.AllocsPerRun(10, func() { _ = utf16ToString(u) }); n > 1 {
		t.Errorf("%v allocations, want 1", n)
	}
}

func TestAppendUTF16z(t *testing.T) {
	for _, s := range []string{"", "abc", "a\x00b", "中文", "😀x", "\xff"} {
		want := append(utf16.Encode([]rune(stripNUL(s))), 0)
		got := appendUTF16z(nil, s)
		if string(utf16.Decode(got)) != string(utf16.Decode(want)) {
			t.Errorf("appendUTF16z(%q) = %#x, want %#x", s, got, want)
		}
		if got[len(got)-1] != 0 {
			t.Errorf("appendUTF16z(%q) is not NUL-terminated", s)
		}
	}
}

func stripNUL(s string) string {
	out := []rune{}
	for _, r := range s {
		if r != 0 {
			out = append(out, r)
		}
	}
	return string(out)
}
