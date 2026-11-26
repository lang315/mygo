package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourRangeTrackBarReportsItsRange(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Choices")
	// The Values group ends the tab: its range's two knobs, then the rating.
	sliders := tt.FindAll(widget.ByRole(ui.RoleSlider))
	sliders[len(sliders)-3].Key(0, ui.KeyRight)
	if !tt.Has(widget.ByLabel("Price 60 to 200")) {
		t.Fatalf("the label did not follow the range: %q", tt.Texts())
	}
}
