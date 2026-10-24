package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourListViewEntry(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 900)
	tt.Find(widget.ByLabel("ListView")).Click()
	if !tt.Has(widget.ByLabel("Kind")) {
		t.Fatalf("Details shows no Kind column: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Large icons")).Click()
	if tt.Has(widget.ByLabel("Kind")) || !tt.Has(widget.ByRole(ui.RoleListItem).Label("Budget")) {
		t.Fatalf("large icons show %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleListItem).Label("Budget")).Click()
	if !tt.Has(widget.ByLabel("1 chosen: [/b]")) {
		t.Fatalf("choosing Budget did not say so: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("List")).Click()
	if !tt.Has(widget.ByRole(ui.RoleListItem).Label("Budget")) {
		t.Fatalf("the list shows %q", tt.Texts())
	}
}
