package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourPopupEditShowsTheChosenColumns(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Dialogs")).Click()
	tt.Find(widget.ByRole(ui.RolePopUpButton)).Click()
	tt.Find(widget.ByLabel("City")).Click()
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if v := tt.Find(widget.ByRole(ui.RolePopUpButton)).Value(); v != "City" {
		t.Fatalf("after choosing City the field shows %q", v)
	}
}
