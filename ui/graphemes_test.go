package ui

import (
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/text"
)

// segmentBound is the most runes one keystroke may hand the segmenter of a
// text input's graphemes, however long its paragraph.
const segmentBound = 16384

// graphemeOracle returns, for every index of s, the end of the grapheme at
// it and the start of the one before, from the whole of each paragraph
// given to a new text.Boundaries, as the buffer splits them: after each
// newline.
func graphemeOracle(s string) (next, prev []int) {
	runes := []rune(s)
	n := len(runes)
	next, prev = make([]int, n+1), make([]int, n+1)
	next[n] = n
	for start := 0; start < n; {
		end := start
		for end < n && runes[end] != '\n' {
			end++
		}
		if end < n {
			end++
		}
		var tb text.Boundaries
		tb.Reset(runes[start:end])
		offs := tb.GraphemeOffsets()
		// Within the paragraph, from its offsets alone.
		k := 0
		for i := start; i < end; i++ {
			for offs[k] <= i-start {
				k++
			}
			next[i] = start + offs[k]
		}
		k = 0
		for i := start + 1; i <= end; i++ {
			for offs[k+1] < i-start {
				k++
			}
			prev[i] = start + offs[k]
		}
		start = end
	}
	return next, prev
}

// checkGraphemes compares next and prev at every index of s, in a buffer of
// each kind, with the oracle.
func checkGraphemes(t *testing.T, s string) {
	t.Helper()
	next, prev := graphemeOracle(s)
	for _, indexed := range []bool{false, true} {
		var b buffer
		if indexed {
			b.setSnapshot(NewTextBuffer(s).Snapshot())
		} else {
			b.set(s)
		}
		if indexed {
			// The windows are the same: only slicing the rope differs.
			walk(t, &b, &graphemes{}, next, prev, 0, b.n)
			continue
		}
		checkRange(t, &b, &graphemes{}, next, prev, 0, b.n)
		if b.n > 8192 {
			continue // under -race, too slow where no anchor is near
		}
		// Asked first where a window reaches just inside or past an end of
		// the text, once for each size it grows to, then on to that end
		// in the same window.
		for h := graphemeWindow; h <= 2*b.n; h *= 2 {
			for d := -1; d <= 1; d++ {
				if i := h + d; i <= b.n {
					walk(t, &b, &graphemes{}, next, prev, i, 0)
				}
				if i := b.n - h + d; i >= 0 {
					walk(t, &b, &graphemes{}, next, prev, i, b.n)
				}
			}
		}
	}
}

// checkRange compares next and prev at every index from a to z with the
// oracle's, walking up, then down: a window is asked near its end on the
// way up and near its start on the way down.
func checkRange(t *testing.T, b *buffer, g *graphemes, next, prev []int, a, z int) {
	t.Helper()
	if walk(t, b, g, next, prev, a, z) {
		walk(t, b, g, next, prev, z, a)
	}
}

// walk compares next and prev at every index from a to z, either way, with
// the oracle's, and reports whether they all equal it.
func walk(t *testing.T, b *buffer, g *graphemes, next, prev []int, a, z int) bool {
	t.Helper()
	bad := 0
	check := func(i int) {
		if got := g.next(b, i); got != next[i] {
			t.Errorf("indexed %v: next(%d) = %d, want %d", b.indexed, i, got, next[i])
			bad++
		}
		if got := g.prev(b, i); got != prev[i] {
			t.Errorf("indexed %v: prev(%d) = %d, want %d", b.indexed, i, got, prev[i])
			bad++
		}
	}
	step := 1
	if z < a {
		step = -1
	}
	for i := a; bad < 5; i += step {
		check(i)
		if i == z {
			break
		}
	}
	return bad == 0
}

// backspaceSegments returns the runes the graphemes of a text input
// holding s segment for one Backspace with the caret in the middle.
func backspaceSegments(t *testing.T, s string) int {
	t.Helper()
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	_, ed := inputEditor(t, tt)
	n := ed.buf.n
	ed.caret, ed.anchor = n/2, n/2
	ed.graphemes.segmented = 0
	tt.Key(0, KeyBackspace)
	if got := utf8.RuneCountInString(s); got != n-1 || ed.caret != n/2-1 {
		t.Fatalf("after Backspace: %d runes, caret %d; want %d and %d", got, ed.caret, n-1, n/2-1)
	}
	return ed.graphemes.segmented
}

// C1: a Backspace in a long paragraph segments a bounded number of runes,
// which does not grow with the paragraph.
func TestGraphemesBackspaceCostPlain(t *testing.T) {
	long := backspaceSegments(t, strings.Repeat("a", 131072))
	if long > segmentBound {
		t.Errorf("Backspace in 131072 letters segmented %d runes, want at most %d", long, segmentBound)
	}
	short := backspaceSegments(t, strings.Repeat("a", 32768))
	t.Logf("Backspace segmented %d runes in 131072 letters, %d in 32768", long, short)
	if long > 2*short || short > 2*long {
		t.Errorf("Backspace segmented %d runes in 131072 letters, %d in 32768: want within 2×", long, short)
	}
}

// C2: the same without ASCII.
func TestGraphemesBackspaceCostCJK(t *testing.T) {
	var sb strings.Builder
	for i := range 131072 {
		sb.WriteRune(0x4E00 + rune(i%20000))
	}
	if got := backspaceSegments(t, sb.String()); got > segmentBound {
		t.Errorf("Backspace in 131072 ideographs segmented %d runes, want at most %d", got, segmentBound)
	}
}

// C3: regional indicators pair from the start of their run.
func TestGraphemesRegionalIndicators(t *testing.T) {
	ri := strings.Repeat("\U0001F1E6", 20000)
	for _, s := range []string{"a" + ri, ri, ri + "\U0001F1E6"} {
		checkGraphemes(t, s)
	}
}

// C4: emoji joined by ZWJ.
func TestGraphemesZWJEmoji(t *testing.T) {
	family := "\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467\U0000200D\U0001F466"
	checkGraphemes(t, strings.Repeat(family, 30000/7+1))
	checkGraphemes(t, strings.Repeat("\U0001F468\U0000200D", 10000)+"\U0001F468")
}

// C5: a letter with many combining marks is one grapheme.
func TestGraphemesCombiningMarks(t *testing.T) {
	s := "a" + strings.Repeat("\U00000301", 20000) + "b"
	var b buffer
	b.set(s)
	var g graphemes
	if got := g.next(&b, 0); got != 20001 {
		t.Errorf("next(0) = %d, want 20001", got)
	}
	if got := g.prev(&b, 20001); got != 0 {
		t.Errorf("prev(20001) = %d, want 0", got)
	}
	checkGraphemes(t, s)
}

// C6: CR LF is one grapheme, and a newline ends its paragraph.
func TestGraphemesCRLF(t *testing.T) {
	s := strings.Repeat("x", 100000) + "\r\ny"
	n := 100000
	var b buffer
	b.set(s)
	var g graphemes
	if got := g.next(&b, n); got != n+2 {
		t.Errorf("next(%d) = %d, want %d", n, got, n+2)
	}
	if got := g.prev(&b, n+2); got != n {
		t.Errorf("prev(%d) = %d, want %d", n+2, got, n)
	}
	// Marks before it leave no anchor but between CR and LF.
	b.set("a" + strings.Repeat("\U00000301", 600) + "\r\ny")
	g = graphemes{}
	if got := g.prev(&b, 603); got != 601 {
		t.Errorf("prev(603) after marks = %d, want 601, the CR", got)
	}
	b.set("ab\ncd")
	g = graphemes{}
	if got := g.prev(&b, 3); got != 2 {
		t.Errorf("prev of the paragraph's start = %d, want 2, its newline", got)
	}
}

// C7: Indic conjuncts.
func TestGraphemesIndicConjuncts(t *testing.T) {
	checkGraphemes(t, strings.Repeat("\U00000915\U0000094D", 10000)+"\U00000915")
}

// C8: short texts and the ends.
func TestGraphemesShortAndEdges(t *testing.T) {
	for _, s := range []string{"", "ab\n\ncd", "0123456789", "é\r\n👍🏽\n\nz"} {
		checkGraphemes(t, s)
		var b buffer
		b.set(s)
		var g graphemes
		if got := g.next(&b, b.n); got != b.n {
			t.Errorf("%q: next(%d) = %d", s, b.n, got)
		}
		if got := g.prev(&b, 0); got != 0 {
			t.Errorf("%q: prev(0) = %d", s, got)
		}
	}
}

// mixed returns n runes of ASCII, C4's emoji and C5's marks.
func mixed(rng *rand.Rand, n int) string {
	var sb strings.Builder
	for k := 0; k < n; {
		switch rng.IntN(3) {
		case 0:
			sb.WriteString("hello ")
			k += 6
		case 1:
			sb.WriteString("\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467")
			k += 5
		case 2:
			sb.WriteString("e\U00000301\U00000301")
			k += 3
		}
	}
	return string([]rune(sb.String())[:n])
}

// C9: after an edit in the second of three paragraphs, each answers from
// its own.
func TestGraphemesSeveralParagraphs(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 9))
	paras := []string{mixed(rng, 50000), mixed(rng, 50000), mixed(rng, 50000)}
	s := strings.Join(paras, "\n")
	for _, indexed := range []bool{false, true} {
		var b buffer
		if indexed {
			b.setSnapshot(NewTextBuffer(s).Snapshot())
		} else {
			b.set(s)
		}
		var g graphemes
		g.next(&b, 75000) // a window before the edit
		b.replace(75000, 75000, "\U00000301x")
		edited := paras[1][:len(string([]rune(paras[1])[:24999]))] + "\U00000301x" + paras[1][len(string([]rune(paras[1])[:24999])):]
		want := strings.Join([]string{paras[0], edited, paras[2]}, "\n")
		if b.string() != want {
			t.Fatal("the edit is not where the test expects it")
		}
		next, prev := graphemeOracle(want)
		checkRange(t, &b, &g, next, prev, 74990, 75010) // where the window was
		for _, r := range [][2]int{{0, 50000}, {50001, 100003}, {100004, b.n}} {
			if indexed {
				walk(t, &b, &g, next, prev, r[0], r[1]) // windows as above
			} else {
				checkRange(t, &b, &g, next, prev, r[0], r[1])
			}
		}
	}
}

// C10: a mark typed after the caret's letter joins it: the window of the
// text before the edit is not used after it.
func TestGraphemesAfterEdit(t *testing.T) {
	s := strings.Repeat("a", 131072)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	_, ed := inputEditor(t, tt)
	n := ed.buf.n
	ed.caret, ed.anchor = n/2, n/2
	if got := ed.graphemes.prev(&ed.buf, n/2); got != n/2-1 {
		t.Fatalf("prev(%d) = %d, want %d", n/2, got, n/2-1)
	}
	tt.Type("\U00000301")
	if ed.caret != n/2+1 {
		t.Fatalf("caret %d after typing a mark, want %d", ed.caret, n/2+1)
	}
	ed.graphemes.segmented = 0
	if got := ed.graphemes.prev(&ed.buf, n/2+1); got != n/2-1 {
		t.Errorf("prev(%d) after the mark = %d, want %d, the start of its letter", n/2+1, got, n/2-1)
	}
	if got := ed.graphemes.segmented; got > segmentBound {
		t.Errorf("prev after the edit segmented %d runes, want at most %d", got, segmentBound)
	}
}

// The window of plain text: every index, the paragraph's start and end and
// each window's edges, equal the oracle; and the windows are smaller than
// the paragraph.
func TestGraphemesWindowEdges(t *testing.T) {
	s := strings.Repeat("abcdefgh", 1024) + "\n" + strings.Repeat("今天天气", 1024)
	checkGraphemes(t, s)
	var b buffer
	b.set(s)
	var g graphemes
	g.next(&b, 4000)
	if g.segmented >= 8193 {
		t.Errorf("one query segmented %d runes, the whole paragraph", g.segmented)
	}
}

// Clusters whose graphemes need what is before them, alone or repeated,
// after and before plain text, and every guard of a window's start.
func TestGraphemesClusters(t *testing.T) {
	plain := strings.Repeat("x", 600)
	for _, c := range []string{
		"\U0001F468\U0000200D\U0001F469",           // E ZWJ E
		"\U0001F600\U0001F600\U0000200D\U0001F600", // E E ZWJ E
		"\U00000915\U0000094D\U00000915",           // C virama C
		"\U00000915\U0000094D\U00000301\U00000915", // C virama extend C
		"\U0001F1E6\U0001F1E6\U0001F1E6",           // odd RI
		"\U0001F1E6\U0001F1E6",                     // even RI
		"\U00001100\U00001161\U000011A8",           // L V T
		"a\U00000301",                              // a letter with a mark
		"\U00000600a",                              // prepend
		"x\U00000E33",                              // spacing mark
		"\r\n",
		"\u0007a",
		"\U0001F44D\U0001F3FD", // a skin tone
		"\U00002764\U0000FE0F", // VS16
	} {
		checkGraphemes(t, plain+c+plain)
		checkGraphemes(t, strings.Repeat(c, 1500/utf8.RuneCountInString(c)))
		checkGraphemes(t, plain+strings.Repeat(c, 1200/utf8.RuneCountInString(c))+plain)
	}
}

// A random text of every kind of rune the rules tell apart.
func TestGraphemesRandom(t *testing.T) {
	alphabet := []rune{'a', ' ', '一', 0x1F1E6, 0x1F600, 0x200D, 0xFE0F, 0x1F3FD, 0x0301,
		0x0915, 0x094D, 0x1100, 0x1161, 0x11A8, 0x0600, '\r', 0x0007, 0x0E33, '\n'}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 4 {
		runes := make([]rune, 5000)
		for i := range runes {
			// Mostly the runes with context, so that windows start among them.
			runes[i] = alphabet[rng.IntN(len(alphabet))]
			if rng.IntN(4) == 0 {
				runes[i] = 'a'
			}
		}
		checkGraphemes(t, string(runes))
	}
}
