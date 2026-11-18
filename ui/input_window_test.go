package ui

import (
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

// laidBy returns the runes the tester's text system laid out while f ran.
func laidBy(tt *Tester, f func()) uint64 {
	before := tt.rt.text.RunesLaid()
	f()
	return tt.rt.text.RunesLaid() - before
}

// windowBound is the most runes a keystroke lays out of a long single-line
// input: its window, each edge moved to a grapheme's, and a composition.
const windowBound = 2*inputWindow + 64

// inputEditor returns the state and editor of the tester's only text input.
func inputEditor(t *testing.T, tt *Tester) (*state, *editor) {
	t.Helper()
	for _, st := range tt.rt.states {
		if st.editor != nil {
			return st, st.editor
		}
	}
	t.Fatal("no text input")
	return nil, nil
}

// A text area as high as its text (Lines) measures its paragraphs only as far
// as its most lines: a keystroke in a long text lays out what is above that,
// not the whole text, and the area is as high as before.
func TestTextAreaLinesKeystrokeLaysOutItsLines(t *testing.T) {
	height := func(s string) float32 {
		tt := NewTester(func(c *Context) { TextArea(c, &s).Lines(1, 8).AutoFocus() }, 300, 400)
		defer tt.rt.close()
		tt.Frame()
		st, _ := inputEditor(t, tt)
		return st.h
	}
	long := strings.Repeat("hello world\n", 5461)
	s := long
	tt := NewTester(func(c *Context) { TextArea(c, &s).Lines(1, 8).AutoFocus() }, 300, 400)
	defer tt.rt.close()
	tt.Frame()
	if got := laidBy(tt, func() { tt.Type("x") }); got > 1024 {
		t.Errorf("a keystroke in %d runes laid out %d runes, want at most 1024", utf8.RuneCountInString(long), got)
	}
	if h, full := height(long), height(strings.Repeat("hello world\n", 20)); h != full {
		t.Errorf("the long text is %v high, 20 paragraphs %v: both are 8 lines", h, full)
	}
	if h, full := height("hi"), height(strings.Repeat("hello world\n", 20)); h >= full {
		t.Errorf("one line is %v high, not less than 8 lines' %v", h, full)
	}
}

// A single-line input lays out a window of a long value around the caret: a
// keystroke in a value of 128K runes, sixteen windows, that the app set lays
// out the window, not the value (R3-5), and editing, moving, selecting,
// copying and pointing go on as in the whole value.
func TestTextInputAppValueKeystrokeLaysOutWindow(t *testing.T) {
	s := strings.Repeat("ab ", 128<<10/3)
	n := utf8.RuneCountInString(s)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	st, ed := inputEditor(t, tt)
	check := func(what string, f func()) {
		t.Helper()
		if got := laidBy(tt, f); got > windowBound {
			t.Errorf("%s laid out %d runes, want at most %d", what, got, windowBound)
		}
	}
	check("End then Backspace", func() { tt.Key(0, KeyEnd); tt.Key(0, KeyBackspace) })
	if got := utf8.RuneCountInString(s); got != n-1 || ed.caret != n-1 {
		t.Fatalf("after Backspace at the end: %d runes, caret %d; want %d and %d", got, ed.caret, n-1, n-1)
	}
	check("Home then Delete", func() { tt.Key(0, KeyHome); tt.Key(0, KeyDelete) })
	if !strings.HasPrefix(s, "b ab") || utf8.RuneCountInString(s) != n-2 {
		t.Fatalf("after Delete at the start: %q…, %d runes", s[:8], utf8.RuneCountInString(s))
	}
	check("Right ×3", func() {
		for range 3 {
			tt.Key(0, KeyRight)
		}
	})
	if ed.caret != 3 {
		t.Errorf("caret %d after Home and three Rights, want 3", ed.caret)
	}
	check("Shift+End", func() { tt.Key(Shift, KeyEnd) })
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != s[3:] {
		t.Errorf("copied %d bytes, want the %d from the caret to the end", len(got), len(s)-3)
	}
	n -= 2
	check("End, Left ×3", func() {
		tt.Key(0, KeyEnd)
		for range 3 {
			tt.Key(0, KeyLeft)
		}
	})
	if ed.caret != n-3 {
		t.Errorf("caret %d after End and three Lefts, want %d", ed.caret, n-3)
	}
	check("End, Left ×2, Shift+Right ×2, Left", func() {
		tt.Key(0, KeyEnd)
		tt.Key(0, KeyLeft)
		tt.Key(0, KeyLeft)
		tt.Key(Shift, KeyRight)
		tt.Key(Shift, KeyRight)
		tt.Key(0, KeyLeft)
	})
	if ed.caret != n-2 || ed.anchor != n-2 {
		t.Errorf("Left collapsed a selection of the last two runes to %d-%d, want %d", ed.anchor, ed.caret, n-2)
	}
	// Selections from out of the window, before it and after it.
	tt.Key(0, KeyHome)
	tt.Key(Shift, KeyEnd)
	tt.Key(Shift, KeyLeft)
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != s[:len(s)-1] {
		t.Errorf("Home, Shift+End, Shift+Left copied %d bytes, want %d", len(got), len(s)-1)
	}
	tt.Key(0, KeyEnd)
	tt.Key(Shift, KeyHome)
	tt.Key(Shift, KeyRight)
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != s[1:] {
		t.Errorf("End, Shift+Home, Shift+Right copied %d bytes, want %d", len(got), len(s)-1)
	}
	if runtime.GOOS == "darwin" {
		// Cmd+Left and Cmd+Right go to the line's edges, not the window's.
		tt.Key(0, KeyEnd)
		tt.Key(Super, KeyLeft)
		if ed.caret != 0 {
			t.Errorf("Cmd+Left from the end put the caret at %d, want 0", ed.caret)
		}
		tt.Key(Super, KeyRight)
		if ed.caret != n {
			t.Errorf("Cmd+Right from the start put the caret at %d, want %d", ed.caret, n)
		}
	}
	check("End then a click", func() {
		tt.Key(0, KeyEnd)
		tt.Frame()
		tt.ClickAt(st.x+ed.originX+5, st.y+st.h/2)
	})
	if ed.caret < n-200 || ed.caret > n {
		t.Errorf("a click at the left of the end of the text put the caret at %d, want it in [%d, %d]", ed.caret, n-200, n)
	}
}

// The window moves as the caret comes near its edge, and the text keeps its
// place in the box: the caret steps a letter, not to the box's edge.
func TestTextInputWindowMovesInPlace(t *testing.T) {
	s := strings.Repeat("ab ", 128<<10/3)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	st, ed := inputEditor(t, tt)
	ed.caret, ed.anchor = 64<<10, 64<<10
	tt.Frame()
	ed.caret, ed.anchor = ed.winEnd-inputWindow/4, ed.winEnd-inputWindow/4
	tt.Frame()
	win, before := ed.win, ed.caretRect(st).X
	tt.Key(0, KeyRight)
	if ed.win == win {
		t.Fatalf("the window stayed at %d with the caret %d runes from its end", win, ed.winEnd-ed.caret)
	}
	if after := ed.caretRect(st).X; after-before < 0 || after-before > 20 {
		t.Errorf("a step right as the window moved took the caret from x %v to %v", before, after)
	}
}

// Without the focus, a long value's window is its start, which the input
// shows; a short value is laid out whole.
func TestTextInputWindowUnfocusedAndShort(t *testing.T) {
	s := strings.Repeat("ab ", 128<<10/3)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	_, ed := inputEditor(t, tt)
	if l := ed.layout.Params.Text; !strings.HasPrefix(l, "ab ab ab ") || utf8.RuneCountInString(l) > windowBound {
		t.Errorf("unfocused, the layout holds %d runes starting %q, want at most %d from the value's start", utf8.RuneCountInString(l), l[:min(len(l), 9)], windowBound)
	}
	short := strings.Repeat("abcd ", 20)
	tt2 := NewTester(func(c *Context) { TextInput(c, &short).Fill().AutoFocus() }, 300, 100)
	defer tt2.rt.close()
	tt2.Frame()
	_, ed2 := inputEditor(t, tt2)
	if ed2.layout.Params.Text != short {
		t.Errorf("a value of 100 runes laid out as %q, want it whole", ed2.layout.Params.Text)
	}
}

// An input method composes in the middle of a long value: the window holds
// the composition at the caret, the commit lands there, and the composition's
// bounds are in the input.
func TestTextInputWindowComposition(t *testing.T) {
	s := strings.Repeat("ab ", 128<<10/3)
	n := utf8.RuneCountInString(s)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	st, ed := inputEditor(t, tt)
	mid := n / 2 // between "ab " and "ab ": n/2 is a multiple of 3 plus 0..2
	mid -= mid % 3
	ed.caret, ed.anchor = mid, mid
	tt.Frame()
	c := tt.h.ime.Client
	if got := laidBy(tt, func() {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
	}); got > windowBound {
		t.Errorf("a composition laid out %d runes, want at most %d", got, windowBound)
	}
	if l := []rune(ed.layout.Params.Text); ed.caret-ed.win < 0 || ed.caret-ed.win >= len(l) || l[ed.caret-ed.win] != 'x' {
		t.Errorf("the layout of %d runes from %d does not hold the composition at the caret %d", len(l), ed.win, ed.caret)
	}
	b, _, visible := c.BoundsForRange(TextInputRange{Start: mid, End: mid + 1})
	if !visible || b.X < 0 || b.X > float64(st.w) {
		t.Errorf("the composition's bounds %+v are not in the input of width %v", b, st.w)
	}
	if at, ok := c.IndexForPoint(b.X+1, b.Y+b.H/2); !ok || at < mid-1 || at > mid+2 {
		t.Errorf("the point of the composition's bounds is at %d (%v), want about %d", at, ok, mid)
	}
	c.UnmarkText()
	tt.Frame()
	if got := utf8.RuneCountInString(s); got != n+1 || !strings.HasPrefix(s[mid:], "xab ") {
		t.Errorf("after the commit: %d runes, %q at the caret; want %d and \"xab \"", got, s[mid:mid+4], n+1)
	}
}

// A long right-to-left value shows its end without the focus: its window is
// there.
func TestTextInputWindowRTLUnfocused(t *testing.T) {
	s := strings.Repeat("שלום ", 13<<10)
	n := utf8.RuneCountInString(s)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Direction(RightToLeft).Fill() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	_, ed := inputEditor(t, tt)
	if ed.winEnd != n || utf8.RuneCountInString(ed.layout.Params.Text) > windowBound {
		t.Errorf("unfocused, right to left: window [%d, %d) of %d runes, want it at the end and at most %d", ed.win, ed.winEnd, n, windowBound)
	}
}

// The app sets a shorter value while the window is far in a long one: the
// window is of the new value.
func TestTextInputWindowAppShortensValue(t *testing.T) {
	s := strings.Repeat("ab ", 128<<10/3)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	_, ed := inputEditor(t, tt)
	ed.caret, ed.anchor = 100<<10, 100<<10
	tt.Frame()
	s = strings.Repeat("cd ", 24<<10/3)
	tt.Frame()
	tt.Frame()
	n := utf8.RuneCountInString(s)
	if l := ed.layout.Params.Text; ed.win >= ed.winEnd || ed.winEnd > n || !strings.HasPrefix(l, "cd ") && !strings.HasPrefix(l, "d ") && !strings.HasPrefix(l, " cd") {
		t.Errorf("window [%d, %d) of %d runes, layout starting %q", ed.win, ed.winEnd, n, l[:min(len(l), 4)])
	}
}

// An input editing a TextBuffer of 4M runes builds no string of it to lay
// out its window: a keystroke allocates about what it does in one of 32K
// runes, whose window is as long, not the 4 MB of a string more.
func TestTextInputBufferWindowBuildsNoString(t *testing.T) {
	keystroke := func(runes int) (laid, bytes uint64) {
		b := NewTextBuffer(strings.Repeat("ab ", runes/3))
		tt := NewTester(func(c *Context) { TextInputBuffer(c, b).Fill().AutoFocus() }, 300, 100)
		defer tt.rt.close()
		tt.Frame()
		tt.Key(0, KeyEnd)
		tt.Frame()
		runtime.GC()
		bytes = allocated(func() { laid = laidBy(tt, func() { tt.Type("x") }) })
		return laid, bytes
	}
	_, small := keystroke(32 << 10)
	laid, large := keystroke(4 << 20)
	if laid > windowBound {
		t.Errorf("a keystroke laid out %d runes, want at most %d", laid, windowBound)
	}
	if large > small+1<<20 {
		t.Errorf("a keystroke allocated %d bytes in a buffer of 4M runes, %d in one of 32K: want at most 1 MB more", large, small)
	}
}

// A value the app set over the cap of a paste takes no paste, which is cut
// to nothing, but takes a rune typed.
func TestTextInputLongAppValuePasteCut(t *testing.T) {
	const n = 1<<19 + 1024
	s := strings.Repeat("a", n)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.Key(0, KeyEnd)
	tt.SetClipboard(strings.Repeat("b", 300))
	tt.Key(Cmd, KeyV)
	if len(s) != n || strings.Contains(s, "b") {
		t.Errorf("a paste of 300 runes into %d gave %d runes, want it cut to nothing", n, len(s))
	}
	tt.Type("x")
	if len(s) != n+1 || !strings.HasSuffix(s, "ax") {
		t.Errorf("a rune typed into %d runes gave %d runes ending %q", n, len(s), s[len(s)-2:])
	}
}

// A value the app set over what typing used to be cut at takes what is typed.
func TestTextInputLongAppValueTakesTyping(t *testing.T) {
	const n = 1<<19 + 1024
	s := strings.Repeat("a", n)
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.Key(0, KeyEnd)
	tt.Type("x")
	if len(s) != n+1 || !strings.HasSuffix(s, "ax") {
		t.Errorf("typing into %d runes gave %d runes ending %q", n, len(s), s[len(s)-2:])
	}
}
