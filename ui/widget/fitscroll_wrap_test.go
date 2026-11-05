package widget

import (
	"testing"
)

const longLine = "The quick brown fox jumps over the lazy dog while the careful reader checks that every word of this sentence stays in view."

// tallDialog is a dialog of width 500 and height 0 whose content, with a
// long Label in a group box, is higher than the work area: it fit-scrolls.
func tallDialog(t *testing.T, label string, rows int) *DialogTester {
	t.Helper()
	g := GroupBox("g").Add(Label(label))
	content := VBox(4).Add(g)
	for i := range rows {
		content.Add(Label("row " + string(rune('a'+i%26))))
	}
	d := NewDialog(nil, "t", "", 500, 300, content)
	dt := TestDialog(t, d)
	d.SetSize(500, 0)
	dt.Settle()
	return dt
}

// Under fit scrolling the width stays the dialog's: a long Label wraps at 500
// and nothing runs past the right edge.
func TestFitScrollKeepsTheDialogsWidthAndWrapsLabels(t *testing.T) {
	dt := tallDialog(t, longLine, 150)
	mu.Lock()
	scrolls := dt.d.fitScroll
	mu.Unlock()
	if !scrolls {
		t.Fatal("the content did not fit-scroll: make it taller")
	}
	r, ok := dt.Tester.Tester.Find(longLine)
	if !ok {
		t.Fatalf("no label in %q", dt.Texts())
	}
	one, ok := dt.Tester.Tester.Find("row a")
	if !ok {
		t.Fatal("no row")
	}
	if r.X < 0 || r.X+r.W > 500 {
		t.Errorf("label at %v overflows the dialog's 500 DIPs", r)
	}
	if r.H < 2*one.H {
		t.Errorf("label %v is one line (%v): it does not wrap", r, one)
	}
}

// The measure of a form does not depend on whether it already fit-scrolls: a
// remeasure (SetSize(w, 0) on a shown dialog) lays the content out at the
// dialog's width in both cases, so wrapped labels count and the fit decision
// agrees with the content it scrolls.
func TestMeasureFitIsTheSameWhetherOrNotItFitScrolls(t *testing.T) {
	f := NewForm("Measure scroll", 500, 300)
	f.SetContent(VBox(4).Add(GroupBox("g").Add(Label(longLine)), Label("row")))
	h := func(scroll bool) int {
		mu.Lock()
		f.fitScroll = scroll
		mu.Unlock()
		got, _ := f.measureFit(500, nil)
		return got
	}
	if plain, scrolled := h(false), h(true); plain != scrolled {
		t.Errorf("measured %d, and %d once it fit-scrolls", plain, scrolled)
	}
}

// Content that fit-scrolled and then gets short fits again, and stops scrolling.
func TestRefitAfterFitScrollFitsShortContent(t *testing.T) {
	d := NewDialog(nil, "Refit short", "", 500, 300, VBox(4).Add(Label("x")))
	dt := TestDialog(t, d)
	tall := VBox(4)
	for range 150 {
		tall.Add(Label("row"))
	}
	d.SetContent(tall)
	d.SetSize(500, 0)
	dt.Settle()
	mu.Lock()
	scrolls := d.fitScroll
	mu.Unlock()
	if !scrolls {
		t.Fatal("did not fit-scroll")
	}
	short := VBox(4).Add(GroupBox("g").Add(Label(longLine)))
	d.SetContent(short)
	d.SetSize(500, 0)
	dt.Settle()
	want := fitOfForm(500, VBox(4).Add(GroupBox("g").Add(Label(longLine))))
	mu.Lock()
	scrolls = d.fitScroll
	mu.Unlock()
	if dt.h != want || scrolls {
		t.Errorf("viewport height %d (scrolls %v), want %d without scrolling", dt.h, scrolls, want)
	}
}
