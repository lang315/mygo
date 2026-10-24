package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourHasAFontDialogButton(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Dialogs")).Click()
	if !tt.Has(widget.ByLabel("Font dialog")) {
		t.Fatalf("no Font dialog button: %q", tt.Texts())
	}
}
