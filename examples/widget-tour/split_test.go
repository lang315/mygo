package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourSplitContainerEntry(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Find(widget.ByRole(ui.RoleTreeItem).Label("SplitContainer")).Click()
	for _, s := range []string{"Files", "Editor", "Console"} {
		if !tt.Has(widget.ByLabel(s)) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	if !tt.Has(widget.ByRole(ui.RoleSplitter).Nth(1)) {
		t.Error("the tour's split has no inner divider")
	}
}
