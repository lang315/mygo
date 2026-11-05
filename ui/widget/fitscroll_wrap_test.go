package widget

import (
	"testing"
)

const longLine = "The quick brown fox jumps over the lazy dog while the careful reader checks that every word of this sentence stays in view."

// Under fit scrolling the width stays the dialog's: a long Label wraps at it.
// (Bounds are clipped to the viewport, so its width says nothing: its lines do.)
func TestFitScrollKeepsTheDialogsWidthAndWrapsLabels(t *testing.T) {
	// Higher than the work area: it fit-scrolls.
	content := VBox(4).Add(GroupBox("g").Add(Label(longLine)))
	for range 150 {
		content.Add(Label("row"))
	}
	d := NewDialog(nil, "t", "", 500, 300, content)
	dt := TestDialog(t, d)
	d.SetSize(500, 0)
	dt.Settle()
	mu.Lock()
	scrolls := d.fitScroll
	mu.Unlock()
	if !scrolls {
		t.Fatal("the content did not fit-scroll: make it taller")
	}
	r, ok := dt.Tester.Tester.Find(longLine)
	if !ok {
		t.Fatalf("no label in %q", dt.Texts())
	}
	one, ok := dt.Tester.Tester.Find("row")
	if !ok {
		t.Fatal("no row")
	}
	if r.H < 2*one.H {
		t.Errorf("label %v is one line (%v): it does not wrap", r, one)
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
