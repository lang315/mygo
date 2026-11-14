package skin

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// look is one way a contrast case is drawn: Calm or no skin, light or
// dark, at a text scale.
type look struct {
	calm, dark bool
	scale      float32
}

func (l look) String() string {
	s := "none"
	if l.calm {
		s = "calm"
	}
	if l.dark {
		return fmt.Sprintf("%s-dark-%g", s, l.scale)
	}
	return fmt.Sprintf("%s-light-%g", s, l.scale)
}

// looks are every look a case runs in: no skin and Calm, light and dark,
// text at 1 and 1.1 (Windows' fonts are taller).
func looks() []look {
	var out []look
	for _, calm := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			for _, scale := range []float32{1, 1.1} {
				out = append(out, look{calm, dark, scale})
			}
		}
	}
	return out
}

// showIn draws w in the look l and returns the tester, with the theme the
// last frame was built with, checked to be the look's.
func showIn(t *testing.T, w widget.Control, width, height int, l look) (*widget.Tester, *ui.Theme) {
	t.Helper()
	t.Cleanup(func() { Use(nil) })
	th := new(ui.Theme)
	probe := widget.Custom(func(c *ui.Context) ui.Element {
		*th = *c.Theme()
		return ui.Text(c, "probe")
	})
	tt := widget.NewTester(t, widget.VBox(8).Add(w, probe), width, height)
	if l.calm {
		Use(Calm)
	} else {
		Use(nil)
	}
	tt.SetDark(l.dark)
	tt.SetPreferences(ui.Preferences{TextScale: l.scale})
	tt.Frame()
	accent := ui.Hex("#1c71d8")
	if l.dark {
		accent = ui.Hex("#78aeed")
	}
	if th.Dark != l.dark || (th.Accent == accent) != l.calm {
		t.Fatalf("%v: the theme is dark %v with accent %v", l, th.Dark, th.Accent)
	}
	return tt, th
}

// px is the pixel at x, y, opaque.
func px(tt *widget.Tester, x, y float32) ui.Color {
	c := tt.Image().RGBAAt(int(x), int(y))
	return ui.Color{R: c.R, G: c.G, B: c.B, A: 255}
}

// mid is the middle of r.
func mid(r ui.Rect) (x, y float32) { return r.X + r.W/2, r.Y + r.H/2 }

// strongest is the pixel of r farthest in contrast from ground: the ink of
// its text.
func strongest(tt *widget.Tester, r ui.Rect, ground ui.Color) (ui.Color, float64) {
	img := tt.Image()
	var best ui.Color
	most := 0.0
	for y := int(r.Y) + 1; y < int(r.Y+r.H)-1; y++ {
		for x := int(r.X) + 1; x < int(r.X+r.W)-1; x++ {
			c := img.RGBAAt(x, y)
			p := ui.Color{R: c.R, G: c.G, B: c.B, A: 255}
			if k := contrastOf(p, ground); k > most {
				best, most = p, k
			}
		}
	}
	return best, most
}

type contrastRow struct{ ID, Name string }

// A ListBox row selected and focused stands 1.25:1 from a row that is not,
// and its text 7:1 from it, light and dark. With no skin it keeps the
// plain theme's accent row.
func TestCalmSelectedRowShows(t *testing.T) {
	for _, l := range looks() {
		t.Run(l.String(), func(t *testing.T) {
			lb := widget.ListBox(func(r *contrastRow) string { return r.ID }, func(r *contrastRow) string { return r.Name }).
				SetItems([]contrastRow{{"a", "Alpha"}, {"b", "Beta"}, {"c", "Gamma"}})
			tt, th := showIn(t, lb, 400, 300, l)
			item := func(s string) *widget.Node { return tt.Find(widget.ByRole(ui.RoleListItem).Label(s)) }
			item("Beta").Click()
			tt.Frame()
			if !item("Beta").Focused() && !tt.Find(widget.ByRole(ui.RoleList)).Focused() {
				t.Fatalf("%v: the list does not have the focus", l)
			}
			sel, other := item("Beta").Bounds(), item("Gamma").Bounds()
			// Right of the short labels, in the middle of each row.
			fill := px(tt, sel.X+sel.W*3/4, sel.Y+sel.H/2)
			ground := px(tt, other.X+other.W*3/4, other.Y+other.H/2)
			_, text := strongest(tt, sel, fill)
			k := contrastOf(fill, ground)
			t.Logf("%v: selected %v, unselected %v: %.2f:1; text %.2f:1", l, fill, ground, k, text)
			if !l.calm {
				if !isColor(rgba(fill), th.Accent) {
					t.Errorf("%v: the selected row is %v, want the plain accent %v", l, fill, th.Accent)
				}
				return
			}
			if k < 1.25 {
				t.Errorf("%v: the selected row %v stands %.2f:1 from %v, want 1.25", l, fill, k, ground)
			}
			if text < 7 {
				t.Errorf("%v: its text stands %.2f:1 from it, want 7", l, text)
			}
		})
	}
}

func rgba(c ui.Color) color.RGBA { return color.RGBA{c.R, c.G, c.B, 255} }

// Selected text in an editor whose ground is the selected row's, as a cell
// edited in place, stands 1.2:1 from that ground, light and dark. The case
// is Calm's: with no skin the row is the accent, which no editor takes.
func TestCalmTextSelectionInSelectedRow(t *testing.T) {
	for _, l := range looks() {
		if !l.calm {
			continue
		}
		t.Run(l.String(), func(t *testing.T) {
			row := Calm.light.RowSelected
			if l.dark {
				row = Calm.dark.RowSelected
			}
			box := widget.TextBox().SetText("selected").Background(row)
			tt, _ := showIn(t, box, 400, 200, l)
			field := tt.Find(widget.ByRole(ui.RoleTextField))
			r := field.Bounds()
			ground := px(tt, r.X+r.W-6, r.Y+r.H/2)
			field.DoubleClick()
			tt.Frame()
			// The highlight: the commonest color inside the field but its
			// ground.
			counts := map[ui.Color]int{}
			for y := int(r.Y) + 4; y < int(r.Y+r.H)-4; y++ {
				for x := int(r.X) + 4; x < int(r.X+r.W)-4; x++ {
					if c := px(tt, float32(x), float32(y)); contrastOf(c, ground) > 1.01 {
						counts[c]++
					}
				}
			}
			var hl ui.Color
			for c, n := range counts {
				if n > counts[hl] {
					hl = c
				}
			}
			if counts[hl] < 10 {
				t.Fatalf("%v: no highlight in the field (ground %v, colors %v)", l, ground, counts)
			}
			k := contrastOf(hl, ground)
			t.Logf("%v: highlight %v on %v: %.2f:1", l, hl, ground, k)
			if k < 1.2 {
				t.Errorf("%v: the selected text's highlight %v stands %.2f:1 from the row %v, want 1.2", l, hl, k, ground)
			}
		})
	}
}

// The menu bar's File, hovered, stands 1.15:1 from the bar under Calm,
// light and dark; with no skin it stays as on main: 1.21 and 1.39.
func TestCalmMenuBarHoverShows(t *testing.T) {
	for _, l := range looks() {
		t.Run(l.String(), func(t *testing.T) {
			bar := widget.MenuBar(widget.MenuItem{Text: "File", Children: []widget.MenuItem{{Text: "Open"}}},
				widget.MenuItem{Text: "Edit", Children: []widget.MenuItem{{Text: "Undo"}}})
			tt, _ := showIn(t, bar, 400, 200, l)
			item := tt.Find(widget.ByRole(ui.RoleMenuItem).Label("File")).Bounds()
			// In the item's side padding, at mid-height.
			x, y := item.X+3, item.Y+item.H/2
			rest := px(tt, x, y)
			tt.Move(mid(item))
			tt.Frame()
			hover := px(tt, x, y)
			k := contrastOf(hover, rest)
			t.Logf("%v: hover %v on bar %v: %.2f:1", l, hover, rest, k)
			switch {
			case l.calm && k < 1.15:
				t.Errorf("%v: the hovered item %v stands %.2f:1 from the bar %v, want 1.15", l, hover, k, rest)
			case !l.calm && !l.dark && (k < 1.19 || k > 1.23):
				t.Errorf("%v: %.2f:1, want main's 1.21", l, k)
			case !l.calm && l.dark && (k < 1.37 || k > 1.41):
				t.Errorf("%v: %.2f:1, want main's 1.39", l, k)
			}
		})
	}
}

// A Sidebar's ground stands 1.04:1 from the content beside it in Calm's
// light; Calm's dark keeps #262a30, and no skin the plain Surface.
func TestCalmSidebarStandsOut(t *testing.T) {
	for _, l := range looks() {
		t.Run(l.String(), func(t *testing.T) {
			sb := widget.Sidebar().SetMenuWidth(160).
				Add("a", "Alpha", widget.Label("First page")).
				Add("b", "Beta", widget.Label("Second page"))
			tt, th := showIn(t, sb, 600, 400, l)
			tree := tt.Find(widget.ByRole(ui.RoleTree)).Bounds()
			// Low in each, below the items and the page's label.
			side := px(tt, tree.X+tree.W/2, tree.Y+tree.H*3/4)
			page := px(tt, tree.X+tree.W+(600-tree.X-tree.W)/2, tree.Y+tree.H*3/4)
			k := contrastOf(side, page)
			t.Logf("%v: sidebar %v, content %v: %.3f:1", l, side, page, k)
			switch {
			case !l.calm:
				if !isColor(rgba(side), th.Surface) {
					t.Errorf("%v: the sidebar is %v, want the plain Surface %v", l, side, th.Surface)
				}
			case l.dark:
				if !isColor(rgba(side), ui.Hex("#262a30")) {
					t.Errorf("%v: the sidebar is %v, want #262a30", l, side)
				}
			case k < 1.04:
				t.Errorf("%v: the sidebar %v stands %.3f:1 from the content %v, want 1.04", l, side, k, page)
			}
		})
	}
}

// A checked Toggle, and a selected TagSelect chip, stand 1.4:1 from an
// unchecked one under Calm, light and dark, and carry the accent in their
// border; with no skin they stay as on main: SurfacePressed on Surface.
func TestCalmCheckedToggleShows(t *testing.T) {
	for _, kind := range []string{"Toggle", "TagSelect"} {
		for _, l := range looks() {
			t.Run(kind+"/"+l.String(), func(t *testing.T) {
				var w widget.Control
				if kind == "Toggle" {
					w = widget.HBox(8).Add(widget.Toggle("On").SetActive(true), widget.Toggle("Off"))
				} else {
					w = widget.TagSelect(widget.Option{Value: "on", Text: "On"}, widget.Option{Value: "off", Text: "Off"}).SetValues("on")
				}
				tt, th := showIn(t, w, 400, 200, l)
				on, off := tt.Find(widget.ByLabel("On")).Bounds(), tt.Find(widget.ByLabel("Off")).Bounds()
				// The padding left of the text, at mid-height.
				fillOn, fillOff := px(tt, on.X+5, on.Y+on.H/2), px(tt, off.X+5, off.Y+off.H/2)
				edge := px(tt, on.X+0.5, on.Y+on.H/2)
				k := contrastOf(fillOn, fillOff)
				t.Logf("%v: on %v, off %v: %.2f:1; edge %v", l, fillOn, fillOff, k, edge)
				if !l.calm {
					if !isColor(rgba(fillOn), th.SurfacePressed) || !isColor(rgba(fillOff), th.Surface) {
						t.Errorf("%v: on %v and off %v, want main's %v and %v", l, fillOn, fillOff, th.SurfacePressed, th.Surface)
					}
					return
				}
				if k < 1.4 {
					t.Errorf("%v: the checked fill %v stands %.2f:1 from the unchecked %v, want 1.4", l, fillOn, k, fillOff)
				}
				if !isColor(rgba(edge), th.Accent) {
					t.Errorf("%v: the checked border is %v, want the accent %v", l, edge, th.Accent)
				}
			})
		}
	}
}

// An own Background on a checked Toggle wins over Calm's checked look.
func TestCalmCheckedToggleKeepsOwnStyle(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	for _, dark := range []bool{false, true} {
		tt, _ := showIn(t, widget.Toggle("Mine").SetActive(true).Background(red), 400, 200, look{calm: true, dark: dark, scale: 1})
		b := tt.Find(widget.ByLabel("Mine")).Bounds()
		if c := px(tt, b.X+5, b.Y+b.H/2); !isColor(rgba(c), red) {
			t.Errorf("dark %v: the checked toggle with its own background is %v, want red", dark, c)
		}
	}
}
