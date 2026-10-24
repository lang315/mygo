package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourDragChipOntoTheZone(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Find(widget.ByRole(ui.RoleTreeItem).Label("Drag and drop")).Click()
	a, b := tt.Find(widget.ByLabel("Drag me")).Bounds(), tt.Find(widget.ByLabel("Drop here")).Bounds()
	ax, ay, bx, by := a.X+a.W/2, a.Y+a.H/2, b.X+b.W/2, b.Y+b.H/2
	tt.Press(ax, ay)
	for i := float32(1); i <= 4; i++ {
		tt.Move(ax+(bx-ax)*i/4, ay+(by-ay)*i/4)
		tt.Frame()
	}
	tt.Release(bx, by)
	tt.Settle()
	if !tt.Has(widget.ByLabel("Dropped a chip")) {
		t.Fatalf("texts %q", tt.Texts())
	}
}
