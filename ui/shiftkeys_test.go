package ui

import (
	"runtime"
	"slices"
	"testing"
)

// Cmd+Shift with the arrows (Ctrl+Shift on Linux and Windows) extends the
// choice from the pivot and keeps the rows chosen before, as Explorer does;
// going back takes out the rows it left, and the keys that move the choice
// further (Home, End) add up to there.
func TestListCmdShiftKeysExtendKeepingTheOthers(t *testing.T) {
	var sel Selection[int]
	s := ListState{Selection: &sel}
	changes := 0
	tt := coreNewTester(severalView(&s, &changes), 300, 400)
	tt.Click("Row 2")
	tt.ClickWith(Cmd, "Row 5")
	expectChosen(t, &sel, 2, 5)
	tt.Key(Cmd|Shift, KeyDown)
	tt.Key(Cmd|Shift, KeyDown)
	expectChosen(t, &sel, 2, 5, 6, 7)
	tt.Key(Cmd|Shift, KeyUp)
	expectChosen(t, &sel, 2, 5, 6)
	for range 3 {
		tt.Key(Cmd|Shift, KeyUp)
	}
	expectChosen(t, &sel, 2, 3, 4, 5)
	tt.Key(Cmd|Shift, KeyEnd)
	if got, want := chosenRows(&sel), append([]int{2}, seq(5, 19)...); !slices.Equal(got, want) {
		t.Fatalf("Cmd+Shift+End from the pivot 5 chose %v, want 2 and 5 to 19", got)
	}
	tt.Key(Cmd|Shift, KeyHome)
	if got := chosenRows(&sel); !slices.Equal(got, []int{0, 1, 2, 3, 4, 5}) {
		t.Fatalf("Cmd+Shift+Home from the pivot 5 chose %v, want 0 to 5", got)
	}
	// Shift alone still takes the place of the others: the key moves from
	// Row 0, where the choice is, to Row 1.
	tt.Key(Shift, KeyDown)
	expectChosen(t, &sel, 1, 2, 3, 4, 5)
}

func shiftKeysGrid(items int) (*Tester, *GridState, *Selection[int]) {
	sel, chosen := -1, &Selection[int]{}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, items, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	return tt, s, chosen
}

// A grid takes Shift with Home and End, as a list does.
func TestGridViewShiftHomeEnd(t *testing.T) {
	tt, s, chosen := shiftKeysGrid(30)
	tt.Click("Item 13")
	tt.Key(Shift, KeyEnd)
	want := make([]int, 0, 17)
	for i := 13; i < 30; i++ {
		want = append(want, i)
	}
	if got := chosenRows(chosen); !slices.Equal(got, want) {
		t.Fatalf("Shift+End from Item 13 chose %v, want 13 to 29", got)
	}
	tt.Key(Shift, KeyHome)
	want = want[:0]
	for i := 0; i <= 13; i++ {
		want = append(want, i)
	}
	if got := chosenRows(chosen); !slices.Equal(got, want) {
		t.Fatalf("Shift+Home from the pivot Item 13 chose %v, want 0 to 13", got)
	}
	if s.Pivot() != 13 {
		t.Fatalf("Pivot %d, want 13", s.Pivot())
	}
	// Without Shift, End and Home choose the item alone, as before.
	tt.Key(0, KeyEnd)
	expectChosen(t, chosen, 29)
}

// Shift with Page Down and Page Up extends the choice by the rows in view, in
// the column of the item the keys move from.
func TestGridViewShiftPageKeys(t *testing.T) {
	tt, s, chosen := shiftKeysGrid(60)
	tt.Click("Item 1")
	first, last := s.rows.Visible()
	cols := s.cols
	if last-first < 2 || cols < 2 {
		t.Fatalf("test grid shows rows %d to %d in %d columns", first, last, cols)
	}
	if runtime.GOOS == "darwin" {
		// Page Down and Page Up scroll on macOS, as in a list.
		tt.Key(Shift, KeyPageDown)
		if got := chosenRows(chosen); !slices.Equal(got, []int{1}) {
			t.Fatalf("Shift+Page Down chose %v on macOS, want Item 1 alone: it scrolls", got)
		}
		return
	}
	tt.Key(Shift, KeyPageDown)
	to := last*cols + 1
	if got := chosenRows(chosen); !slices.Equal(got, seq(1, to)) {
		t.Fatalf("Shift+Page Down from Item 1 chose %v, want 1 to %d (the last row in view)", got, to)
	}
	tt.Key(Shift, KeyPageDown)
	to2 := *s.Selected
	if to2 <= to || to2%cols != 1 {
		t.Fatalf("Shift+Page Down again moved to %d from %d, want further down in the same column", to2, to)
	}
	if got := chosenRows(chosen); !slices.Equal(got, seq(1, to2)) {
		t.Fatalf("Shift+Page Down again chose %v, want 1 to %d", got, to2)
	}
	tt.Key(Shift, KeyPageUp)
	if back := *s.Selected; back >= to2 || back%cols != 1 {
		t.Fatalf("Shift+Page Up moved to %d from %d", back, to2)
	}
	for range 10 {
		tt.Key(Shift, KeyPageUp)
	}
	if got := chosenRows(chosen); !slices.Equal(got, []int{1}) {
		t.Fatalf("Shift+Page Up back to the pivot's row chose %v, want Item 1 alone", got)
	}
}

// A grid takes Cmd+Shift with the arrows as a list does: extending from the
// pivot and keeping the items chosen before.
func TestGridViewCmdShiftKeysExtendKeepingTheOthers(t *testing.T) {
	tt, s, chosen := shiftKeysGrid(30)
	cols := s.cols
	tt.Click("Item 0")
	tt.ClickWith(Cmd, "Item 9")
	tt.Key(Cmd|Shift, KeyRight)
	tt.Key(Cmd|Shift, KeyRight)
	if got := chosenRows(chosen); !slices.Equal(got, []int{0, 9, 10, 11}) {
		t.Fatalf("Cmd+Shift+Right twice from Item 9 chose %v, want [0 9 10 11]", got)
	}
	tt.Key(Cmd|Shift, KeyLeft)
	if got := chosenRows(chosen); !slices.Equal(got, []int{0, 9, 10}) {
		t.Fatalf("Cmd+Shift+Left chose %v, want [0 9 10]: going back takes out the item it left", got)
	}
	tt.Key(Cmd|Shift, KeyDown)
	want := append([]int{0}, seq(9, 10+cols)...)
	if got := chosenRows(chosen); !slices.Equal(got, want) {
		t.Fatalf("Cmd+Shift+Down chose %v, want %v", got, want)
	}
}

// seq returns the integers from lo to hi.
func seq(lo, hi int) []int {
	var out []int
	for i := lo; i <= hi; i++ {
		out = append(out, i)
	}
	return out
}
