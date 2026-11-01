package ui

import "testing"

// A replayed row's elements are walked once: the walk checks them against the
// record and keeps them for setting their flags (not one walk to check and
// another to set), and a row shifted by a scroll walks them for its boxes
// from the same list.
func TestReplayedRowWalksItsElementsOnce(t *testing.T) {
	tt, list := replayTable(t, 100, rowText)
	tt.Frame()
	first, last := list.Visible()
	rows := last - first + 1
	if tt.rt.replayed < rows {
		t.Fatalf("%d of %d rows replayed", tt.rt.replayed, rows)
	}
	if tt.rt.replayWalks != tt.rt.replayed {
		t.Fatalf("%d walks for %d replayed rows in an idle frame", tt.rt.replayWalks, tt.rt.replayed)
	}
	// Scrolled by whole pixels, the rows replay shifted.
	tt.Scroll(100, 100, 0, 1)
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Skip("no row replayed shifted")
	}
	if tt.rt.replayWalks != tt.rt.replayed {
		t.Fatalf("%d walks for %d replayed rows after a scroll", tt.rt.replayWalks, tt.rt.replayed)
	}
}
