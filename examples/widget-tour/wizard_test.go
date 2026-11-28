package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Containers tab's wizard: the empty name keeps the first step, a name
// moves on, Enter goes to the last step, and Done finishes.
func TestTourWizard(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Find(widget.ByLabel("Wizard")).Click()
	button := func(s string) *widget.Node { return tt.Find(widget.ByRole(ui.RoleButton).Label(s)) }
	if !button("Previous").Disabled() {
		t.Errorf("Previous is enabled on the first step")
	}
	button("Next").Click()
	name := widget.ByRole(ui.RoleTextField).Label("Name")
	if !tt.Has(name) || tt.Has(widget.ByLabel("The free plan")) {
		t.Fatalf("an empty name left the first step: %q", tt.Texts())
	}
	tt.Find(name).Type("Ada")
	button("Next").Click()
	if !tt.Has(widget.ByLabel("The free plan")) {
		t.Fatalf("a name did not move on: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	button("Done").Click()
	if !tt.Has(widget.ByLabel("Finished")) {
		t.Fatalf("Done did not finish: %q", tt.Texts())
	}
}
