package ui

import (
	"sort"
	"testing"
)

// SetDefaultBorder gives the default themes of both appearances the border
// colors, for apps with no Context of their own to set a Theme with, and a
// change of appearance keeps them.
func TestSetDefaultBorder(t *testing.T) {
	var got Color
	view := func(c *context) { got = c.Theme().Border }
	light, dark := Hex("#9ca3af"), Hex("#6b7280")

	tt := coreNewTester(view, 100, 100)
	if got != LightTheme().Border {
		t.Fatalf("before the call, the light border is %v, want the theme's %v", got, LightTheme().Border)
	}

	SetDefaultBorder(light, dark)
	t.Cleanup(func() { SetDefaultBorder(Color{}, Color{}) })

	tt.Frame()
	if got != light {
		t.Errorf("a window open when it was called shows the light border %v, want %v", got, light)
	}
	tt.SetDark(true)
	if got != dark {
		t.Errorf("after the appearance went dark, the border is %v, want %v", got, dark)
	}
	tt.SetDark(false)
	if got != light {
		t.Errorf("after it came back, the border is %v, want %v", got, light)
	}

	other := coreNewTester(view, 100, 100)
	other.SetDark(true)
	if got != dark {
		t.Errorf("a window opened later, dark, shows %v, want %v", got, dark)
	}

	// A Theme the view sets itself is its own: the call changes the defaults only.
	own := LightTheme()
	coreNewTester(func(c *context) {
		c.SetTheme(own)
		got = c.Theme().Border
	}, 100, 100)
	if got != own.Border {
		t.Errorf("a theme set by the view has the border %v, want its own %v", got, own.Border)
	}

	// A zero color leaves a theme's own, in each appearance.
	SetDefaultBorder(Color{}, dark)
	coreNewTester(view, 100, 100)
	if got != LightTheme().Border {
		t.Errorf("with no light color, the light border is %v, want the theme's %v", got, LightTheme().Border)
	}
}

// The rows a list keeps (RowVersion) were built with the theme the pass lent
// them: a change of the default border builds them again (themeGen), so that
// their lines show the new color rather than the one they were built with.
func TestSetDefaultBorderRebuildsKeptRows(t *testing.T) {
	s := &ListState{GridLines: true, Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
	cols := []TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 80}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, s, cols, 6, func(row, col int) { coreBox(c).Size(4, 4) }).Grow(1)
	}, 300, 300)
	tt.Frame()
	tt.Frame()
	if tt.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	// The line at the end of the first cell of the first row.
	line := func() Color {
		var cells []TestNode
		for _, n := range tt.Nodes() {
			if n.Role == RoleCell {
				cells = append(cells, n)
			}
		}
		sort.Slice(cells, func(i, j int) bool {
			if cells[i].Bounds.Y != cells[j].Bounds.Y {
				return cells[i].Bounds.Y < cells[j].Bounds.Y
			}
			return cells[i].Bounds.X < cells[j].Bounds.X
		})
		for _, n := range cells {
			if n.Bounds.Y > cells[0].Bounds.Y { // below the header
				c := tt.Image().RGBAAt(int(n.Bounds.X+n.Bounds.W)-1, int(n.Bounds.Y)+2)
				return Color{R: c.R, G: c.G, B: c.B}
			}
		}
		t.Fatal("no cell below the header")
		return Color{}
	}
	if got, want := line(), LightTheme().Border; got.R != want.R || got.G != want.G || got.B != want.B {
		t.Fatalf("before the call the line is %v, want the theme's %v", got, want)
	}
	red := Hex("#ff0000")
	SetDefaultBorder(red, red)
	t.Cleanup(func() { SetDefaultBorder(Color{}, Color{}) })
	before := tt.rt.rowsBuilt
	tt.Frame()
	if got := line(); got.R != red.R || got.G != red.G || got.B != red.B {
		t.Errorf("a kept row shows the line %v after the default border changed, want %v (rows built: %d more)", got, red, tt.rt.rowsBuilt-before)
	}
}
