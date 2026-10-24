package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourShowsTheCharts(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Extras")).Click()
	if !tt.Has(widget.ByLabel("Sales")) || !tt.Has(widget.ByLabel("Users")) || !tt.Has(widget.ByLabel("Mar")) {
		t.Fatalf("no charts: %q", tt.Texts())
	}
}
