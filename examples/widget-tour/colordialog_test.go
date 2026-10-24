package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourHasAColorDialogButton(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Dialogs")).Click()
	if !tt.Has(widget.ByLabel("Color dialog")) {
		t.Fatalf("no Color dialog button: %q", tt.Texts())
	}
}
