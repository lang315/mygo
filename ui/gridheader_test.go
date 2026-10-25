package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// headerGrid shows a grid of n items in four columns, those in heads being
// headers ("H5") and the others items ("Item 7"), choosing several.
func headerGrid(n int, heads []int) (*Tester, *GridState, *Selection[int]) {
	sel, chosen := -1, &Selection[int]{}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { return slices.Contains(heads, i) }
	s.Label = func(i int) string {
		if s.Header(i) {
			return fmt.Sprintf("H%d", i)
		}
		return fmt.Sprintf("Item %d", i)
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, n, 100, 40, func(i int) {
				coreText(c, s.Label(i))
			}).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	return tt, s, chosen
}

func rowY(t *testing.T, tt *Tester, text string) float32 {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q on the screen", text)
	}
	return r.Y
}

// A header takes a row of its own, at the width of the grid, and the items
// after it start a row: the last row of a group may be short.
func TestGridViewHeaderRows(t *testing.T) {
	tt, s, _ := headerGrid(18, []int{0, 5, 6, 12})
	if s.cols != 4 {
		t.Fatalf("the test grid has %d columns, want 4", s.cols)
	}
	y := func(text string) float32 { return rowY(t, tt, text) }
	same := func(a, b string) {
		t.Helper()
		if y(a) != y(b) {
			t.Errorf("%s (y %v) and %s (y %v) are not in one row", a, y(a), b, y(b))
		}
	}
	below := func(a, b string) {
		t.Helper()
		if y(a) >= y(b) {
			t.Errorf("%s (y %v) is not above %s (y %v)", a, y(a), b, y(b))
		}
	}
	same("Item 1", "Item 4")
	same("Item 7", "Item 10")
	same("Item 13", "Item 16")
	below("H0", "Item 1")
	below("Item 4", "H5")
	below("H5", "H6")
	below("H6", "Item 7")
	below("Item 10", "Item 11")
	below("Item 11", "H12")
	below("H12", "Item 13")
	below("Item 16", "Item 17")
	// The header spans the grid, which the items of a row share.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	h := accessNode(t, tt.h.access, platform.RoleListItem, "H5")
	item := accessNode(t, tt.h.access, platform.RoleListItem, "Item 1")
	if h.Bounds.W < 3*item.Bounds.W {
		t.Errorf("the header is %v wide, an item %v: it does not span the grid", h.Bounds.W, item.Bounds.W)
	}
}

// Assistive technology sees the headers as items of the grid, as it does a
// list's: named, and counted in the position and the size of the set.
func TestGridViewHeaderAccessibility(t *testing.T) {
	tt, _, _ := headerGrid(18, []int{0, 5, 6, 12})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	h := accessNode(t, tt.h.access, platform.RoleListItem, "H6")
	if h.PosInSet != 7 || h.SetSize != 18 {
		t.Errorf("the header H6 is %d of %d, want 7 of 18", h.PosInSet, h.SetSize)
	}
	item := accessNode(t, tt.h.access, platform.RoleListItem, "Item 7")
	if item.PosInSet != 8 || item.SetSize != 18 {
		t.Errorf("Item 7 is %d of %d, want 8 of 18", item.PosInSet, item.SetSize)
	}
	list := accessNode(t, tt.h.access, platform.RoleList, "")
	if list.SetSize != 18 {
		t.Errorf("the list has %d items, want 18", list.SetSize)
	}
	if h.States&platform.AccessChecked != 0 || h.States&platform.AccessSelectable != 0 {
		t.Errorf("the header %+v is choosable", h)
	}
}

// The arrows pass the headers by, and the up and down arrows keep the
// column in a group and go on to the next, in the last item of a short row.
func TestGridViewHeaderKeys(t *testing.T) {
	tt, s, chosen := headerGrid(18, []int{0, 5, 6, 12})
	at := func(want int, what string) {
		t.Helper()
		if got := *s.Selected; got != want {
			t.Fatalf("%s: item %d, want %d", what, got, want)
		}
		if got := chosenRows(chosen); !slices.Equal(got, []int{want}) {
			t.Fatalf("%s: chose %v, want only %d", what, got, want)
		}
	}
	tt.Click("Item 4")
	at(4, "click")
	tt.Key(0, KeyRight)
	at(7, "Right past the headers 5 and 6")
	tt.Key(0, KeyLeft)
	at(4, "Left past them")
	tt.Key(0, KeyDown)
	at(10, "Down from Item 4 (the last column) to the next group's row")
	tt.Key(0, KeyDown)
	at(11, "Down to a short row, the last item of it")
	tt.Key(0, KeyDown)
	at(13, "Down past the header 12")
	tt.Key(0, KeyUp)
	at(11, "Up past the header 12 to the short row")
	tt.Key(0, KeyUp)
	at(7, "Up to the first row of the group, its first column")
	tt.Key(0, KeyUp)
	at(1, "Up past the headers 5 and 6")
	tt.Key(0, KeyUp)
	at(1, "Up from the first row stays")
	tt.Key(0, KeyEnd)
	at(17, "End")
	tt.Key(0, KeyHome)
	at(1, "Home goes to the first item, not the header 0")
}

// Headers are not chosen: Shift extends over them, Cmd+A leaves them out,
// and a click on one does nothing.
func TestGridViewHeadersAreNotChosen(t *testing.T) {
	tt, s, chosen := headerGrid(18, []int{0, 5, 6, 12})
	tt.Click("Item 2")
	tt.ClickWith(Shift, "Item 8")
	if got := chosenRows(chosen); !slices.Equal(got, []int{2, 3, 4, 7, 8}) {
		t.Fatalf("Shift-click from Item 2 to Item 8 chose %v, want the items between, not the headers 5 and 6", got)
	}
	tt.Key(Shift, KeyEnd)
	if got := chosenRows(chosen); !slices.Equal(got, []int{2, 3, 4, 7, 8, 9, 10, 11, 13, 14, 15, 16, 17}) {
		t.Fatalf("Shift+End chose %v", got)
	}
	tt.Key(Cmd, KeyA)
	if chosen.Len() != 14 || chosen.Has(0) || chosen.Has(5) || chosen.Has(6) || chosen.Has(12) {
		t.Fatalf("Cmd+A chose %d items %v, want the 14 items, no header", chosen.Len(), chosenRows(chosen))
	}
	tt.Click("Item 2")
	before := *s.Selected
	tt.Click("H5")
	if *s.Selected != before || !slices.Equal(chosenRows(chosen), []int{2}) {
		t.Fatalf("a click on a header changed the choice: item %d, chose %v", *s.Selected, chosenRows(chosen))
	}
	// Cmd+Shift extends and keeps the others, over the headers too: Up from
	// Item 10 (the last column of its row) goes to Item 4, the rows between
	// being 7 to 9 and the headers.
	tt.ClickWith(Cmd, "Item 10")
	tt.Key(Cmd|Shift, KeyUp)
	if got := chosenRows(chosen); !slices.Equal(got, []int{2, 4, 7, 8, 9, 10}) {
		t.Fatalf("Cmd+Shift+Up from Item 10 chose %v, want [2 4 7 8 9 10]", got)
	}
}

// With nothing chosen, the arrows start at the first item, not a header.
func TestGridViewHeaderKeysWithoutAChoice(t *testing.T) {
	tt, s, chosen := headerGrid(18, []int{0, 5, 6, 12})
	tt.Key(0, KeyTab) // focuses the grid
	tt.Key(0, KeyDown)
	if *s.Selected != 1 || !slices.Equal(chosenRows(chosen), []int{1}) {
		t.Fatalf("Down with nothing chosen: item %d, chose %v, want Item 1", *s.Selected, chosenRows(chosen))
	}
}

// A grid scrolls to the item the keys reach, whatever the headers before it
// add to the rows.
func TestGridViewHeadersScrollToTheItem(t *testing.T) {
	var heads []int
	for h := 0; h < 400; h += 10 {
		heads = append(heads, h)
	}
	tt, s, _ := headerGrid(400, heads)
	tt.Click("Item 1")
	tt.Key(0, KeyEnd)
	if *s.Selected != 399 {
		t.Fatalf("End chose %d, want 399", *s.Selected)
	}
	if _, ok := tt.Find("Item 399"); !ok {
		t.Fatal("End did not scroll to Item 399")
	}
	tt.Key(0, KeyHome)
	if _, ok := tt.Find("Item 1"); !ok {
		t.Fatal("Home did not scroll back to Item 1")
	}
	s.ScrollTo(251, Center)
	tt.Frame()
	if _, ok := tt.Find("Item 251"); !ok {
		t.Fatal("ScrollTo(251) did not show Item 251")
	}
}

// The grid finds out that the headers moved when the number of items stays
// and a header it knew is none; ReloadHeaders is for a new header among
// the items that were not.
func TestGridViewHeadersFollowTheData(t *testing.T) {
	heads := []int{0, 5}
	sel, chosen := -1, &Selection[int]{}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { return slices.Contains(heads, i) }
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, 12, 100, 40, func(i int) {
				if s.Header(i) {
					coreTextf(c, "H%d", i)
				} else {
					coreTextf(c, "Item %d", i)
				}
			}).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	if rowY(t, tt, "H5") <= rowY(t, tt, "Item 4") {
		t.Fatal("setup: H5 is not below Item 4")
	}
	heads = []int{0, 7} // the header moved: the old position is no header
	tt.Frame()
	if rowY(t, tt, "H7") <= rowY(t, tt, "Item 4") {
		t.Errorf("after the header moved to 7, H7 is not below Item 4")
	}
	if rowY(t, tt, "Item 1") != rowY(t, tt, "Item 4") || rowY(t, tt, "Item 5") != rowY(t, tt, "Item 6") {
		t.Errorf("after the header moved to 7, Items 1 to 6 are not laid out in rows of 4 and 2")
	}
	heads = []int{0, 7, 9} // an item became a header, the number of items stays
	s.ReloadHeaders()
	tt.Frame()
	if rowY(t, tt, "H9") <= rowY(t, tt, "Item 8") {
		t.Errorf("after ReloadHeaders, H9 is not below Item 8")
	}
}

// Headers are not dragged, and the items go before the item or the header
// they are dropped on.
func TestGridViewHeaderReorder(t *testing.T) {
	var moves [][]int
	sel := -1
	s := &GridState{Selected: &sel, Key: func(i int) any { return i }, Reorder: func(items []int, to int) {
		moves = append(moves, append(slices.Clone(items), to))
	}}
	s.Header = func(i int) bool { return i == 0 || i == 5 }
	tt := coreNewTester(func(c *context) {
		coreGridView(c, s, 10, 100, 60, func(i int) {
			if s.Header(i) {
				coreTextf(c, "H%d", i)
			} else {
				coreTextf(c, "Item %d", i)
			}
		}).Grow(1)
	}, 448, 400)
	tt.Frame()
	from, _ := tt.Find("Item 2")
	to, _ := tt.Find("Item 7")
	drag(tt, from.X, from.Y, to.X-20, to.Y)
	if len(moves) != 1 || !slices.Equal(moves[0], []int{2, 7}) {
		t.Fatalf("moves %v, want Item 2 before Item 7", moves)
	}
	h, _ := tt.Find("H5")
	drag(tt, h.X, h.Y, to.X-20, to.Y)
	if len(moves) != 1 {
		t.Errorf("a header was dragged: moves %v", moves)
	}
	// Dropped on a header, which takes the whole row: before it on its upper
	// half, after it (first in its group) on the lower, not at the end of the
	// grid, wherever along the row the pointer is.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	hn := accessNode(t, tt.h.access, platform.RoleListItem, "H5")
	upper, lower := float32(hn.Bounds.Y+2), float32(hn.Bounds.Y+hn.Bounds.H-2)
	for _, x := range []float32{float32(hn.Bounds.X) + 10, 400} {
		moves = moves[:1]
		drag(tt, from.X, from.Y, x, upper)
		if len(moves) != 2 || !slices.Equal(moves[1], []int{2, 5}) {
			t.Errorf("Item 2 dropped on the upper half of H5 at x %v: moves %v, want before it", x, moves)
		}
		drag(tt, from.X, from.Y, x, lower)
		if len(moves) != 3 || !slices.Equal(moves[2], []int{2, 6}) {
			t.Errorf("Item 2 dropped on the lower half of H5 at x %v: moves %v, want after it", x, moves)
		}
	}
}

// Without Header the grid is laid out in rows of its columns, as it was.
func TestGridViewWithoutHeaderKeepsItsRows(t *testing.T) {
	sel, chosen := -1, &Selection[int]{}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, 10, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	if rowY(t, tt, "Item 0") != rowY(t, tt, "Item 3") || rowY(t, tt, "Item 3") >= rowY(t, tt, "Item 4") || rowY(t, tt, "Item 8") != rowY(t, tt, "Item 9") {
		t.Fatal("a grid without Header is not in rows of 4")
	}
}
