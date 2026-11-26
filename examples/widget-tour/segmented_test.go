package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourSegmentedReportsItsChoice(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Choices")
	tt.Find(widget.ByLabel("Grid")).Click()
	if !tt.Has(widget.ByLabel("View Grid")) {
		t.Fatalf("the label did not follow the segments: %q", tt.Texts())
	}
}
