package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourShowsEveryTab(t *testing.T) {
	tt := ui.NewTester(widget.View(tour(nil)), 1000, 700)
	tt.Frame()
	for _, tab := range []string{"Layout", "Inputs", "Choices", "Containers", "Dialogs", "Extras", "App shell", "DataGridView"} {
		if err := tt.Click(tab); err != nil {
			t.Fatalf("tab %s: %v (texts %q)", tab, err, tt.Texts())
		}
		tt.Frame()
	}
}

func TestTourTableTab(t *testing.T) {
	tt := ui.NewTester(widget.View(tour(nil)), 1000, 700)
	tt.Frame()
	if err := tt.Click("DataGridView"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, s := range []string{"Rows: 100000", "acct000000", "Search"} {
		// The tester lists texts off the window too: the panel must fit.
		if r, ok := tt.Find(s); !ok || r.Y+r.H > 700 {
			t.Errorf("%q at %v (found %v), not in the window; texts %q", s, r, ok, tt.Texts())
		}
	}
}

func TestTourTrackBarMovesItsLabel(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	tt.Find(widget.ByRole(ui.RoleSlider).Nth(0)).Key(0, ui.KeyEnd)
	if !tt.Has(widget.ByLabel("Volume 100")) {
		t.Fatalf("the label did not follow the track bar: %q", tt.Texts())
	}
}
