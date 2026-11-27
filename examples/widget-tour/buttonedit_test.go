package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourButtonEdit(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Inputs")).Click()
	tt.Find(widget.ByRole(ui.RoleButton).Label("Browse")).Click()
	if !tt.Has(widget.ByLabel("File: report.pdf")) {
		t.Fatalf("status after Browse: %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleButton).Label("Clear")).Click()
	if !tt.Has(widget.ByLabel("File: none")) {
		t.Fatalf("status after Clear: %q", tt.Texts())
	}
	if tt.Has(widget.ByRole(ui.RoleButton).Label("Clear")) {
		t.Fatal("Clear shows while the field is empty")
	}
}
