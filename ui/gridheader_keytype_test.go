package ui

import "testing"

// A header's key need not be of the type of the Selection's keys: the grid does
// not ask the Selection about a header, in Copy, a context menu, Space or
// anywhere else it lists the items chosen.
func TestGridViewHeaderKeyOfAnotherTypeIsNotAskedOfTheSelection(t *testing.T) {
	type headerKey struct{ title string }
	sel, chosen := -1, &Selection[int]{}
	var copied [][]int
	s := &GridState{Selected: &sel, Selection: chosen}
	s.Header = func(i int) bool { return i%4 == 0 }
	s.Key = func(i int) any {
		if s.Header(i) {
			return headerKey{"group"}
		}
		return i
	}
	s.Copy = func(items []int) string {
		copied = append(copied, append([]int(nil), items...))
		return "copied"
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, 12, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 300)
	tt.Frame()
	tt.Click("Item 2")
	tt.Command("copy")
	if tt.Clipboard() != "copied" || len(copied) != 1 || len(copied[0]) != 1 || copied[0][0] != 2 {
		t.Fatalf("Copy of Item 2 gave %q for %v", tt.Clipboard(), copied)
	}
	tt.Key(Cmd, KeyA)
	tt.Command("copy")
	if len(copied) != 2 || len(copied[1]) != 9 {
		t.Fatalf("Copy of all gave %v, want the nine items and no header", copied)
	}
}
