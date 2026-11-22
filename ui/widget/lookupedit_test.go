package widget

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// cust is the record of the cases.
type cust struct {
	ID               int
	Code, Name, City string
}

// custs are the records of the cases.
func custs() []cust {
	return []cust{{1, "C01", "Acme", "Hanoi"}, {2, "C02", "Beta", "Hue"}, {3, "C03", "Acorn", "Hanoi"}}
}

func custKey(c *cust) int     { return c.ID }
func custName(c *cust) string { return c.Name }

// custCols are the columns of the cases: Code 60 wide, Name sharing the
// room left, City 80 wide.
func custCols() []Column[cust] {
	return []Column[cust]{
		{Title: "Code", Text: func(c *cust) string { return c.Code }, Width: 60},
		{Title: "Name", Text: custName},
		{Title: "City", Text: func(c *cust) string { return c.City }, Width: 80},
	}
}

// custLookUp is the LookUpEdit of the cases, holding custs.
func custLookUp() *LookUpEditWidget[cust, int] {
	return LookUpEdit(custKey, custName, custCols()...).SetItems(custs())
}

// pick is a run of OnSelect.
type pick struct {
	k  int
	ok bool
}

// picks records the runs of OnSelect.
type picks struct {
	mu  sync.Mutex
	got []pick
}

func (p *picks) add(k int, ok bool) { p.mu.Lock(); p.got = append(p.got, pick{k, ok}); p.mu.Unlock() }

func (p *picks) all(t *testing.T) []pick {
	t.Helper()
	flushEvents(t)
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.got)
}

// lookTester shows w in a form of 400 by 300, above a button "Other", with
// a control that keeps the frame's theme in the returned Theme.
func lookTester(t *testing.T, w Control) (*Tester, *ui.Theme) {
	t.Helper()
	calmOff()
	th := new(ui.Theme)
	keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
	tt := NewTester(t, VBox(0).Add(keep, w, Button("Other")), 400, 300)
	tt.Settle()
	return tt, th
}

// lookField is the LookUpEdit's input.
func lookField(tt *Tester) *Node { return tt.Find(ByRole(ui.RoleComboBox)) }

// lookKeys presses keys where the focus is, settling after each. A Node's
// Key would focus the field first, which fails while an option is
// highlighted: assistive technology then has the focus on the option
// (ActiveDescendant), not on the input.
func lookKeys(tt *Tester, keys ...ui.Key) {
	for _, k := range keys {
		tt.Tester.Key(0, k)
		tt.Settle()
	}
}

// lookType types s where the focus is, and settles.
func lookType(tt *Tester, s string) {
	tt.Tester.Type(s)
	tt.Settle()
}

// lookOpen reports whether the drop-down shows.
func lookOpen(tt *Tester) bool { return tt.Has(ByRole(ui.RoleList)) }

// lookRows are the drop-down's record rows.
func lookRows(tt *Tester) []*Node { return tt.FindAll(ByRole(ui.RoleListItem)) }

// textAt is the box of the text s, failing the test without one.
func textAt(t *testing.T, tt *Tester, s string) ui.Rect {
	t.Helper()
	r, ok := tt.Tester.Find(s)
	if !ok {
		t.Fatalf("no text %q among %q", s, tt.Texts())
	}
	return r
}

func within(a, b, d float32) bool { return math.Abs(float64(a-b)) <= float64(d) }

// wantSelected fails the test unless Selected is (k, ok).
func wantSelected(t *testing.T, l *LookUpEditWidget[cust, int], k int, ok bool) {
	t.Helper()
	if gk, gok := l.Selected(); gk != k || gok != ok {
		t.Fatalf("Selected (%v, %v), want (%v, %v)", gk, gok, k, ok)
	}
}

// Case 1.
func TestLookUpEditOpensColumns(t *testing.T) {
	l := custLookUp()
	tt, th := lookTester(t, l)
	lookField(tt).Click()
	if !lookOpen(tt) {
		t.Fatal("a click on the field did not open the drop-down")
	}
	if n := len(lookRows(tt)); n != 3 {
		t.Fatalf("%d rows, want 3", n)
	}
	code, name, city := textAt(t, tt, "Code"), textAt(t, tt, "Name"), textAt(t, tt, "City")
	if !within(code.Y, name.Y, 0.5) || !within(code.Y, city.Y, 0.5) {
		t.Fatalf("the titles are not on one row: %v %v %v", code, name, city)
	}
	first := lookRows(tt)[0].Bounds()
	if code.Y+code.H > first.Y+0.5 {
		t.Fatalf("the header (to %v) is not above the first row (at %v)", code.Y+code.H, first.Y)
	}
	gap := space(2)
	for _, r := range [][3]string{{"C01", "Acme", "Hanoi"}, {"C02", "Beta", "Hue"}, {"C03", "Acorn", "Hanoi"}} {
		c0, c1, c2 := textAt(t, tt, r[0]), textAt(t, tt, r[1]), textAt(t, tt, r[2])
		if !within(c0.X, code.X, 1) || !within(c1.X, name.X, 1) || !within(c2.X, city.X, 1) {
			t.Errorf("row %q starts its cells at %v %v %v, the header at %v %v %v", r, c0.X, c1.X, c2.X, code.X, name.X, city.X)
		}
	}
	// Code is 60 wide: Name starts 60 and a gap after it.
	if got := name.X - code.X; !within(got, 60+gap, 1) {
		t.Errorf("Name starts %v after Code, want 60 + the gap %v", got, gap)
	}
	// City is 80 wide, at the end of the row, inside its padding.
	if end := first.X + first.W - space(2.5); !within(city.X, end-80, 1) {
		t.Errorf("City starts at %v, want 80 before the row's end %v", city.X, end)
	}
	if city.X-name.X <= space(15) {
		t.Errorf("Name takes %v, less than the grid's least %v", city.X-name.X-gap, space(15))
	}
	list, in := tt.Find(ByRole(ui.RoleList)).Bounds(), lookField(tt).Bounds()
	if list.W < in.W {
		t.Errorf("the drop-down is %v wide, narrower than the field's %v", list.W, in.W)
	}
	// Semibold: as wide as ui's text of weight 600, not as its regular one.
	width := func(weight int) float32 {
		ref := ui.NewTester(func(c *ui.Context) {
			ui.Row(c).Children(func() { ui.Text(c, "Code").FontWeight(weight) })
		}, 200, 100)
		r, _ := ref.Find("Code")
		return r.W
	}
	semi, regular := width(600), width(400)
	if within(semi, regular, 0.5) {
		t.Fatalf("semibold and regular Code are both %v wide: the case cannot tell them apart", semi)
	}
	if !within(code.W, semi, 0.5) {
		t.Errorf("the title Code is %v wide, semibold is %v, regular %v", code.W, semi, regular)
	}
	_ = th
}

// Case 2.
func TestLookUpEditTypingFilters(t *testing.T) {
	l := custLookUp()
	tt, _ := lookTester(t, l)
	f := lookField(tt)
	f.Type("ac")
	order := func(want ...string) {
		t.Helper()
		if n := len(lookRows(tt)); n != len(want) {
			t.Fatalf("%d rows, want %q (texts %q)", n, want, tt.Texts())
		}
		for i := 1; i < len(want); i++ {
			if a, b := textAt(t, tt, want[i-1]), textAt(t, tt, want[i]); a.Y >= b.Y {
				t.Errorf("%s at %v is not above %s at %v", want[i-1], a.Y, want[i], b.Y)
			}
		}
	}
	order("Acme", "Acorn")
	if tt.HasText("Beta") {
		t.Fatal("Beta shows for ac")
	}
	lookKeys(tt, ui.KeyBackspace, ui.KeyBackspace)
	lookType(tt, "hanoi")
	order("Acme", "Acorn")
	for range 5 {
		lookKeys(tt, ui.KeyBackspace)
	}
	lookType(tt, "zz")
	if n := len(lookRows(tt)); n != 0 {
		t.Fatalf("%d rows for zz", n)
	}
	if !tt.HasText(l10n.NoResults()) {
		t.Fatalf("no %q row for zz: %q", l10n.NoResults(), tt.Texts())
	}
	if tt.HasText("Code") {
		t.Errorf("the header shows with no row under it")
	}
}

// Case 2: records whose text starts with the typed text come first, then
// those of which text or a column holds it, in their order.
func TestLookUpEditStartsFirst(t *testing.T) {
	l := LookUpEdit(custKey, custName, custCols()...).SetItems([]cust{
		{1, "X1", "Banana", "Ca"}, {2, "X2", "Cab", "Hue"}, {3, "X3", "Arcade", "Hue"},
	})
	tt, _ := lookTester(t, l)
	lookField(tt).Type("ca")
	rows := lookRows(tt)
	if len(rows) != 3 {
		t.Fatalf("%d rows, want 3: %q", len(rows), tt.Texts())
	}
	cab, ban, arc := textAt(t, tt, "Cab"), textAt(t, tt, "Banana"), textAt(t, tt, "Arcade")
	if !(cab.Y < ban.Y && ban.Y < arc.Y) {
		t.Fatalf("Cab %v, Banana %v, Arcade %v: want Cab (starts) then Banana and Arcade in order", cab.Y, ban.Y, arc.Y)
	}
}

// Case 3.
func TestLookUpEditEnterChoosesOnce(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt)
	f.Type("be")
	lookKeys(tt, ui.KeyEnter)
	if v := f.Value(); v != "Beta" {
		t.Fatalf("the field shows %q, want Beta", v)
	}
	wantSelected(t, l, 2, true)
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}}) {
		t.Fatalf("OnSelect ran %v, want (2, true) once", got)
	}
	// Down opens on the first, Down moves to Beta, Enter chooses it again.
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 2, true)
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}}) {
		t.Fatalf("choosing Beta again ran OnSelect: %v", got)
	}
}

// Case 4. ui/combobox.go: a click opens the popup with no option
// highlighted (open(true, false, -1)); Down then moves the highlight to
// min(highlight+1, n-1): the first option, then the second.
func TestLookUpEditArrows(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 2, true)
	if v := f.Value(); v != "Beta" {
		t.Fatalf("the field shows %q, want Beta", v)
	}
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}}) {
		t.Fatalf("OnSelect ran %v", got)
	}
}

// Case 5.
func TestLookUpEditFreeTextGoesBack(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	tt.Find(ByRole(ui.RoleListItem).Nth(0)).Click()
	wantSelected(t, l, 1, true)
	lookType(tt, "xyz")
	if v := f.Value(); v == "Acme" {
		t.Fatalf("typing did not change the field: %q", v)
	}
	lookKeys(tt, ui.KeyTab)
	if f.Focused() {
		t.Fatal("Tab left the focus in the field")
	}
	if v := f.Value(); v != "Acme" {
		t.Fatalf("after Tab the field shows %q, want Acme", v)
	}
	wantSelected(t, l, 1, true)
	if got := p.all(t); !slices.Equal(got, []pick{{1, true}}) {
		t.Fatalf("OnSelect ran %v, want the choice of Acme alone", got)
	}
}

// Case 6.
func TestLookUpEditEmptiedSelectsNone(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Type("acm")
	lookKeys(tt, ui.KeyEnter)
	wantSelected(t, l, 1, true)
	lookKeys(tt, ui.KeyEnd, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace)
	if v := f.Value(); v != "" {
		t.Fatalf("the field holds %q, want it empty", v)
	}
	lookKeys(tt, ui.KeyTab)
	wantSelected(t, l, 0, false)
	if v := f.Value(); v != "" {
		t.Fatalf("the field shows %q", v)
	}
	if got := p.all(t); !slices.Equal(got, []pick{{1, true}, {0, false}}) {
		t.Fatalf("OnSelect ran %v, want (1, true) then (0, false)", got)
	}
	// Emptied again with nothing selected: no event.
	f.Focus()
	lookKeys(tt, ui.KeyTab)
	if got := p.all(t); len(got) != 2 {
		t.Fatalf("leaving an empty field with nothing selected ran OnSelect: %v", got)
	}
}

// Case 7.
func TestLookUpEditSetters(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	l.SetSelected(3)
	tt.Settle()
	wantSelected(t, l, 3, true)
	if v := lookField(tt).Value(); v != "Acorn" {
		t.Fatalf("SetSelected(3): the field shows %q", v)
	}
	l.SetSelected(9)
	tt.Settle()
	wantSelected(t, l, 3, true)
	if v := lookField(tt).Value(); v != "Acorn" {
		t.Fatalf("SetSelected(9): the field shows %q", v)
	}
	l.Unselect()
	tt.Settle()
	wantSelected(t, l, 0, false)
	if v := lookField(tt).Value(); v != "" {
		t.Fatalf("Unselect: the field shows %q", v)
	}
	if got := p.all(t); len(got) != 0 {
		t.Fatalf("the setters ran OnSelect: %v", got)
	}
}

// Case 7's bounds: the first record and the last, and the keys around them.
func TestLookUpEditSetSelectedBounds(t *testing.T) {
	l := custLookUp()
	tt, _ := lookTester(t, l)
	for _, k := range []int{1, 3} {
		l.SetSelected(k)
		tt.Settle()
		wantSelected(t, l, k, true)
	}
	for _, k := range []int{0, 4, -1} {
		l.SetSelected(k)
		tt.Settle()
		wantSelected(t, l, 3, true)
	}
}

// Case 8.
func TestLookUpEditSetItemsKeepsTheSelection(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	l.SetSelected(3)
	tt.Settle()
	l.SetItems(custs()[:2])
	tt.Settle()
	wantSelected(t, l, 0, false)
	if v := lookField(tt).Value(); v != "" {
		t.Fatalf("record 3 gone: the field shows %q", v)
	}
	l.SetSelected(1)
	tt.Settle()
	items := custs()
	items[0].Name = "Acme Ltd"
	l.SetItems(items)
	tt.Settle()
	wantSelected(t, l, 1, true)
	if v := lookField(tt).Value(); v != "Acme Ltd" {
		t.Fatalf("record 1 renamed: the field shows %q", v)
	}
	if got := p.all(t); len(got) != 0 {
		t.Fatalf("SetItems ran OnSelect: %v", got)
	}
}

// SetItems copies the slice: the caller's changes after it do not show.
func TestLookUpEditSetItemsCopies(t *testing.T) {
	items := custs()
	l := LookUpEdit(custKey, custName, custCols()...).SetItems(items).SetSelected(1)
	tt, _ := lookTester(t, l)
	items[0].Name = "Changed"
	tt.Settle()
	if v := lookField(tt).Value(); v != "Acme" {
		t.Fatalf("the field shows %q after the caller changed its slice", v)
	}
}

// Case 9, no column.
func TestLookUpEditNoColumns(t *testing.T) {
	l := LookUpEdit(custKey, custName).SetItems(custs())
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	if n := len(lookRows(tt)); n != 3 {
		t.Fatalf("%d rows, want 3", n)
	}
	for _, s := range []string{"Acme", "Beta", "Acorn"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q: %q", s, tt.Texts())
		}
	}
	for _, s := range []string{"C01", "Hanoi", "Code"} {
		if tt.HasText(s) {
			t.Fatalf("%q shows with no column: %q", s, tt.Texts())
		}
	}
	list, first := tt.Find(ByRole(ui.RoleList)).Bounds(), lookRows(tt)[0].Bounds()
	if first.Y-list.Y > space(1)+2 {
		t.Fatalf("the first row is %v below the drop-down's top: a header above it?", first.Y-list.Y)
	}
}

// Case 9, a hidden column.
func TestLookUpEditHiddenColumn(t *testing.T) {
	cols := custCols()
	cols[0].Hidden = true
	l := LookUpEdit(custKey, custName, cols...).SetItems(custs())
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	if tt.HasText("Code") || tt.HasText("C01") {
		t.Fatalf("the hidden column shows: %q", tt.Texts())
	}
	if !tt.HasText("Name") || !tt.HasText("Hanoi") {
		t.Fatalf("the shown columns are missing: %q", tt.Texts())
	}
	f.Type("c01")
	if n := len(lookRows(tt)); n != 0 {
		t.Fatalf("c01 matched %d rows through the hidden column", n)
	}
}

// Case 10.
func TestLookUpEditNoTitlesNoHeader(t *testing.T) {
	cols := custCols()
	for i := range cols {
		cols[i].Title = ""
	}
	l := LookUpEdit(custKey, custName, cols...).SetItems(custs())
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	if !tt.HasText("C01") {
		t.Fatalf("no cells: %q", tt.Texts())
	}
	list, first := tt.Find(ByRole(ui.RoleList)).Bounds(), lookRows(tt)[0].Bounds()
	if first.Y-list.Y > space(1)+2 {
		t.Fatalf("the first row is %v below the drop-down's top: a header row above it", first.Y-list.Y)
	}
}

// Case 11.
func TestLookUpEditFieldsRow(t *testing.T) {
	calmOff()
	withMac(t, false)
	l := custLookUp()
	tt := formOf(t, Fields().Row("&Customer", l).Row("Other", Button("B")))
	f := tt.Find(ByRole(ui.RoleComboBox).Label("Customer"))
	l.Focus()
	tt.Settle()
	if !f.Focused() {
		t.Fatal("Focus did not focus the input")
	}
	tt.Find(ByLabel("B")).Focus()
	altKey(tt, ui.KeyC)
	if !f.Focused() {
		t.Fatal("Alt+C from a button did not focus the input")
	}
}

// Case 11: the input tells assistive technology that its options match
// what was typed.
func TestLookUpEditAutoCompleteList(t *testing.T) {
	tt, _ := lookTester(t, custLookUp())
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleComboBox {
			if a.AutoComplete != ui.AutoCompleteList {
				t.Fatalf("the input's AutoComplete is %v, want AutoCompleteList", a.AutoComplete)
			}
			return
		}
	}
	t.Fatal("no combo box")
}

// Case 12.
func TestLookUpEditEscapeAndClickOutside(t *testing.T) {
	l := custLookUp().SetSelected(2)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	if !lookOpen(tt) {
		t.Fatal("the drop-down did not open")
	}
	lookKeys(tt, ui.KeyEscape)
	if lookOpen(tt) {
		t.Fatal("Escape did not close the drop-down")
	}
	wantSelected(t, l, 2, true)
	if v := f.Value(); v != "Beta" {
		t.Fatalf("after Escape the field shows %q", v)
	}
	f.Click()
	if !lookOpen(tt) {
		t.Fatal("the drop-down did not open again")
	}
	tt.ClickAt(390, 290) // the page, below and beside everything
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("a click outside did not close the drop-down")
	}
	wantSelected(t, l, 2, true)
}

// Case 13.
func TestLookUpEditFrameAllocations(t *testing.T) {
	calmOff()
	combo := ui.NewTester(content(ComboBox(Option{"1", "Acme"}, Option{"2", "Beta"}, Option{"3", "Acorn"}).Editable().SetValue("1")), 400, 300)
	look := ui.NewTester(content(custLookUp().SetSelected(1)), 400, 300)
	for range 10 {
		combo.Frame()
		look.Frame()
	}
	want, got := allocrun.Least(30, combo.Frame), allocrun.Least(30, look.Frame)
	if got > want {
		t.Fatalf("a frame of the LookUpEdit at rest allocates %v, more than an Editable ComboBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, an Editable ComboBox's %v", got, want)
}

// Case 14.
func TestLookUpEditSettersFromAnotherGoroutine(t *testing.T) {
	l := custLookUp()
	tt, _ := lookTester(t, l)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			items := custs()
			items[2].Name = fmt.Sprintf("Acorn %d", i)
			l.SetItems(items)
			l.SetSelected(1 + i%3)
			if i%5 == 0 {
				l.Unselect()
			}
		}
		l.SetSelected(3)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	wantSelected(t, l, 3, true)
	if v := lookField(tt).Value(); v != "Acorn 39" {
		t.Fatalf("the field shows %q, want Acorn 39", v)
	}
}

// Review Focus 1.
func TestLookUpEditTextScale(t *testing.T) {
	long := "A customer name far too long for its column"
	items := custs()
	items[0].Name = long
	l := LookUpEdit(custKey, custName, custCols()...).SetItems(items).SetSize(160, 0)
	tt, th := lookTester(t, l)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Settle()
	lookField(tt).Click()
	list := tt.Find(ByRole(ui.RoleList)).Bounds()
	need := (60+80)*1.5 + space(15) + 2*space(2) + 2*space(2.5) + 2*space(1)
	if list.W < need {
		t.Errorf("the drop-down is %v wide, less than its columns' %v", list.W, need)
	}
	if list.X < 0 || list.X+list.W > 400 || list.Y < 0 || list.Y+list.H > 300 {
		t.Errorf("the drop-down at %v leaves the window of 400 by 300", list)
	}
	code, city, first := textAt(t, tt, "C01"), textAt(t, tt, "Hanoi"), lookRows(tt)[0].Bounds()
	name := textAt(t, tt, "Name")
	if got := name.X - code.X; !within(got, 60*1.5+space(2), 1) {
		t.Errorf("Name starts %v after Code, want 90 and the gap", got)
	}
	if end := first.X + first.W - space(2.5); !within(city.X, end-80*1.5, 1) {
		t.Errorf("City starts at %v, want 120 before the row's end %v", city.X, end)
	}
	// The long name draws nothing in the gap before City.
	img, bg := tt.Image(), th.Background
	x := int(city.X - space(2)/2)
	for y := int(first.Y + 2); y < int(first.Y+first.H-2); y++ {
		if c := img.RGBAAt(x, y); !near(c, bg) {
			t.Fatalf("the long name draws %v at (%d, %d), in the gap before City", c, x, y)
		}
	}
}

// Review Focus 2.
func TestLookUpEditScrollsToTheHighlight(t *testing.T) {
	var items []cust
	for i := range 200 {
		items = append(items, cust{i + 1, fmt.Sprintf("K%03d", i), fmt.Sprintf("Name %d", i), "Hue"})
	}
	l := LookUpEdit(custKey, custName, custCols()...).SetItems(items)
	tt, _ := lookTester(t, l)
	tt.SetSize(400, 900)
	tt.Settle()
	lookField(tt).Click()
	list := tt.Find(ByRole(ui.RoleList)).Bounds()
	if max := space(70); list.H > max+0.5 {
		t.Fatalf("the drop-down is %v high, more than ui's %v", list.H, max)
	}
	for range 30 {
		lookKeys(tt, ui.KeyDown)
	}
	var lit *ui.AccessInfo
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleListItem && a.Focused { // the option the arrows are on (ActiveDescendant)
			lit = &a
			break
		}
	}
	if lit == nil {
		t.Fatal("no highlighted row")
	}
	if b := lit.Bounds; b.Y < list.Y-0.5 || b.Y+b.H > list.Y+list.H+0.5 {
		t.Fatalf("the highlighted row %q at %v is outside the drop-down %v", lit.Name, b, list)
	}
	if !strings.Contains(lit.Name, "Name 29") {
		t.Fatalf("the highlighted row is %q, want Name 29's", lit.Name)
	}
}

// Review Focus 3.
func TestLookUpEditOnSelectCallsSetters(t *testing.T) {
	var l *LookUpEditWidget[cust, int]
	l = custLookUp().OnSelect(func(k int, ok bool) {
		items := custs()
		items[2].Name = "Acorn Co"
		l.SetItems(items).SetSelected(3)
	})
	tt, _ := lookTester(t, l)
	lookField(tt).Type("be")
	lookKeys(tt, ui.KeyEnter)
	wantSelected(t, l, 3, true)
	if v := lookField(tt).Value(); v != "Acorn Co" {
		t.Fatalf("the field shows %q, want Acorn Co", v)
	}
}

// lookInk is the pixel of the text s farthest from bg: its ink.
func lookInk(t *testing.T, tt *Tester, s string, bg ui.Color) ui.Color {
	t.Helper()
	c := navInk(t, tt, s, bg)
	return ui.Color{R: c.R, G: c.G, B: c.B, A: 255}
}

// Review Focus 4.
func TestLookUpEditContrast(t *testing.T) {
	for _, skin := range []string{"", "Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			name := skin
			if name == "" {
				name = "none"
			}
			t.Run(fmt.Sprintf("%s/dark=%v", name, dark), func(t *testing.T) {
				th := new(ui.Theme)
				keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
				if skin != "" {
					useSkinNamed(t, skin, false)
				} else {
					calmOff()
				}
				tt := NewTester(t, VBox(0).Add(keep, custLookUp()), 400, 300)
				tt.SetDark(dark)
				tt.Settle()
				if th.Dark != dark {
					t.Fatalf("theme dark %v, want %v", th.Dark, dark)
				}
				lookField(tt).Click()
				list := tt.Find(ByRole(ui.RoleList)).Bounds()
				px := tt.Image().RGBAAt(int(list.X+list.W-3), int(list.Y+list.H/2))
				bg := ui.Color{R: px.R, G: px.G, B: px.B, A: 255}
				for _, s := range []string{"Code", "City", "Hue", "C02"} {
					if r := contrastOf(lookInk(t, tt, s, bg), bg); r < 4.5 {
						t.Errorf("%q reaches %.2f:1 on the drop-down, less than 4.5", s, r)
					}
				}
			})
		}
	}
}

// Review Focus 5.
func TestLookUpEditDisabled(t *testing.T) {
	var p picks
	l := custLookUp().OnSelect(p.add).SetDisabled(true)
	tt, _ := lookTester(t, l)
	b := lookField(tt).Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("a click opened the drop-down of a disabled LookUpEdit")
	}
	tt.Type("be")
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if v := lookField(tt).Value(); v != "" {
		t.Fatalf("typing reached a disabled LookUpEdit: %q", v)
	}
	wantSelected(t, l, 0, false)
	if got := p.all(t); len(got) != 0 {
		t.Fatalf("OnSelect ran: %v", got)
	}
}

// No input: a LookUpEdit with no record draws, opens on "No results", and
// its setters answer.
func TestLookUpEditEmpty(t *testing.T) {
	var p picks
	l := LookUpEdit(custKey, custName, custCols()...).OnSelect(p.add)
	tt, _ := lookTester(t, l)
	l.SetSelected(1)
	tt.Settle()
	wantSelected(t, l, 0, false)
	lookField(tt).Click()
	if n := len(lookRows(tt)); n != 0 {
		t.Fatalf("%d rows with no record", n)
	}
	lookKeys(tt, ui.KeyEnter)
	wantSelected(t, l, 0, false)
	l.SetItems(custs()[:1])
	tt.Settle()
	lookKeys(tt, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 1, true)
	if got := p.all(t); !slices.Equal(got, []pick{{1, true}}) {
		t.Fatalf("OnSelect ran %v", got)
	}
}

// One record: Down past it stays on it.
func TestLookUpEditOneRecord(t *testing.T) {
	l := LookUpEdit(custKey, custName, custCols()...).SetItems(custs()[1:2])
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 2, true)
}

// A click on a record chooses it, and OnSelect runs on the event goroutine.
func TestLookUpEditClickRunsOnTheEventGoroutine(t *testing.T) {
	on := make(chan bool, 4)
	l := custLookUp().OnSelect(func(int, bool) { on <- onEvents() })
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	tt.Find(ByRole(ui.RoleListItem).Nth(2)).Click()
	wantSelected(t, l, 3, true)
	if !received(t, on) {
		t.Fatal("OnSelect ran off the event goroutine")
	}
}
