package ui

import "testing"

// The accent of the widgets that are the fork's (a grid's chosen item, the
// highlighted item of a popover menu, the editor of a cell) fades in a window
// that is not drawn as active, as that of the widgets of Window.WindowActive's
// doc does (TestWindowActiveFadesAccent), and a kept row of a grid is built
// again when the window changes.
func TestWindowActiveFadesTheForksWidgets(t *testing.T) {
	light := LightTheme()
	gray := light.inactiveAccent()
	// Accent pixels, in its border or its background, of the region r.
	shows := func(tt *Tester, r Rect, c Color) bool { return pixelsOf(tt.Image(), r, c) > 0 }

	t.Run("grid", func(t *testing.T) {
		g := &gridScene{version: true, n: 300}
		p := newPair(t, 500, 400, g.mk())
		p.idle(3)
		p.both(func(tt *Tester) { _ = tt.Click("Item 5") })
		p.idle(3)
		r, ok := p.built.Find("Item 5")
		if !ok {
			t.Fatalf("no Item 5: %q", p.built.Texts())
		}
		// On the top border of its cell, which is 2 pixels wide.
		at := func() Color { return pixel(p.built.Image(), r.X+30, r.Y) }
		if got := at(); got != light.Accent {
			t.Fatalf("the border of the chosen item of a focused grid is %v, want the accent %v", got, light.Accent)
		}
		p.both(func(tt *Tester) { tt.SetWindowActive(false) })
		p.frame("inactive")
		p.frame("inactive +1")
		if got := at(); got != gray {
			t.Errorf("in a window not active the border is %v, want %v", got, gray)
		}
		p.both(func(tt *Tester) { tt.SetWindowActive(true) })
		p.frame("active again")
		if got := at(); got != light.Accent {
			t.Errorf("the border did not get the accent back with the window: %v", got)
		}
	})

	t.Run("menu", func(t *testing.T) {
		a := &menuApp{}
		tt := coreNewTester(a.view, 400, 300)
		r, _ := tt.Find("Report.pdf")
		tt.RightClickAt(r.X+10, r.Y+r.H/2)
		tt.Key(0, KeyDown) // the first item
		open, ok := tt.Find("Open")
		if !ok {
			t.Fatalf("no menu: %q", tt.Texts())
		}
		region := Rect{open.X - 8, open.Y - 2, open.W + 16, open.H + 4}
		if !shows(tt, region, light.Accent) {
			t.Fatal("the highlighted item of a popover menu does not show the accent")
		}
		tt.SetWindowActive(false)
		tt.Frame()
		if shows(tt, region, light.Accent) || !shows(tt, region, gray) {
			t.Error("the highlighted item of a menu in a window not active still shows the accent")
		}
	})

	t.Run("cell editor", func(t *testing.T) {
		draft := "report"
		tt := NewTester(func(c *Context) {
			Row(c).Padding(8).Children(func() { CellInput(c, &draft, CellStart{Fresh: true, SelectAll: true}) })
		}, 300, 80)
		tt.Frame()
		// The focus ring is the accent too and stays: the border is what changes.
		region := Rect{0, 0, 300, 80}
		active := pixelsOf(tt.Image(), region, light.Accent)
		grayActive := pixelsOf(tt.Image(), region, gray)
		if active == 0 {
			t.Fatal("the editor of a cell does not have an accent border")
		}
		tt.SetWindowActive(false)
		tt.Frame()
		if n := pixelsOf(tt.Image(), region, light.Accent); n >= active {
			t.Errorf("in a window not active %d pixels show the accent, %d when active: the border kept it", n, active)
		}
		if n := pixelsOf(tt.Image(), region, gray); n <= grayActive {
			t.Errorf("in a window not active %d pixels are gray, %d when active: the border did not fade", n, grayActive)
		}
	})
}
