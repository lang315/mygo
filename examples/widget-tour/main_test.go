package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourShowsEveryTab(t *testing.T) {
	tt := ui.NewTester(widget.View(tour(nil)), 1000, 700)
	tt.Frame()
	for _, tab := range []string{"Layout", "Inputs", "Choices", "Containers", "Dialogs", "Extras"} {
		if err := tt.Click(tab); err != nil {
			t.Fatalf("tab %s: %v (texts %q)", tab, err, tt.Texts())
		}
		tt.Frame()
	}
}
