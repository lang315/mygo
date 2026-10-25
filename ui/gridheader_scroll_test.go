package ui

import (
	"slices"
	"testing"
)

// When the width changes the columns, the grid keeps the rows it showed on
// the same items, with the headers between them.
func TestGridViewHeaderReflowKeepsTheItemsInView(t *testing.T) {
	var heads []int
	for h := 0; h < 400; h += 10 {
		heads = append(heads, h)
	}
	tt, s, _ := headerGrid(400, heads)
	s.ScrollTo(301, Start)
	tt.Frame()
	if _, ok := tt.Find("Item 301"); !ok {
		t.Fatal("setup: ScrollTo(301, Start) did not show Item 301")
	}
	tt.SetSize(230, 400)
	tt.Frame()
	tt.Frame()
	if s.cols != 2 {
		t.Fatalf("the narrow grid has %d columns, want 2", s.cols)
	}
	if _, ok := tt.Find("Item 301"); !ok {
		t.Fatal("after the grid got narrower, Item 301 is out of view")
	}
}

// A header as the pivot (an app's SetPivot) extends from nothing: the item
// shift-clicked is chosen alone.
func TestGridViewHeaderAsPivotChoosesTheItemAlone(t *testing.T) {
	tt, s, chosen := headerGrid(18, []int{0, 5, 6, 12})
	tt.Click("Item 2")
	s.SetPivot(5)
	tt.Frame()
	tt.ClickWith(Shift, "Item 8")
	if got := chosenRows(chosen); !slices.Equal(got, []int{8}) {
		t.Fatalf("Shift-click with the header 5 as the pivot chose %v, want Item 8 alone", got)
	}
}
