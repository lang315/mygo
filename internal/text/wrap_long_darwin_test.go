package text

import (
	"strings"
	"testing"
)

// A paragraph without spaces, as a base64 blob or a minified line, wraps from
// one line in steps linear in its length, into the lines Core Text makes
// (R2-15): each line used to walk the rest of the paragraph.
func TestCoreTextWrapUnbrokenLinear(t *testing.T) {
	s := newSystem()
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.engine().(*coreText)
	style := Style{Size: 13}
	const width = 300
	for _, blob := range []string{"aB3x", "aB3+"} {
		for _, prefix := range []string{"", "hello world "} {
			steps := func(n int) uint64 {
				text := []rune(prefix + strings.Repeat(blob, n/4) + " end")
				e.resetScratch()
				font := e.ctFont(style)
				e.u16, e.index = appendUTF16(e.u16[:0], e.index[:0], text)
				typesetter := e.newTypesetter(font, e.u16, text, style, nil, false, "")
				defer ct.release(typesetter)
				kept := &keptTypesetter{typesetter: typesetter, font: font}
				if f := e.wrapFonts[font]; f != nil {
					f.checked, f.lines = wrapCheckedFirst, 1 // no check by Core Text
				}
				before := e.wrapSteps
				got := e.wrapKept(kept, text, style, width)
				walked := e.wrapSteps - before
				if got == nil {
					t.Fatalf("%q×%d: left to Core Text", blob, n/4)
				}
				e.u16, e.index = appendUTF16(e.u16[:0], e.index[:0], text)
				want := e.typesetLines(typesetter, len(e.u16), width, nil, false, false, e.index, style.Italic)
				if !sameLines(got, want) {
					report(t, text, style, width, got, want)
				}
				return walked
			}
			small, large := steps(16<<10), steps(64<<10)
			if large > small*9/2 {
				t.Errorf("%q%q: wrapping 4× the runes walked %d runes, %.1f× the %d of 16K", prefix, blob, large, float64(large)/float64(small), small)
			}
		}
	}
}
