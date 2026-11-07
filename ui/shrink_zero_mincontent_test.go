package ui

import "testing"

// A row that grows keeps at least the width of its children that cannot
// shrink, when they are wider than their own min-content (a Button's is
// only its padding). Otherwise they spill over
// the sibling after the row.
func TestGrowingRowHoldsItsUnshrinkableChildren(t *testing.T) {
	var group, last, title *node
	tt := coreNewTester(func(c *context) {
		coreRow(c).Fill().Gap(8).Children(func() {
			group = coreRow(c).Grow(1).Gap(8)
			group.Children(func() {
				coreButton(c, "Alpha")
				coreButton(c, "Bravo")
				last = coreButton(c, "Charlie")
			})
			title = coreText(c, "title").Width(300).Shrink(0)
		})
	}, 500, 100)
	tt.Frame()
	g, l, ti := group.Bounds(), last.Bounds(), title.Bounds()
	if l.X+l.W > g.X+g.W+0.5 {
		t.Errorf("last child %v spills out of its row %v", l, g)
	}
	if g.X+g.W > ti.X+0.5 {
		t.Errorf("row %v runs under the sibling %v", g, ti)
	}
}
