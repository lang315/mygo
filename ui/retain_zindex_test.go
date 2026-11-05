package ui

import "testing"

// The order a replayed row's children paint in is part of what a replay keeps:
// a child whose ZIndex changes paints anew (a Table with a constant
// RowVersion keeps its rows; replayCheck paints each replayed row as well and
// counts the pixels that differ).
func TestRetainReplayFollowsAChildsZIndex(t *testing.T) {
	front := 0
	var list ListState
	tt := NewTester(func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}}, 20, func(r, col int) {
			Row(c).Height(50).Children(func() {
				Box(c).Size(80, 40).Background(RGB(255, 0, 0)).ZIndex(front)
				Box(c).Size(80, 40).Margin(0, 0, 0, -40).Background(RGB(0, 0, 255)).ZIndex(1 - front)
			})
		}).Grow(1)
	}, 300, 400)
	for range 4 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	fresh := tt.rt.freshRows
	front = 1
	tt.Frame()
	if tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("zindex: %d rows painted anew, %d mismatches", tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}
