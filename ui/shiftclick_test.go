package ui

import (
	"slices"
	"testing"
)

// A Shift-click replaces the choice with the rows from the pivot to the row
// clicked, and a Cmd+Shift-click adds them, on every platform: macOS does not
// keep the rows chosen before, as Finder does.
func TestTableShiftClickReplacesAndCmdShiftAdds(t *testing.T) {
	names := []string{"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7"}
	cols := []TableColumn{{Title: "Name"}}
	var chosen Selection[int]
	sel := -1
	s := ListState{Selected: &sel, Selection: &chosen, Key: func(i int) any { return i + 100 }}
	tt := coreNewTester(tableView(&s, cols, names), 500, 400)
	got := func() []int { return slices.Sorted(chosen.All()) }
	tt.Click("r1")
	tt.ClickWith(Cmd, "r3")
	tt.ClickWith(Shift, "r6")
	if want := []int{103, 104, 105, 106}; !slices.Equal(got(), want) {
		t.Fatalf("Shift-click from r3 to r6 after a Cmd-click chose %v, want %v", got(), want)
	}
	tt.ClickWith(Shift, "r4")
	if want := []int{103, 104}; !slices.Equal(got(), want) {
		t.Fatalf("Shift-click back to r4 chose %v, want %v", got(), want)
	}
	tt.ClickWith(Cmd|Shift, "r6")
	if want := []int{103, 104, 105, 106}; !slices.Equal(got(), want) {
		t.Fatalf("Cmd+Shift-click to r6 chose %v, want %v: it adds", got(), want)
	}
	tt.Click("r0")
	tt.ClickWith(Cmd, "r2")
	tt.ClickWith(Cmd|Shift, "r5")
	if want := []int{100, 102, 103, 104, 105}; !slices.Equal(got(), want) {
		t.Fatalf("Cmd+Shift-click kept the rows chosen before: chose %v, want %v", got(), want)
	}
}

// A grid's Shift-click replaces the choice, and a Cmd+Shift-click adds the
// items from the pivot to the one clicked to it.
func TestGridViewCmdShiftClickAddsTheRange(t *testing.T) {
	sel, chosen := -1, Selection[int]{}
	s := GridState{Selected: &sel, Selection: &chosen, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, &s, 8, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	got := func() []int { return chosenKeys(&chosen, 8) }
	tt.Click("Item 0")
	tt.ClickWith(Cmd, "Item 2")
	tt.ClickWith(Cmd|Shift, "Item 5")
	if want := []int{0, 2, 3, 4, 5}; !slices.Equal(got(), want) {
		t.Fatalf("Cmd+Shift-click from item 2 to 5 chose %v, want %v: it adds", got(), want)
	}
	if sel != 5 || s.Pivot() != 2 {
		t.Fatalf("Selected %d and Pivot %d, want 5 and 2", sel, s.Pivot())
	}
	tt.ClickWith(Shift, "Item 7")
	if want := []int{2, 3, 4, 5, 6, 7}; !slices.Equal(got(), want) {
		t.Fatalf("Shift-click to item 7 chose %v, want %v", got(), want)
	}
}
