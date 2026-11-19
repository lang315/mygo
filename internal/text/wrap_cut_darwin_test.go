//go:build darwin && !ios

package text

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

// A plain word wider than a line is cut where Core Text cuts it, after the
// last letter that fits, from the glyphs of the text's one line: its lines
// are those of the whole typesetter, and Core Text finds none of them.

// cutWrap wraps text from its one line as wrapKept does, with the font's
// check by Core Text off, and returns its lines, Core Text's lines of the
// same typesetter, and the calls of suggestLineBreak and typesetterCreateLine
// wrapKept made.
func cutWrap(t *testing.T, e *coreText, text []rune, width float32) (got, want []shapedLine, suggests, lines int) {
	t.Helper()
	style := Style{Size: 13}
	e.resetScratch()
	font := e.ctFont(style)
	u16, index := utf16Text(text)
	typesetter := e.newTypesetter(font, u16, text, style, nil, false, "")
	t.Cleanup(func() { ct.release(typesetter) })
	f := e.wrapFonts[font]
	if f == nil {
		f = &wrapFont{}
		e.wrapFonts[font] = f
	}
	f.checked, f.lines = wrapCheckedFirst, 1 // no check by Core Text
	suggest, createLine := ct.suggestLineBreak, ct.typesetterCreateLine
	ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
		suggests++
		return suggest(typesetter, start, width)
	}
	ct.typesetterCreateLine = func(typesetter uintptr, r cfRange) uintptr {
		lines++
		return createLine(typesetter, r)
	}
	got = e.wrapKept(&keptTypesetter{typesetter: typesetter, font: font}, text, style, width)
	ct.suggestLineBreak, ct.typesetterCreateLine = suggest, createLine
	if got == nil || f.disagreed {
		t.Fatalf("%d runes at %v: left to Core Text", len(text), width)
	}
	want = e.typesetLines(typesetter, len(u16), width, nil, false, false, index, style.Italic)
	return got, want, suggests, lines
}

// sameGlyphs tells the first line where got differs from want in its range,
// its glyph IDs, their X or their advances, or "".
func sameGlyphs(text []rune, got, want []shapedLine) string {
	glyphs := func(l shapedLine) string {
		var sb strings.Builder
		for _, r := range l.runs {
			for _, g := range r.glyphs {
				fmt.Fprintf(&sb, " %d@%d:%.3f/%.3f", g.ID, g.Cluster, g.X, g.Advance)
			}
		}
		return sb.String()
	}
	for i := range min(len(got), len(want)) {
		a, b := got[i], want[i]
		same := a.start == b.start && a.end == b.end && len(a.runs) == len(b.runs)
		for r := 0; same && r < len(a.runs); r++ {
			x, y := a.runs[r].glyphs, b.runs[r].glyphs
			same = len(x) == len(y)
			for k := 0; same && k < len(x); k++ {
				same = x[k].ID == y[k].ID && x[k].Cluster == y[k].Cluster &&
					math.Abs(float64(x[k].X-y[k].X)) <= wrapTolerance && math.Abs(float64(x[k].Advance-y[k].Advance)) <= wrapTolerance
			}
		}
		if !same {
			return fmt.Sprintf("line %d: %d-%d %q%s\n  Core Text's %d-%d %q%s", i,
				a.start, a.end, string(text[a.start:a.end]), glyphs(a), b.start, b.end, string(text[b.start:b.end]), glyphs(b))
		}
	}
	if len(got) != len(want) {
		return fmt.Sprintf("%d lines, Core Text's %d", len(got), len(want))
	}
	return ""
}

// plainRun is n letters, and digits every third rune when digits.
func plainRun(n int, digits bool) []rune {
	text := make([]rune, n)
	for i := range text {
		text[i] = 'a' + rune(i%26)
		if digits && i%3 == 0 {
			text[i] = '0' + rune(i%10)
		}
	}
	return text
}

func coreTextEngine(t *testing.T) *coreText {
	s := newSystem()
	s.mu.Lock()
	t.Cleanup(s.mu.Unlock)
	return s.engine().(*coreText)
}

// P1: one unbroken word, across the size where Core Text's typesetter stops
// kerning (10,240).
func TestCoreTextCutWord(t *testing.T) {
	e := coreTextEngine(t)
	for _, digits := range []bool{false, true} {
		for _, n := range []int{8192, 10240, 10241, 20000} {
			text := plainRun(n, digits)
			for _, width := range []float32{40, 400, 541} {
				got, want, _, _ := cutWrap(t, e, text, width)
				if d := sameGlyphs(text, got, want); d != "" {
					t.Fatalf("digits %v, %d runes at %v: %s", digits, n, width, d)
				}
			}
		}
	}
}

// Latin-1 letters among ASCII ones, in one unbroken word.
func TestCoreTextCutWordLatin1(t *testing.T) {
	e := coreTextEngine(t)
	text := []rune(strings.Repeat("aéxßyüz", 8192/7))
	for _, width := range []float32{40, 400} {
		got, want, suggests, _ := cutWrap(t, e, text, width)
		if d := sameGlyphs(text, got, want); d != "" {
			t.Fatalf("at %v: %s", width, d)
		}
		if suggests != 0 {
			t.Errorf("at %v: Core Text found %d lines", width, suggests)
		}
	}
}

// P2: Core Text finds no line of an unbroken plain word, and shapes it once.
func TestCoreTextCutWordCalls(t *testing.T) {
	e := coreTextEngine(t)
	cutWrap(t, e, plainRun(10, false), 541) // the font's space, measured once
	for _, n := range []int{8192, 32768, 131072} {
		_, _, suggests, lines := cutWrap(t, e, plainRun(n, false), 541)
		if suggests != 0 || lines > 1 {
			t.Errorf("%d letters: %d calls of suggestLineBreak, %d of typesetterCreateLine; want 0 and at most 1", n, suggests, lines)
		}
	}
}

// P3: a word of 500 letters between spaced text.
func TestCoreTextCutWordBetweenSpaces(t *testing.T) {
	e := coreTextEngine(t)
	spaced := plainRun(600, false)
	for i := 6; i < len(spaced); i += 7 {
		spaced[i] = ' '
	}
	text := slices.Concat(spaced, []rune{' '}, plainRun(500, false), []rune{' '}, spaced)
	for _, width := range []float32{40, 400} {
		got, want, _, _ := cutWrap(t, e, text, width)
		if d := sameGlyphs(text, got, want); d != "" {
			t.Fatalf("at %v: %s", width, d)
		}
		cut := false
		for _, l := range got {
			cut = cut || l.start >= 601 && l.end < 1101 && l.end > l.start
		}
		if !cut {
			t.Errorf("at %v: no line inside the word", width)
		}
	}
}

// P4: a width narrower than any glyph holds a rune a line.
func TestCoreTextCutWordNarrow(t *testing.T) {
	e := coreTextEngine(t)
	text := plainRun(300, true)
	got, want, suggests, _ := cutWrap(t, e, text, 0.5)
	if d := sameGlyphs(text, got, want); d != "" {
		t.Fatal(d)
	}
	if suggests != 0 {
		t.Errorf("Core Text found %d lines", suggests)
	}
	for i, l := range got {
		if l.end-l.start != 1 {
			t.Fatalf("line %d holds %d runes", i, l.end-l.start)
		}
	}
}

// The last letter of a word, alone on a line too narrow for it, before one
// space or several and more text.
func TestCoreTextCutWordLastLetter(t *testing.T) {
	e := coreTextEngine(t)
	for _, spaces := range []string{" ", "    "} {
		text := []rune("abcdefW" + spaces + "next words" + spaces + "andW" + spaces + "end")
		for _, width := range []float32{0.5, 3} {
			got, want, _, _ := cutWrap(t, e, text, width)
			if d := sameGlyphs(text, got, want); d != "" {
				t.Fatalf("%d spaces at %v: %s", len(spaces), width, d)
			}
		}
	}
}

// P5: a width of exactly the first k advances, the tie Core Text decides.
func TestCoreTextCutWordTie(t *testing.T) {
	e := coreTextEngine(t)
	text := plainRun(400, false)
	style := Style{Size: 13}
	font := e.ctFont(style)
	u16, _ := utf16Text(text)
	typesetter := e.newTypesetter(font, u16, text, style, nil, false, "")
	w := e.shapeWhole(typesetter, text, false)
	ct.release(typesetter)
	if w == nil {
		t.Fatal("not one run")
	}
	for _, k := range []int{1, 2, 37} {
		sum := w.pens[k] - w.pens[0]
		// The width, and the width whose limit, 1/1024 wider, is the sum.
		tie := float32(sum - 1.0/1024)
		if float64(tie)+1.0/1024 != sum {
			t.Fatalf("%d letters: no width whose limit is their sum %v", k, sum)
		}
		for _, width := range []float32{float32(sum), tie} {
			got, want, _, _ := cutWrap(t, e, text, width)
			if d := sameGlyphs(text, got, want); d != "" {
				t.Fatalf("width of %d letters, %v: %s", k, width, d)
			}
		}
	}
}

// P6: a word that may break inside itself goes to Core Text.
func TestCoreTextCutWordNotPlain(t *testing.T) {
	e := coreTextEngine(t)
	// Words that end in a letter or a digit, with the other mark inside.
	for _, blob := range []string{"aB3+", "aB3/", "a+b/", "aB3+x", "ab/cd", "a-b"} {
		text := []rune(strings.Repeat(blob, 8192/len(blob)))
		for _, width := range []float32{40, 400} {
			got, want, suggests, _ := cutWrap(t, e, text, width)
			if d := sameGlyphs(text, got, want); d != "" {
				t.Fatalf("%q at %v: %s", blob, width, d)
			}
			if suggests == 0 {
				t.Errorf("%q at %v: Core Text found no line", blob, width)
			}
		}
	}
}
