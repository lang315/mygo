package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourTrackBarMovesItsLabel(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	tt.Find(widget.ByRole(ui.RoleSlider).Nth(0)).Key(0, ui.KeyEnd)
	if !tt.Has(widget.ByLabel("Volume 100")) {
		t.Fatalf("the label did not follow the track bar: %q", tt.Texts())
	}
}
