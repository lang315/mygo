package ui

import "testing"

// A percentage max-width of a grid item is of its cell, not of the grid: a
// width wider than the room is capped inside its column. (This case holds
// without the change to the grid too, as an Fr column does not grow with its
// content; the widget tests of Fields and Grid are what failed.)
func TestGridItemMaxWidthPercentIsOfItsCell(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Grid().ColumnTracks(FitContent(), Fr(1)).GapX(10).Children(func() {
			coreBox(c).Label("narrow").Size(100, 20)
			coreBox(c).Label("wide").Width(900).Height(20).MaxWidthPercent(100)
		})
	}, 300, 100)
	narrow, _ := tt.Find("narrow")
	wide, _ := tt.Find("wide")
	if wide.X+wide.W > 300.5 || wide.X < narrow.X+narrow.W-0.5 {
		t.Fatalf("narrow %v, wide %v: the wide item must sit in the room that is left of 300", narrow, wide)
	}
}
