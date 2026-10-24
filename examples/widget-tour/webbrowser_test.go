package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourHasAWebBrowserThatDrawsAnEmptyBoxWithoutAWindow(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Extras")).Click()
	if !tt.Has(widget.ByLabel("Reload")) {
		t.Fatalf("no browser row: %q", tt.Texts())
	}
}
