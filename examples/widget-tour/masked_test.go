package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourMaskedPhone(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Inputs")).Click()
	var phone *widget.Node
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleTextField)) {
		if n.Value() == "(___) ___-____" {
			phone = n
		}
	}
	if phone == nil {
		t.Fatalf("no phone box among %q", tt.Texts())
	}
	phone.Focus().Key(ui.Cmd, ui.KeyHome).Type("5551234567")
	if !tt.Has(widget.ByLabel("Phone: (555) 123-4567")) {
		t.Fatalf("status after typing: %q", tt.Texts())
	}
}
