package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourTabControlStripStyle(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 700)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Find(widget.ByLabel("TabControl")).Click()
	second := tt.Find(widget.ByLabel("Second"))
	b := second.Bounds()
	if c := tt.Image().RGBAAt(int(b.X+b.W+40), int(b.Y+b.H/2)); c.R != 0xdb || c.G != 0xea || c.B != 0xfe {
		t.Fatalf("the tab strip past its last tab is %v, want #dbeafe", c)
	}
	second.Click()
	if !tt.Has(widget.ByLabel("The second page")) {
		t.Fatalf("the second tab shows %q", tt.Texts())
	}
}
