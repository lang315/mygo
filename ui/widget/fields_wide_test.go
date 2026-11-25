package widget

import (
	"bytes"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Wide rows (P2d): a row whose control is under its label, across the form
// or the card. The forms are 400 DIPs wide; a card's inside runs from 13
// (its line and Space(3)) to 387.
const (
	wideKeys = "Danh sách"
	wideLong = "Danh sách các khóa được phép, mỗi dòng một khóa, dài hơn mọi nhãn khác của biểu mẫu này"
)

// cardLeft and cardRight are a card's inner edges, in a form 400 wide.
var cardLeft, cardRight = 1 + space(3), 400 - 1 - space(3)

// fieldBox returns the bounds of the n-th text field, from 0.
func fieldBox(tt *Tester, n int) ui.Rect {
	return tt.Find(ByRole(ui.RoleTextField).Nth(n)).Bounds()
}

// wideList returns the bounds of the list.
func wideList(tt *Tester) ui.Rect { return tt.Find(ByRole(ui.RoleList)).Bounds() }

// sameDrawing reports whether a and b draw the same pixels and the same
// nodes, but their ids, which differ between forms.
func sameDrawing(a, b *Tester) bool {
	if !bytes.Equal(a.Image().Pix, b.Image().Pix) {
		return false
	}
	na, nb := a.Nodes(), b.Nodes()
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		x, y := na[i], nb[i]
		x.ID, y.ID = 0, 0
		if x.Role != y.Role || x.Label != y.Label || x.Value != y.Value || x.Bounds != y.Bounds {
			return false
		}
	}
	return true
}

// dividersOutside returns the dividers of a card (rules) that are not in r,
// whose own lines rules would take for some.
func dividersOutside(tt *ui.Tester, r ui.Rect) (ys []int) {
	for _, y := range rules(tt, 40) {
		if float32(y) < r.Y-0.5 || float32(y) > r.Y+r.H+0.5 {
			ys = append(ys, y)
		}
	}
	return ys
}

// Case 1: in the plain grid a wide row's label starts where the other
// labels do, and its control spans both columns, from the labels' edge to
// the controls'; the rows after it keep the shared label column.
func TestWideRowSpansTheGrid(t *testing.T) {
	f := Fields().Row("Tên", TextBox()).Row(wideKeys, ListBox(skinRowKey, skinRowName).SetItems(skinRows)).Wide().
		Row("Mã", TextBox())
	tt := NewTester(t, f, 400, 400)
	name, keys := find(t, tt.Tester, "Tên"), find(t, tt.Tester, wideKeys)
	tb, list, tb2 := fieldBox(tt, 0), wideList(tt), fieldBox(tt, 1)
	if !approx(keys.X, name.X) {
		t.Errorf("the wide label at x %v, %q at %v", keys.X, "Tên", name.X)
	}
	// The label is as wide as its text, as a stacked label, not the row.
	if keys.W > 100 {
		t.Errorf("the wide label %v is wider than its text", keys)
	}
	if !approx(list.X, 0) || !approx(list.X+list.W, 400) || !approx(list.X+list.W, tb.X+tb.W) {
		t.Errorf("the list %v, want from 0 to the text box's right edge %v (400)", list, tb.X+tb.W)
	}
	if list.Y < keys.Y+keys.H-0.5 {
		t.Errorf("the list %v is not under its label %v", list, keys)
	}
	if !approx(tb2.X, tb.X) || tb.X < name.X+name.W {
		t.Errorf("the text boxes at x %v and %v, want both beside the label column", tb.X, tb2.X)
	}
}

// Case 2: a wide label longer than every other does not widen the label
// column: the text box is where it is without the wide row.
func TestWideLabelLeavesTheColumn(t *testing.T) {
	alone := fieldBox(NewTester(t, Fields().Row("Tên", TextBox()), 400, 400), 0)
	tt := NewTester(t, Fields().Row("Tên", TextBox()).Row(wideLong, ListBox(skinRowKey, skinRowName).SetItems(skinRows)).Wide(), 400, 400)
	if tb := fieldBox(tt, 0); !approx(tb.X, alone.X) || !approx(tb.W, alone.W) {
		t.Errorf("the text box is %v beside a long wide label, %v without it", tb, alone)
	}
	// So in a card: the other rows' controls keep their place.
	card := wbar("c1", 60, 24)
	tc := NewTester(t, Fields().SettingsRows().Row("Tên", card).Row(wideLong, MultilineTextBox()).Wide(), 400, 400)
	if c := find(t, tc.Tester, "c1"); !approx(c.X, cardRight-60) {
		t.Errorf("in a card beside a long wide label c1 is at x %v, want %v", c.X, cardRight-60)
	}
}

// Case 3: a wide row with an empty label builds no label: no node with an
// empty name, and its control's top is the row's.
func TestWideRowWithNoLabel(t *testing.T) {
	f := Fields().Row("Tên", TextBox()).Row("", Button("Thêm")).Wide()
	tt := NewTester(t, f, 400, 400)
	for _, n := range tt.Nodes() {
		if n.Label == "" && n.Role == ui.RoleText {
			t.Errorf("a text node with no name: %+v", n)
		}
	}
	tb, b := fieldBox(tt, 0), tt.Find(ByRole(ui.RoleButton)).Bounds()
	if top := tb.Y + tb.H + space(2); !approx(b.Y, top) {
		t.Errorf("the button's top is %v, want the row's %v", b.Y, top)
	}
	if !approx(b.X, 0) || !approx(b.W, 400) {
		t.Errorf("the button is %v, want across the form", b)
	}
	// A label given text later shows on the next frame.
	f.Label(1).SetText("Hàng")
	tt.Frame()
	if !tt.HasText("Hàng") {
		t.Errorf("the label set later does not show: %q", tt.Texts())
	}
}

// The label of a wide row that has no text is held all the same: giving it
// text redraws the window that shows the form.
func TestWideEmptyLabelRedraws(t *testing.T) {
	f := Fields().Row("Tên", TextBox()).Row("", Button("Thêm")).Wide()
	_, w := showFrame(t, f)
	redrawn := recordInvalidations(t)
	f.Label(1).SetText("Hàng")
	wantRedrawn(t, redrawn(), w)
}

// Case 4: a wide row's label, its description in TextMuted, then its
// control, top to bottom, Space(1) apart; the control's description is the
// text.
func TestWideRowDescribed(t *testing.T) {
	for _, settings := range []bool{false, true} {
		f := Fields().Row(wideKeys, MultilineTextBox()).Wide().Describe(nameDesc)
		if settings {
			f.SettingsRows()
		}
		tt := NewTester(t, f, 400, 400)
		l, d, c := find(t, tt.Tester, wideKeys), find(t, tt.Tester, nameDesc), fieldBox(tt, 0)
		if gap := d.Y - (l.Y + l.H); !approx(gap, space(1)) {
			t.Errorf("settings %v: the description %v is %v under the label %v, want %v", settings, d, gap, l, space(1))
		}
		if gap := c.Y - (d.Y + d.H); !approx(gap, space(1)) {
			t.Errorf("settings %v: the control %v is %v under the description %v, want %v", settings, c, gap, d, space(1))
		}
		if !approx(d.X, l.X) || !approx(c.X, l.X) {
			t.Errorf("settings %v: label %v, description %v and control %v do not start at one x", settings, l, d, c)
		}
		if got := descriptionOf(t, tt, wideKeys); got != nameDesc {
			t.Errorf("settings %v: the control's description is %q", settings, got)
		}
		g := Fields().Row(wideKeys, MultilineTextBox()).Wide().Describe(nameDesc)
		if settings {
			g.SettingsRows()
		}
		if muted, text := descColors(t, g); !muted || text {
			t.Errorf("settings %v: the description draws TextMuted %v and Text %v", settings, muted, text)
		}
	}
	// A hidden label takes its description with it; the control keeps it.
	h := Fields().Row(wideKeys, MultilineTextBox()).Wide().Describe(nameDesc)
	h.Label(0).SetVisible(false)
	th := NewTester(t, h, 400, 400)
	if th.HasText(nameDesc) || descriptionOf(t, th, "") != nameDesc {
		t.Errorf("a hidden wide label: texts %q, description %q", th.Texts(), descriptionOf(t, th, ""))
	}
	// Gap's column value replaces Space(1), as stacked.
	tt := NewTester(t, Fields().Gap(4, 20).Row(wideKeys, MultilineTextBox()).Wide(), 400, 400)
	if l, c := find(t, tt.Tester, wideKeys), fieldBox(tt, 0); !approx(c.Y-(l.Y+l.H), 20) {
		t.Errorf("with Gap(4, 20) the control is %v under its label, want 20", c.Y-(l.Y+l.H))
	}
}

// Case 5 and Review Focus 4: in a card, a wide row's control spans its
// inside; a divider between each pair of rows, none above the first or
// below the last, as for ordinary rows.
func TestWideRowInACard(t *testing.T) {
	f := Fields().SettingsRows().Row("A", Switch()).Row("Keys", MultilineTextBox()).Wide().Row("B", Switch())
	tt := NewTester(t, f, 400, 400)
	if n := len(cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)); n != 1 {
		t.Fatalf("%d cards, want 1", n)
	}
	m := fieldBox(tt, 0)
	if !approx(m.X, cardLeft) || !approx(m.X+m.W, cardRight) {
		t.Errorf("the text box %v, want from %v to %v", m, cardLeft, cardRight)
	}
	ys := dividersOutside(tt.Tester, m)
	if len(ys) != 2 {
		t.Fatalf("dividers at %v, want 2", ys)
	}
	a, k, b := find(t, tt.Tester, "A"), find(t, tt.Tester, "Keys"), find(t, tt.Tester, "B")
	if y := float32(ys[0]); y < a.Y+a.H || y > k.Y {
		t.Errorf("the first divider at %v is not between A %v and Keys %v", y, a, k)
	}
	if y := float32(ys[1]); y < m.Y+m.H || y > b.Y {
		t.Errorf("the second divider at %v is not between the text box %v and B %v", y, m, b)
	}

	// Wide rows first and last: no divider above the first, nor below the last.
	g := Fields().SettingsRows().Row("Keys", MultilineTextBox()).Wide().Row("B", Switch()).Row("Notes", MultilineTextBox()).Wide()
	tg := NewTester(t, g, 400, 500)
	first, last := fieldBox(tg, 0), fieldBox(tg, 1)
	var out []int
	for _, y := range dividersOutside(tg.Tester, first) {
		if float32(y) < last.Y-0.5 || float32(y) > last.Y+last.H+0.5 {
			out = append(out, y)
		}
	}
	if len(out) != 2 {
		t.Fatalf("wide first and last: dividers at %v, want 2", out)
	}
	ks := cardsOf(t, tg.Tester, cardX, ui.LightTheme().Border)
	if len(ks) != 1 || float32(out[0]) < first.Y+first.H || float32(out[1]) > last.Y {
		t.Errorf("wide first and last: dividers %v, text boxes %v and %v, card %+v", out, first, last, ks)
	}
}

// Case 6: under Calm a wide row in a card is at least SettingsRowHeight
// high, as every row is.
func TestWideRowHeightUnderCalm(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	least := skinhook.Get(false).SettingsRowHeight
	if least < 40 {
		t.Fatalf("Calm's SettingsRowHeight is %v", least)
	}
	f := Fields().SettingsRows().Row("", wbar("w", 60, 10)).Wide().Row("B", wbar("b", 60, 10))
	tt := NewTester(t, f, 400, 400)
	w, b := find(t, tt.Tester, "w"), find(t, tt.Tester, "b")
	if b.Y-w.Y < least {
		t.Errorf("the wide row is %v high, want at least SettingsRowHeight %v", b.Y-w.Y, least)
	}
	// The wide row's control starts the card's inside, the other ends it.
	if w.X > b.X-100 {
		t.Errorf("the wide control at x %v, the other at %v: want the wide one at the card's left", w.X, b.X)
	}
}

// Case 7 and Review Focus 5: a hidden wide row has no row and no divider, as
// for an ordinary row, and comes back with its divider when shown; a card
// whose only shown rows are wide draws.
func TestWideRowHidden(t *testing.T) {
	mtb := MultilineTextBox()
	f := Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Row("Keys", mtb).Wide().Row("B", wbar("c2", 60, 24))
	tt := NewTester(t, f, 400, 400)
	mtb.SetVisible(false)
	tt.Frame()
	without := NewTester(t, Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Row("B", wbar("c2", 60, 24)), 400, 400)
	if tt.HasText("Keys") || !sameDrawing(tt, without) {
		t.Errorf("hidden, the wide row still draws: %q", tt.Texts())
	}
	mtb.SetVisible(true)
	tt.Frame()
	m := fieldBox(tt, 0)
	if ys := dividersOutside(tt.Tester, m); !tt.HasText("Keys") || len(ys) != 2 {
		t.Errorf("shown again: texts %q, dividers %v, want Keys and 2", tt.Texts(), ys)
	}
	if !approx(m.X, cardLeft) || !approx(m.X+m.W, cardRight) {
		t.Errorf("shown again, the text box %v is not across the card", m)
	}

	only := NewTester(t, Fields().SettingsRows().Row("Keys", MultilineTextBox()).Wide().Row("B", Switch().SetVisible(false)), 400, 400)
	if n := len(cardsOf(t, only.Tester, cardX, ui.LightTheme().Border)); n != 1 {
		t.Errorf("a card of only a wide row: %d cards, want 1", n)
	}
	if m := fieldBox(only, 0); !approx(m.X, cardLeft) || !approx(m.X+m.W, cardRight) {
		t.Errorf("a card of only a wide row: the text box %v is not across it", m)
	}
}

// Case 8: stacked rows are all wide: Wide changes nothing there.
func TestWideRowStacked(t *testing.T) {
	build := func(wide bool) *Tester {
		f := Fields().Stacked().Row("Tên", TextBox())
		if wide {
			f.Wide()
		}
		f.Row(wideKeys, MultilineTextBox()).Describe(nameDesc)
		if wide {
			f.Wide()
		}
		return NewTester(t, f, 400, 400)
	}
	if !sameDrawing(build(true), build(false)) {
		t.Error("stacked, Wide changed the form")
	}
}

// Case 9: a wide row's mnemonic focuses its control, which its label names.
func TestWideRowMnemonic(t *testing.T) {
	withMac(t, false)
	for _, settings := range []bool{false, true} {
		g := Fields().Row("&Tên", TextBox()).Row("&Danh sách", MultilineTextBox()).Wide()
		if settings {
			g.SettingsRows()
		}
		f := NewForm("m", 400, 300)
		f.SetContent(g)
		tt := TestForm(t, f)
		tt.Settle()
		if !tt.Has(ByRole(ui.RoleTextField).Label(wideKeys)) {
			t.Errorf("settings %v: no text field named %q among %q", settings, wideKeys, fieldNodes(tt.Tester))
		}
		tt.Key(ui.Alt, ui.KeyD)
		tt.Settle()
		if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
			t.Errorf("settings %v: Alt+D did not focus the wide row's control", settings)
		}
	}
}

// Case 10: a control's Align places it across a wide row, in its own width;
// without one it stretches. VAlign does not move it across.
func TestWideRowAlign(t *testing.T) {
	for _, settings := range []bool{false, true} {
		left, right := float32(0), float32(400)
		if settings {
			left, right = cardLeft, cardRight
		}
		at := func(b *ButtonWidget) ui.Rect {
			f := Fields().Row("Tên", TextBox()).Row("", b).Wide()
			if settings {
				f.SettingsRows()
			}
			return NewTester(t, f, 400, 400).Find(ByRole(ui.RoleButton)).Bounds()
		}
		if b := at(Button("Thêm").Align(Start).SetSize(200, 0)); !approx(b.X, left) || !approx(b.W, 200) {
			t.Errorf("settings %v: Align(Start) puts the button at %v, want 200 wide at %v", settings, b, left)
		}
		if b := at(Button("Thêm").Align(End).SetSize(200, 0)); !approx(b.X+b.W, right) || !approx(b.W, 200) {
			t.Errorf("settings %v: Align(End) puts the button at %v, want 200 wide ending at %v", settings, b, right)
		}
		if b := at(Button("Thêm")); !approx(b.X, left) || !approx(b.X+b.W, right) {
			t.Errorf("settings %v: without Align the button is %v, want from %v to %v", settings, b, left, right)
		}
		if b := at(Button("Thêm").VAlign(End)); !approx(b.X, left) || !approx(b.X+b.W, right) {
			t.Errorf("settings %v: VAlign(End) moved the button across: %v", settings, b)
		}
	}
}

// A wide label's own Align places it across the cell, as an ordinary
// label's does.
func TestWideLabelAlign(t *testing.T) {
	for _, settings := range []bool{false, true} {
		left, right := float32(0), float32(400)
		if settings {
			left, right = cardLeft, cardRight
		}
		for _, a := range []Align{Start, Center, End} {
			f := Fields().Row(wideKeys, MultilineTextBox()).Wide()
			if settings {
				f.SettingsRows()
			}
			f.Label(0).Align(a)
			l := find(t, NewTester(t, f, 400, 400).Tester, wideKeys)
			var ok bool
			switch a {
			case Start:
				ok = approx(l.X, left)
			case Center:
				ok = approx(l.X+l.W/2, (left+right)/2)
			case End:
				ok = approx(l.X+l.W, right)
			}
			if !ok || l.W > 100 {
				t.Errorf("settings %v: a wide label with Align %v is at %v, across [%v, %v]", settings, a, l, left, right)
			}
		}
	}
}

// A row after a wide one places its control as it does with no wide row
// before it: its Align across its column, its VAlign along the row.
func TestWideRowLeavesTheNextRow(t *testing.T) {
	for _, settings := range []bool{false, true} {
		for _, name := range []string{"Align(Start)", "Align(Center)", "VAlign(End)", "VAlign(Center)"} {
			// at returns where the control of row B is, from its label's
			// top left, with a wide row before B or an ordinary one.
			at := func(wide bool) (dx, dy float32) {
				c := wbar("cb", 60, 10)
				switch name {
				case "Align(Start)":
					c.Align(Start)
				case "Align(Center)":
					c.Align(Center)
				case "VAlign(End)":
					c.VAlign(End)
				case "VAlign(Center)":
					c.VAlign(Center)
				}
				f := Fields().Row("A", wbar("ca", 60, 24))
				if wide {
					f.Wide()
				}
				// The description makes B's label taller than its control.
				f.Row("B", c).Describe(nameDesc + " " + nameDesc)
				if settings {
					f.SettingsRows()
				}
				tt := NewTester(t, f, 400, 400)
				b, cb := find(t, tt.Tester, "B"), find(t, tt.Tester, "cb")
				return cb.X - b.X, cb.Y - b.Y
			}
			wx, wy := at(true)
			px, py := at(false)
			if !approx(wx, px) || !approx(wy, py) {
				t.Errorf("settings %v, %s: after a wide row B's control is at (%v, %v) from its label, else (%v, %v)", settings, name, wx, wy, px, py)
			}
		}
	}
}

// Case 11: Wide before any row changes nothing; twice is once.
func TestWideEdges(t *testing.T) {
	a := NewTester(t, Fields().Wide().Row("Tên", TextBox()), 400, 300)
	b := NewTester(t, Fields().Row("Tên", TextBox()), 400, 300)
	if !sameDrawing(a, b) {
		t.Error("Wide before any row changed the form")
	}
	once := NewTester(t, Fields().Row("Tên", TextBox()).Wide(), 400, 300)
	twice := NewTester(t, Fields().Row("Tên", TextBox()).Wide().Wide(), 400, 300)
	if !sameDrawing(once, twice) {
		t.Error("Wide twice differs from once")
	}
	if sameDrawing(once, b) {
		t.Error("Wide changed nothing")
	}
}

// Wide makes the last row wide, not the one before it.
func TestWideTheLastRow(t *testing.T) {
	tt := NewTester(t, Fields().Row("Tên", TextBox()).Row("Mã", TextBox()).Wide(), 400, 300)
	a, b := fieldBox(tt, 0), fieldBox(tt, 1)
	if a.X < find(t, tt.Tester, "Tên").W || !approx(b.X, 0) || !approx(b.W, 400) {
		t.Errorf("text boxes %v and %v: want the first beside its label, the second across", a, b)
	}
}

// Case 12: a form with no wide row lays out as before Wide: its controls at
// the places main puts them.
// The controls are bars of a fixed size, so that the places do not hang on
// the font: in the plain grid each starts the control column, Space(3) after
// the widest of the labels and the description, a row under the one before;
// in cards each ends at the card's inner edge.
func TestWideNoneKeepsTheLayout(t *testing.T) {
	plain := NewTester(t, Fields().Row("Tên", wbar("c1", 60, 24)).Row(wideKeys, wbar("c2", 60, 24)).Describe(nameDesc).
		Row("Mã", wbar("c3", 60, 24)), 400, 400)
	col := max(find(t, plain.Tester, "Tên").W, find(t, plain.Tester, wideKeys).W, find(t, plain.Tester, nameDesc).W) + space(3)
	prev := float32(-100) // above the first row
	for _, s := range []string{"c1", "c2", "c3"} {
		c := find(t, plain.Tester, s)
		if !approx(c.X, col) {
			t.Errorf("plain: %s at x %v, want the control column's %v", s, c.X, col)
		}
		if c.Y < prev+24+space(2)-0.5 {
			t.Errorf("plain: %s at y %v, want a row under the one before, at %v", s, c.Y, prev)
		}
		prev = c.Y
	}
	cards := NewTester(t, Fields().SettingsRows().Row("Tên", wbar("c1", 60, 24)).Row(wideKeys, wbar("c2", 60, 24)).Describe(nameDesc).
		Row("Mã", wbar("c3", 60, 24)), 400, 400)
	for _, s := range []string{"c1", "c2", "c3"} {
		if c := find(t, cards.Tester, s); !approx(c.X, cardRight-60) {
			t.Errorf("cards: %s at x %v, want %v", s, c.X, cardRight-60)
		}
	}
	if ys := rules(cards.Tester, int(cardRight-60)-4); len(ys) != 2 {
		t.Errorf("cards: dividers at %v, want 2", ys)
	}
}

// Review Focus 1: a long wide label wraps at the form's width, not at the
// label column's.
func TestWideLabelWrapsAtTheForm(t *testing.T) {
	tt := NewTester(t, Fields().Row("Tên", TextBox()).Row(longLine, MultilineTextBox()).Wide(), 300, 400)
	l, tb := find(t, tt.Tester, longLine), fieldBox(tt, 0)
	one := find(t, tt.Tester, "Tên").H
	if l.H < 2*one || l.W > 300.5 {
		t.Errorf("the long label %v: want wrapped, within 300", l)
	}
	if l.W < 200 || l.W <= tb.X {
		t.Errorf("the long label %v wraps at the label column (the text box at %v)", l, tb.X)
	}
	if m := fieldBox(tt, 1); m.Y < l.Y+l.H-0.5 || !approx(m.X, 0) || !approx(m.W, 300) {
		t.Errorf("the wide control %v is not under the wrapped label %v, across the form", m, l)
	}
}

// Review Focus 2: at text scale 1.5 nothing of a wide row in a card leaves
// it.
func TestWideRowAtTextScale(t *testing.T) {
	f := Fields().SettingsRows().Section("S").Row(wideKeys, MultilineTextBox()).Wide().Describe(nameDesc)
	tt := NewTester(t, f, 400, 500)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	k := cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)[0]
	l, d, m := find(t, tt.Tester, wideKeys), find(t, tt.Tester, nameDesc), fieldBox(tt, 0)
	for _, r := range []ui.Rect{l, d, m} {
		if !k.inside(r) {
			t.Errorf("%v leaves the card %+v at text scale 1.5", r, k)
		}
	}
	if m.Y < d.Y+d.H || !approx(m.X, cardLeft) || !approx(m.X+m.W, cardRight) {
		t.Errorf("at text scale 1.5 the text box %v is not under the description %v, across the card", m, d)
	}
}

// Review Focus 3: a list in a wide row of a card fills the card's inside,
// not past its padding.
func TestWideListFillsTheCard(t *testing.T) {
	f := Fields().SettingsRows().Row("A", Switch()).Row("Members", ListBox(skinRowKey, skinRowName).SetItems(skinRows)).Wide()
	tt := NewTester(t, f, 400, 400)
	if b := wideList(tt); !approx(b.X, cardLeft) || !approx(b.X+b.W, cardRight) {
		t.Errorf("the list %v, want from %v to %v", b, cardLeft, cardRight)
	}
	lv := ListView(skinRowKey, Column[skinRow]{Title: "Name", Text: skinRowName}).SetRows(skinRows)
	g := Fields().SettingsRows().Row("A", Switch()).Row("People", lv).Wide()
	tg := NewTester(t, g, 400, 400)
	if b := nodeOf(t, tg.Tester, ui.RoleTable, "People").Bounds; !approx(b.X, cardLeft) || !approx(b.X+b.W, cardRight) {
		t.Errorf("the list view %v, want from %v to %v", b, cardLeft, cardRight)
	}
}

// Wide may be called from any goroutine while the form draws.
func TestWideFromGoroutines(t *testing.T) {
	f := Fields().Row("Tên", TextBox())
	tt := NewTester(t, f, 400, 300)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() { f.Wide() })
	}
	tt.Frame()
	wg.Wait()
	tt.Frame()
	if b := fieldBox(tt, 0); !approx(b.X, 0) || !approx(b.W, 400) {
		t.Errorf("the text box %v, want across the form", b)
	}
}

// wideAllocs returns the allocations of a frame of a form of n rows of
// multiline text boxes, in cards of 5, described, wide or not.
func wideAllocs(n int, wide bool) float64 {
	f := Fields().SettingsRows()
	for i := range n {
		if i%5 == 0 {
			f.Section("Section")
		}
		f.Row("Name", MultilineTextBox().SetTooltip("Shown to others.")).Describe("As the account shows it.")
		if wide {
			f.Wide()
		}
	}
	tt := ui.NewTester(content(ScrollPanel(f)), 1280, 800)
	for range 10 {
		tt.Frame()
	}
	return allocrun.Least(30, tt.Frame)
}

// A form of wide rows, once built, allocates per row no more in a frame than
// the same form without Wide: the slope, 200 rows less 100, per row.
func TestWideAllocatesNoMore(t *testing.T) {
	plain := (wideAllocs(200, false) - wideAllocs(100, false)) / 100
	wide := (wideAllocs(200, true) - wideAllocs(100, true)) / 100
	t.Logf("%.2f allocations per wide row in a frame, %.2f per ordinary row", wide, plain)
	if wide > plain+0.2 {
		t.Errorf("%.2f allocations per wide row in a frame, %.2f per ordinary row", wide, plain)
	}
	// testing.AllocsPerRun on a built form of wide rows: as few as without
	// (none, here), but for one of the runtime's or another goroutine's.
	build := func(wide bool) func() {
		f := Fields().SettingsRows()
		for range 20 {
			f.Row("Name", MultilineTextBox()).Describe("As the account shows it.")
			if wide {
				f.Wide()
			}
		}
		tt := ui.NewTester(content(ScrollPanel(f)), 1280, 800)
		for range 10 {
			tt.Frame()
		}
		return tt.Frame
	}
	p, w := allocrun.Least(20, build(false)), allocrun.Least(20, build(true))
	t.Logf("AllocsPerRun: %.1f with 20 wide rows, %.1f without", w, p)
	if w > p+1 {
		t.Errorf("AllocsPerRun: %.1f with 20 wide rows, %.1f without", w, p)
	}
}
