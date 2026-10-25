package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The grid lays its rows out again when the number of items changes: a header
// added at the end, and one moved by an insert before it.
func TestGridViewHeadersFollowTheNumberOfItems(t *testing.T) {
	n, heads := 12, []int{0, 5}
	sel, chosen := -1, &Selection[int]{}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { return slices.Contains(heads, i) }
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, n, 100, 40, func(i int) {
				if s.Header(i) {
					coreTextf(c, "H%d", i)
				} else {
					coreTextf(c, "Item %d", i)
				}
			}).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	n, heads = 14, []int{0, 5, 12} // two items added, one of them a header
	tt.Frame()
	if rowY(t, tt, "H12") <= rowY(t, tt, "Item 11") || rowY(t, tt, "Item 13") <= rowY(t, tt, "H12") {
		t.Errorf("after two items were added, H12 is not between Item 11 and Item 13")
	}
	n, heads = 15, []int{0, 1, 6} // an item inserted at the start: everything moves on
	tt.Frame()
	if rowY(t, tt, "H1") <= rowY(t, tt, "H0") || rowY(t, tt, "Item 2") <= rowY(t, tt, "H1") || rowY(t, tt, "H6") <= rowY(t, tt, "Item 2") {
		t.Errorf("after the insert, H0, H1, Item 2 and H6 are not in rows in order")
	}
}

// An item in the short row at the end of a group is as wide as the others.
func TestGridViewHeaderShortRowsKeepTheItemsWidth(t *testing.T) {
	tt, _, _ := headerGrid(18, []int{0, 5, 6, 12})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	full := accessNode(t, tt.h.access, platform.RoleListItem, "Item 1")
	for _, name := range []string{"Item 11", "Item 17"} {
		short := accessNode(t, tt.h.access, platform.RoleListItem, name)
		// Within a few DIPs, as the last row of a grid without headers is:
		// an item that was not given its empty places would fill the row.
		if short.Bounds.W < full.Bounds.W || short.Bounds.W > full.Bounds.W+4 {
			t.Errorf("%s is %v wide, Item 1 %v: a short row's items are not as wide as the others", name, short.Bounds.W, full.Bounds.W)
		}
	}
}
