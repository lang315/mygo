package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourMenuBarPicksFromASubmenu(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("App shell")).Click()
	tt.Find(widget.ByRole(ui.RoleMenuItem).Label("File")).Click()
	tt.Find(widget.ByRole(ui.RoleMenuItem).Label("Open recent")).Click()
	tt.Find(widget.ByRole(ui.RoleMenuItem).Label("notes.txt")).Click()
	if !tt.Has(widget.ByLabel("Picked notes.txt")) {
		t.Fatalf("texts %q", tt.Texts())
	}
	// While File shows, Help is a hover away.
	tt.Find(widget.ByRole(ui.RoleMenuItem).Label("File")).Click()
	r := tt.Find(widget.ByRole(ui.RoleMenuItem).Label("Help")).Bounds()
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Settle()
	tt.Find(widget.ByRole(ui.RoleMenuItem).Label("About")).Click()
	if !tt.Has(widget.ByLabel("Picked About")) {
		t.Fatalf("texts %q", tt.Texts())
	}
}
