package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The TreeView page's tree shows its folders by name, the search box above it
// filters them, and F2 renames the node chosen.
func TestTourTreeOps(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	visit(tt, "Lists", "TreeView")
	y := func(s string) float32 {
		t.Helper()
		r, ok := tt.Tester.Find(s)
		if !ok {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
		return r.Y
	}
	if !(y("Documents") < y("Music") && y("Music") < y("Pictures")) {
		t.Fatalf("the roots are not by name: %q", tt.Texts())
	}
	if n := len(tt.FindAll(widget.ByRole(ui.RoleTextField))); n != 1 {
		t.Fatalf("%d text fields on the page: want the search box", n)
	}
	tt.Find(widget.ByRole(ui.RoleTextField)).Type("mus")
	tt.Settle()
	if tt.Has(widget.ByLabel("Pictures")) || !tt.Has(widget.ByLabel("Music")) {
		t.Fatalf("filtered by mus: %q", tt.Texts())
	}
	if err := tt.Tester.Click("Music"); err != nil {
		t.Fatal(err)
	}
	tt.Settle()
	tt.Tester.Key(0, ui.KeyF2)
	tt.Settle()
	tt.Tester.Type("Songs")
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	if !tt.HasText("Renamed to Songs") {
		t.Fatalf("F2 and Enter did not rename Music: %q", tt.Texts())
	}
	// Music no longer passes the filter "mus".
	if tt.Has(widget.ByLabel("Songs")) {
		t.Fatalf("Songs shows under the filter mus: %q", tt.Texts())
	}
}
