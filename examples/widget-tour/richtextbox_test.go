package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourRichTextBoxBoldButton(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Extras")).Click()
	box := tt.Find(widget.ByRole(ui.RoleTextField).Label("Notes"))
	if box.Value() != "Edit rich text" {
		t.Fatalf("the box shows %q", box.Value())
	}
}
