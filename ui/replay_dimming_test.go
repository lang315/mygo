package ui

import "testing"

// A theme that stops dimming disabled elements (Theme.DisableAutoDimming)
// repaints the rows that replayed them dimmed: painting reads the theme, not
// only the elements, which are built as before.
func TestRowsPaintAgainWhenTheThemeStopsDimming(t *testing.T) {
	var list ListState
	dimmed, undimmed := LightTheme(), LightTheme()
	undimmed.DisableAutoDimming = true
	theme := dimmed
	tt := coreNewTester(func(c *context) {
		c.SetTheme(theme)
		coreList(c, &list, 100, func(i int) {
			coreBox(c).Size(20, 20).Background(RGB(0, 0, 0)).Disabled(true)
		}).Grow(1)
	}, 300, 400)
	for range 4 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	theme = undimmed
	tt.Frame()
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
	}
	if px := tt.Image().RGBAAt(5, 5); px.R != 0 {
		t.Fatalf("disabled box in a row at %v, want black once the theme stops dimming", px)
	}
}
