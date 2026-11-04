package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourComboSubmenuAndSelectableTitle(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Extras")).Click()
	tt.Find(widget.ByLabel("Any food")).Click()
	tt.Find(widget.ByLabel("Fruits").Nth(0)).Click()
	tt.Find(widget.ByLabel("Pear")).Click()
	if !tt.Has(widget.ByLabel("Pear")) || tt.Has(widget.ByLabel("Kale")) {
		t.Fatalf("after choosing Pear in the submenu: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Pear")).Click()
	tt.Find(widget.ByLabel("Vegetables").Nth(0)).Click()
	if tt.Has(widget.ByLabel("Kale")) || !tt.Has(widget.ByLabel("Vegetables")) {
		t.Fatalf("the title Vegetables was not chosen: %q", tt.Texts())
	}
}
