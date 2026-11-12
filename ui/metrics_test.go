package ui

import (
	"math"
	"testing"
	"time"
)

// metricsParts builds one of each part Theme.Metrics sizes, each findable by
// its role and label (partOf), in a window that draws its menus.
type metricsParts struct {
	text, sel, opt     string
	num                float64
	date               time.Time
	check, sw, toggled bool
	radio, tab, seg    int
	rows, cells, free  ListState
	outline            OutlineState[string]
	place              string
	buttonHeight       float32 // the plain button's own Height, when not 0
	buttonMinZero      bool    // and MinHeight(0) after it
	label              string  // the plain button's label
}

func newMetricsParts() *metricsParts {
	p := &metricsParts{sel: "Apple", label: "OK", place: "mail",
		date: time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC)}
	p.rows.Selected = new(int)
	p.cells.CellSelect, p.cells.Selection = true, &Selection[int]{}
	p.outline.List.Selected = new(int)
	return p
}

// group wraps fn's part in a box assistive technology sees labeled label,
// as tall as the part: for parts whose own node is not named, as a combo
// box's field around its input.
func group(c *context, label string, fn func()) {
	coreColumn(c).AlignItems(Start).Role(RoleGroup).Label(label).Children(fn)
}

func (p *metricsParts) view(c *context) {
	c.SetMenuPresentation(PopoverMenus)
	opts := []string{"Apple", "Banana", "Cherry"}
	coreRow(c).Fill().AlignItems(Start).Gap(10).Padding(10).Children(func() {
		coreColumn(c).Gap(8).AlignItems(Start).Width(260).Children(func() {
			b := coreButton(c, p.label).Label("button")
			if p.buttonHeight > 0 {
				b.Height(p.buttonHeight)
				if p.buttonMinZero {
					b.MinHeight(0)
				}
			}
			corePrimaryButton(c, "Send").Label("primary")
			coreToolbar(c, func() { coreButton(c, "Cut").Label("toolbar button") }).Label("toolbar")
			coreTextInput(c, &p.text).Label("text input")
			group(c, "combo box", func() { coreCombobox(c, &p.opt, opts) })
			coreSelect(c, &p.sel, opts).Label("select")
			group(c, "number", func() { coreSpinBox(c, &p.num, 0, 10, 1) })
			coreDateInput(c, &p.date).Label("date")
			coreTimeInput(c, &p.date).Label("time")
			coreCheckbox(c, &p.check, "").Label("check")
			coreRadio(c, &p.radio, 1, "").Label("radio")
			coreSwitch(c, &p.sw).Label("switch")
			coreTabs(c, &p.tab, "One", "Two")
			coreSegmented(c, &p.seg, "Left", "Right").Label("segmented")
			coreSplitButton(c, "Split").Group.Label("split")
			coreToggleGroup(c, func() { coreToggle(c, &p.toggled, "Bold") }).Label("toggle group")
			coreMenuButton(c, "Menu", func(m *Menu) { m.Item("Copy"); m.Item("Paste") }).Label("menu")
		})
		coreColumn(c).Gap(8).Width(300).Children(func() {
			coreTable(c, &p.rows, []TableColumn{{Title: "Name"}}, 3, func(row, col int) {
				coreText(c, []string{"Row A", "Row B", "Row C"}[row])
			}).Height(200).Label("table")
			coreTree(c, func() {
				coreTreeItem(c, "Leaf", nil, nil).Selected(true)
				coreTreeItem(c, "Other", nil, nil)
			}).Label("tree")
			coreOutline(c, &p.outline, []string{"Root", "Next"}, func(string) []string { return nil }, func(s string) {
				coreText(c, s)
			}).Height(70).Label("outline")
			coreSidebar(c, &p.place, func() {
				coreSidebarSection(c, "Places", nil, func() {
					coreSidebarItem(c, "mail", nil, "Mail")
					coreSidebarItem(c, "notes", nil, "Notes")
				})
			}).Width(160).Label("sidebar")
			coreTable(c, &p.cells, []TableColumn{{Title: "Cell"}}, 2, func(row, col int) {
				coreText(c, []string{"Cell A", "Cell B"}[row])
			}).Height(110).Label("cells")
			// Without a choice, a table's rows show the pointer themselves
			// (tableRowStyle).
			coreTable(c, &p.free, []TableColumn{{Title: "Free"}}, 2, func(row, col int) {
				coreText(c, []string{"Free A", "Free B"}[row])
			}).Height(110).Label("free")
		})
	})
}

// metricsTester returns a Tester of p's view, light, at text scale 1, with
// motion reduced so that hover colors show at once.
func metricsTester(t *testing.T, p *metricsParts) *Tester {
	t.Helper()
	tt := coreNewTester(p.view, 900, 900)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	return tt
}

// partOf returns the first node of role r labeled label.
func partOf(t *testing.T, tt *Tester, r Role, label string) TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && n.Label == label {
			return n
		}
	}
	t.Fatalf("no %v %q", r, label)
	return TestNode{}
}

// itemOf returns the first node labeled label that is not its text: a row,
// an option or a menu item.
func itemOf(t *testing.T, tt *Tester, label string) TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role != RoleText && n.Label == label {
			return n
		}
	}
	t.Fatalf("no item %q", label)
	return TestNode{}
}

// nodeOf returns the element of the last frame with the ID id.
func nodeOf(t *testing.T, tt *Tester, id uint64) *node {
	t.Helper()
	var found *node
	var walk func(e *node)
	walk = func(e *node) {
		for ; e != nil && found == nil; e = e.next {
			if e.id == id {
				found = e
				return
			}
			walk(e.first)
		}
	}
	walk(tt.rt.c.root)
	if found == nil {
		t.Fatalf("no element has the ID %x", id)
	}
	return found
}

// fillOf returns the color of the last frame inside n's box, at its right
// end, away from its text.
func fillOf(tt *Tester, n TestNode) Color {
	r := n.Bounds
	return pixel(tt.Image(), r.X+r.W-6, r.Y+r.H/2)
}

// lineOf returns the height of a line of text in the theme's FontSize.
func lineOf(t *testing.T, tt *Tester) float32 {
	t.Helper()
	r, ok := tt.Find("Row A")
	if !ok {
		t.Fatal("no Row A")
	}
	return r.H
}

// plainSizes are the heights, and widths where they are fixed, of the parts
// at f9ec6f97, before Theme.Metrics: in DIPs, as the text's line height L
// plus a constant, so that they hold for every platform's font. Measured on
// macOS (L 15.31, FontSize 13): button 29.31, primary 27.31, toolbar button
// 27.31, text input 29.31, combo box 29.31, select 29.31, number 29.31, date
// 29.31, time 29.31, check box 16 × 16, radio 16 × 16, switch 36 × 20, tab
// 31.31, segmented 29.31, split 29.31, toggle group 29.31, popup item 27.31,
// select option 27.31, table row and header 32, tree row 28, outline row 24.
func plainSizes(line float32) map[string]float32 {
	return map[string]float32{
		"button": line + 14, "primary": line + 12, "toolbar button": line + 12,
		"text input": line + 14, "combo box": line + 14, "select": line + 14, "number": line + 14,
		"date": line + 14, "time": line + 14,
		"check": 16, "check width": 16, "radio": 16, "radio width": 16, "switch": 20, "switch width": 36,
		"tab": line + 16, "segmented": line + 14, "split": line + 14, "toggle group": line + 14,
		"popup item": line + 12, "select option": line + 12,
		"table row": 32, "table header": 32, "tree row": 28,
		// A list's rows take whole DIPs.
		"outline row": float32(math.Ceil(float64(line + 8))),
	}
}

// sizesOf measures the parts plainSizes names, opening the select and the
// menu to measure their items, and closing them.
func sizesOf(t *testing.T, tt *Tester) map[string]float32 {
	t.Helper()
	got := map[string]float32{}
	for _, p := range []struct {
		name string
		role Role
	}{
		{"button", RoleButton}, {"primary", RoleButton}, {"toolbar button", RoleButton},
		{"text input", RoleTextField}, {"combo box", RoleGroup}, {"select", RolePopUpButton},
		{"number", RoleGroup}, {"date", RolePopUpButton}, {"time", RoleGroup},
		{"check", RoleCheckBox}, {"radio", RoleRadio}, {"switch", RoleSwitch},
		{"segmented", RoleRadioGroup}, {"split", RoleGroup}, {"toggle group", RoleGroup},
	} {
		got[p.name] = partOf(t, tt, p.role, p.name).Bounds.H
	}
	got["check width"] = partOf(t, tt, RoleCheckBox, "check").Bounds.W
	got["radio width"] = partOf(t, tt, RoleRadio, "radio").Bounds.W
	got["switch width"] = partOf(t, tt, RoleSwitch, "switch").Bounds.W
	got["tab"] = partOf(t, tt, RoleTab, "One").Bounds.H
	got["table header"] = partOf(t, tt, RoleColumnHeader, "Name").Bounds.H
	got["table row"] = partOf(t, tt, RoleRow, "Row B").Bounds.H
	got["tree row"] = partOf(t, tt, RoleTreeItem, "Leaf").Bounds.H
	got["outline row"] = partOf(t, tt, RoleTreeItem, "Root").Bounds.H
	if err := tt.Click("select"); err != nil {
		t.Fatal(err)
	}
	got["select option"] = itemOf(t, tt, "Banana").Bounds.H
	tt.Key(0, KeyEscape)
	if err := tt.Click("menu"); err != nil {
		t.Fatal(err)
	}
	got["popup item"] = itemOf(t, tt, "Copy").Bounds.H
	tt.CloseMenu()
	return got
}

// sameSize reports whether a and b are the same size, to rounding.
func sameSize(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }

// calmMetrics are Calm's.
var calmMetrics = Metrics{ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, TabHeight: 40, CheckSize: 15, SwitchWidth: 40, SwitchHeight: 22}

// withMetrics makes m the metrics of both default themes until the test
// ends.
func withMetrics(t *testing.T, m Metrics) {
	t.Helper()
	light, dark := LightTheme(), DarkTheme()
	light.Metrics, dark.Metrics = m, m
	SetDefaultThemes(light, dark)
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
}

// Without Metrics every part is as large as at f9ec6f97 and rows and buttons
// take today's colors, light and dark.
func TestPlainLookUnchanged(t *testing.T) {
	for _, dark := range []bool{false, true} {
		p := newMetricsParts()
		tt := metricsTester(t, p)
		tt.SetDark(dark)
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		want := plainSizes(lineOf(t, tt))
		for name, h := range sizesOf(t, tt) {
			if !sameSize(h, want[name]) {
				t.Errorf("dark=%v: %s is %v, want %v", dark, name, h, want[name])
			}
		}
		row := partOf(t, tt, RoleRow, "Row A")
		if e := nodeOf(t, tt, row.ID); e.bg != th.Accent || e.ts.color != th.AccentText {
			t.Errorf("dark=%v: a selected table row fills %v with text %v, want the accent %v and %v", dark, e.bg, e.ts.color, th.Accent, th.AccentText)
		}
		b := partOf(t, tt, RoleButton, "button")
		if e := nodeOf(t, tt, b.ID); e.borderC != th.Outline() {
			t.Errorf("dark=%v: a plain button's border is %v, want Outline %v", dark, e.borderC, th.Outline())
		}
		for _, name := range []string{"Row C", "Free B"} {
			hovered := partOf(t, tt, RoleRow, name)
			tt.Move(hovered.Bounds.X+hovered.Bounds.W/2, hovered.Bounds.Y+hovered.Bounds.H/2)
			if got := fillOf(tt, hovered); got != th.SurfaceHover {
				t.Errorf("dark=%v: hovered %s fills %v, want SurfaceHover %v", dark, name, got, th.SurfaceHover)
			}
		}
	}
}

// Calm's Metrics size every part: controls 36 high, rows, headers, popup
// items and options 30, tab headers 40, check boxes and radios 15, switches
// 40 × 22. The outer box of a combo box, a number input, a segmented
// control, a toggle group and a split button is 36, not 36 and an inner
// part.
func TestMetricsSizeControls(t *testing.T) {
	withMetrics(t, calmMetrics)
	tt := metricsTester(t, newMetricsParts())
	want := map[string]float32{
		"button": 36, "primary": 36, "toolbar button": 36, "text input": 36, "combo box": 36, "select": 36,
		"number": 36, "date": 36, "time": 36, "check": 15, "check width": 15, "radio": 15, "radio width": 15,
		"switch": 22, "switch width": 40, "tab": 40, "segmented": 36, "split": 36, "toggle group": 36,
		"popup item": 30, "select option": 30, "table row": 30, "table header": 30, "tree row": 30, "outline row": 30,
	}
	got := sizesOf(t, tt)
	for name, h := range want {
		if !sameSize(got[name], h) {
			t.Errorf("%s is %v, want %v", name, got[name], h)
		}
	}
	// The toolbar holds its 36 buttons, and the segments their track.
	if h := partOf(t, tt, RoleRadio, "Left").Bounds.H; h >= 36 {
		t.Errorf("a segment is %v high, want less than its track's 36", h)
	}
}

// Each Metrics field sizes its parts alone: one set, the others zero, the
// parts of the others keep their plain size.
func TestMetricsFieldsAreApart(t *testing.T) {
	withMetrics(t, Metrics{TabHeight: 40})
	tt := metricsTester(t, newMetricsParts())
	plain := plainSizes(lineOf(t, tt))
	got := sizesOf(t, tt)
	for name, h := range got {
		want := plain[name]
		if name == "tab" {
			want = 40
		}
		if !sameSize(h, want) {
			t.Errorf("with TabHeight alone, %s is %v, want %v", name, h, want)
		}
	}
}

// A size the app sets wins where it is larger, and a row height of its own
// in either direction; a smaller Height alone keeps the metric, the least,
// until the app takes the minimum away too.
func TestAppSizesWinOverMetrics(t *testing.T) {
	withMetrics(t, calmMetrics)
	for _, c := range []struct {
		height  float32
		minZero bool
		want    float32
	}{{50, false, 50}, {24, false, 36}, {24, true, 24}} {
		p := newMetricsParts()
		p.buttonHeight, p.buttonMinZero = c.height, c.minZero
		tt := metricsTester(t, p)
		if h := partOf(t, tt, RoleButton, "button").Bounds.H; !sameSize(h, c.want) {
			t.Errorf("Height(%v), MinHeight(0) %v: the button is %v, want %v", c.height, c.minZero, h, c.want)
		}
	}
	for _, rh := range []float32{24, 40} {
		p := newMetricsParts()
		p.rows.RowHeight = rh
		tt := metricsTester(t, p)
		if h := partOf(t, tt, RoleRow, "Row B").Bounds.H; !sameSize(h, rh) {
			t.Errorf("ListState.RowHeight %v: a row is %v", rh, h)
		}
	}
}

// Metrics scale with the user's text size and come back with it; a label of
// stacked marks fits its button.
func TestMetricsFollowTextScale(t *testing.T) {
	withMetrics(t, calmMetrics)
	p := newMetricsParts()
	p.label = "Ữ Ặ Ỗ"
	tt := metricsTester(t, p)
	check := func(scale, button, row, box float32) {
		t.Helper()
		tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: scale})
		b := partOf(t, tt, RoleButton, "button").Bounds
		if !sameSize(b.H, button) {
			t.Errorf("TextScale %v: the button is %v, want %v", scale, b.H, button)
		}
		if h := partOf(t, tt, RoleRow, "Row B").Bounds.H; !sameSize(h, row) {
			t.Errorf("TextScale %v: a table row is %v, want %v", scale, h, row)
		}
		if h := partOf(t, tt, RoleCheckBox, "check").Bounds.H; !sameSize(h, box) {
			t.Errorf("TextScale %v: the check box is %v, want %v", scale, h, box)
		}
		text, ok := tt.Find("Ữ Ặ Ỗ")
		pad := nodeOf(t, tt, partOf(t, tt, RoleButton, "button").ID).padY()
		if !ok || b.H < text.H+pad || text.Y < b.Y || text.Y+text.H > b.Y+b.H {
			t.Errorf("TextScale %v: the label %v (found %v) does not fit its button %v with padding %v", scale, text, ok, b, pad)
		}
	}
	check(1.5, 54, 45, 22.5)
	check(1, 36, 30, 15)
	check(2, 72, 60, 30)
	check(1, 36, 30, 15)
}

// The row colors color rows, tree items, highlighted popup items and
// options, the chosen sidebar item, focused or not, and a table's block of
// cells; the chosen day of a calendar keeps the accent.
func TestRowColors(t *testing.T) {
	light, dark := LightTheme(), DarkTheme()
	hover, sel, text := Hex("#f5f7fa"), Hex("#e8f0fd"), Hex("#102030")
	light.RowHover, light.RowSelected, light.RowSelectedText = hover, sel, text
	SetDefaultThemes(light, dark)
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	tt := metricsTester(t, newMetricsParts())

	for _, name := range []string{"Row A", "Leaf"} {
		e := nodeOf(t, tt, itemOf(t, tt, name).ID)
		if e.bg != sel || e.ts.color != text {
			t.Errorf("selected %s fills %v with text %v, want %v and %v", name, e.bg, e.ts.color, sel, text)
		}
	}
	if err := tt.Click("Cell B"); err != nil {
		t.Fatal(err)
	}
	cell := blockCellOf(t, tt)
	if cell.bg != sel || cell.ts.color != text {
		t.Errorf("the block's cell fills %v with text %v, want %v and %v", cell.bg, cell.ts.color, sel, text)
	}
	if e := nodeOf(t, tt, itemOf(t, tt, "Mail").ID); e.bg != sel || e.ts.color != text {
		t.Errorf("the chosen sidebar item, not focused, fills %v with text %v, want %v and %v", e.bg, e.ts.color, sel, text)
	}
	for _, name := range []string{"Row C", "Free B", "Other"} {
		r := itemOf(t, tt, name)
		tt.Move(r.Bounds.X+r.Bounds.W/2, r.Bounds.Y+r.Bounds.H/2)
		if got := fillOf(tt, r); got != hover {
			t.Errorf("hovered %s fills %v, want %v", name, got, hover)
		}
	}
	if err := tt.Click("Mail"); err != nil {
		t.Fatal(err)
	}
	if e := nodeOf(t, tt, itemOf(t, tt, "Mail").ID); e.bg != sel || e.ts.color != text {
		t.Errorf("the chosen sidebar item, focused, fills %v with text %v, want %v and %v", e.bg, e.ts.color, sel, text)
	}

	if err := tt.Click("select"); err != nil {
		t.Fatal(err)
	}
	r := itemOf(t, tt, "Banana")
	tt.Move(r.Bounds.X+r.Bounds.W/2, r.Bounds.Y+r.Bounds.H/2)
	if got := fillOf(tt, r); got != sel {
		t.Errorf("the highlighted option fills %v, want %v", got, sel)
	}
	tt.Key(0, KeyEscape)
	if err := tt.Click("menu"); err != nil {
		t.Fatal(err)
	}
	r = itemOf(t, tt, "Paste")
	tt.Move(r.Bounds.X+r.Bounds.W/2, r.Bounds.Y+r.Bounds.H/2)
	if got := fillOf(tt, r); got != sel {
		t.Errorf("the highlighted popup item fills %v, want %v", got, sel)
	}
	tt.CloseMenu()

	if err := tt.Click("date"); err != nil {
		t.Fatal(err)
	}
	day := chosenDayOf(t, tt)
	if day.bg != light.Accent {
		t.Errorf("the chosen day fills %v, want the accent %v", day.bg, light.Accent)
	}
}

// blockCellOf returns the cell of the "cells" table's block: the element
// whose checked is 2 among the cells.
func blockCellOf(t *testing.T, tt *Tester) *node {
	t.Helper()
	return findNode(t, tt, "the block's cell", func(e *node) bool { return e.role == RoleCell && e.checked == 2 })
}

// chosenDayOf returns the calendar's chosen day.
func chosenDayOf(t *testing.T, tt *Tester) *node {
	t.Helper()
	return findNode(t, tt, "the chosen day", func(e *node) bool {
		return e.checked == 2 && e.first != nil && e.first.text == "15"
	})
}

// findNode returns the first element of the last frame, overlays included,
// that ok accepts.
func findNode(t *testing.T, tt *Tester, what string, ok func(e *node) bool) *node {
	t.Helper()
	var found *node
	var walk func(e *node)
	walk = func(e *node) {
		for ; e != nil && found == nil; e = e.next {
			if ok(e) {
				found = e
				return
			}
			walk(e.first)
		}
	}
	walk(tt.rt.c.root)
	if found == nil {
		t.Fatalf("no %s", what)
	}
	return found
}

// With RowSelected set and RowSelectedText left zero, chosen rows and the
// highlighted popup item show Text on the tint.
func TestRowSelectedTextDefaultsToText(t *testing.T) {
	light := LightTheme()
	light.RowSelected = Hex("#e8f0fd")
	SetDefaultThemes(light, nil)
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	tt := metricsTester(t, newMetricsParts())
	if e := nodeOf(t, tt, itemOf(t, tt, "Row A").ID); e.bg != light.RowSelected || e.ts.color != light.Text {
		t.Errorf("a selected row fills %v with text %v, want %v and Text %v", e.bg, e.ts.color, light.RowSelected, light.Text)
	}
	if err := tt.Click("menu"); err != nil {
		t.Fatal(err)
	}
	r := itemOf(t, tt, "Paste")
	tt.Move(r.Bounds.X+r.Bounds.W/2, r.Bounds.Y+r.Bounds.H/2)
	item := findNode(t, tt, "the highlighted popup item", func(e *node) bool { return e.label == "Paste" && e.bg == light.RowSelected })
	if item.ts.color != light.Text {
		t.Errorf("the highlighted popup item's text is %v, want Text %v", item.ts.color, light.Text)
	}
}

// A selected row keeps RowSelected in a window not drawn as active; without
// it the accent fades as before.
func TestRowSelectedInInactiveWindow(t *testing.T) {
	for _, tinted := range []bool{false, true} {
		light := LightTheme()
		if tinted {
			light.RowSelected = Hex("#e8f0fd")
		}
		SetDefaultThemes(light, nil)
		tt := metricsTester(t, newMetricsParts())
		tt.SetWindowActive(false)
		want := light.inactiveAccent()
		if tinted {
			want = light.RowSelected
		}
		if e := nodeOf(t, tt, itemOf(t, tt, "Row A").ID); e.bg != want {
			t.Errorf("RowSelected set %v: in an inactive window a selected row fills %v, want %v", tinted, e.bg, want)
		}
		SetDefaultThemes(nil, nil)
	}
}

// ButtonBorder outlines plain buttons, not inputs, and High Contrast
// strengthens it as it does ControlBorder.
func TestButtonBorder(t *testing.T) {
	light := LightTheme()
	light.ButtonBorder = Hex("#d2d7de")
	SetDefaultThemes(light, nil)
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	tt := metricsTester(t, newMetricsParts())
	check := func(contrast bool, button, input Color) {
		t.Helper()
		tt.SetPreferences(Preferences{ReduceMotion: true, HighContrast: contrast, TextScale: 1})
		if e := nodeOf(t, tt, partOf(t, tt, RoleButton, "button").ID); e.borderC != button {
			t.Errorf("HighContrast %v: a plain button's border is %v, want %v", contrast, e.borderC, button)
		}
		if e := nodeOf(t, tt, partOf(t, tt, RoleTextField, "text input").ID); e.borderC != input {
			t.Errorf("HighContrast %v: a text input's border is %v, want %v", contrast, e.borderC, input)
		}
	}
	check(false, light.ButtonBorder, light.ControlBorder)
	check(true, light.ButtonBorder.Mix(light.Text, 0.45), light.ControlBorder.Mix(light.Text, 0.45))
	check(false, light.ButtonBorder, light.ControlBorder)
}

// A table's kept rows take new Metrics in the next frame, and the plain
// size back.
func TestKeptRowsTakeNewMetrics(t *testing.T) {
	var rows ListState
	rows.RowVersion = func(int) uint64 { return 1 }
	tt := coreNewTester(func(c *context) {
		coreTable(c, &rows, []TableColumn{{Title: "Name"}}, 50, func(row, col int) {
			coreText(c, "Row")
		}).Fill()
	}, 300, 400)
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	heights := func() []float32 {
		var hs []float32
		for _, n := range tt.Nodes() {
			if n.Role == RoleRow && n.Label == "Row" && n.Bounds.H > 0 {
				hs = append(hs, n.Bounds.H)
			}
		}
		return hs
	}
	for range 3 {
		tt.Frame()
	}
	check := func(want float32) {
		t.Helper()
		hs := heights()
		if len(hs) < 6 {
			t.Fatalf("%d rows show", len(hs))
		}
		for i, h := range hs[:5] { // the last may show in part

			if !sameSize(h, want) {
				t.Fatalf("row %d is %v, want %v", i, h, want)
			}
		}
	}
	check(32)
	withMetrics(t, Metrics{RowHeight: 30})
	tt.Frame()
	check(30)
	SetDefaultThemes(nil, nil)
	tt.Frame()
	check(32)
}

// WithPreferences scales each Metrics field set by TextScale, leaves zero
// ones zero, and puts them back when the scale goes back; a Metrics the app
// set since stays.
func TestMetricsWithPreferences(t *testing.T) {
	th := LightTheme()
	th.Metrics = Metrics{ControlHeight: 36, RowHeight: 30}
	u := th.WithPreferences(Preferences{TextScale: 1.5})
	if u.Metrics != (Metrics{ControlHeight: 54, RowHeight: 45}) {
		t.Errorf("at 1.5, Metrics %+v", u.Metrics)
	}
	if v := u.WithPreferences(Preferences{TextScale: 1}); v.Metrics != th.Metrics {
		t.Errorf("back at 1, Metrics %+v, want %+v", v.Metrics, th.Metrics)
	}
	w := *u
	w.Metrics.RowHeight = 28
	if v := w.WithPreferences(Preferences{TextScale: 1}); v.Metrics.RowHeight != 28 {
		t.Errorf("a Metrics set since is %+v, want RowHeight 28 kept", v.Metrics)
	}
	if v := LightTheme().WithPreferences(Preferences{TextScale: 2}); v.Metrics != (Metrics{}) {
		t.Errorf("zero Metrics at 2 are %+v", v.Metrics)
	}
	b := LightTheme()
	b.ButtonBorder = Hex("#d2d7de")
	hc := b.WithPreferences(Preferences{HighContrast: true, TextScale: 1})
	if back := hc.WithPreferences(Preferences{TextScale: 1}); back.ButtonBorder != b.ButtonBorder {
		t.Errorf("ButtonBorder after High Contrast goes is %v, want %v", back.ButtonBorder, b.ButtonBorder)
	}
	if z := LightTheme().WithPreferences(Preferences{HighContrast: true, TextScale: 1}); z.ButtonBorder != (Color{}) {
		t.Errorf("a zero ButtonBorder under High Contrast is %v, want zero", z.ButtonBorder)
	}
}
