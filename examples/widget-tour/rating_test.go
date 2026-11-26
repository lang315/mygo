package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourRatingReportsItsStars(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Choices")
	// The rating is the last slider of the tab, in its Values group.
	sliders := tt.FindAll(widget.ByRole(ui.RoleSlider))
	sliders[len(sliders)-1].Key(0, ui.KeyEnd)
	if !tt.Has(widget.ByLabel("Rated 5")) {
		t.Fatalf("the label did not follow the rating: %q", tt.Texts())
	}
}
