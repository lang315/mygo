package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourCheckButtonReportsItsState(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Choices")
	tt.Find(widget.ByLabel("Bold")).Click()
	if !tt.Has(widget.ByLabel("Bold true")) {
		t.Fatalf("the label did not follow the check button: %q", tt.Texts())
	}
}
