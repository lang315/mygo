package ui

import (
	"slices"
	"testing"
)

// A destructive control that does not fit the toolbar goes into the overflow
// menu as a destructive item; the others, and a control made destructive and
// then not, as normal ones.
func TestToolbarOverflowKeepsADestructiveControlDestructive(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Toolbar(c, func() {
			Button(c, "Alpha")
			Button(c, "Beta")
			Button(c, "Delete all").Destructive(true)
			Button(c, "Reset").Destructive(true).Destructive(false)
		})
	}, 110, 100)
	tt.Frame()
	if err := tt.Click("More"); err != nil {
		t.Fatalf("no overflow button (texts %q): %v", tt.Texts(), err)
	}
	tt.Frame()
	menu := tt.Menu()
	if !slices.Contains(menu, "Delete all") || !slices.Contains(menu, "Reset") {
		t.Fatalf("overflow menu %q: want Delete all and Reset in it", menu)
	}
	if !tt.MenuItemDestructive("Delete all") {
		t.Error("Delete all is a normal item in the overflow menu")
	}
	if tt.MenuItemDestructive("Reset") {
		t.Error("Reset, not destructive any more, is a destructive item")
	}
}
