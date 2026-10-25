package ui

import "testing"

// A page from any item lands on an item, never on a header, in the column of
// the item it started from, or the last of a shorter row, and going the other
// way does not skip the header rows' neighbours. Tested on page itself, as
// the keys do not take Page Up and Page Down on macOS.
func TestGridViewHeaderPageLandsOnItems(t *testing.T) {
	var heads []int
	for h := 0; h < 120; h += 6 {
		heads = append(heads, h)
	}
	_, s, _ := headerGrid(120, heads)
	const n = 120
	for at := 0; at < n; at++ {
		if s.isHeader(at) {
			continue
		}
		first, _, _ := s.rowItems(s.itemRow(at), n)
		col := at - first
		for _, d := range []int{1, -1} {
			to := s.page(at, d, n)
			if s.isHeader(to) {
				t.Fatalf("a page from item %d (direction %d) landed on the header %d", at, d, to)
			}
			tf, te, _ := s.rowItems(s.itemRow(to), n)
			if want := min(tf+col, te-1); to != want && to != at {
				t.Fatalf("a page from item %d (direction %d) landed on %d, want the column %d of its row, %d", at, d, to, col, want)
			}
			if d > 0 && to < at || d < 0 && to > at {
				t.Fatalf("a page from item %d (direction %d) went the other way, to %d", at, d, to)
			}
		}
	}
}
