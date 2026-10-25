package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourListViewGroupsByKind(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 900)
	tt.Find(widget.ByLabel("ListView")).Click()
	tt.Find(widget.ByLabel("Group by kind")).Click()
	tt.Find(widget.ByLabel("Large icons")).Click()
	for _, header := range []string{"Document (3)", "Sheet (3)", "Image (3)"} {
		if !tt.Has(widget.ByRole(ui.RoleListItem).Label(header)) {
			t.Fatalf("large icons show no header %q: %q", header, tt.Texts())
		}
	}
	// Choosing a row of a group says so, and the headers choose nothing.
	tt.Find(widget.ByRole(ui.RoleListItem).Label("Budget")).Click()
	if !tt.Has(widget.ByLabel("1 chosen: [/b]")) {
		t.Fatalf("choosing Budget did not say so: %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleListItem).Label("Image (3)")).Click()
	if !tt.Has(widget.ByLabel("1 chosen: [/b]")) {
		t.Fatalf("a click on a header changed the choice: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("No groups")).Click()
	if tt.Has(widget.ByRole(ui.RoleListItem).Label("Image (3)")) {
		t.Fatalf("the headers are still there after No groups: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Details")).Click()
	tt.Find(widget.ByLabel("Group by kind")).Click()
	if tt.Has(widget.ByRole(ui.RoleListItem).Label("Image (3)")) || !tt.Has(widget.ByLabel("Kind")) {
		t.Fatalf("Details shows groups: %q", tt.Texts())
	}
}
