package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// The settings cards (P2b): each section of SettingsRows is a card, its title
// above it. The forms are 400 DIPs wide; a card's padding is Space(3) across
// and Space(1) down, inside a line of 1, so its right padding holds the
// column x 389, where neither a control (up to 387) nor a divider reaches.
const cardX = 389

// lineColor reports whether p is c, but for the smoothing a round card's
// edge has this near its corners: within 60 over the three channels.
func lineColor(p color.RGBA, c ui.Color) bool {
	d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	return d(p.R, c.R)+d(p.G, c.G)+d(p.B, c.B) <= 60
}

// cardLines returns the y of each horizontal line of color c in the column x,
// with its thickness: the top and bottom edges of the cards.
func cardLines(tt *ui.Tester, x int, c ui.Color) (ys, ws []int) {
	img := tt.Image()
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if !lineColor(img.RGBAAt(x, y), c) {
			continue
		}
		if n := len(ys); n > 0 && ys[n-1]+ws[n-1] == y {
			ws[n-1]++
			continue
		}
		ys, ws = append(ys, y), append(ws, 1)
	}
	return ys, ws
}

// card is a card's box as cardLines finds it: the y of its top line and of
// its bottom line, and their thickness.
type cardBox struct{ top, bottom, line int }

// cardsOf returns the cards whose edges are lines of color c in the column x.
func cardsOf(t *testing.T, tt *ui.Tester, x int, c ui.Color) []cardBox {
	t.Helper()
	ys, ws := cardLines(tt, x, c)
	if len(ys)%2 != 0 {
		t.Fatalf("%d edges of color %v at x %d (at %v): not pairs of a top and a bottom", len(ys), c, x, ys)
	}
	var cs []cardBox
	for i := 0; i < len(ys); i += 2 {
		cs = append(cs, cardBox{ys[i], ys[i+1], ws[i]})
	}
	return cs
}

// inside reports whether r lies within the card's box, between its lines.
func (k cardBox) inside(r ui.Rect) bool {
	return r.Y >= float32(k.top+k.line) && r.Y+r.H <= float32(k.bottom)+0.5
}

func cardForm() (*FieldsWidget, [4]*CustomWidget) {
	cs := [4]*CustomWidget{wbar("c1", 60, 24), wbar("c2", 120, 24), wbar("c3", 90, 24), wbar("c4", 60, 24)}
	f := Fields().SettingsRows().
		Section("Alpha").Row("One", cs[0]).Row("Two", cs[1]).
		Section("Beta").Row("Three", cs[2]).Row("Four", cs[3])
	return f, cs
}

// Case 1: two sections of two rows → two cards, each the theme's Surface in a
// line of 1 of its Border, rounded by its Radius; each title above its card,
// outside its box, a heading.
func TestSettingsCardsOnePerSection(t *testing.T) {
	f, _ := cardForm()
	tt := NewTester(t, f, 400, 400)
	th := ui.LightTheme()
	ks := cardsOf(t, tt.Tester, cardX, th.Border)
	if len(ks) != 2 {
		t.Fatalf("%d cards, want 2", len(ks))
	}
	for i, k := range ks {
		if k.line != 1 {
			t.Errorf("card %d: its top line is %d high, want 1", i, k.line)
		}
		if c := skinPx(tt.Tester, cardX, float32(k.top+1)); !near(c, th.Surface) {
			t.Errorf("card %d is %v under its top line, want Surface %v", i, c, th.Surface)
		}
		if c := skinPx(tt.Tester, 0.5, float32(k.top+k.bottom)/2); !near(c, th.Border) {
			t.Errorf("card %d: its left edge is %v, want Border %v", i, c, th.Border)
		}
		// The corner is round: its top left pixel is the page's, and 3 down
		// the left edge is on the line of a radius of 6.
		if c := skinPx(tt.Tester, 0.5, float32(k.top)+0.5); c != page(tt.Tester) {
			t.Errorf("card %d: its top left corner is %v, want the page %v (round)", i, c, page(tt.Tester))
		}
		if c := skinPx(tt.Tester, 0.5, float32(k.top)+3.5); c == page(tt.Tester) {
			t.Errorf("card %d: 3 down its left edge is the page: rounder than the theme's Radius", i)
		}
	}
	rows := [][]string{{"One", "Two"}, {"Three", "Four"}}
	for i, title := range []string{"Alpha", "Beta"} {
		h := find(t, tt.Tester, title)
		if h.Y+h.H > float32(ks[i].top)+0.5 {
			t.Errorf("title %q %v is not above its card at %d", title, h, ks[i].top)
		}
		if i > 0 && h.Y < float32(ks[i-1].bottom) {
			t.Errorf("title %q %v is not below the card before it, at %d", title, h, ks[i-1].bottom)
		}
		if h.X > 2 {
			t.Errorf("title %q at x %v, want at the cards' left edge", title, h.X)
		}
		if _, ok := tt.AccessByRole(ui.RoleHeading, title); !ok {
			t.Errorf("no heading %q", title)
		}
		for _, r := range rows[i] {
			if l := find(t, tt.Tester, r); !ks[i].inside(l) {
				t.Errorf("row %q %v is not in card %d %+v", r, l, i, ks[i])
			}
		}
	}
}

// Case 2: under a skin, a card is rounded by RadiusCard (10, Calm's) and has
// CardShadow under it, as Card has.
func TestSettingsCardsTakeTheSkinsCard(t *testing.T) {
	tk := useSkin(t)
	// The test tokens' Surface is their Background: give the cards their own.
	tk.Surface = ui.Hex("#e0f7fa")
	th := ui.LightTheme()
	th.Background, th.Surface, th.Border, th.Text = tk.Background, tk.Surface, tk.Border, tk.Text
	ui.SetDefaultThemes(th, th)
	f, _ := cardForm()
	tt := NewTester(t, f, 400, 400)
	ks := cardsOf(t, tt.Tester, cardX, tk.Border)
	if len(ks) != 2 {
		t.Fatalf("%d cards, want 2", len(ks))
	}
	pg := page(tt.Tester)
	for i, k := range ks {
		if c := skinPx(tt.Tester, 0.5, float32(k.top)+3.5); c != pg {
			t.Errorf("card %d: 3 down its left edge is %v, want the page %v: outside a radius of 10", i, c, pg)
		}
		if c := skinPx(tt.Tester, cardX, float32(k.top+2)); !near(c, tk.Surface) {
			t.Errorf("card %d is %v inside, want the skin's Surface %v", i, c, tk.Surface)
		}
		if c := skinPx(tt.Tester, 200, float32(k.bottom+3)); c == pg {
			t.Errorf("card %d: 3 under it is the page: no CardShadow", i)
		}
	}
	offSkin()
	tt.Frame()
	if c := skinPx(tt.Tester, 200, float32(cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)[0].bottom+3)); c != page(tt.Tester) {
		t.Errorf("the skin off, 3 under a card is %v: a shadow stays", c)
	}
}

// Case 3: rows before the first Section make a card with no title, first;
// with no Section at all, one card holds every row.
func TestSettingsCardsUntitledFirst(t *testing.T) {
	th := ui.LightTheme()
	f := Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Row("B", wbar("c2", 60, 24)).
		Section("S").Row("C", wbar("c3", 60, 24))
	tt := NewTester(t, f, 400, 400)
	ks := cardsOf(t, tt.Tester, cardX, th.Border)
	if len(ks) != 2 {
		t.Fatalf("%d cards, want 2", len(ks))
	}
	if ks[0].top > 2 {
		t.Errorf("the untitled card starts at %d, want at the top: no title above it", ks[0].top)
	}
	for _, s := range []string{"A", "B"} {
		if !ks[0].inside(find(t, tt.Tester, s)) {
			t.Errorf("row %q is not in the first card", s)
		}
	}
	if s := find(t, tt.Tester, "S"); s.Y < float32(ks[0].bottom) || s.Y+s.H > float32(ks[1].top)+0.5 {
		t.Errorf("title S %v is not between the cards %+v", s, ks)
	}
	if !ks[1].inside(find(t, tt.Tester, "C")) {
		t.Error("row C is not in the second card")
	}

	one := NewTester(t, Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Row("B", wbar("c2", 60, 24)).Row("C", wbar("c3", 60, 24)), 400, 400)
	ks = cardsOf(t, one.Tester, cardX, th.Border)
	if len(ks) != 1 {
		t.Fatalf("no Section: %d cards, want 1", len(ks))
	}
	for _, s := range []string{"A", "B", "C"} {
		if !ks[0].inside(find(t, one.Tester, s)) {
			t.Errorf("no Section: row %q is not in the card", s)
		}
	}
}

// Case 4: Section("") ends a card and starts one with no title, the section
// gap between them, and no node for assistive technology.
func TestSettingsCardsBreak(t *testing.T) {
	th := ui.LightTheme()
	f := Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Section("").Row("B", wbar("c2", 60, 24))
	tt := NewTester(t, f, 400, 400)
	ks := cardsOf(t, tt.Tester, cardX, th.Border)
	if len(ks) != 2 {
		t.Fatalf("%d cards, want 2", len(ks))
	}
	if gap := float32(ks[1].top - (ks[0].bottom + 1)); !approx(gap, space(4)) {
		t.Errorf("%v between the cards, want the section gap %v", gap, space(4))
	}
	if !ks[0].inside(find(t, tt.Tester, "A")) || !ks[1].inside(find(t, tt.Tester, "B")) {
		t.Errorf("rows A and B are not one in each card %+v", ks)
	}
	if a, ok := tt.AccessByRole(ui.RoleHeading, ""); ok {
		t.Errorf("the break has a heading node %+v", a)
	}
}

// Case 5 and Review Focus 3: the dividers are inside a card, between its
// rows that show: none above a card's first row, none below its last. A
// hidden row takes its divider away; a section whose rows are all hidden,
// even later, draws no card and no title; shown again, both come back.
func TestSettingsCardsDividers(t *testing.T) {
	th := ui.LightTheme()
	cs := [5]*CustomWidget{wbar("c1", 60, 24), wbar("c2", 60, 24), wbar("c3", 60, 24), wbar("c4", 60, 24), wbar("c5", 60, 24)}
	f := Fields().SettingsRows().
		Section("Alpha").Row("One", cs[0]).Row("Two", cs[1]).Row("Three", cs[2]).
		Section("Beta").Row("Four", cs[3]).Row("Five", cs[4])
	tt := NewTester(t, f, 400, 400)
	left := int(find(t, tt.Tester, "c1").X) - 4
	// between checks that a divider is at each of pairs, between its two rows.
	between := func(state string, pairs [][2]string) {
		t.Helper()
		ys := rules(tt.Tester, left)
		if len(ys) != len(pairs) {
			t.Fatalf("%s: %d dividers at %v, want %d", state, len(ys), ys, len(pairs))
		}
		for i, p := range pairs {
			a, b := find(t, tt.Tester, p[0]), find(t, tt.Tester, p[1])
			if y := float32(ys[i]); y < a.Y+a.H || y > b.Y {
				t.Errorf("%s: divider %d at %v is not between %q %v and %q %v", state, i, y, p[0], a, p[1], b)
			}
		}
	}
	between("all shown", [][2]string{{"One", "Two"}, {"Two", "Three"}, {"Four", "Five"}})
	if n := len(cardsOf(t, tt.Tester, cardX, th.Border)); n != 2 {
		t.Fatalf("%d cards, want 2", n)
	}

	cs[0].SetVisible(false)
	tt.Frame()
	between("One hidden", [][2]string{{"Two", "Three"}, {"Four", "Five"}})
	cs[2].SetVisible(false)
	tt.Frame()
	between("One and Three hidden", [][2]string{{"Four", "Five"}})

	cs[3].SetVisible(false)
	cs[4].SetVisible(false)
	tt.Frame()
	if tt.HasText("Beta") {
		t.Errorf("the title of a section whose rows are hidden shows: %q", tt.Texts())
	}
	if n := len(cardsOf(t, tt.Tester, cardX, th.Border)); n != 1 {
		t.Errorf("%d cards with Beta's rows hidden, want 1", n)
	}
	between("only Two", nil)

	for _, c := range cs {
		c.SetVisible(true)
	}
	tt.Frame()
	if !tt.HasText("Beta") || len(cardsOf(t, tt.Tester, cardX, th.Border)) != 2 {
		t.Errorf("shown again: texts %q, want Beta and two cards", tt.Texts())
	}
	between("shown again", [][2]string{{"One", "Two"}, {"Two", "Three"}, {"Four", "Five"}})
}

// Case 6: under a skin, each row is at least SettingsRowHeight high; the
// card's padding is outside it.
func TestSettingsCardsRowHeight(t *testing.T) {
	tk := useSkin(t)
	tk.SettingsRowHeight = 56
	one := NewTester(t, Fields().SettingsRows().Row("One", wbar("c1", 60, 24)), 400, 400)
	ks := cardsOf(t, one.Tester, cardX, tk.Border)
	if len(ks) != 1 {
		t.Fatalf("%d cards, want 1", len(ks))
	}
	// A line, the padding, the row, the padding and the line.
	if h, want := float32(ks[0].bottom+1-ks[0].top), 1+space(1)+56+space(1)+1; !approx(h, want) {
		t.Errorf("a card of one row is %v high, want %v: the row %v and the padding outside it", h, want, 56)
	}
	if c := find(t, one.Tester, "c1"); !approx(c.Y+12, float32(ks[0].top+ks[0].bottom+1)/2) {
		t.Errorf("the control %v is not at the card's middle %+v", c, ks[0])
	}
	two := NewTester(t, Fields().SettingsRows().Row("One", wbar("c1", 60, 24)).Row("Two", wbar("c2", 60, 24)), 400, 400)
	if a, b := find(t, two.Tester, "c1"), find(t, two.Tester, "c2"); b.Y-a.Y < 56 {
		t.Errorf("rows %v apart, want at least SettingsRowHeight 56", b.Y-a.Y)
	}
}

// Case 7: across cards the controls share one right edge, at the cards'
// padding, and the labels start at one x.
func TestSettingsCardsShareTheEdges(t *testing.T) {
	f, _ := cardForm()
	tt := NewTester(t, f, 400, 400)
	edge := float32(400) - 1 - space(3)
	for text, w := range map[string]float32{"c1": 60, "c2": 120, "c3": 90, "c4": 60} {
		if got := find(t, tt.Tester, text).X + w; !approx(got, edge) {
			t.Errorf("%s ends at %v, want the cards' inner edge %v", text, got, edge)
		}
	}
	x := 1 + space(3)
	for _, l := range []string{"One", "Two", "Three", "Four"} {
		if got := find(t, tt.Tester, l).X; !approx(got, x) {
			t.Errorf("label %q at x %v, want %v", l, got, x)
		}
	}
}

// Each card measures its own columns: a control that asks for Start sits at
// the left of the widest control of its card, not of the form (ui has no
// track shared between grids).
func TestSettingsCardsColumnsPerCard(t *testing.T) {
	f := Fields().SettingsRows().
		Section("Alpha").Row("One", wbar("c1", 60, 24).Align(Start)).Row("Two", wbar("c2", 120, 24)).
		Section("Beta").Row("Three", wbar("c3", 60, 24).Align(Start)).Row("Four", wbar("c4", 90, 24))
	tt := NewTester(t, f, 400, 400)
	edge := float32(400) - 1 - space(3)
	if c1 := find(t, tt.Tester, "c1"); !approx(c1.X, edge-120) {
		t.Errorf("c1 at x %v, want %v: the left of its card's widest control", c1.X, edge-120)
	}
	if c3 := find(t, tt.Tester, "c3"); !approx(c3.X, edge-90) {
		t.Errorf("c3 at x %v, want %v: the left of its card's widest control", c3.X, edge-90)
	}
}

// Review Focus 4: a list in a settings row stays inside its card's padding.
func TestSettingsCardsListStaysInside(t *testing.T) {
	lb := ListBox(skinRowKey, skinRowName).SetItems(skinRows)
	f := Fields().SettingsRows().Section("People").Row("Members", lb)
	tt := NewTester(t, f, 400, 400)
	b := tt.Find(ByRole(ui.RoleList)).Bounds()
	if b.X < 1+space(3)-0.5 || b.X+b.W > 400-1-space(3)+0.5 {
		t.Errorf("the list %v leaves the card's padding, [%v, %v]", b, 1+space(3), 400-1-space(3))
	}
}

// Review Focus 5: the mnemonics of labels in different cards focus their
// controls.
func TestSettingsCardsMnemonics(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 300)
	f.SetContent(Fields().SettingsRows().Section("A").Row("&Name", TextBox()).Section("B").Row("E&mail", TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	for _, n := range []string{"Name", "Email"} {
		if !tt.Has(ByRole(ui.RoleTextField).Label(n)) {
			t.Errorf("no text field named %q among %q", n, fieldNodes(tt.Tester))
		}
	}
	tt.Key(ui.Alt, ui.KeyM)
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+M did not focus the text box of the second card")
	}
}

// settingsAllocs returns the allocations of a frame of a form of n rows of
// text boxes, in sections of 5: settings rows or a plain form.
func settingsAllocs(n int, settings bool) float64 {
	f := Fields()
	if settings {
		f.SettingsRows()
	}
	for i := range n {
		if i%5 == 0 {
			f.Section("Section")
		}
		f.Row("Name", TextBox())
	}
	tt := ui.NewTester(content(ScrollPanel(f)), 1280, 800)
	for range 10 {
		tt.Frame()
	}
	return allocrun.Least(30, tt.Frame)
}

// A settings form, once built, allocates per row no more in a frame than a
// plain form of the same controls: the slope, 200 rows less 100, per row.
func TestSettingsCardsAllocateNoMoreThanAPlainForm(t *testing.T) {
	plain := (settingsAllocs(200, false) - settingsAllocs(100, false)) / 100
	cards := (settingsAllocs(200, true) - settingsAllocs(100, true)) / 100
	if cards > plain+0.2 {
		t.Errorf("%.2f allocations per settings row in a frame, %.2f per plain row", cards, plain)
	}
}
