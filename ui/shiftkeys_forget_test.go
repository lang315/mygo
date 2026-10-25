package ui

import (
	"slices"
	"testing"
)

// moveKey moves key k to index at in keys, as an app sorting its rows does.
func moveKey(keys []int, k, at int) []int {
	keys = slices.DeleteFunc(slices.Clone(keys), func(x int) bool { return x == k })
	return slices.Insert(keys, at, k)
}

// What an extension chose is forgotten when the rows move under it: going
// back works on keys, not on the indices they had, and takes nothing out.
func TestListCmdShiftKeysForgetWhenTheRowsMove(t *testing.T) {
	keys := seq(0, 19)
	var sel Selection[int]
	s := ListState{Selection: &sel, Key: func(i int) any { return keys[i] }}
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, len(keys), func(i int) { coreTextf(c, "K%d", keys[i]).Height(30) }).Grow(1)
	}, 300, 400)
	tt.Click("K10")
	tt.ClickWith(Cmd, "K0")
	for range 3 {
		tt.Key(Cmd|Shift, KeyDown)
	}
	expectChosen(t, &sel, 0, 1, 2, 3, 10)
	keys = moveKey(keys, 10, 2)
	tt.Frame()
	tt.Key(Cmd|Shift, KeyHome)
	expectChosen(t, &sel, 0, 1, 2, 3, 10)
}

func TestGridViewCmdShiftKeysForgetWhenTheItemsMove(t *testing.T) {
	keys := seq(0, 29)
	var chosen Selection[int]
	cur := -1
	s := GridState{Selected: &cur, Selection: &chosen, Key: func(i int) any { return keys[i] }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, &s, len(keys), 100, 40, func(i int) { coreTextf(c, "K%d", keys[i]) }).Grow(1)
		})
	}, 448, 300)
	tt.Click("K10")
	tt.ClickWith(Cmd, "K0")
	for range 3 {
		tt.Key(Cmd|Shift, KeyRight)
	}
	if got := chosenRows(&chosen); !slices.Equal(got, []int{0, 1, 2, 3, 10}) {
		t.Fatalf("setup chose %v", got)
	}
	keys = moveKey(keys, 10, 2)
	tt.Frame()
	tt.Key(Cmd|Shift, KeyHome)
	if got := chosenRows(&chosen); !slices.Equal(got, []int{0, 1, 2, 3, 10}) {
		t.Fatalf("Cmd+Shift+Home after the items moved chose %v, want [0 1 2 3 10]", got)
	}
}

// What an extension chose is forgotten when the app changes the choice: going
// back from there takes out what it chooses afresh.
func TestListCmdShiftKeysForgetWhenTheAppClearsTheChoice(t *testing.T) {
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
	sel.Clear()
	tt.Frame()
	for range 3 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	expectChosen(t, &sel, 3, 4)
}

func TestGridViewCmdShiftKeysForgetWhenTheAppClearsTheChoice(t *testing.T) {
	tt, _, chosen := shiftKeysGrid(30)
	tt.Click("Item 6")
	tt.ClickWith(Cmd, "Item 3")
	for range 4 {
		tt.Key(Cmd|Shift, KeyRight)
	}
	chosen.Clear()
	tt.Frame()
	for range 3 {
		tt.Key(Cmd|Shift, KeyLeft)
	}
	if got := chosenRows(chosen); !slices.Equal(got, []int{3, 4}) {
		t.Fatalf("Cmd+Shift+Left three times after Clear chose %v, want [3 4]", got)
	}
}

// A grid whose pivot is lost (an app's SetPivot, rows gone) starts afresh: the
// exceptions of the extension before do not stay.
func TestGridViewCmdShiftKeysWithoutAPivotForgetTheExceptions(t *testing.T) {
	tt, s, chosen := shiftKeysGrid(30)
	tt.Click("Item 2")
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd|Shift, KeyEnd)
	if len(s.except) != 27 {
		t.Fatalf("setup: %d exceptions, want the 27 items chosen in the way", len(s.except))
	}
	s.pivot = -1
	tt.Key(Shift, KeyLeft)
	if len(s.except) != 0 {
		t.Fatalf("%d exceptions after the pivot was lost, want none", len(s.except))
	}
	if chosen.Len() == 0 {
		t.Fatal("nothing chosen")
	}
}

// An app that tells the list where the pivot is on every build (SetPivot, as
// a view keeping it by key does) does not make it forget the extension, while
// the pivot stays where it is; moving it does.
func TestListSetPivotToTheSameRowKeepsTheExtension(t *testing.T) {
	var sel Selection[int]
	s := ListState{Selection: &sel}
	changes := 0
	tt := coreNewTester(severalView(&s, &changes), 300, 400)
	tt.Click("Row 6")
	tt.ClickWith(Cmd, "Row 3")
	tt.Key(Cmd|Shift, KeyDown)
	tt.Key(Cmd|Shift, KeyDown)
	expectChosen(t, &sel, 3, 4, 5, 6)
	s.SetPivot(s.Pivot())
	tt.Frame()
	tt.Key(Cmd|Shift, KeyUp)
	expectChosen(t, &sel, 3, 4, 6)
	s.SetPivot(3 + 2)
	tt.Frame()
	tt.Key(Cmd|Shift, KeyUp)
	// The pivot moved to Row 5: the extension is forgotten, and the rows from
	// it to Row 3 are chosen, nothing taken out.
	expectChosen(t, &sel, 3, 4, 5, 6)
}
