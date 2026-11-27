package main

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// dragText drags the text from onto the text to, letting go just above its
// bottom edge.
func dragText(t *testing.T, tt *ui.Tester, from, to string) {
	t.Helper()
	a, ok := tt.Find(from)
	b, ok2 := tt.Find(to)
	if !ok || !ok2 {
		t.Fatalf("no %q or %q in %q", from, to, tt.Texts())
	}
	ax, ay := a.X+a.W/2, a.Y+a.H/2
	bx, by := b.X+b.W/2, b.Y+b.H-1
	tt.Press(ax, ay)
	for i := 1; i <= 4; i++ {
		f := float32(i) / 4
		tt.Move(ax+(bx-ax)*f, ay+(by-ay)*f)
		tt.Frame()
	}
	tt.Release(bx, by)
	settle(tt)
	settle(tt)
}

// The table's rows reorder with Reorder rows on, and drag out onto the bin
// with Drag rows out on.
func TestTourTableDragsRows(t *testing.T) {
	tt := tourTableTab(t)
	if err := tt.Click("Reorder rows"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	dragText(t, tt, "acct000000", "acct000002")
	texts := tt.Texts()
	at := func(s string) int { return slices.Index(texts, s) }
	if a, b, c := at("acct000001"), at("acct000002"), at("acct000000"); !(a >= 0 && a < b && b < c) {
		t.Fatalf("want acct000001, acct000002, acct000000 in that order: %q", texts)
	}
	if !tt.HasText("Rows moved") {
		t.Fatalf("OnRowsReordered not shown; texts %q", tt.Texts())
	}
	if err := tt.Click("Drag rows out"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	dragText(t, tt, "acct000001", "Drop rows here")
	if !tt.HasText("Dropped 1 rows") {
		t.Fatalf("the bin got no rows; texts %q", tt.Texts())
	}
}
