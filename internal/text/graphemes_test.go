package text

import "testing"

// NextGrapheme and PrevGrapheme answer from the grapheme offsets at every
// index, the ends included.
func TestBoundariesGraphemeSearch(t *testing.T) {
	for _, s := range []string{"", "a", "éx\r\n👍🏽🇹🇭🇹", "\U00000915\U0000094D\U00000915ab"} {
		var b Boundaries
		b.Reset([]rune(s))
		offs := b.GraphemeOffsets()
		n := len([]rune(s))
		for i := 0; i <= n; i++ {
			next, prev := n, 0
			for _, g := range offs {
				if g > i {
					next = g
					break
				}
			}
			for _, g := range offs {
				if g < i {
					prev = g
				}
			}
			if got := b.NextGrapheme(i); got != next {
				t.Errorf("%q: NextGrapheme(%d) = %d, want %d", s, i, got, next)
			}
			if got := b.PrevGrapheme(i); got != prev {
				t.Errorf("%q: PrevGrapheme(%d) = %d, want %d", s, i, got, prev)
			}
		}
		// Past the ends, as the scans gave.
		if got := b.NextGrapheme(n + 1); got != n {
			t.Errorf("%q: NextGrapheme(%d) = %d, want %d", s, n+1, got, n)
		}
		if got := b.PrevGrapheme(-1); got != 0 {
			t.Errorf("%q: PrevGrapheme(-1) = %d, want 0", s, got)
		}
	}
}
