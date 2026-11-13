package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// TestMain runs the tests on the fake backend, as package widget's do: the
// appearance (widget.SetTheme) needs a running app.
func TestMain(m *testing.M) {
	platform.TestBackend = fake.New()
	mygo.App.SetName("SkinStudioTest")
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		go func() { os.Exit(m.Run()) }()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pages are the studio's tabs, in order.
var pages = []string{
	"Buttons and toggles", "Text and choices", "Numbers and dates", "Grid",
	"Lists and trees", "Containers", "Chart and progress", "Dialogs and messages",
}

// reset puts the skin and the appearance back when the test ends: they are
// the process's.
func reset(t *testing.T) {
	t.Cleanup(func() {
		skin.Use(nil)
		widget.SetTheme(widget.ThemeSystem)
	})
}

// same reports whether a and b are the same size, to rounding.
func same(a, b float32) bool { return a-b < 0.01 && b-a < 0.01 }

// toolStrip holds the three combo boxes of the tool strip, by their names.
var toolStrip = []string{"Skin", "Palette", "Mode"}

// heights returns the heights of the tool strip's combo boxes, by name.
func heights(t *testing.T, tt *widget.Tester) map[string]float32 {
	t.Helper()
	got := map[string]float32{}
	for _, name := range toolStrip {
		got[name] = tt.Find(widget.ByLabel(name)).Bounds().H
	}
	return got
}

// pick opens the combo box named combo and chooses item: the last element
// of that name, which is the one in the popup.
func pick(tt *widget.Tester, combo, item string) {
	tt.Find(widget.ByLabel(combo)).Click()
	all := tt.FindAll(widget.ByLabel(item))
	all[len(all)-1].Click()
}

// named are the roles whose label is the text they show: a text cut by one of
// these is a clipped caption.
var named = map[ui.Role]bool{
	ui.RoleButton: true, ui.RoleLink: true, ui.RoleCheckBox: true, ui.RoleRadio: true, ui.RoleTab: true,
	ui.RoleMenuButton: true, ui.RoleToggleButton: true, ui.RoleColumnHeader: true,
}

// clipped returns the captions of the last frame that stick out of the
// control they name: buttons, tabs, check boxes and the like, whose label is
// their text. A control scrolled off the window is left alone.
func clipped(tt *widget.Tester, w, h float32) []string {
	var out []string
	for _, n := range tt.Nodes() {
		b := n.Bounds
		if !named[n.Role] || n.Label == "" || b.W <= 0 || b.H <= 0 || b.X+b.W > w || b.Y+b.H > h {
			continue
		}
		r, ok := tt.Tester.Find(n.Label) // the text's box; widget's Find takes a Query
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if !ok || cx < b.X || cx > b.X+b.W || cy < b.Y || cy > b.Y+b.H {
			continue // another element shows this text first
		}
		if r.X < b.X-0.5 || r.Y < b.Y-0.5 || r.X+r.W > b.X+b.W+0.5 || r.Y+r.H > b.Y+b.H+0.5 {
			out = append(out, fmt.Sprintf("%q at %v is outside its %v at %v", n.Label, r, n.Role, b))
		}
	}
	return out
}

// With Calm, the grid page has rows and a header of 30 and the tool strip's
// controls are 36 (spec P1 gate), or the platform's plain height where a
// taller line makes that the larger.
func TestStudioGridPage(t *testing.T) {
	reset(t)
	// What the controls are without a skin, on this platform: the plain
	// formula, line height and padding.
	plain := heights(t, widget.TestForm(t, newStudio()))

	skin.Use(skin.Calm)
	tt := widget.TestForm(t, newStudio())
	tt.Find(widget.ByLabel("Grid")).Click()
	tt.Frame()
	for name, h := range heights(t, tt) {
		if want := max(36, plain[name]); !same(h, want) {
			t.Errorf("the %s combo box is %v high, want max(36, plain %v) = %v", name, h, plain[name], want)
		}
	}
	rows := 0
	var lastEnd float32 // where the grid ends: the row there may be cut by it
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTable {
			lastEnd = n.Bounds.Y + n.Bounds.H
		}
	}
	if lastEnd == 0 {
		t.Fatalf("the grid page has no table")
	}
	if lastEnd > formHeight {
		t.Errorf("the grid ends at %v, below the window's %d: the page does not fit", lastEnd, formHeight)
	}
	for _, n := range tt.Nodes() {
		switch n.Role {
		case ui.RoleRow:
			if n.Bounds.H > 0 {
				rows++
				// The row at the grid's bottom edge may be cut by it: its visible box is short.
				if !same(n.Bounds.H, 30) && n.Bounds.Y+n.Bounds.H < lastEnd-0.01 {
					t.Errorf("grid row %q is %v high, want 30 (at %v, the lowest row ends at %v)", n.Label, n.Bounds.H, n.Bounds, lastEnd)
				}
			}
		case ui.RoleColumnHeader:
			if !same(n.Bounds.H, 30) {
				t.Errorf("column header %q is %v high, want 30", n.Label, n.Bounds.H)
			}
		}
	}
	if rows < 8 {
		t.Errorf("the grid page shows %d rows, want at least 8 of its 40", rows)
	}
	if !tt.Has(widget.ByRole(ui.RoleColumnHeader).Label("Status")) {
		t.Errorf("the grid has no Status column: %q", tt.Texts())
	}
}

// Every page builds and none has a text cut by its box, with no skin and with
// Calm in each palette, light and dark.
func TestStudioEveryPage(t *testing.T) {
	reset(t)
	type config struct {
		name string
		s    *skin.Skin
		dark bool
	}
	var configs []config
	for _, dark := range []bool{false, true} {
		mode := map[bool]string{false: "light", true: "dark"}[dark]
		configs = append(configs, config{"None " + mode, nil, dark})
		for _, p := range []*skin.Palette{skin.Cham, skin.Indigo, skin.Ink, skin.Teal} {
			configs = append(configs, config{"Calm " + p.Name() + " " + mode, skin.Calm.WithPalette(p), dark})
		}
	}
	for _, c := range configs {
		t.Run(c.name, func(t *testing.T) {
			reset(t)
			skin.Use(c.s)
			f := newStudio()
			tt := widget.TestForm(t, f)
			tt.SetDark(c.dark)
			tt.Frame()
			w, h := float32(formWidth), float32(formHeight)
			for _, page := range pages {
				tt.Find(widget.ByLabel(page)).Click()
				tt.Frame()
				for _, s := range clipped(tt, w, h) {
					t.Errorf("%s: %s", page, s)
				}
			}
		})
	}
}

// The Buttons page has the long Vietnamese labels, whole in their buttons.
func TestStudioLongLabels(t *testing.T) {
	reset(t)
	for _, s := range []*skin.Skin{nil, skin.Calm} {
		skin.Use(s)
		tt := widget.TestForm(t, newStudio())
		for _, label := range []string{"Đăng nhập lại tài khoản đã khoá", "Xuất danh sách ra Excel"} {
			b := tt.Find(widget.ByLabel(label)).Bounds()
			r, ok := tt.Tester.Find(label)
			if !ok || r.X < b.X-0.5 || r.X+r.W > b.X+b.W+0.5 || r.Y < b.Y-0.5 || r.Y+r.H > b.Y+b.H+0.5 {
				t.Errorf("skin %v: %q is %v in its button %v", s != nil, label, r, b)
			}
		}
	}
}

// Choosing an item in each tool strip combo box changes the skin, the palette
// or the appearance at once.
func TestStudioSwitches(t *testing.T) {
	reset(t)
	tt := widget.TestForm(t, newStudio())
	if skin.Active() != nil {
		t.Fatalf("the studio starts with the skin %v, want none", skin.Active().Name())
	}

	pick(tt, "Skin", "Calm")
	if s := skin.Active(); s == nil || s.Name() != "Calm" || s.Palette() != skin.Cham {
		t.Fatalf("after Skin > Calm the skin is %v, want Calm with Chàm", s)
	}
	for _, p := range []*skin.Palette{skin.Indigo, skin.Ink, skin.Teal, skin.Cham} {
		pick(tt, "Palette", p.Name())
		if s := skin.Active(); s == nil || s.Palette() != p {
			t.Errorf("after Palette > %s the skin's palette is %v", p.Name(), s)
		}
	}
	pick(tt, "Palette", "Teal")
	pick(tt, "Skin", "None")
	if skin.Active() != nil {
		t.Errorf("after Skin > None the skin is %v, want none", skin.Active().Name())
	}
	pick(tt, "Skin", "Calm")
	if s := skin.Active(); s == nil || s.Palette() != skin.Teal {
		t.Errorf("Calm after the palette Teal has the palette %v, want Teal kept", s)
	}

	for _, c := range []struct {
		item string
		want mygo.ThemeSource
	}{{"Dark", mygo.ThemeDark}, {"Light", mygo.ThemeLight}, {"System", mygo.ThemeSystem}} {
		pick(tt, "Mode", c.item)
		if got := mygo.Theme.Source(); got != c.want {
			t.Errorf("after Mode > %s the appearance is %q, want %q", c.item, got, c.want)
		}
	}
}

// Every tab is there, and the combo boxes show what is on.
func TestStudioStartsPlain(t *testing.T) {
	reset(t)
	tt := widget.TestForm(t, newStudio())
	for _, page := range pages {
		if !tt.Has(widget.ByLabel(page)) {
			t.Errorf("no tab %q: %q", page, tt.Texts())
		}
	}
	for _, v := range []string{"None", "Chàm", "System"} {
		if !tt.HasText(v) {
			t.Errorf("the tool strip does not show %q: %q", v, tt.Texts())
		}
	}
}
