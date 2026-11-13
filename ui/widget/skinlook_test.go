package widget

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// skinTestTokens are tokens with Calm's sizes and colors far from the plain
// themes' and from each other, which near tells apart: Calm's own chrome
// and hover are 1 apart, and its card shadow 4 % black.
func skinTestTokens() *skinhook.Tokens {
	return &skinhook.Tokens{
		Chrome:       ui.Hex("#ffd54f"),
		Content:      ui.Hex("#ffffff"),
		Control:      ui.Hex("#ffffff"),
		Line:         ui.Hex("#e91e63"),
		LineStrong:   ui.Hex("#3f51b5"),
		Text:         ui.Hex("#202020"),
		ControlHover: ui.Hex("#4dd0e1"),
		Elevation:    [4]skinhook.Shadow{{}, {Y: 6, Blur: 2, Color: ui.RGBA(0, 0, 0, 0.6)}},

		ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, TabStripHeight: 40,
		MenuBarStripHeight: 34, MenuBarItemHeight: 28, RadiusControl: 6, RadiusCard: 10,
		ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
		Icon: 16, ToolbarIcon: 17, CheckBox: 15, SwitchWidth: 40, SwitchHeight: 22,
	}
}

// useSkin turns on the test tokens, in the hook and in the default themes as
// skin.Use does, until the test ends.
func useSkin(t *testing.T) *skinhook.Tokens {
	t.Helper()
	tk := skinTestTokens()
	th := ui.LightTheme()
	th.Background, th.Surface, th.Border, th.Text = tk.Content, tk.Control, tk.Line, tk.Text
	th.Metrics = ui.Metrics{ControlHeight: tk.ControlHeight, RowHeight: tk.RowHeight, HeaderHeight: tk.HeaderHeight,
		TabHeight: tk.TabStripHeight, CheckSize: tk.CheckBox, SwitchWidth: tk.SwitchWidth, SwitchHeight: tk.SwitchHeight}
	skinhook.Set(tk, tk)
	ui.SetDefaultThemes(th, th)
	t.Cleanup(offSkin)
	return tk
}

// offSkin turns the test tokens off.
func offSkin() {
	skinhook.Set(nil, nil)
	ui.SetDefaultThemes(nil, nil)
}

// space is the plain theme's Space(n).
func space(n float32) float32 { return ui.LightTheme().Space(n) }

func sameSize(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }

func skinPx(tt *ui.Tester, x, y float32) color.RGBA { return tt.Image().RGBAAt(int(x), int(y)) }

// lineOf is the height of a line of text at the text size scale, on this
// platform: a label's.
func lineOf(t *testing.T, scale float32) float32 {
	t.Helper()
	tt := tester(t, Label("Xy"))
	tt.SetPreferences(ui.Preferences{TextScale: scale})
	tt.Frame()
	return find(t, tt, "Xy").H
}

// darkRows returns the first and the last pixel row of box holding a dark
// pixel: the ink of its text.
func darkRows(tt *ui.Tester, box ui.Rect) (first, last int) {
	first, last = -1, -1
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if c := tt.Image().RGBAAt(x, y); int(c.R)+int(c.G)+int(c.B) < 300 {
				if first < 0 {
					first = y
				}
				last = y
				break
			}
		}
	}
	return first, last
}

// labeled returns the node labeled label, whatever its role.
func labeled(t *testing.T, tt *ui.Tester, label string) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Label == label {
			return n
		}
	}
	t.Fatalf("no node labeled %q", label)
	return ui.TestNode{}
}

// skinMenuBar is a menu bar of two menus.
func skinMenuBar() *MenuBarWidget {
	return MenuBar(MenuItem{Text: "File", Children: []MenuItem{{Text: "Open"}}}, MenuItem{Text: "Edit", Children: []MenuItem{{Text: "Undo"}}})
}

// The menu bar: with a skin, the strip at least 34 on Chrome, its items at
// least 28 with their text centered, hovered in ControlHover; without, as
// before skins.
func TestSkinMenuBar(t *testing.T) {
	tt := tester(t, skinMenuBar())
	line := lineOf(t, 1)
	check := func(on bool, tk *skinhook.Tokens) {
		t.Helper()
		strip, item := nodeOf(t, tt, ui.RoleMenuBar, "").Bounds, nodeOf(t, tt, ui.RoleMenuItem, "File").Bounds
		wantItem, wantStrip, fill := line+2*space(1), line+2*space(1)+2*space(0.5), ui.LightTheme().Background
		if on {
			wantItem = max(28, line)
			wantStrip = max(34, wantItem+2*space(0.5))
			fill = tk.Chrome
		}
		if !sameSize(item.H, wantItem) || !sameSize(strip.H, wantStrip) {
			t.Errorf("skin %v: item %v and strip %v high, want %v and %v (line %v)", on, item.H, strip.H, wantItem, wantStrip, line)
		}
		// The text's ink is in the item's middle.
		first, last := darkRows(tt, item)
		if d := float32(first+last+1)/2 - (item.Y + item.H/2); d > 1.5 || d < -1.5 {
			t.Errorf("skin %v: the text's ink (rows %d to %d) is %v off the item's middle", on, first, last, d)
		}
		if c := skinPx(tt, strip.X+strip.W-4, strip.Y+strip.H/2); !near(c, fill) {
			t.Errorf("skin %v: the strip is %v past its items, want %v", on, c, fill)
		}
		hover := ui.LightTheme().SurfaceHover
		if on {
			hover = tk.ControlHover
		}
		tt.Move(item.X+item.W/2, item.Y+item.H/2)
		tt.Frame()
		if c := skinPx(tt, item.X+3, item.Y+item.H/2); !near(c, hover) {
			t.Errorf("skin %v: the hovered item is %v, want %v", on, c, hover)
		}
		tt.Move(strip.X+strip.W-2, strip.Y+strip.H+100)
		tt.Frame()
	}
	check(false, nil)
	tk := useSkin(t)
	tt.Frame()
	check(true, tk)
	offSkin()
	tt.Frame()
	check(false, nil)
}

// The menu bar's strip and items grow with the text size.
func TestSkinMenuBarFollowsTextScale(t *testing.T) {
	useSkin(t)
	tt := tester(t, skinMenuBar())
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	strip, item := nodeOf(t, tt, ui.RoleMenuBar, "").Bounds, nodeOf(t, tt, ui.RoleMenuItem, "File").Bounds
	if want := max(42, lineOf(t, 1.5)); !sameSize(item.H, want) {
		t.Errorf("at 150 %%, the item is %v high, want %v", item.H, want)
	}
	if strip.H < 51-0.01 || strip.H < item.H {
		t.Errorf("at 150 %%, the strip is %v high around an item of %v, want at least 51", strip.H, item.H)
	}
}

// skinIcon is a square icon whose ink fills its box.
var skinIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="currentColor"/></svg>`))

// skinToolForm is a form with a tool strip of three buttons, a separator
// between the second and the third, and an icon button.
func skinToolForm() *Form {
	return NewForm("Shell", 800, 600).SetContent(Label("content")).ToolStrip(ToolStrip(
		ToolButton("Cut"), ToolButton("Copy"), Separator(), ToolButton("Paste"),
		ToolButton("").SetIcon(skinIcon).SetTooltip("Icon")))
}

// inkHeight is how many pixel rows of box hold ink of color c.
func inkHeight(tt *ui.Tester, box ui.Rect, c ui.Color) int {
	n := 0
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if near(tt.Image().RGBAAt(x, y), c) {
				n++
				break
			}
		}
	}
	return n
}

// The tool strip: with a skin, Chrome, padded 10 above and 12 below, 10
// between controls and 22 across a separator with its line in the middle,
// icons of 17 and a line of LineStrong under it; without, as before skins.
func TestSkinToolStrip(t *testing.T) {
	tt := formTester(t, skinToolForm())
	check := func(on bool, tk *skinhook.Tokens) {
		t.Helper()
		bar := nodeOf(t, tt, ui.RoleToolbar, "").Bounds
		cut, cp, paste := nodeOf(t, tt, ui.RoleButton, "Cut").Bounds, nodeOf(t, tt, ui.RoleButton, "Copy").Bounds, nodeOf(t, tt, ui.RoleButton, "Paste").Bounds
		top, bottom, gap, across := space(1), space(1), space(1), 2*space(1)+1
		fill := ui.LightTheme().Background
		if on {
			top, bottom, gap, across, fill = 10, 12+1, 10, 22, tk.Chrome
		}
		if got := cut.Y - bar.Y; !sameSize(got, top) {
			t.Errorf("skin %v: %v above the controls, want %v", on, got, top)
		}
		if got := bar.Y + bar.H - (cut.Y + cut.H); !sameSize(got, bottom) {
			t.Errorf("skin %v: %v below the controls, want %v", on, got, bottom)
		}
		if got := cp.X - (cut.X + cut.W); !sameSize(got, gap) {
			t.Errorf("skin %v: %v between two controls, want %v", on, got, gap)
		}
		if got := paste.X - (cp.X + cp.W); !sameSize(got, across) {
			t.Errorf("skin %v: %v across the separator, want %v", on, got, across)
		}
		// The separator's line, in the theme's Border, in the middle.
		line, y := ui.LightTheme().Border, cut.Y+cut.H/2
		if on {
			line = tk.Line
		}
		lx := float32(-1)
		for x := int(cp.X + cp.W); x < int(paste.X); x++ {
			if near(tt.Image().RGBAAt(x, int(y)), line) {
				lx = float32(x) + 0.5
			}
		}
		if mid := (cp.X + cp.W + paste.X) / 2; lx < 0 || lx-mid > 1 || mid-lx > 1 {
			t.Errorf("skin %v: the separator's line is at %v, want it at %v", on, lx, mid)
		}
		if c := skinPx(tt, bar.X+bar.W-4, bar.Y+2); !near(c, fill) {
			t.Errorf("skin %v: the strip is %v, want %v", on, c, fill)
		}
		under := skinPx(tt, bar.X+bar.W-4, bar.Y+bar.H-0.5)
		if on && !near(under, tk.LineStrong) {
			t.Errorf("the strip's bottom line is %v, want LineStrong %v", under, tk.LineStrong)
		}
		if !on && near(under, ui.LightTheme().Border) {
			t.Errorf("without a skin, the strip has a bottom line %v", under)
		}
		icon := nodeOf(t, tt, ui.RoleButton, "Icon").Bounds
		if on {
			if h := inkHeight(tt, icon, tk.Text); h != 17 {
				t.Errorf("the icon is %v high, want 17", h)
			}
		}
	}
	check(false, nil)
	plainIcon := inkHeight(tt, nodeOf(t, tt, ui.RoleButton, "Icon").Bounds, ui.LightTheme().Text)
	tk := useSkin(t)
	tt.Frame()
	check(true, tk)
	offSkin()
	tt.Frame()
	check(false, nil)
	if h := inkHeight(tt, nodeOf(t, tt, ui.RoleButton, "Icon").Bounds, ui.LightTheme().Text); h != plainIcon {
		t.Errorf("after the skin, the icon is %v high, want its plain %v", h, plainIcon)
	}
}

// The tool strip's sizes grow with the text size.
func TestSkinToolStripFollowsTextScale(t *testing.T) {
	useSkin(t)
	tt := formTester(t, skinToolForm())
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	bar := nodeOf(t, tt, ui.RoleToolbar, "").Bounds
	cut, cp, paste := nodeOf(t, tt, ui.RoleButton, "Cut").Bounds, nodeOf(t, tt, ui.RoleButton, "Copy").Bounds, nodeOf(t, tt, ui.RoleButton, "Paste").Bounds
	if got := cut.Y - bar.Y; !sameSize(got, 15) {
		t.Errorf("at 150 %%, %v above the controls, want 15", got)
	}
	if got := cp.X - (cut.X + cut.W); !sameSize(got, 15) {
		t.Errorf("at 150 %%, %v between two controls, want 15", got)
	}
	if got := paste.X - (cp.X + cp.W); !sameSize(got, 33) {
		t.Errorf("at 150 %%, %v across the separator, want 33", got)
	}
}

// skinStatusForm is a form with a status strip, and a view that keeps the
// theme in *th.
func skinStatusForm(th *ui.Theme) *Form {
	return NewForm("Shell", 800, 600).SetContent(Custom(func(c *ui.Context) ui.Element {
		*th = *c.Theme()
		return ui.Text(c, "content")
	})).StatusStrip(StatusStrip(StatusLabel("Ready")))
}

// statusLine returns the color of the status strip's top line, the last row
// above its text that is not the strip's fill, scanning up from the text.
func statusLine(t *testing.T, tt *ui.Tester, fill ui.Color) (color.RGBA, bool) {
	t.Helper()
	r := find(t, tt, "Ready")
	for y := int(r.Y) - 1; y > int(r.Y)-16; y-- {
		if c := tt.Image().RGBAAt(700, y); !near(c, fill) {
			return c, true
		}
	}
	return color.RGBA{}, false
}

// The status strip: with a skin, Chrome under its line of the theme's
// Border; without, as before skins.
func TestSkinStatusStrip(t *testing.T) {
	var th ui.Theme
	tt := formTester(t, skinStatusForm(&th))
	check := func(on bool, tk *skinhook.Tokens) {
		t.Helper()
		r := find(t, tt, "Ready")
		fill := ui.LightTheme().Background
		if on {
			fill = tk.Chrome
		}
		if c := skinPx(tt, 700, r.Y+r.H/2); !near(c, fill) {
			t.Errorf("skin %v: the strip is %v, want %v", on, c, fill)
		}
		if c, ok := statusLine(t, tt, fill); !ok || !near(c, th.Border) {
			t.Errorf("skin %v: the line above the strip is %v (found %v), want the theme's Border %v", on, c, ok, th.Border)
		}
	}
	check(false, nil)
	tk := useSkin(t)
	tt.Frame()
	check(true, tk)
	offSkin()
	tt.Frame()
	check(false, nil)
}

// Under High Contrast, the status strip's line is the theme's strengthened
// Border, and the tool strip's bottom LineStrong mixed with Text.
func TestSkinLinesUnderHighContrast(t *testing.T) {
	tk := useSkin(t)
	var th ui.Theme
	f := skinStatusForm(&th).ToolStrip(ToolStrip(ToolButton("Cut")))
	tt := formTester(t, f)
	tt.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	tt.Frame()
	if want := tk.Line.Mix(tk.Text, 0.45); th.Border != want {
		t.Fatalf("under High Contrast the theme's Border is %v, want %v", th.Border, want)
	}
	if c, ok := statusLine(t, tt, tk.Chrome); !ok || !near(c, th.Border) {
		t.Errorf("the status strip's line is %v (found %v), want the strengthened Border %v", c, ok, th.Border)
	}
	bar := nodeOf(t, tt, ui.RoleToolbar, "").Bounds
	if c, want := skinPx(tt, bar.X+bar.W-4, bar.Y+bar.H-0.5), tk.LineStrong.Mix(tk.Text, 0.45); !near(c, want) {
		t.Errorf("the tool strip's bottom line is %v, want %v", c, want)
	}
}

// The header bar: with a skin, on Chrome, its height unchanged; without, as
// before skins.
func TestSkinHeaderBar(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content"))
	f.UseHeaderBar("")
	tt := formTester(t, f)
	plain := find(t, tt, "content").Y
	check := func(on bool, tk *skinhook.Tokens) {
		t.Helper()
		title := find(t, tt, "Shell")
		fill := ui.LightTheme().Background
		if on {
			fill = tk.Chrome
		}
		if c := skinPx(tt, 8, title.Y+title.H/2); !near(c, fill) {
			t.Errorf("skin %v: the header bar is %v, want %v", on, c, fill)
		}
		if y := find(t, tt, "content").Y; !sameSize(y, plain) {
			t.Errorf("skin %v: the content starts at %v, want %v: the bar's height changed", on, y, plain)
		}
	}
	check(false, nil)
	if plain < 32 {
		t.Fatalf("the content starts at %v, inside the header bar's least height of 32", plain)
	}
	tk := useSkin(t)
	tt.Frame()
	check(true, tk)
	offSkin()
	tt.Frame()
	check(false, nil)
}

// hasColor reports whether a pixel of color c is in the column x between y0
// and y1.
func hasColor(tt *ui.Tester, x float32, y0, y1 float32, c ui.Color) bool {
	for y := int(y0); y <= int(y1); y++ {
		if near(tt.Image().RGBAAt(int(x), y), c) {
			return true
		}
	}
	return false
}

// The tab strip: with a skin and no StripStyle, Chrome across the control
// and a line under it; an app's StripStyle replaces both; without a skin,
// as before skins.
func TestSkinTabStrip(t *testing.T) {
	nb := TabControl().AddPage("A", Label("page A")).AddPage("B", Label("page B"))
	tt := tester(t, nb)
	check := func(on bool, tk *skinhook.Tokens, fill ui.Color, line bool) {
		t.Helper()
		b, p := find(t, tt, "B"), find(t, tt, "page A")
		x := b.X + b.W + 40
		if c := skinPx(tt, x, b.Y+b.H/2); !near(c, fill) {
			t.Errorf("skin %v: the strip past the last tab is %v, want %v", on, c, fill)
		}
		border := ui.LightTheme().Border
		if tk != nil {
			border = tk.Line
		}
		if got := hasColor(tt, x, b.Y+b.H/2, p.Y-1, border); got != line {
			t.Errorf("skin %v: a line under the strip is %v, want %v", on, got, line)
		}
	}
	plain := ui.LightTheme().Background
	check(false, nil, plain, false)
	tk := useSkin(t)
	tt.Frame()
	check(true, tk, tk.Chrome, true)
	red := ui.RGB(200, 0, 0)
	nb.StripStyle(NewStyle().Background(Fixed(red)))
	tt.Frame()
	check(true, tk, red, false)
	nb.StripStyle()
	offSkin()
	tt.Frame()
	check(false, nil, plain, false)
}

// The card: with a skin, rounded by RadiusCard, Elevation[1]'s shadow under
// it and a line of 1 around it; without, as before skins.
func TestSkinCard(t *testing.T) {
	card := Card().SetTitle(Label("title")).SetBody(Label("body"))
	card.SetAccessibleName("card")
	tt := tester(t, VBox(0).Add(card).SetSize(300, 0))
	check := func(on bool, tk *skinhook.Tokens) {
		t.Helper()
		r := nodeOf(t, tt, ui.RoleGroup, "card").Bounds
		page, surface, border := ui.LightTheme().Background, ui.LightTheme().Surface, ui.LightTheme().Border
		if on {
			page, surface, border = tk.Content, tk.Control, tk.Line
		}
		// On the left edge, 3.5 down: outside a radius of 10, on the line of
		// the plain radius.
		corner := skinPx(tt, r.X+0.5, r.Y+3.5)
		if on && !near(corner, page) || !on && near(corner, page) {
			t.Errorf("skin %v: the left edge 3.5 down is %v (page %v)", on, corner, page)
		}
		if c := skinPx(tt, r.X+r.W/2, r.Y+0.5); !near(c, border) {
			t.Errorf("skin %v: the card's top line is %v, want %v", on, c, border)
		}
		if c := skinPx(tt, r.X+r.W/2, r.Y+r.H/2-1); !near(c, surface) && !hasColor(tt, r.X+r.W-6, r.Y+4, r.Y+r.H-4, surface) {
			t.Errorf("skin %v: the card is %v inside, want %v", on, c, surface)
		}
		below := skinPx(tt, r.X+r.W/2, r.Y+r.H+3)
		if shadow := !near(below, page); shadow != on {
			t.Errorf("skin %v: 3 under the card is %v, a shadow %v", on, below, shadow)
		}
	}
	check(false, nil)
	tk := useSkin(t)
	tt.Frame()
	check(true, tk)
	offSkin()
	tt.Frame()
	check(false, nil)
}

type skinRow struct{ ID, Name string }

var skinRows = []skinRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}

func skinRowKey(r *skinRow) string  { return r.ID }
func skinRowName(r *skinRow) string { return r.Name }

// The rows of list boxes and checked list boxes: with a skin, exactly
// Metrics.RowHeight, 45 at 150 %; without, as before skins.
func TestSkinListBoxRows(t *testing.T) {
	box := ListBox(skinRowKey, skinRowName).SetItems(skinRows).SetSize(200, 150)
	checked := CheckedListBox(skinRowKey, skinRowName).SetItems(skinRows).SetSize(200, 150)
	tt := tester(t, HBox(8).Add(box, checked))
	line := lineOf(t, 1)
	rows := func() (a, b ui.TestNode) {
		t.Helper()
		for _, n := range tt.Nodes() {
			if n.Role == ui.RoleListItem && n.Label == "Grace" {
				if a.ID == 0 {
					a = n
				} else {
					b = n
				}
			}
		}
		if a.ID == 0 || b.ID == 0 {
			t.Fatalf("no two list items Grace in %v", tt.Texts())
		}
		return a, b
	}
	check := func(on bool, want float32) {
		t.Helper()
		a, b := rows()
		if !on {
			want = line + 2*space(1)
			if !sameSize(a.Bounds.H, want) {
				t.Errorf("without a skin, the list box's row is %v, want %v", a.Bounds.H, want)
			}
			if b.Bounds.H < want-0.01 {
				t.Errorf("without a skin, the checked list box's row is %v, want at least %v", b.Bounds.H, want)
			}
			return
		}
		if !sameSize(a.Bounds.H, want) || !sameSize(b.Bounds.H, want) {
			t.Errorf("with a skin, the rows are %v and %v, want %v", a.Bounds.H, b.Bounds.H, want)
		}
	}
	check(false, 0)
	_, plainChecked := rows()
	useSkin(t)
	tt.Frame()
	check(true, 30)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	check(true, 45)
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	offSkin()
	tt.Frame()
	check(false, 0)
	if _, b := rows(); !sameSize(b.Bounds.H, plainChecked.Bounds.H) {
		t.Errorf("after the skin, the checked list box's row is %v, want its plain %v", b.Bounds.H, plainChecked.Bounds.H)
	}
}

// The rows of a list view in details and list modes: with a skin,
// max(Metrics.RowHeight, image + 8); the icon views keep their sizes;
// without, as before skins.
func TestSkinListViewRows(t *testing.T) {
	cols := []Column[skinRow]{{Title: "Name", Text: skinRowName}}
	views := map[ViewMode]*ListViewWidget[skinRow, string]{}
	box := HBox(8)
	for _, m := range []ViewMode{ViewDetails, ViewList, ViewSmallIcon, ViewLargeIcon} {
		lv := ListView(skinRowKey, cols...).SetRows(skinRows).SetView(m).SetSize(180, 200)
		lv.SetAccessibleName(m.String())
		views[m] = lv
		box.Add(lv)
	}
	tt := tester(t, box)
	// heights returns the height of the item Grace in each view, by its x.
	heights := func() map[ViewMode]float32 {
		t.Helper()
		out := map[ViewMode]float32{}
		var xs []ui.TestNode
		for _, n := range tt.Nodes() {
			if (n.Role == ui.RoleRow || n.Role == ui.RoleListItem) && n.Label == "Grace" {
				xs = append(xs, n)
			}
		}
		if len(xs) != 4 {
			t.Fatalf("%d items Grace, want 4", len(xs))
		}
		for _, m := range []ViewMode{ViewDetails, ViewList, ViewSmallIcon, ViewLargeIcon} {
			v := labeled(t, tt, m.String()).Bounds
			for _, n := range xs {
				if n.Bounds.X >= v.X && n.Bounds.X < v.X+v.W {
					out[m] = n.Bounds.H
				}
			}
		}
		return out
	}
	plain := heights()
	if !sameSize(plain[ViewList], 30) || !sameSize(plain[ViewSmallIcon], 30) || plain[ViewDetails] < space(8)-0.01 {
		t.Errorf("without a skin, the rows are %v, want list and small icons 30, details at least %v", plain, space(8))
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	plain15 := heights()
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	useSkin(t)
	tt.Frame()
	got := heights()
	for m, want := range map[ViewMode]float32{ViewDetails: 30, ViewList: 30, ViewSmallIcon: 30, ViewLargeIcon: plain[ViewLargeIcon]} {
		if !sameSize(got[m], want) {
			t.Errorf("with a skin, the %v row is %v, want %v", m, got[m], want)
		}
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	got = heights()
	for m, want := range map[ViewMode]float32{ViewDetails: 45, ViewList: 45, ViewSmallIcon: plain15[ViewSmallIcon], ViewLargeIcon: plain15[ViewLargeIcon]} {
		if !sameSize(got[m], want) {
			t.Errorf("with a skin at 150 %%, the %v row is %v, want %v", m, got[m], want)
		}
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	offSkin()
	tt.Frame()
	if after := heights(); after[ViewList] != plain[ViewList] || after[ViewDetails] != plain[ViewDetails] || after[ViewLargeIcon] != plain[ViewLargeIcon] {
		t.Errorf("after the skin, the rows are %v, want their plain %v", after, plain)
	}
}

// The domain up-down: with a skin, Metrics.ControlHeight high; an app's
// SetSize wins; without, as before skins.
func TestSkinDomainUpDown(t *testing.T) {
	// Each control's height is the room between the labels around it: its
	// own node is the stepper's, the field has none.
	tt := tester(t, VBox(0).Add(Label("a"), DomainUpDown("North", "South"), Label("b"),
		DomainUpDown("East", "West").SetSize(0, 24), Label("c")))
	height := func(name string) float32 {
		t.Helper()
		above, below := map[string]string{"dom": "a", "sized": "b"}[name], map[string]string{"dom": "b", "sized": "c"}[name]
		a := find(t, tt, above)
		return find(t, tt, below).Y - (a.Y + a.H)
	}
	plain := height("dom")
	// The field: its line, padded by Space(1.5) above and below, and its
	// border; the stepper beside it may be taller.
	if want := lineOf(t, 1) + 2*space(1.5) + 2; plain < want-0.01 {
		t.Errorf("without a skin, the domain up-down is %v, want at least %v", plain, want)
	}
	useSkin(t)
	tt.Frame()
	if h := height("dom"); !sameSize(h, 36) {
		t.Errorf("with a skin, the domain up-down is %v, want 36", h)
	}
	if h := height("sized"); !sameSize(h, 24) {
		t.Errorf("with a skin, SetSize(0, 24) gives %v, want 24", h)
	}
	// The field fills the control's height: its outline is on the control's
	// first row.
	for _, c := range []struct{ above, item string }{{"a", "North"}, {"b", "East"}} {
		top, x := find(t, tt, c.above), find(t, tt, c.item).X
		if px := skinPx(tt, x, top.Y+top.H+0.5); near(px, ui.LightTheme().Background) {
			t.Errorf("with a skin, the field of %s does not reach the control's top: %v there", c.item, px)
		}
	}
	offSkin()
	tt.Frame()
	if h := height("dom"); !sameSize(h, plain) {
		t.Errorf("after the skin, the domain up-down is %v, want its plain %v", h, plain)
	}
}
