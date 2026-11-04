package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// StripStyle colors the tab strip across the control, past its last tab,
// and not the page under it.
func TestTabControlStripStyleColorsTheStrip(t *testing.T) {
	red := ui.RGB(200, 0, 0)
	nb := TabControl().AddPage("A", Label("page A")).AddPage("B", Label("page B")).
		StripStyle(NewStyle().Background(Fixed(red)))
	tt := tester(t, nb)
	b := find(t, tt, "B")
	if c := tt.Image().RGBAAt(int(b.X+b.W+40), int(b.Y+b.H/2)); !near(c, red) {
		t.Fatalf("strip past the last tab is %v, want %v", c, red)
	}
	p := find(t, tt, "page A")
	if c := tt.Image().RGBAAt(int(p.X+p.W+40), int(p.Y+p.H/2)); near(c, red) {
		t.Fatalf("the page took the strip's background %v", c)
	}
	// StripStyle() takes it off.
	nb.StripStyle()
	tt.Frame()
	if c := tt.Image().RGBAAt(int(b.X+b.W+40), int(b.Y+b.H/2)); near(c, red) {
		t.Fatalf("strip still %v after StripStyle()", c)
	}
}
