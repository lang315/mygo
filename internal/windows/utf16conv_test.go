package windows

import (
	"math/rand/v2"
	"strconv"
	"strings"
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

func TestPostBufferAllocatesOnceForABigReply(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    int
		// allocs a post takes once the window has posted one of the same size
		steady float64
	}{
		{"300 KiB", 300 << 10, 0},
		{"2 MiB, more than the buffer is kept", 2 << 20, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := strings.Repeat("a", tc.n)
			var p postBuffer
			u, owned := p.take(s)
			p.give(u, owned)
			if got := testing.AllocsPerRun(5, func() { u, owned := p.take(s); p.give(u, owned) }); got != tc.steady {
				t.Errorf("%v allocations a post, want %v", got, tc.steady)
			}
			// The first post of a window grows no buffer by doubling.
			var q postBuffer
			if got := testing.AllocsPerRun(5, func() { q = postBuffer{}; u, owned := q.take(s); q.give(u, owned) }); got != 1 {
				t.Errorf("%v allocations for the first post, want 1", got)
			}
		})
	}
}

func TestPostBufferNestedPostGetsItsOwn(t *testing.T) {
	var p postBuffer
	a, aOwned := p.take("outer")
	b, bOwned := p.take("inner")
	if !aOwned || bOwned {
		t.Fatalf("owned = %v, %v; want true, false", aOwned, bOwned)
	}
	if string(utf16.Decode(a[:len(a)-1])) != "outer" || string(utf16.Decode(b[:len(b)-1])) != "inner" {
		t.Errorf("the inner post wrote over the outer: %q, %q", utf16.Decode(a), utf16.Decode(b))
	}
	p.give(b, bOwned)
	if !p.busy {
		t.Error("giving back the inner copy freed the outer's buffer")
	}
	p.give(a, aOwned)
	if p.busy {
		t.Error("the buffer is still busy once the outer copy is given back")
	}
}

// BenchmarkPostBuffer is the first post of a window (or one after a huge reply),
// the one that had the buffer grow by doubling.
func BenchmarkPostBuffer(b *testing.B) {
	for _, n := range []int{300 << 10, 2 << 20} {
		s := strings.Repeat("a", n)
		b.Run(strconv.Itoa(n>>10)+"KiB", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var p postBuffer
				u, owned := p.take(s)
				p.give(u, owned)
			}
		})
	}
}
