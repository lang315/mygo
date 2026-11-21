package widget

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// A row's action (P2h): a small control at the right end of a wide row's
// label line, or after a grid row's control. The forms are 400 DIPs wide.

// help is the action of the tests: a link that runs its handler.
func help() *LinkLabelWidget { return LinkLabel("Help", "").OnClick(func() {}) }

// linkBox returns the bounds of the link.
func linkBox(tt *Tester) ui.Rect { return tt.Find(ByRole(ui.RoleLink)).Bounds() }

// midY is the vertical centre of r.
func midY(r ui.Rect) float32 { return r.Y + r.H/2 }

// wantLabelLine checks case 1 in a form whose inside runs from left to right:
// the link ends the label's line, centred on it, and the box is under both,
// across the inside.
func wantLabelLine(t *testing.T, tt *Tester, left, right float32) {
	t.Helper()
	l, k, b := find(t, tt.Tester, "Content"), linkBox(tt), fieldBox(tt, 0)
	if !approx(k.X+k.W, right) {
		t.Errorf("the link %v ends at %v, want the right edge %v", k, k.X+k.W, right)
	}
	if !approx(midY(k), midY(l)) {
		t.Errorf("the link's centre at y %v, the label's at %v", midY(k), midY(l))
	}
	if !approx(l.X, left) || l.X+l.W > k.X {
		t.Errorf("the label %v, want from %v, left of the link %v", l, left, k)
	}
	if b.Y < l.Y+l.H-0.5 || b.Y < k.Y+k.H-0.5 {
		t.Errorf("the box %v is not under the label %v and the link %v", b, l, k)
	}
	if !approx(b.X, left) || !approx(b.X+b.W, right) {
		t.Errorf("the box %v, want from %v to %v", b, left, right)
	}
}

// Case 1.
func TestActionEndsAWideRowsLabelLine(t *testing.T) {
	f := Fields().Row("Tên", TextBox()).Row("Content", MultilineTextBox()).Wide().Action(help())
	tt := NewTester(t, f, 400, 400)
	l, b := find(t, tt.Tester, "Content"), fieldBox(tt, 1)
	if !approx(linkBox(tt).X+linkBox(tt).W, 400) || !approx(midY(linkBox(tt)), midY(l)) {
		t.Errorf("the link %v, the label %v: want the link at the right edge, on the label's line", linkBox(tt), l)
	}
	if b.Y < linkBox(tt).Y+linkBox(tt).H-0.5 || !approx(b.X, 0) || !approx(b.W, 400) {
		t.Errorf("the box %v is not under the link, across the form", b)
	}
	// A button, taller than the label, is centred on it too.
	tb := NewTester(t, Fields().Row("Content", MultilineTextBox()).Wide().Action(Button("+")), 400, 400)
	if l, k := find(t, tb.Tester, "Content"), tb.Find(ByLabel("+")).Bounds(); k.H <= l.H || !approx(midY(k), midY(l)) || !approx(k.X+k.W, 400) {
		t.Errorf("the button %v, the label %v: want the button at the right edge, centred on the label", k, l)
	}
	// Alone in the form, every edge.
	wantLabelLine(t, NewTester(t, Fields().Row("Content", MultilineTextBox()).Wide().Action(help()), 400, 400), 0, 400)
}

// Case 2.
func TestActionEndsAStackedRowsLabelLine(t *testing.T) {
	f := Fields().Stacked().Row("Content", MultilineTextBox()).Action(help())
	wantLabelLine(t, NewTester(t, f, 400, 400), 0, 400)
}

// Case 3: inside the card's padding.
func TestActionEndsAWideRowsLabelLineInACard(t *testing.T) {
	f := Fields().SettingsRows().Row("Content", MultilineTextBox()).Wide().Action(help())
	wantLabelLine(t, NewTester(t, f, 400, 400), cardLeft, cardRight)
}

// Case 4: a grid row's action follows its control, a column gap after it,
// and the control starts where it does with no action.
func TestActionFollowsAGridRowsControl(t *testing.T) {
	alone := NewTester(t, Fields().Row("Kind", combo()), 400, 300).Find(ByRole(ui.RolePopUpButton)).Bounds()
	tt := NewTester(t, Fields().Row("Kind", combo()).Action(Button("+")), 400, 300)
	c, b := tt.Find(ByRole(ui.RolePopUpButton)).Bounds(), tt.Find(ByLabel("+")).Bounds()
	if !approx(c.X, alone.X) {
		t.Errorf("the combo at x %v, %v with no action", c.X, alone.X)
	}
	if !approx(b.X, c.X+c.W+space(3)) {
		t.Errorf("the button at x %v, want one column gap (%v) past the combo's right edge %v", b.X, space(3), c.X+c.W)
	}
	if !approx(b.X+b.W, 400) || b.W > 100 {
		t.Errorf("the button %v, want its own width at the form's right edge", b)
	}
}

// Beside a label made tall by its description, a grid row's control keeps
// the place it has with no action: the row's top, or where its VAlign puts
// it; in a card, the label's middle.
func TestActionKeepsTheControlsPlaceDown(t *testing.T) {
	desc := nameDesc + " " + nameDesc
	// at returns the control and the label's cell, from the label's top to
	// its description's bottom.
	at := func(settings bool, v Align, act bool) (c, cell ui.Rect) {
		w := wbar("c", 60, 10)
		if v != 0 {
			w.VAlign(v)
		}
		f := Fields().Row("Kind", w).Describe(desc)
		if act {
			f.Action(Button("+"))
		}
		if settings {
			f.SettingsRows()
		}
		tt := NewTester(t, f, 240, 300)
		l, d := find(t, tt.Tester, "Kind"), find(t, tt.Tester, desc)
		return find(t, tt.Tester, "c"), ui.Rect{X: l.X, Y: l.Y, W: l.W, H: d.Y + d.H - l.Y}
	}
	if c, _ := at(false, 0, true); !approx(c.Y, 0) {
		t.Errorf("the control is at y %v beside an action, want the row's top, 0", c.Y)
	}
	for _, v := range []Align{0, Center, End} {
		with, _ := at(false, v, true)
		if alone, _ := at(false, v, false); !approx(with.Y, alone.Y) {
			t.Errorf("VAlign %v: the control at y %v beside an action, %v alone", v, with.Y, alone.Y)
		}
	}
	// In a card the label column narrows by the action's width, so the
	// label wraps further: the control is at the middle of the label's cell,
	// or at its bottom with VAlign(End), as alone.
	for _, v := range []Align{0, Center, End} {
		c, cell := at(true, v, true) // c is the bar's text, at its top: the bar is 10 high
		ok := approx(c.Y+5, midY(cell))
		if v == End {
			ok = approx(c.Y+10, cell.Y+cell.H)
		}
		if !ok {
			t.Errorf("settings, VAlign %v: the control %v beside the label's cell %v", v, c, cell)
		}
	}
}

// Gap's column value is the gap before a grid row's action.
func TestActionGapFollowsGap(t *testing.T) {
	tt := NewTester(t, Fields().Gap(4, 20).Row("Kind", combo()).Action(Button("+")), 400, 300)
	c, b := tt.Find(ByRole(ui.RolePopUpButton)).Bounds(), tt.Find(ByLabel("+")).Bounds()
	if !approx(b.X-(c.X+c.W), 20) {
		t.Errorf("the button is %v past the combo, want 20", b.X-(c.X+c.W))
	}
}

// In SettingsRows an ordinary row's control and action end at the card's
// inner edge, the control a column gap before the action.
func TestActionInASettingsRow(t *testing.T) {
	f := Fields().SettingsRows().Row("A", wbar("c1", 60, 24)).Action(wbar("a1", 30, 24))
	tt := NewTester(t, f, 400, 300)
	c, a := find(t, tt.Tester, "c1"), find(t, tt.Tester, "a1")
	if !approx(a.X, cardRight-30) {
		t.Errorf("the action at x %v, want %v", a.X, cardRight-30)
	}
	if !approx(c.X, cardRight-30-space(3)-60) {
		t.Errorf("the control at x %v, want %v", c.X, cardRight-30-space(3)-60)
	}
	// In a narrow card the text box gives the action its room: the two and
	// the gap take the width the box takes alone, and the action keeps its
	// own and ends at the card's inner edge (220 less its line and padding).
	alone := fieldBox(NewTester(t, Fields().SettingsRows().Row("Notifications for every message", TextBox()), 220, 300), 0)
	tn := NewTester(t, Fields().SettingsRows().Row("Notifications for every message", TextBox()).Action(Button("+")), 220, 300)
	tb, b := fieldBox(tn, 0), tn.Find(ByLabel("+")).Bounds()
	if !approx(b.X+b.W, 220-1-space(3)) || !approx(tb.X, alone.X) || !approx(b.X, tb.X+tb.W+space(3)) {
		t.Errorf("in a narrow card: the text box %v and the button %v, the box alone %v", tb, b, alone)
	}
	if wide := NewTester(t, Fields().SettingsRows().Row("A", Button("+")), 400, 300).Find(ByLabel("+")).Bounds(); !approx(b.W, wide.W) {
		t.Errorf("in a narrow card the button is %v wide, %v in a wide one", b.W, wide.W)
	}}

// Case 5: before any row Action does nothing; a second replaces the first;
// nil removes it.
func TestActionEdges(t *testing.T) {
	plain := func() *Tester { return NewTester(t, Fields().Row("Content", TextBox()), 400, 300) }
	before := NewTester(t, Fields().Action(help()).Row("Content", TextBox()), 400, 300)
	if before.Has(ByRole(ui.RoleLink)) || !sameDrawing(before, plain()) {
		t.Error("Action before any row changed the form")
	}
	two := LinkLabel("Two", "")
	g := Fields().Row("Content", TextBox()).Action(LinkLabel("One", "")).Action(two)
	tt := NewTester(t, g, 400, 300)
	if tt.HasText("One") || !tt.HasText("Two") || len(tt.FindAll(ByRole(ui.RoleLink))) != 1 {
		t.Errorf("after a second Action: texts %q, want Two alone", tt.Texts())
	}
	if cs := g.children(); len(cs) != 3 || cs[2] != two {
		t.Errorf("children %d after a second Action, want the label, the box and Two", len(g.children()))
	}
	none := NewTester(t, Fields().Row("Content", TextBox()).Action(help()).Action(nil), 400, 300)
	if none.Has(ByRole(ui.RoleLink)) || !sameDrawing(none, plain()) {
		t.Error("Action(nil) left the form changed")
	}
	if n := len(Fields().Row("Content", TextBox()).Action(help()).Action(nil).children()); n != 2 {
		t.Errorf("%d children after Action(nil), want 2", n)
	}
	wide := NewTester(t, Fields().Row("Content", TextBox()).Wide().Action(help()).Action(nil), 400, 300)
	if !sameDrawing(wide, NewTester(t, Fields().Row("Content", TextBox()).Wide(), 400, 300)) {
		t.Error("Action(nil) on a wide row left the form changed")
	}
	// No row is built around a grid row's control: beside a label made tall
	// by its description, it keeps the top of its row, where the grid puts it.
	tall := NewTester(t, Fields().Row("Kind", wbar("c", 60, 10)).Describe(nameDesc+" "+nameDesc).Action(help()).Action(nil), 200, 300)
	if c := find(t, tall.Tester, "c"); !approx(c.Y, 0) {
		t.Errorf("after Action(nil) the control is at y %v, want the row's top, 0", c.Y)
	}
	// No row is built around the label: it is as wide as its text, not the form.
	if l := find(t, wide.Tester, "Content"); l.W > 100 {
		t.Errorf("after Action(nil) the wide label is %v, wider than its text", l)
	}
}

// Case 6: the row's key focuses the box, not the link, and its name goes to
// the box only.
func TestActionRowKeyFocusesTheControl(t *testing.T) {
	withMac(t, false)
	for name, lay := range fieldsKinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, lay(Fields().Row("&Content", TextBox())).Action(help()))
			altKey(tt, ui.KeyC)
			if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
				t.Error("Alt+C did not focus the box")
			}
			if tt.Find(ByRole(ui.RoleLink)).Focused() {
				t.Error("Alt+C focused the link")
			}
			if !tt.Has(ByRole(ui.RoleTextField).Label("Content")) || !tt.Has(ByRole(ui.RoleLink).Label("Help")) {
				t.Errorf("names: %q", fieldNodes(tt.Tester))
			}
			if n := len(tt.FindAll(ByLabel("Content"))); n != 2 {
				t.Errorf("%d nodes carry the label, want the label and the box", n)
			}
		})
	}
}

// focusOrder returns the roles that take the focus, from the first Tab on,
// for n Tabs.
func focusOrder(tt *Tester, n int) (rs []ui.Role) {
	for range n {
		tt.Key(0, ui.KeyTab)
		tt.Settle()
		for _, nd := range tt.Tester.Nodes() {
			if nd.Focused {
				rs = append(rs, nd.Role)
			}
		}
	}
	return rs
}

// before reports whether a comes before b in rs, both in it.
func before(rs []ui.Role, a, b ui.Role) bool {
	ia, ib := -1, -1
	for i, r := range rs {
		if r == a && ia < 0 {
			ia = i
		}
		if r == b && ib < 0 {
			ib = i
		}
	}
	return ia >= 0 && ib >= 0 && ia < ib
}

// Case 7: a wide row's link comes before its box (reading order), a grid
// row's control before its action.
func TestActionTabOrder(t *testing.T) {
	wide := formOf(t, Fields().Row("Content", TextBox()).Wide().Action(help()))
	if rs := focusOrder(wide, 3); !before(rs, ui.RoleLink, ui.RoleTextField) {
		t.Errorf("wide: focus went %v, want the link then the box", rs)
	}
	stacked := formOf(t, Fields().Stacked().Row("Content", TextBox()).Action(help()))
	if rs := focusOrder(stacked, 3); !before(rs, ui.RoleLink, ui.RoleTextField) {
		t.Errorf("stacked: focus went %v, want the link then the box", rs)
	}
	grid := formOf(t, Fields().Row("Content", TextBox()).Action(help()))
	if rs := focusOrder(grid, 3); !before(rs, ui.RoleTextField, ui.RoleLink) {
		t.Errorf("grid: focus went %v, want the box then the link", rs)
	}
}

// Case 8: a hidden control hides its row, the action too, and the row comes
// back with it.
func TestActionHiddenWithItsControl(t *testing.T) {
	for _, wide := range []bool{false, true} {
		box := TextBox()
		f := Fields().Row("Tên", TextBox()).Row("Content", box).Action(help())
		w := Fields().Row("Tên", TextBox())
		if wide {
			f.Wide()
		}
		box.SetVisible(false)
		tt := NewTester(t, f, 400, 300)
		if tt.Has(ByRole(ui.RoleLink)) || !sameDrawing(tt, NewTester(t, w, 400, 300)) {
			t.Errorf("wide %v: the hidden row still draws: %q", wide, tt.Texts())
		}
		box.SetVisible(true)
		tt.Frame()
		if !tt.Has(ByRole(ui.RoleLink)) {
			t.Errorf("wide %v: shown again, no link: %q", wide, tt.Texts())
		}
	}
}

// The action of a hidden row is held: changing it redraws the window that
// shows the form.
func TestActionOfAHiddenRowRedraws(t *testing.T) {
	box, l := TextBox(), help()
	f := Fields().Row("Tên", TextBox()).Row("Content", box).Action(l)
	box.SetVisible(false)
	_, w := showFrame(t, f)
	redrawn := recordInvalidations(t)
	l.SetVisible(false)
	wantRedrawn(t, redrawn(), w)
}

// A hidden action is held: showing it redraws the window that shows the form.
func TestActionHiddenRedrawsWhenShown(t *testing.T) {
	l := help().SetVisible(false)
	_, w := showFrame(t, Fields().Row("Content", TextBox()).Wide().Action(l))
	redrawn := recordInvalidations(t)
	l.SetVisible(true)
	wantRedrawn(t, redrawn(), w)
}

// A hidden action leaves its row as it is with none.
func TestActionHiddenItself(t *testing.T) {
	for _, wide := range []bool{false, true} {
		f := Fields().Row("Content", TextBox()).Action(help().SetVisible(false))
		g := Fields().Row("Content", TextBox())
		if wide {
			f.Wide()
			g.Wide()
		}
		if !sameDrawing(NewTester(t, f, 400, 300), NewTester(t, g, 400, 300)) {
			t.Errorf("wide %v: a hidden action changed the row", wide)
		}
	}
	// No row is built around a grid row's control: it keeps its row's top.
	tall := NewTester(t, Fields().Row("Kind", wbar("c", 60, 10)).Describe(nameDesc+" "+nameDesc).Action(help().SetVisible(false)), 200, 300)
	if c := find(t, tall.Tester, "c"); !approx(c.Y, 0) {
		t.Errorf("beside a hidden action the control is at y %v, want the row's top, 0", c.Y)
	}
}

// A wide row with no label text: the link alone, at the right end.
func TestActionOfAWideRowWithNoLabel(t *testing.T) {
	tt := NewTester(t, Fields().Row("", MultilineTextBox()).Wide().Action(help()), 400, 300)
	if k, b := linkBox(tt), fieldBox(tt, 0); !approx(k.X+k.W, 400) || b.Y < k.Y+k.H-0.5 {
		t.Errorf("the link %v, the box %v: want the link at the right edge, above the box", k, b)
	}
}

// Case 9.
func TestActionClickRunsItsHandlerOnce(t *testing.T) {
	var n atomic.Int32
	l := LinkLabel("Help", "").OnClick(func() { n.Add(1) })
	tt := NewTester(t, Fields().Row("Content", MultilineTextBox()).Wide().Action(l), 400, 300)
	tt.Find(ByRole(ui.RoleLink)).Click()
	<-posted()
	if got := n.Load(); got != 1 {
		t.Errorf("the handler ran %d times, want 1", got)
	}
}

// Review Focus 1: at text scale 1.5 in a form 240 wide, the label wraps
// before it reaches the link, which keeps its width.
func TestActionAtTextScaleInANarrowForm(t *testing.T) {
	scaled := func(w int, label string) *Tester {
		tt := NewTester(t, Fields().Row(label, MultilineTextBox()).Wide().Action(help()), w, 400)
		tt.SetPreferences(ui.Preferences{TextScale: 1.5})
		tt.Frame()
		return tt
	}
	natural := linkBox(scaled(400, "C"))
	tt := scaled(240, longLine)
	l, k := find(t, tt.Tester, longLine), linkBox(tt)
	if l.X+l.W > k.X+0.5 {
		t.Errorf("the label %v reaches the link %v", l, k)
	}
	if !approx(k.W, natural.W) || !approx(k.H, natural.H) || !approx(k.X+k.W, 240) {
		t.Errorf("the link %v, want %vx%v at the right edge 240", k, natural.W, natural.H)
	}
	if one := find(t, scaled(240, "C").Tester, "C").H; l.H < 2*one {
		t.Errorf("the label %v did not wrap", l)
	}
	// A label of one word too long to fit clips, and the link keeps its width.
	word := scaled(240, "Kiểmtrađộdàicủamộttừkhôngthểngắtdòngđược")
	k = linkBox(word)
	if !approx(k.W, natural.W) || !approx(k.X+k.W, 240) {
		t.Errorf("beside a long word the link is %v, want %v wide at the right edge 240", k, natural.W)
	}
	// The gap before the link is empty: the word is clipped there.
	img := word.Image()
	bg := img.RGBAAt(1, img.Bounds().Max.Y-1)
	inked := false // the word draws left of the gap: the image is in DIPs
	for x := 0; x < int(k.X-space(3)) && !inked; x++ {
		for y := int(k.Y); y < int(k.Y+k.H); y++ {
			inked = inked || img.RGBAAt(x, y) != bg
		}
	}
	if !inked {
		t.Fatal("the long word draws nothing left of the link")
	}
	for x := int(k.X - space(3) + 1); x < int(k.X)-1; x++ {
		for y := int(k.Y); y < int(k.Y+k.H); y++ {
			if img.RGBAAt(x, y) != bg {
				t.Fatalf("the long word draws at (%d, %d), in the gap before the link %v", x, y, k)
			}
		}
	}
	// A button beside the wrapped label keeps its own height, centred on it.
	tb := NewTester(t, Fields().Row(longLine, MultilineTextBox()).Wide().Action(Button("+")), 240, 400)
	tb.SetPreferences(ui.Preferences{TextScale: 1.5})
	tb.Frame()
	one := NewTester(t, Button("+"), 240, 400)
	one.SetPreferences(ui.Preferences{TextScale: 1.5})
	one.Frame()
	b, bl, own := tb.Find(ByLabel("+")).Bounds(), find(t, tb.Tester, longLine), one.Find(ByLabel("+")).Bounds()
	if !approx(b.H, own.H) || !approx(midY(b), midY(bl)) {
		t.Errorf("the button %v beside the wrapped label %v, want %v high, centred on it", b, bl, own.H)
	}
}

// Review Focus 2: a disabled form disables the action with its row.
func TestActionDisabledWithTheForm(t *testing.T) {
	var n atomic.Int32
	l := LinkLabel("Help", "").OnClick(func() { n.Add(1) })
	f := Fields().Row("Content", TextBox()).Wide().Action(l)
	f.SetDisabled(true)
	tt := NewTester(t, f, 400, 300)
	if nd := nodeOf(t, tt.Tester, ui.RoleLink, "Help"); !nd.Disabled {
		t.Error("the link is not disabled")
	}
	k := linkBox(tt)
	tt.ClickAt(k.X+k.W/2, midY(k))
	tt.Settle()
	<-posted()
	if n.Load() != 0 {
		t.Error("a click ran the disabled link's handler")
	}
}

// Review Focus 3: Action from goroutines after the first frame shows on the
// next.
func TestActionFromGoroutines(t *testing.T) {
	f := Fields().Row("Content", TextBox()).Wide()
	tt := NewTester(t, f, 400, 300)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() { f.Action(help()) })
	}
	tt.Frame()
	wg.Wait()
	tt.Frame()
	if n := len(tt.FindAll(ByRole(ui.RoleLink))); n != 1 || !approx(linkBox(tt).X+linkBox(tt).W, 400) {
		t.Errorf("%d links, want 1 at the right edge", n)
	}
}

// builtFrame returns the frame of c, built: drawn 10 times first.
func builtFrame(c Control) func() {
	tt := ui.NewTester(content(ScrollPanel(c)), 1280, 800)
	for range 10 {
		tt.Frame()
	}
	return tt.Frame
}

// actionForm is a form of 20 rows, every other one wide, each with the
// action act makes, or none when act is nil.
func actionForm(lay func(*FieldsWidget) *FieldsWidget, act func() Control) *FieldsWidget {
	f := lay(Fields())
	for i := range 20 {
		f.Row("Name", MultilineTextBox())
		if act != nil {
			f.Action(act())
		}
		if i%2 == 0 {
			f.Wide()
		}
	}
	return f
}

// Review Focus 4: a built form with actions allocates nothing in a frame, in
// each layout. A link allocates once a frame wherever it is (ui.Link's own,
// before P2h): in a form, the same as in a box.
func TestActionAllocatesNothingInAFrame(t *testing.T) {
	// Widget ids past 255, as in an app: Go boxes a smaller integer with no
	// allocation, a larger one with one.
	for range 300 {
		Label("")
	}
	button := func() Control { return Button("+") }
	for name, lay := range map[string]func(*FieldsWidget) *FieldsWidget{
		"grid":     func(f *FieldsWidget) *FieldsWidget { return f },
		"stacked":  func(f *FieldsWidget) *FieldsWidget { return f.Stacked() },
		"settings": func(f *FieldsWidget) *FieldsWidget { return f.SettingsRows() },
	} {
		for _, act := range []func() Control{nil, button} {
			frame := builtFrame(actionForm(lay, act))
			if n := allocrun.Least(30, frame); n != 0 {
				t.Errorf("%s, action %v: %v allocations in a frame, want 0", name, act != nil, n)
			}
			if n := testing.AllocsPerRun(20, frame); n >= 1 {
				t.Errorf("%s, action %v: AllocsPerRun %v, want 0", name, act != nil, n)
			}
		}
	}
	links := VBox(0)
	for range 20 {
		links.Add(LinkLabel("Help", "https://example.com"))
	}
	inBox := allocrun.Least(30, builtFrame(links))
	inForm := allocrun.Least(30, builtFrame(actionForm(fieldsKinds["grid"], func() Control { return LinkLabel("Help", "https://example.com") })))
	t.Logf("20 links: %v allocations a frame in a box, %v in a form", inBox, inForm)
	if inForm != inBox {
		t.Errorf("20 links: %v allocations a frame in a form, %v in a box", inForm, inBox)
	}
}

// Review Focus 5: a row's key with a container as its control focuses the
// container's first control, never the action.
func TestActionRowKeyOfAContainer(t *testing.T) {
	withMac(t, false)
	for name, lay := range fieldsKinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, lay(Fields().Row("&Kịch bản", comboPlus())).Action(help()))
			altKey(tt, ui.KeyK)
			if !tt.Find(ByRole(ui.RolePopUpButton)).Focused() {
				t.Error("Alt+K did not focus the combo")
			}
			if tt.Find(ByRole(ui.RoleLink)).Focused() {
				t.Error("Alt+K focused the action")
			}
		})
	}
}
