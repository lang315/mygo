package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourListBoxEntry(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 900)
	tt.Find(widget.ByLabel("ListBox")).Click()
	item := func(s string) widget.Query { return widget.ByRole(ui.RoleListItem).Label(s) }
	for _, s := range []string{"Vietnam", "Japan", "Cheese", "Basil"} {
		if !tt.Has(item(s)) {
			t.Fatalf("no list item %q in %q", s, tt.Texts())
		}
	}
	tt.Find(item("Japan")).Click()
	if !tt.Has(widget.ByLabel("1 chosen: [jp]")) {
		t.Fatalf("choosing Japan did not say so: %q", tt.Texts())
	}
}
