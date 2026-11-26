package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourColorEditSetsTheTint(t *testing.T) {
	tt := navigationPage(t)
	tt.Find(widget.ByRole(ui.RoleColorWell)).Click()
	tt.Find(widget.ByLabel("Green")).Click()
	if !tt.Has(widget.ByLabel("Tint #22c55e")) {
		t.Fatalf("a pick of Green did not set the tint: %q", tt.Texts())
	}
}
