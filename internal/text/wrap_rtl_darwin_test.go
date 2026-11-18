package text

import (
	"slices"
	"strings"
	"testing"
)

// A text Core Text lays out, as a right-to-left paragraph, breaks into lines
// at the spaces before plain words from the advances of its one line, found
// in Go: Core Text lays each line out, from the whole typesetter, and finds
// none of them. Its lines are those of typesetLines.

// arabic are Arabic letters, without the tatweel.
var arabic = []rune("بتثجحخدذرزسشصضطظعغفقكلمنهوي")

// arabSpaced is n Arabic letters with a space every 40 runes.
func arabSpaced(n int) []rune {
	text := make([]rune, n)
	for i := range text {
		text[i] = arabic[i%len(arabic)]
		if i%40 == 39 {
			text[i] = ' '
		}
	}
	return text
}

// arabWords is the words of ws, each of letters from arabic, of the lengths in
// ws, with a space after each but the last.
func arabWords(ws ...int) []rune {
	var text []rune
	for i, w := range ws {
		for j := range w {
			text = append(text, arabic[(i*7+j)%len(arabic)])
		}
		if i < len(ws)-1 {
			text = append(text, ' ')
		}
	}
	return text
}

// laidWrap wraps text through shapeText with the fonts' checks by Core Text
// off, and returns its lines, the typesetter kept of it, the calls of
// suggestLineBreak and the runes wrapKept walked.
func laidWrap(t *testing.T, e *coreText, text []rune, style Style, width float32, rtl bool) (got []shapedLine, kept *keptTypesetter, suggests int, steps uint64) {
	t.Helper()
	font := e.ctFont(style)
	for _, fonts := range []map[uintptr]*wrapFont{e.wrapFonts, e.laidFonts} {
		f := fonts[font]
		if f == nil {
			f = &wrapFont{}
			fonts[font] = f
		}
		f.checked, f.lines = wrapCheckedFirst, 1 // no check by Core Text
	}
	e.resetScratch()
	suggest := ct.suggestLineBreak
	defer func() { ct.suggestLineBreak = suggest }()
	ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
		suggests++
		return suggest(typesetter, start, width)
	}
	before := e.wrapSteps
	got = e.shapeText(text, style, nil, width, rtl, false)
	steps = e.wrapSteps - before
	kept = e.typesetters[typesetterKey{font, rtl, "", style.LetterSpacing, noKerning(style.Features), textHash(e.textSeed, text)}]
	return got, kept, suggests, steps
}

// cuts counts the lines that end inside a word: those Core Text finds, of a
// word too long for a line.
func cuts(text []rune, lines []shapedLine) int {
	n := 0
	for _, l := range lines {
		if l.end < len(text) && text[l.end-1] != ' ' {
			n++
		}
	}
	return n
}

// laidSame wraps text as laidWrap does, fails the test where its lines are
// not Core Text's, and returns the calls of suggestLineBreak and the lines.
func laidSame(t *testing.T, e *coreText, text []rune, width float32, rtl bool) (int, []shapedLine) {
	t.Helper()
	style := Style{Size: 13}
	got, kept, suggests, _ := laidWrap(t, e, text, style, width, rtl)
	if kept == nil {
		t.Fatalf("%d runes at %v: no typesetter kept", len(text), width)
	}
	want := e.typesetLines(kept.typesetter, len(text), width, nil, false, false, runeIndex(nil, len(text)), style.Italic)
	if d := sameGlyphs(text, got, want); d != "" {
		t.Fatalf("%d runes at %v: %s", len(text), width, d)
	}
	if !sameLines(got, want) {
		t.Fatalf("%d runes at %v: the lines' fonts, runs or directions differ from Core Text's", len(text), width)
	}
	return suggests, got
}

// runeWidth is the mean advance of a rune of text in its one line, which a
// text of more than 10,240 units shapes otherwise than a shorter one.
func runeWidth(e *coreText, text []rune, rtl bool) float32 {
	e.resetScratch()
	var w float32
	for _, l := range e.shapeText(text, Style{Size: 13}, nil, 0, rtl, false) {
		for _, r := range l.runs {
			for _, g := range r.glyphs {
				w += g.Advance
			}
		}
	}
	return w / float32(len(text))
}

// 1: Arabic with a space every 40 runes, at about 80 runes a line, and at
// widths of two to five words.
func TestCoreTextLaidArabic(t *testing.T) {
	e := coreTextEngine(t)
	text := arabSpaced(8192)
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{80, 85, 120, 163, 200} {
		if s, _ := laidSame(t, e, text, rw*runes, true); s != 0 {
			t.Errorf("%v runes a line: Core Text found %d lines", runes, s)
		}
	}
}

// 2: the same past the 10,240 units where Core Text shapes otherwise.
func TestCoreTextLaidArabicLong(t *testing.T) {
	e := coreTextEngine(t)
	text := arabSpaced(32768)
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{80, 120, 200} {
		if s, _ := laidSame(t, e, text, rw*runes, true); s != 0 {
			t.Errorf("%v runes a line: Core Text found %d lines", runes, s)
		}
	}
}

// 3: two and three spaces between words, and a text ending in spaces. At 40
// runes a line, words of 39 letters wider than a line go to Core Text.
func TestCoreTextLaidSpaces(t *testing.T) {
	e := coreTextEngine(t)
	for _, spaces := range []string{"  ", "   "} {
		text := []rune(strings.ReplaceAll(string(arabSpaced(4000)), " ", spaces) + spaces)
		rw := runeWidth(e, text, true)
		for _, runes := range []float32{40, 80, 81, 82, 120} {
			if s, got := laidSame(t, e, text, rw*runes, true); s != cuts(text, got) {
				t.Errorf("%d spaces, %v runes a line: Core Text found %d lines, want the %d inside words", len(spaces), runes, s, cuts(text, got))
			}
		}
	}
}

// 4: a word of 500 letters between spaced text goes to Core Text, and only
// the words too long for a line.
func TestCoreTextLaidLongWord(t *testing.T) {
	e := coreTextEngine(t)
	spaced := arabSpaced(600)
	long := arabWords(500)
	text := slices.Concat(spaced, []rune{' '}, long, []rune{' '}, spaced)
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{40, 400} {
		s, got := laidSame(t, e, text, rw*runes, true)
		inWord := 0
		for _, l := range got {
			if l.start >= 601 && l.start < 1101 && l.end < 1101 {
				inWord++
			}
		}
		if inWord == 0 || s != cuts(text, got) {
			t.Errorf("%v runes a line: Core Text found %d lines, want the %d inside words, %d of them in the long word", runes, s, cuts(text, got), inWord)
		}
	}
}

// 5: Arabic words among Latin words and numbers, as 123-456, and a Latin word
// before a number. Core Text finds the lines that end before a word that may
// break inside itself, as 123-456.
func TestCoreTextLaidMixed(t *testing.T) {
	e := coreTextEngine(t)
	var sb strings.Builder
	for i := range 60 {
		sb.WriteString(string(arabWords(5+i%7, 3, 9)))
		switch i % 4 {
		case 0:
			sb.WriteString(" hello world ")
		case 1:
			sb.WriteString(" 123-456 ")
		case 2:
			sb.WriteString(" version 2024 ")
		case 3:
			sb.WriteString(" ٣٤٥ ")
		}
	}
	text := []rune(strings.TrimSpace(sb.String()))
	rw := runeWidth(e, text, true)
	width := rw * float32(len(text)) / 20
	s, got := laidSame(t, e, text, width, true)
	if len(got) < 20 {
		t.Errorf("%d lines, want 20 or more", len(got))
	}
	before := 0
	for _, l := range got {
		if strings.HasPrefix(string(text[l.end:]), "123-456") {
			before++
		}
	}
	// The path takes the text, and leaves to Core Text only the lines ending
	// before 123-456: some, not all.
	if s != before || before == 0 || before == len(got) {
		t.Errorf("Core Text found %d of %d lines, want the %d ending before 123-456, and at least one", s, len(got), before)
	}
}

// 6: Core Text finds no line of a long Arabic paragraph, and the runes walked
// grow as the paragraph.
func TestCoreTextLaidCalls(t *testing.T) {
	e := coreTextEngine(t)
	const perRune = 4
	for _, n := range []int{16384, 131072} {
		text := arabSpaced(n)
		_, _, suggests, steps := laidWrap(t, e, text, Style{Size: 13}, runeWidth(e, text, true)*80, true)
		t.Logf("%d runes: %d calls of suggestLineBreak, %d runes walked", n, suggests, steps)
		if suggests != 0 || steps > perRune*uint64(n) {
			t.Errorf("%d runes: %d calls of suggestLineBreak and %d runes walked; want 0 and at most %d", n, suggests, steps, perRune*n)
		}
	}
}

// Arabic with its marks, which go to Core Text when they open the next word.
func TestCoreTextLaidMarks(t *testing.T) {
	e := coreTextEngine(t)
	var sb strings.Builder
	for i := range 300 {
		sb.WriteString(string(arabWords(4 + i%5)))
		sb.WriteRune(rune(0x064B + i%8))
		sb.WriteString(string(arabWords(3)))
		sb.WriteString(" ")
	}
	text := []rune(sb.String())
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{40, 80} {
		if s, got := laidSame(t, e, text, rw*runes, true); s <= cuts(text, got) {
			t.Errorf("at %v runes a line: Core Text found %d lines, only the %d inside words", runes, s, cuts(text, got))
		}
	}
}

// lam-alef, one glyph for two runes, and the tatweel.
func TestCoreTextLaidLigatures(t *testing.T) {
	e := coreTextEngine(t)
	for _, word := range []string{"سلام", "لالالا", "كـتـاب", "بلا"} {
		text := []rune(strings.Repeat(word+" "+string(arabWords(6))+" ", 400))
		rw := runeWidth(e, text, true)
		for _, runes := range []float32{30, 80} {
			if s, _ := laidSame(t, e, text, rw*runes, true); s != 0 {
				t.Errorf("%q at %v runes a line: Core Text found %d lines", word, runes, s)
			}
		}
	}
}

// Persian with a zero-width non-joiner inside words, and Arabic punctuation
// after them: words that go to Core Text when they are the next.
func TestCoreTextLaidPersianPunctuation(t *testing.T) {
	e := coreTextEngine(t)
	for _, word := range []string{"می‌خواهم", "کتاب‌ها", "نعم،", "لماذا؟", "قال؛"} {
		t.Run(word, func(t *testing.T) {
			var sb strings.Builder
			for i := range 400 {
				sb.WriteString(string(arabWords(2 + i%7)))
				sb.WriteString(" " + word + " ")
			}
			text := []rune(sb.String())
			rw := runeWidth(e, text, true)
			all := 0
			for _, runes := range []float32{30, 37, 80} {
				s, got := laidSame(t, e, text, rw*runes, true)
				before := 0
				for _, l := range got {
					if strings.HasPrefix(string(text[l.end:]), word) {
						before++
					}
				}
				if s != before+cuts(text, got) {
					t.Errorf("at %v runes a line: Core Text found %d lines, want the %d ending before it and the %d inside words", runes, s, before, cuts(text, got))
				}
				all += before
			}
			if all == 0 {
				t.Error("no line ends before it")
			}
		})
	}
}

// Hebrew, Arabic in a left-to-right paragraph, and Latin in a right-to-left
// one.
func TestCoreTextLaidScripts(t *testing.T) {
	e := coreTextEngine(t)
	for _, c := range []struct {
		name string
		text []rune
		rtl  bool
	}{
		{"hebrew", []rune(strings.Repeat("שלום עולם גדול מאוד ", 400)), true},
		{"arabic ltr", arabSpaced(8192), false},
		{"latin rtl", []rune(strings.Repeat("hello world, wide words ", 400)), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			rw := runeWidth(e, c.text, c.rtl)
			for _, runes := range []float32{40, 80} {
				if s, got := laidSame(t, e, c.text, rw*runes, c.rtl); s != cuts(c.text, got) {
					t.Errorf("at %v runes a line: Core Text found %d lines, want the %d inside words", runes, s, cuts(c.text, got))
				}
			}
		})
	}
}

// One kept text wrapped at two widths, one after the other, and at widths of
// exactly a line of Core Text's, the tie it decides.
func TestCoreTextLaidWidths(t *testing.T) {
	e := coreTextEngine(t)
	text := arabSpaced(4000)
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{80, 120, 80} {
		if s, _ := laidSame(t, e, text, rw*runes, true); s != 0 {
			t.Errorf("%v runes a line: Core Text found %d lines", runes, s)
		}
	}
	// The widths of Core Text's first lines, without the space ending them,
	// and the widths whose limit, 1/1024 wider, is that sum.
	e.resetScratch()
	u16, index := utf16Text(text)
	style := Style{Size: 13}
	ts := e.newTypesetter(e.ctFont(style), u16, text, style, nil, true, "")
	defer ct.release(ts)
	whole := e.measureWhole(ts, text)
	ties := 0
	for _, l := range e.typesetLines(ts, len(text), rw*120, nil, false, false, index, false)[:20] {
		sum := whole.pens[l.end-1] - whole.pens[l.start]
		tie := float32(sum - 1.0/1024)
		if float64(tie)+1.0/1024 != sum {
			continue
		}
		ties++
		for _, width := range []float32{float32(sum), tie} {
			if s, _ := laidSame(t, e, text, width, true); s != 0 {
				t.Errorf("width %v: Core Text found %d lines", width, s)
			}
		}
	}
	t.Logf("%d of 20 lines give a tie", ties)
	if ties == 0 {
		t.Error("no width whose limit is a line's sum")
	}
}

// What wrapping keeps of a font, in either path, goes with the fonts: its key
// is the font's address, which another font may take.
func TestCoreTextLaidForget(t *testing.T) {
	for _, c := range []struct {
		name   string
		forget func(e *coreText)
	}{
		{"forgetFonts", (*coreText).forgetFonts},
		{"forgetPrimary", (*coreText).forgetPrimary},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := coreTextEngine(t)
			style := Style{Size: 13}
			e.resetScratch()
			e.shapeText(arabSpaced(400), style, nil, 300, true, false)
			e.shapeText([]rune(strings.Repeat("plain words ", 40)), style, nil, 300, false, false)
			if len(e.laidFonts) == 0 || len(e.wrapFonts) == 0 {
				t.Fatalf("%d laid fonts and %d Latin ones kept, want both", len(e.laidFonts), len(e.wrapFonts))
			}
			c.forget(e)
			if len(e.laidFonts) != 0 || len(e.wrapFonts) != 0 {
				t.Errorf("%d laid fonts and %d Latin ones kept after %s, want none", len(e.laidFonts), len(e.wrapFonts), c.name)
			}
		})
	}
}

// laidText takes the runes at its bounds, and none beside them.
func TestCoreTextLaidTextRunes(t *testing.T) {
	in := []rune{0x20, 0x7E, 0xC0, 0xFF, 0x060C, 0x061B, 0x061F, 0x0621, 0x0669, 0x0670, 0x06D3, 0x05D0, 0x05EA, 0x200C}
	out := []rune{0x1F, 0x7F, 0xBF, 0x100, 0x060B, 0x060D, 0x061A, 0x061C, 0x061E, 0x0620, 0x066A, 0x066F, 0x06D4, 0x05CF, 0x05EB, 0x200B, 0x200D}
	for _, r := range in {
		if !laidText([]rune{'a', r}) {
			t.Errorf("U+%04X left out", r)
		}
	}
	for _, r := range out {
		if laidText([]rune{'a', r}) {
			t.Errorf("U+%04X taken", r)
		}
	}
}

// Words of the runes at laidText's bounds wrap as Core Text wraps them.
func TestCoreTextLaidEdgeRunes(t *testing.T) {
	e := coreTextEngine(t)
	// U+0670, a mark, is laidText's too, but a word holding it goes to Core
	// Text as the next word, as TestCoreTextLaidMarks holds.
	edges := []string{"ءبت", "بت٩", "ۓبت", "שלת", "ÀéÿÖ"}
	var sb strings.Builder
	for i := range 400 {
		sb.WriteString(string(arabWords(2 + i%6)))
		sb.WriteString(" " + edges[i%len(edges)] + " ")
	}
	text := []rune(sb.String())
	rw := runeWidth(e, text, true)
	for _, runes := range []float32{30, 37, 80} {
		if s, got := laidSame(t, e, text, rw*runes, true); s != cuts(text, got) {
			t.Errorf("at %v runes a line: Core Text found %d lines, want the %d inside words", runes, s, cuts(text, got))
		}
	}
}

// Letter spacing, spans and Han are left to Core Text.
func TestCoreTextLaidLeft(t *testing.T) {
	e := coreTextEngine(t)
	text := arabSpaced(4000)
	t.Run("letter spacing", func(t *testing.T) {
		got, _, s, _ := laidWrap(t, e, text, Style{Size: 13, LetterSpacing: 1}, 300, true)
		if s != len(got) {
			t.Errorf("Core Text found %d of %d lines", s, len(got))
		}
	})
	t.Run("han", func(t *testing.T) {
		han := []rune(strings.Repeat("中文字 ", 1000))
		got, _, s, _ := laidWrap(t, e, han, Style{Size: 13}, 300, true)
		if s != len(got) {
			t.Errorf("Core Text found %d of %d lines", s, len(got))
		}
	})
	t.Run("spans", func(t *testing.T) {
		e.resetScratch()
		suggest := ct.suggestLineBreak
		defer func() { ct.suggestLineBreak = suggest }()
		s := 0
		ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
			s++
			return suggest(typesetter, start, width)
		}
		got := e.shapeText(text, Style{Size: 13}, []Span{{End: 10, Size: 15}}, 300, true, false)
		if s != len(got) {
			t.Errorf("Core Text found %d of %d lines", s, len(got))
		}
	})
}

// The fonts' checks: a font where Core Text's lines differ from those found
// in Go is left to it, for texts it lays out only.
func TestCoreTextLaidCheck(t *testing.T) {
	e := coreTextEngine(t)
	style := Style{Size: 13}
	font := e.ctFont(style)
	e.wrapFonts[font] = &wrapFont{checked: wrapCheckedFirst, lines: 1}
	delete(e.laidFonts, font)
	text := arabSpaced(2000)
	e.resetScratch()
	suggest := ct.suggestLineBreak
	defer func() { ct.suggestLineBreak = suggest }()
	lied := false
	ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
		n := suggest(typesetter, start, width)
		if !lied && n > 1 {
			lied = true
			return n - 1
		}
		return n
	}
	e.shapeText(text, style, nil, 300, true, false)
	ct.suggestLineBreak = suggest
	if f := e.laidFonts[font]; f == nil || !f.disagreed {
		t.Fatal("Core Text's lines differed, but the font is not left to it")
	}
	if e.wrapFonts[font].disagreed {
		t.Error("the font is left to Core Text for plain Latin text too")
	}
	// Plain Latin text still wraps in Go.
	_, _, s, _ := laidWrap(t, e, []rune(strings.Repeat("plain words ", 400)), style, 300, false)
	if s != 0 {
		t.Errorf("plain Latin: Core Text found %d lines", s)
	}
}
