package ui

import (
	"runtime"
	"slices"
	"testing"
)

// The rows of any grid with headers partition its items in order: a header
// alone in its row, the items of a group in rows of the columns, the last of
// them short, and itemRow and rowItems agree. Every set of headers of grids
// of up to ten items, in one to four columns.
func TestGridViewHeaderRowsPartitionTheItems(t *testing.T) {
	for n := 0; n <= 10; n++ {
		for mask := 0; mask < 1<<n; mask++ {
			heads := func(i int) bool { return mask>>i&1 == 1 }
			for cols := 1; cols <= 4; cols++ {
				s := &GridState{Header: heads, cols: cols}
				s.prepare(n, cols)
				next := 0
				rows := s.rowCount(n, cols)
				for r := 0; r < rows; r++ {
					first, end, header := s.rowItems(r, n)
					if first != next || end <= first || end > n {
						t.Fatalf("n %d, headers %b, %d columns: row %d has items %d to %d, want from %d", n, mask, cols, r, first, end, next)
					}
					if header {
						if end != first+1 || !heads(first) {
							t.Fatalf("n %d, headers %b, %d columns: row %d is a header row of %d to %d", n, mask, cols, r, first, end)
						}
					} else {
						for i := first; i < end; i++ {
							if heads(i) {
								t.Fatalf("n %d, headers %b, %d columns: header %d in the items' row %d", n, mask, cols, i, r)
							}
						}
						if end-first < cols && end < n && !heads(end) {
							t.Fatalf("n %d, headers %b, %d columns: row %d is short but the group goes on", n, mask, cols, r)
						}
					}
					for i := first; i < end; i++ {
						if got := s.itemRow(i); got != r {
							t.Fatalf("n %d, headers %b, %d columns: itemRow(%d) = %d, want %d", n, mask, cols, i, got, r)
						}
					}
					next = end
				}
				if next != n {
					t.Fatalf("n %d, headers %b, %d columns: the rows cover %d items", n, mask, cols, next)
				}
			}
		}
	}
}

// Page Down and Page Up with Shift extend by the rows in view, passing the
// headers by. On macOS they scroll, as in a list.
func TestGridViewHeaderShiftPageKeys(t *testing.T) {
	var heads []int
	for h := 0; h < 120; h += 6 {
		heads = append(heads, h)
	}
	tt, s, chosen := headerGrid(120, heads)
	tt.Click("Item 1")
	tt.Key(Shift, KeyPageDown)
	if runtime.GOOS == "darwin" {
		if got := chosenRows(chosen); !slices.Equal(got, []int{1}) {
			t.Fatalf("Shift+Page Down chose %v on macOS, want Item 1 alone", got)
		}
		return
	}
	got := chosenRows(chosen)
	if len(got) < 3 || got[0] != 1 {
		t.Fatalf("Shift+Page Down from Item 1 chose %v", got)
	}
	for _, i := range got {
		if i%6 == 0 {
			t.Fatalf("Shift+Page Down chose the header %d: %v", i, got)
		}
	}
	if *s.Selected != got[len(got)-1] {
		t.Fatalf("the keys moved to %d, the last chosen is %d", *s.Selected, got[len(got)-1])
	}
	tt.Key(Shift, KeyPageUp)
	tt.Key(Shift, KeyPageUp)
	if got := chosenRows(chosen); !slices.Equal(got, []int{1}) {
		t.Fatalf("Shift+Page Up back to the pivot's row chose %v, want Item 1 alone", got)
	}
}

// Typing the first letters of an item chooses it, and a header with the same
// letters is passed by.
func TestGridViewHeaderTypeSelectPassesHeaders(t *testing.T) {
	sel, chosen := -1, &Selection[int]{}
	names := []string{"Fruit", "Apple", "Avocado", "Veg", "Asparagus", "Vine"}
	s := &GridState{Selected: &sel, Selection: chosen, Key: func(i int) any { return i },
		Header: func(i int) bool { return names[i] == "Fruit" || names[i] == "Veg" },
		Label:  func(i int) string { return names[i] }}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, s, len(names), 100, 40, func(i int) { coreText(c, names[i]) }).Grow(1)
	}, 448, 300)
	tt.Frame()
	tt.Click("Apple")
	tt.Key(0, KeyA)
	tt.Key(0, KeyS)
	if sel != 4 {
		t.Fatalf("typing \"as\" chose item %d, want 4 (Asparagus)", sel)
	}
	tt.Advance(typePause) // what was typed is forgotten
	tt.Click("Apple")
	tt.Key(0, KeyV)
	if sel != 5 {
		t.Fatalf("typing \"v\" chose item %d, want 5 (Vine): Veg, item 3, is a header", sel)
	}
}

// A narrower window means fewer columns: the rows are laid out again with
// the headers in place.
func TestGridViewHeaderRowsFollowTheWidth(t *testing.T) {
	tt, s, _ := headerGrid(18, []int{0, 5, 6, 12})
	tt.SetSize(230, 400)
	tt.Frame()
	tt.Frame()
	if s.cols != 2 {
		t.Fatalf("the narrow grid has %d columns, want 2", s.cols)
	}
	y := func(text string) float32 { return rowY(t, tt, text) }
	if y("Item 1") != y("Item 2") || y("Item 2") >= y("Item 3") || y("Item 4") >= y("H5") {
		t.Errorf("the group 1 to 4 is not in rows of 2 above H5")
	}
	if y("H5") >= y("H6") || y("H6") >= y("Item 7") || y("Item 7") != y("Item 8") {
		t.Errorf("H5, H6 and the group from 7 are not laid out in rows")
	}
	if y("Item 11") >= y("H12") || y("H12") >= y("Item 13") {
		t.Errorf("Item 11 (alone), H12 and Item 13 are not in rows in order")
	}
}
