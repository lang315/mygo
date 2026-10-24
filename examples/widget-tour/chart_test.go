package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourShowsTheCharts(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1600)
	tt.Find(widget.ByLabel("Extras")).Click()
	if !tt.Has(widget.ByLabel("2026, Jan: 10")) || !tt.Has(widget.ByLabel("Win: 60 (60%)")) {
		t.Fatalf("no charts: %d marks", len(tt.FindAll(widget.ByRole(ui.RoleButton))))
	}
}
