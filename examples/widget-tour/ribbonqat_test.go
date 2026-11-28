package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Navigation page's ribbon shows Save and Undo in its Quick Access
// Toolbar, and its Format tab only while the picture is selected.
func TestTourRibbonQATAndContextualTab(t *testing.T) {
	tt := navigationPage(t)
	for _, name := range []string{"Save", "Undo"} {
		if !tt.Has(widget.ByLabel(name).Role(ui.RoleButton)) {
			t.Fatalf("no QAT button %q: %q", name, tt.Texts())
		}
	}
	format := widget.ByLabel("Format").Role(ui.RoleTab)
	if tt.Has(format) || tt.Has(widget.ByLabel("Picture Tools")) {
		t.Fatalf("the Format tab shows with no picture selected: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Picture selected")).Click()
	if !tt.Has(format) || !tt.Has(widget.ByLabel("Picture Tools")) {
		t.Fatalf("the Format tab does not show with the picture selected: %q", tt.Texts())
	}
	tt.Find(format).Click()
	if !tt.Has(widget.ByLabel("Rotate")) {
		t.Fatalf("Format's commands do not show: %q", tt.Texts())
	}
}
