package ui

import (
	"slices"
	"testing"
)

// Going back with Cmd+Shift takes out only the rows the extension chose: rows
// chosen before it began stay, even those in the way back.
func TestListCmdShiftKeysBackKeepsRowsChosenBefore(t *testing.T) {
	var sel Selection[int]
	s := ListState{Selection: &sel}
	changes := 0
	tt := coreNewTester(severalView(&s, &changes), 300, 400)
	tt.Click("Row 6")
	tt.ClickWith(Cmd, "Row 3")
	for range 4 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	expectChosen(t, &sel, 3, 4, 5, 6, 7)
	for range 4 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	expectChosen(t, &sel, 3, 6)
	// All chosen, then one step on and back: all stay.
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd|Shift, KeyDown)
	tt.Key(Cmd|Shift, KeyUp)
	if sel.Len() != 20 {
		t.Fatalf("Cmd+A, Cmd+Shift+Down and back left %v chosen, want all 20", chosenRows(&sel))
	}
}

// What an extension chose is forgotten when the pivot moves: rows an earlier
// one chose are chosen before the next, and going back leaves them.
func TestListCmdShiftKeysForgetWhatAnEarlierPivotChose(t *testing.T) {
	var sel Selection[int]
	s := ListState{Selection: &sel}
	changes := 0
	tt := coreNewTester(severalView(&s, &changes), 300, 400)
	tt.Click("Row 1")
	for range 3 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	expectChosen(t, &sel, 1, 2, 3, 4)
	tt.ClickWith(Cmd, "Row 8")
	for range 6 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	expectChosen(t, &sel, 1, 2, 3, 4, 5, 6, 7, 8)
	for range 6 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	expectChosen(t, &sel, 1, 2, 3, 4, 8)
}

func TestGridViewCmdShiftKeysForgetWhatAnEarlierPivotChose(t *testing.T) {
	tt, _, chosen := shiftKeysGrid(30)
	tt.Click("Item 1")
	for range 3 {
		tt.Key(Cmd|Shift, KeyRight)
	}
	tt.ClickWith(Cmd, "Item 8")
	for range 6 {
		tt.Key(Cmd|Shift, KeyLeft)
	}
	if got := chosenRows(chosen); !slices.Equal(got, seq(1, 8)) {
		t.Fatalf("Cmd+Shift+Left six times from Item 8 chose %v, want 1 to 8", got)
	}
	for range 6 {
		tt.Key(Cmd|Shift, KeyRight)
	}
	if got := chosenRows(chosen); !slices.Equal(got, []int{1, 2, 3, 4, 8}) {
		t.Fatalf("back to Item 8 left %v chosen, want [1 2 3 4 8]", got)
	}
}

// The same for a grid.
func TestGridViewCmdShiftKeysBackKeepsItemsChosenBefore(t *testing.T) {
	tt, _, chosen := shiftKeysGrid(30)
	tt.Click("Item 12")
	tt.ClickWith(Cmd, "Item 9")
	for range 4 {
		tt.Key(Cmd|Shift, KeyRight)
	}
	if got := chosenRows(chosen); !slices.Equal(got, []int{9, 10, 11, 12, 13}) {
		t.Fatalf("Cmd+Shift+Right four times from Item 9 chose %v", got)
	}
	for range 4 {
		tt.Key(Cmd|Shift, KeyLeft)
	}
	if got := chosenRows(chosen); !slices.Equal(got, []int{9, 12}) {
		t.Fatalf("back to Item 9 left %v chosen, want [9 12]: Item 12 was chosen before", got)
	}
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd|Shift, KeyRight)
	tt.Key(Cmd|Shift, KeyLeft)
	if chosen.Len() != 30 {
		t.Fatalf("Cmd+A, Cmd+Shift+Right and back left %v chosen, want all 30", chosenRows(chosen))
	}
}

// Shift with End in a grid that has chosen nothing chooses the first item, as
// its arrows do; Cmd+Shift with End adds up to the last item.
func TestGridViewShiftEndWithoutAChoice(t *testing.T) {
	tt, _, chosen := shiftKeysGrid(30)
	tt.Key(0, KeyTab) // focuses the grid, which has chosen nothing
	if chosen.Len() != 0 {
		t.Fatalf("setup: %v chosen", chosenRows(chosen))
	}
	tt.Key(Shift, KeyEnd)
	if got := chosenRows(chosen); !slices.Equal(got, []int{0}) {
		t.Fatalf("Shift+End with nothing chosen chose %v, want [0]", got)
	}
	tt.Click("Item 2")
	tt.ClickWith(Cmd, "Item 5")
	tt.Key(Cmd|Shift, KeyEnd)
	if got, want := chosenRows(chosen), append([]int{2}, seq(5, 29)...); !slices.Equal(got, want) {
		t.Fatalf("Cmd+Shift+End from the pivot Item 5 chose %v, want 2 and 5 to 29", got)
	}
}
