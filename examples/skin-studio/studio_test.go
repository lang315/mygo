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

// describe names a skin and its palette, "none" for nil.
func describe(s *skin.Skin) string {
	if s == nil {
		return "none"
	}
	return s.Name() + " with " + s.Palette().Name()
}

// pick opens the combo box named combo and chooses item: the last element
// of that name, which is the one in the popup.
func pick(tt *widget.Tester, combo, item string) {
	tt.Find(widget.ByLabel(combo)).Click()
	all := tt.FindAll(widget.ByLabel(item))
	if len(all) == 0 {
		tt.Find(widget.ByLabel(item)) // fails the test, naming what the popup offers
	}
	all[len(all)-1].Click()
}

// captions are the captions the studio draws in controls of their own (an
// accessible name is not always the text drawn, as a month's days or a
// stepper's arrows): buttons, tabs and toggles.
var captions = map[string]bool{
	"Header action": true, "Plain": true, "Suggested": true, "Destructive": true, "Disabled": true, "Drop down": true,
	"Menu": true, "Link": true, "Đăng nhập lại tài khoản đã khoá": true, "Xuất danh sách ra Excel": true,
	"Toggle on": true, "Toggle off": true, "Info": true, "Warning": true, "Error": true, "Question": true,
	"Input text": true, "Input number": true, "Input choice": true, "Dialog": true,
	"Chip 1": true, "Chip 2": true, "Chip 3": true, "Chip 4": true, "Chip 5": true, "Chip 6": true,
	"Buttons and toggles": true, "Text and choices": true, "Numbers and dates": true, "Grid": true,
	"Lists and trees": true, "Containers": true, "Chart and progress": true, "Dialogs and messages": true,
	"First": true, "Second": true,
}

// clipped returns the captions of the last frame that are cut by the control
// they are in: sticking out of it, or narrower than the text takes. A control
// scrolled off the window is left alone.
func clipped(t *testing.T, tt *widget.Tester, w, h float32) []string {
	t.Helper()
	var out []string
	for _, n := range tt.Nodes() {
		b := n.Bounds
		if !captions[n.Label] || b.W <= 0 || b.H <= 0 || b.X+b.W > w || b.Y+b.H > h {
			continue
		}
		r, ok := tt.Tester.Find(n.Label) // the text's box; widget's Find takes a Query
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		if !ok || cx < b.X || cx > b.X+b.W || cy < b.Y || cy > b.Y+b.H {
			continue // another element shows this text first
		}
		if r.X < b.X-0.5 || r.Y < b.Y-0.5 || r.X+r.W > b.X+b.W+0.5 || r.Y+r.H > b.Y+b.H+0.5 {
			out = append(out, fmt.Sprintf("%q at %v is outside its %v at %v", n.Label, r, n.Role, b))
		} else if nat := natural(t, n.Label); r.W < nat-0.5 {
			out = append(out, fmt.Sprintf("%q is cut to %v wide of %v in its %v at %v", n.Label, r.W, nat, n.Role, b))
		}
	}
	return out
}

// natural returns the width s takes in a label with room to spare: a caption
// narrower than that is cut. Control text is as wide or wider, never narrower.
func natural(t *testing.T, s string) float32 {
	t.Helper()
	tt := widget.NewTester(t, widget.HBox(0).Add(widget.Label(s)), 1500, 100)
	r, ok := tt.Tester.Find(s)
	if !ok {
		t.Fatalf("a label %q shows no text", s)
	}
	return r.W
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
				// It is cut at the inner edge of the grid's frame, a border of 1 DIP inside its box.
				const frame = 1
				if !same(n.Bounds.H, 30) && n.Bounds.Y+n.Bounds.H < lastEnd-frame-0.01 {
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
		for _, p := range []*skin.Palette{skin.Cham, skin.Indigo, skin.Ink, skin.Teal, skin.Blue} {
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
				for _, s := range clipped(t, tt, w, h) {
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
			if !ok || r.W < natural(t, label)-0.5 || r.X < b.X-0.5 || r.X+r.W > b.X+b.W+0.5 || r.Y < b.Y-0.5 || r.Y+r.H > b.Y+b.H+0.5 {
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
	if s := skin.Active(); s == nil || s.Name() != "Calm" || s.Palette() != skin.Blue {
		t.Fatalf("after Skin > Calm the skin is %s, want Calm with Blue", describe(s))
	}
	for _, p := range []*skin.Palette{skin.Cham, skin.Indigo, skin.Ink, skin.Teal, skin.Blue} {
		pick(tt, "Palette", p.Name())
		if s := skin.Active(); s == nil || s.Palette() != p {
			t.Errorf("after Palette > %s the skin is %s", p.Name(), describe(s))
		}
	}
	pick(tt, "Palette", "Teal")
	pick(tt, "Skin", "None")
	if skin.Active() != nil {
		t.Errorf("after Skin > None the skin is %v, want none", skin.Active().Name())
	}
	pick(tt, "Skin", "Calm")
	if s := skin.Active(); s == nil || s.Palette() != skin.Teal {
		t.Errorf("Calm after the palette Teal is %s, want Teal kept", describe(s))
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

// The chart page: nothing a chart draws (its plot, tick labels or legend) goes
// below the top of the first progress bar, at scale 1 and 2. A chart's box is
// not a node: its drawn parts are the nodes whose top is above the bar.
func TestStudioChartsClearProgress(t *testing.T) {
	reset(t)
	for _, scale := range []float32{1, 2} {
		tt := widget.TestForm(t, newStudio())
		tt.SetSize(1280, 800)
		tt.SetScale(scale)
		tt.Find(widget.ByLabel("Chart and progress")).Click()
		tt.Frame()
		bar := tt.Find(widget.ByRole(ui.RoleProgress).Nth(0)).Bounds()
		line := tt.Find(widget.ByLabel("Line")).Bounds()
		seen := 0
		for _, n := range tt.Nodes() {
			b := n.Bounds
			if n.Label == "" || b.Y < line.Y-10 || b.Y >= bar.Y || n.Role == ui.RoleProgress || b.W <= 0 || b.H <= 0 {
				continue
			}
			seen++
			if b.Y+b.H > bar.Y {
				t.Errorf("scale %v: %v %q at %v reaches below the top of the progress bar, %v", scale, n.Role, n.Label, b, bar.Y)
			}
		}
		if seen < 20 {
			t.Errorf("scale %v: only %d nodes above the progress bar, the charts' labels are missing: %q", scale, seen, tt.Texts())
		}
	}
}

// The containers page: the sidebar's page starts Space(2) = 8 DIPs right of
// the sidebar's divider, and the nested tabs are not squeezed to the right.
func TestStudioContainersSpacing(t *testing.T) {
	reset(t)
	tt := widget.TestForm(t, newStudio())
	tt.SetSize(1280, 800)
	tt.Find(widget.ByLabel("Containers")).Click()
	tt.Frame()
	edge := float32(-1)
	for _, d := range tt.FindAll(widget.ByLabel("Divider")) {
		if b := d.Bounds(); b.Y > 400 { // the sidebar's, below the menu bar
			edge = b.X
		}
	}
	if edge < 0 {
		t.Fatalf("no sidebar divider: %q", tt.Texts())
	}
	page, ok := tt.Tester.Find("Mail page")
	if !ok {
		t.Fatalf("no text Mail page: %q", tt.Texts())
	}
	if page.X < edge+8 {
		t.Errorf("the sidebar's page starts at %v, want 8 or more right of the sidebar's edge %v", page.X, edge)
	}
	if first := tt.Find(widget.ByLabel("First")).Bounds(); first.X > 900 {
		t.Errorf("the nested tabs start at %v, pushed to the far right", first.X)
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
	for _, v := range []string{"None", "Blue", "System"} {
		if !tt.HasText(v) {
			t.Errorf("the tool strip does not show %q: %q", v, tt.Texts())
		}
	}
}
