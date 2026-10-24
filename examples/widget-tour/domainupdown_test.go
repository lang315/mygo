package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourDomainUpDownSteps(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	if !tt.Has(widget.ByLabel("Mon")) {
		t.Fatalf("no day chosen: %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleStepper).Label("")).Key(0, ui.KeyDown)
	if !tt.Has(widget.ByLabel("Tue")) {
		t.Fatalf("Down did not go to the next day: %q", tt.Texts())
	}
}
