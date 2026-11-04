package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourNumericUpDownArrowsAndKeysStep(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	tt.Find(widget.ByLabel("Increase").Nth(0)).Click()
	if !tt.Has(widget.ByLabel("31%")) {
		t.Fatalf("the up arrow did not step the number: %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleTextField).Nth(0)).Key(0, ui.KeyDown).Key(0, ui.KeyDown)
	if !tt.Has(widget.ByLabel("29%")) {
		t.Fatalf("Down, Down did not step the number: %q", tt.Texts())
	}
}
