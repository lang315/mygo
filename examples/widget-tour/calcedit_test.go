package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Choices tab's CalcEdit: its calculator works out an amount, which the
// label beside it says.
func TestTourCalcEdit(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	field := tt.Find(widget.ByRole(ui.RoleTextField).Label("Total"))
	if v := field.Value(); v != "120.00" {
		t.Fatalf("the CalcEdit shows %q, want 120.00", v)
	}
	field.Key(ui.Alt, ui.KeyDown)
	tt.Type("*3+4=")
	tt.Settle()
	if v := field.Value(); v != "364.00" {
		t.Fatalf("120, *3+4= shows %q, want 364.00", v)
	}
	if !tt.Has(widget.ByLabel("Total 364.00")) {
		t.Fatalf("the label did not follow the CalcEdit: %q", tt.Texts())
	}
	if tt.Has(widget.ByRole(ui.RoleButton).Label("=")) {
		t.Fatal("= left the calculator open")
	}
}
