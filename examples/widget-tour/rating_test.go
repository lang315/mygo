package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourRatingReportsItsStars(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Values")).Click()
	// The range's two knobs come first.
	tt.Find(widget.ByRole(ui.RoleSlider).Nth(2)).Key(0, ui.KeyEnd)
	if !tt.Has(widget.ByLabel("Rated 5")) {
		t.Fatalf("the label did not follow the rating: %q", tt.Texts())
	}
}
