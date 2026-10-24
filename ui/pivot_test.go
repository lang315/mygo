package ui

import (
	"fmt"
	"slices"
	"testing"
)

func chosenKeys(sel *Selection[int], n int) []int {
	var got []int
	for i := range n {
		if sel.Has(i) {
			got = append(got, i)
		}
	}
	return got
}

// A grid whose user has chosen nothing has no item to extend from: a
// Shift-click as the first click chooses that item alone, not the items from
// the first.
func TestGridViewShiftClickWithoutAPivotChoosesOnlyTheItem(t *testing.T) {
	sel, cur := Selection[int]{}, -1
	s := GridState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, &s, 6, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	if s.Pivot() != -1 {
		t.Fatalf("a new grid has pivot %d, want -1", s.Pivot())
	}
	if err := tt.ClickWith(Shift, "Item 3"); err != nil {
		t.Fatal(err)
	}
	if got := chosenKeys(&sel, 6); !slices.Equal(got, []int{3}) {
		t.Fatalf("Shift-click first chose %v, want only 3", got)
	}
}

// An app that shows the same items in another view tells the grid where
// Shift extends from, without changing the choice.
func TestGridStateSetPivotIsWhereShiftExtendsFrom(t *testing.T) {
	sel, cur := Selection[int]{}, 3
	sel.Add(3)
	s := GridState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, &s, 6, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	s.SetPivot(1)
	if s.Pivot() != 1 || !slices.Equal(chosenKeys(&sel, 6), []int{3}) {
		t.Fatalf("SetPivot(1): Pivot %d, chosen %v: the choice must stay", s.Pivot(), chosenKeys(&sel, 6))
	}
	if err := tt.ClickWith(Shift, "Item 5"); err != nil {
		t.Fatal(err)
	}
	if got := chosenKeys(&sel, 6); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("Shift-click from the pivot 1 to 5 chose %v", got)
	}
	if s.Pivot() != 1 {
		t.Fatalf("Pivot %d after Shift-click, want it kept at 1", s.Pivot())
	}
}

func TestListStateSetPivotIsWhereShiftExtendsFrom(t *testing.T) {
	sel, cur := Selection[int]{}, 3
	sel.Add(3)
	s := ListState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 6, func(i int) { coreText(c, fmt.Sprint("Row ", i)) }).Grow(1)
	}, 300, 300)
	if s.Pivot() != -1 {
		t.Fatalf("a new list has pivot %d, want -1", s.Pivot())
	}
	s.SetPivot(1)
	if s.Pivot() != 1 || !slices.Equal(chosenKeys(&sel, 6), []int{3}) {
		t.Fatalf("SetPivot(1): Pivot %d, chosen %v", s.Pivot(), chosenKeys(&sel, 6))
	}
	if err := tt.ClickWith(Shift, "Row 5"); err != nil {
		t.Fatal(err)
	}
	if got := chosenKeys(&sel, 6); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("Shift-click from the pivot 1 to 5 chose %v", got)
	}
}

// With AccentChoice a grid shows the items chosen in the accent color, as a
// table's rows are, focused or not; without it, as a tint or a grey.
func TestGridViewAccentChoiceShowsChosenItemsInTheAccent(t *testing.T) {
	for _, accent := range []bool{false, true} {
		sel, cur := Selection[int]{}, 1
		sel.Add(1)
		s := GridState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }, AccentChoice: accent}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				coreGridView(c, &s, 3, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
			})
		}, 448, 200)
		tt.SetDark(false)
		tt.Frame()
		r, _ := tt.Find("Item 1")
		px := tt.Image().RGBAAt(int(r.X)+3, int(r.Y)+3)
		want := LightTheme().Accent
		near := func(a, b uint8) bool { d := int(a) - int(b); return d > -9 && d < 9 }
		isAccent := near(px.R, want.R) && near(px.G, want.G) && near(px.B, want.B)
		if isAccent != accent {
			t.Errorf("AccentChoice %v: the chosen item's corner is %v (accent %v)", accent, px, want)
		}
	}
}
