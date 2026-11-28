package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Inputs tab's background button shows its SuperTip once the pointer
// rests on it, and assistive technology reads it as its description.
func TestTourSuperTip(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Inputs")).Click()
	b := tt.Find(widget.ByRole(ui.RoleButton).Label("Count in the background")).Bounds()
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	for _, s := range []string{"Counts to five on a goroutine", "Setters are safe from any goroutine."} {
		if !tt.HasText(s) {
			t.Errorf("no %q in the tip; texts %q", s, tt.Texts())
		}
	}
	a, ok := tt.AccessByRole(ui.RoleButton, "Count in the background")
	if want := "Count in the background. Counts to five on a goroutine, setting the greeting's text from there: the window redraws. Setters are safe from any goroutine."; !ok || a.Description != want {
		t.Errorf("description %q, want %q", a.Description, want)
	}
}
