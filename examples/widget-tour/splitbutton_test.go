package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourSplitButtonRunsTheButtonAndTheMenu(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Choices")
	tt.Find(widget.ByLabel("Save")).Click()
	if !tt.Has(widget.ByLabel("Saved")) {
		t.Fatalf("the button did not run: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("More options")).Click()
	if err := tt.ChooseMenuItem("Save All"); err != nil {
		t.Fatal(err)
	}
	tt.Settle()
	if !tt.Has(widget.ByLabel("Saved all")) {
		t.Fatalf("the menu's item did not run: %q", tt.Texts())
	}
}
