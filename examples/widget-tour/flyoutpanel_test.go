package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Containers tab's flyout: the button slides the panel in, and Escape
// closes it, which the status line says.
func TestTourFlyoutPanel(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Find(widget.ByLabel("FlyoutPanel")).Click()
	if tt.Has(widget.ByLabel("Show hidden files")) {
		t.Fatalf("the panel shows before the button: %q", tt.Texts())
	}
	tt.Find(widget.ByRole(ui.RoleButton).Label("Show the flyout")).Click()
	tt.Advance(time.Second)
	tt.Settle()
	if !tt.Has(widget.ByLabel("Show hidden files")) || !tt.Has(widget.ByLabel("Open")) {
		t.Fatalf("the button did not show the panel: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	tt.Advance(time.Second)
	tt.Settle()
	if tt.Has(widget.ByLabel("Show hidden files")) || !tt.Has(widget.ByLabel("Closed by the user")) {
		t.Fatalf("Escape did not close the panel: %q", tt.Texts())
	}
}
