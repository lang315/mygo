package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourRangeTrackBarReportsItsRange(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Values")).Click()
	tt.Find(widget.ByRole(ui.RoleSlider).Nth(0)).Key(0, ui.KeyRight)
	if !tt.Has(widget.ByLabel("Price 60 to 200")) {
		t.Fatalf("the label did not follow the range: %q", tt.Texts())
	}
}
