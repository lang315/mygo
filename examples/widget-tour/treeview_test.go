package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourTreeViewEntry(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("TreeView")).Click()
	for _, s := range []string{"Documents", "Pictures", "Music"} {
		if !tt.Has(widget.ByLabel(s)) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	if tt.Has(widget.ByLabel("Loading…")) {
		t.Fatal("a closed folder shows its Loading row")
	}
	r, ok := tt.Tester.Find("Documents")
	if !ok {
		t.Fatal("no Documents row")
	}
	tt.ClickAt(r.X+16, r.Y+r.H/2) // the arrow
	tt.Frame()
	if !tt.Has(widget.ByLabel("Loading…")) {
		t.Fatalf("opening Documents shows no Loading row: %q", tt.Texts())
	}
}
