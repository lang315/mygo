package widget

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// lookTree is what assistive technology reads of the window, one element a
// line: role, name and value. Boxes are left out: they follow the system's
// fonts, which differ from one OS to the next.
func lookTree(tt *Tester) string {
	var b strings.Builder
	for _, a := range tt.AccessAll() {
		fmt.Fprintf(&b, "%s %q %q\n", roleName(a.Role), a.Name, a.Value)
	}
	return b.String()
}

// The trees of main (2023d053f) for a click opening the drop-down and for ac
// typed then, as lookTree printed them there.
const (
	mainClickTree = `combo box "" ""
button "Other" ""
list "" ""
text "Code" ""
text "Name" ""
text "City" ""
list item "C01 Acme Hanoi" ""
text "C01" ""
text "Acme" ""
text "Hanoi" ""
list item "C02 Beta Hue" ""
text "C02" ""
text "Beta" ""
text "Hue" ""
list item "C03 Acorn Hanoi" ""
text "C03" ""
text "Acorn" ""
text "Hanoi" ""
`
	mainTypedTree = `combo box "" "ac"
button "Other" ""
list "" ""
text "Code" ""
text "Name" ""
text "City" ""
list item "C01 Acme Hanoi" ""
text "C01" ""
text "Acme" ""
text "Hanoi" ""
list item "C03 Acorn Hanoi" ""
text "C03" ""
text "Acorn" ""
text "Hanoi" ""
`
)

func custCode(c *cust) string { return c.Code }

// searchLookUp is the LookUpEdit of the cases with the search panel on.
func searchLookUp() *LookUpEditWidget[cust, int] { return custLookUp().SetSearchPanel(true) }

// searchBox is the drop-down's search box.
func searchBox(tt *Tester) *Node { return tt.Find(ByRole(ui.RoleTextField)) }

// rowNames are the names of the drop-down's rows, in order.
func rowNames(tt *Tester) []string {
	var out []string
	for _, n := range lookRows(tt) {
		out = append(out, n.Label())
	}
	return out
}

// wantRows fails the test unless the drop-down shows the rows named want.
func wantRows(t *testing.T, tt *Tester, q string, want ...string) {
	t.Helper()
	if got := rowNames(tt); !slices.Equal(got, want) {
		t.Fatalf("for %q the rows are %q, want %q", q, got, want)
	}
}

// Case L1: a row shows when a visible column's text holds the query, case
// ignored, in the records' order; with no column, the text alone.
func TestLookUpSearchAnyColumn(t *testing.T) {
	// The field shows the code: Beta matches by its second column alone.
	l := LookUpEdit(custKey, custCode, custCols()...).SetItems(custs()).SetSearchPanel(true)
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	if !lookOpen(tt) {
		t.Fatal("a click did not open the drop-down")
	}
	wantRows(t, tt, "", "C01 Acme Hanoi", "C02 Beta Hue", "C03 Acorn Hanoi")
	box := searchBox(tt)
	box.Type("BETA")
	wantRows(t, tt, "BETA", "C02 Beta Hue")
	lookKeys(tt, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace)
	lookType(tt, "hAnOi")
	wantRows(t, tt, "hAnOi", "C01 Acme Hanoi", "C03 Acorn Hanoi")
	for range 5 {
		lookKeys(tt, ui.KeyBackspace)
	}
	lookType(tt, "zz")
	wantRows(t, tt, "zz")
	if !tt.HasText(l10n.NoResults()) {
		t.Fatalf("no %q row for zz: %q", l10n.NoResults(), tt.Texts())
	}

	// With columns, the text is not searched unless a column shows it.
	l3 := LookUpEdit(custKey, func(c *cust) string { return "Text " + c.Name }, custCols()[:1]...).
		SetItems(custs()).SetSearchPanel(true)
	tt3, _ := lookTester(t, l3)
	lookField(tt3).Click()
	searchBox(tt3).Type("text")
	wantRows(t, tt3, "text")

	// With no column, the text alone.
	l2 := LookUpEdit(custKey, custName).SetItems(custs()).SetSearchPanel(true)
	tt2, _ := lookTester(t, l2)
	lookField(tt2).Click()
	searchBox(tt2).Type("ACO")
	wantRows(t, tt2, "ACO", "Acorn")
}

// Case L2: Down, Down, Enter picks the second match and runs OnSelect once;
// Escape closes with the choice as it was.
func TestLookUpSearchKeys(t *testing.T) {
	var p picks
	l := searchLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	searchBox(tt).Type("c0") // the three match, by their code
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEnter)
	if lookOpen(tt) {
		t.Fatal("Enter did not close the drop-down")
	}
	wantSelected(t, l, 2, true)
	if v := f.Value(); v != "Beta" {
		t.Fatalf("the field shows %q, want Beta", v)
	}
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}}) {
		t.Fatalf("OnSelect ran %v, want (2, true) once", got)
	}

	// Up moves back; the highlight starts afresh when the query changes.
	f.Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyDown, ui.KeyUp)
	lookType(tt, "a") // Acme, Beta, Acorn hold an a: the highlight is on none
	lookKeys(tt, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 1, true)
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}, {1, true}}) {
		t.Fatalf("OnSelect ran %v, want (2, true) then (1, true)", got)
	}

	// Escape closes and keeps the choice.
	f.Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEscape)
	if lookOpen(tt) {
		t.Fatal("Escape did not close the drop-down")
	}
	wantSelected(t, l, 1, true)
	if v := f.Value(); v != "Acme" {
		t.Fatalf("after Escape the field shows %q", v)
	}
	if got := p.all(t); len(got) != 2 {
		t.Fatalf("Escape ran OnSelect: %v", got)
	}

	// Down stops at the last match, Up at the first.
	f.Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyDown, ui.KeyDown, ui.KeyDown, ui.KeyEnter)
	wantSelected(t, l, 3, true)
	f.Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyUp, ui.KeyUp, ui.KeyUp, ui.KeyEnter)
	wantSelected(t, l, 1, true)
	if got := p.all(t); !slices.Equal(got, []pick{{2, true}, {1, true}, {3, true}, {1, true}}) {
		t.Fatalf("OnSelect ran %v", got)
	}

	// Enter with no row highlighted picks nothing and keeps it open.
	f.Click()
	lookKeys(tt, ui.KeyEnter)
	wantSelected(t, l, 1, true)
	if got := p.all(t); len(got) != 4 {
		t.Fatalf("Enter on no row ran OnSelect: %v", got)
	}
}

// Case L3: with the panel on, typing in the field changes nothing.
func TestLookUpSearchFieldNoTyping(t *testing.T) {
	var p picks
	l := searchLookUp().SetSelected(2).OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Focus()
	lookType(tt, "zz")
	if v := f.Value(); v != "Beta" {
		t.Fatalf("typing in the field made it %q, want Beta", v)
	}
	if tt.Has(ByRole(ui.RoleTextField)) {
		t.Fatal("the field is a text field: it takes typing")
	}
	wantSelected(t, l, 2, true)
	if got := p.all(t); len(got) != 0 {
		t.Fatalf("typing in the field ran OnSelect: %v", got)
	}
}

// Case L4: a hidden column's text does not match.
func TestLookUpSearchHiddenColumn(t *testing.T) {
	cols := custCols()
	cols[2].Hidden = true // City
	l := LookUpEdit(custKey, custName, cols...).SetItems(custs()).SetSearchPanel(true)
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	searchBox(tt).Type("hanoi")
	wantRows(t, tt, "hanoi")
	for range 5 {
		lookKeys(tt, ui.KeyBackspace)
	}
	lookType(tt, "acme")
	wantRows(t, tt, "acme", "C01 Acme")
}

// largeCusts are n records: C00000 Name 0 Hanoi, C00001 Name 1 Hue, ...
func largeCusts(n int) []cust {
	cities := []string{"Hanoi", "Hue", "Saigon"}
	out := make([]cust, n)
	for i := range out {
		out[i] = cust{i, fmt.Sprintf("C%05d", i), fmt.Sprintf("Name %d", i), cities[i%3]}
	}
	return out
}

// lookBuiltMax is the most rows the drop-down may build: those its list
// shows, ten, with the margins ui's List builds beyond its edges.
const lookBuiltMax = 16

// Case L5: of 10,000 records, opening the drop-down builds the rows shown and
// a margin, and so does typing a query; the highlight scrolls into view.
func TestLookUpSearchLarge(t *testing.T) {
	var p picks
	l := LookUpEdit(custKey, custName, custCols()...).SetItems(largeCusts(10000)).SetSearchPanel(true).OnSelect(p.add)
	tt, _ := lookTester(t, l)
	tt.SetSize(400, 600)
	tt.Settle()
	lookField(tt).Click()
	rows := rowNames(tt)
	t.Logf("opening built %d rows", len(rows))
	if len(rows) == 0 || len(rows) > lookBuiltMax {
		t.Fatalf("opening built %d rows, want 1 to %d", len(rows), lookBuiltMax)
	}
	if rows[0] != "C00000 Name 0 Hanoi" {
		t.Fatalf("the first row is %q", rows[0])
	}
	searchBox(tt).Type("name 1")
	rows = rowNames(tt)
	if len(rows) == 0 || len(rows) > lookBuiltMax {
		t.Fatalf("typing built %d rows, want 1 to %d", len(rows), lookBuiltMax)
	}
	if rows[0] != "C00001 Name 1 Hue" || rows[1] != "C00010 Name 10 Hue" {
		t.Fatalf("the rows for name 1 start %q", rows[:2])
	}
	// The matches are 1, 10 to 19, 100 to 199, ...: the 30th is Name 118,
	// out of view until Down reaches it.
	if tt.Has(ByRole(ui.RoleListItem).Label("C00118 Name 118 Hue")) {
		t.Fatalf("Name 118 is built before Down reaches it: %q", rowNames(tt))
	}
	for range 30 {
		lookKeys(tt, ui.KeyDown)
	}
	a, ok := tt.Tester.AccessByRole(ui.RoleListItem, "C00118 Name 118 Hue")
	if !ok || a.Offscreen {
		t.Fatalf("the highlighted row is not in view (found %v): %q", ok, rowNames(tt))
	}
	if rows := rowNames(tt); len(rows) > lookBuiltMax {
		t.Fatalf("scrolling built %d rows, more than %d", len(rows), lookBuiltMax)
	}
	// Up scrolls back: the 15th match, Name 103, comes into view again.
	for range 15 {
		lookKeys(tt, ui.KeyUp)
	}
	if a, ok := tt.Tester.AccessByRole(ui.RoleListItem, "C00103 Name 103 Hue"); !ok || a.Offscreen {
		t.Fatalf("Up did not bring the highlighted row into view (found %v): %q", ok, rowNames(tt))
	}
	for range 15 {
		lookKeys(tt, ui.KeyDown)
	}
	lookKeys(tt, ui.KeyEnter)
	wantSelected(t, l, 118, true)
	if got := p.all(t); !slices.Equal(got, []pick{{118, true}}) {
		t.Fatalf("OnSelect ran %v", got)
	}
}

// Case L6: a click on a row picks it and closes the drop-down.
func TestLookUpSearchClickRow(t *testing.T) {
	var p picks
	l := searchLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	f := lookField(tt).Click()
	tt.Find(ByRole(ui.RoleListItem).Nth(2)).Click()
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("a click on a row did not close the drop-down")
	}
	wantSelected(t, l, 3, true)
	if v := f.Value(); v != "Acorn" {
		t.Fatalf("the field shows %q", v)
	}
	if got := p.all(t); !slices.Equal(got, []pick{{3, true}}) {
		t.Fatalf("OnSelect ran %v", got)
	}
	// A click on the field toggles it, as PopupEdit's does.
	f.Click()
	f.Click()
	if lookOpen(tt) {
		t.Fatal("a second click on the field did not close the drop-down")
	}
}

// Case L7: the keyboard opens the drop-down: Alt+Down, Down, Enter and Space
// on the field, with the focus in the search box.
func TestLookUpSearchOpenKeys(t *testing.T) {
	keys := []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{ui.Alt, ui.KeyDown}, {0, ui.KeyDown}, {0, ui.KeyEnter}, {0, ui.KeySpace}}
	for _, k := range keys {
		tt, _ := lookTester(t, searchLookUp())
		lookField(tt).Focus()
		tt.Tester.Key(k.mods, k.key)
		tt.Settle()
		if !lookOpen(tt) {
			t.Fatalf("%v %v did not open the drop-down", k.mods, k.key)
		}
		if !searchBox(tt).Focused() {
			t.Fatalf("%v %v: the search box has not the focus", k.mods, k.key)
		}
	}
}

// Case L8: SetItems shrinking the records while the drop-down shows, with a
// row highlighted past the new end, neither panics nor picks a record gone.
func TestLookUpSearchSetItemsWhileOpen(t *testing.T) {
	var p picks
	l := searchLookUp().OnSelect(p.add)
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyDown) // Acorn
	l.SetItems(custs()[:1])
	tt.Settle()
	wantRows(t, tt, "", "C01 Acme Hanoi")
	lookKeys(tt, ui.KeyEnter)
	if got := p.all(t); len(got) > 1 || (len(got) == 1 && got[0] != (pick{1, true})) {
		t.Fatalf("OnSelect ran %v", got)
	}
}

// focusedRows are the names of the rows assistive technology has the focus
// on: the search box's active descendant.
func focusedRows(tt *Tester) []string {
	var out []string
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleListItem && a.Focused {
			out = append(out, a.Name)
		}
	}
	return out
}

// Case L10: the highlighted row is the search box's active descendant, so
// assistive technology reads it; none while no row is highlighted.
func TestLookUpSearchActiveDescendant(t *testing.T) {
	tt, _ := lookTester(t, searchLookUp())
	lookField(tt).Click()
	if got := focusedRows(tt); len(got) != 0 {
		t.Fatalf("on opening, the focus is on rows %q, want none", got)
	}
	searchBox(tt).Type("c0")
	lookKeys(tt, ui.KeyDown, ui.KeyDown)
	if got := focusedRows(tt); !slices.Equal(got, []string{"C02 Beta Hue"}) {
		t.Fatalf("after Down, Down the focus is on rows %q, want the second match", got)
	}
	lookType(tt, "2") // c02: Beta alone, highlighted by none
	if got := focusedRows(tt); len(got) != 0 {
		t.Fatalf("after the query changed, the focus is on rows %q, want none", got)
	}
	if !searchBox(tt).Focused() {
		t.Fatal("the search box lost the focus")
	}
}

// Case L11: right to left, the drop-down mirrors: the first column's
// header and cells are at the right, the last's at the left.
func TestLookUpSearchRightToLeft(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		tt, _ := lookTester(t, searchLookUp())
		if rtl {
			tt.SetDirection(ui.RightToLeft)
			tt.Frame()
		}
		lookField(tt).Click()
		code, city := textAt(t, tt, "Code"), textAt(t, tt, "City")
		c01, hanoi := textAt(t, tt, "C01"), textAt(t, tt, "Hanoi")
		if first := code.X < city.X && c01.X < hanoi.X; first == rtl {
			t.Fatalf("right to left %v: Code at %v, City at %v, C01 at %v, Hanoi at %v", rtl, code.X, city.X, c01.X, hanoi.X)
		}
		if rtl && !within(code.X+code.W, c01.X+c01.W, 1) {
			t.Fatalf("right to left: the header's Code ends at %v, the row's C01 at %v: not lined up at the right", code.X+code.W, c01.X+c01.W)
		}
	}
}

// Case L9: the drop-down closes when its LookUpEdit is hidden, as a tab
// switched away hides it, and stays closed when it shows again.
func TestLookUpSearchHideCloses(t *testing.T) {
	l := searchLookUp()
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	if !lookOpen(tt) {
		t.Fatal("a click did not open the drop-down")
	}
	l.SetVisible(false)
	tt.Settle()
	l.SetVisible(true)
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("the drop-down shows again after a hide")
	}
}

// With the panel off, a click opens the drop-down and typing filters as on
// main, and the drop-down's tree is main's; SetSearchPanel(false) is off.
func TestLookUpSearchOffIsUnchanged(t *testing.T) {
	for _, off := range []bool{false, true} {
		l := custLookUp()
		if off {
			l.SetSearchPanel(true).SetSearchPanel(false)
		}
		tt, _ := lookTester(t, l)
		f := lookField(tt).Click()
		if got := lookTree(tt); got != mainClickTree {
			t.Fatalf("turned off %v: a click shows\n%s\nwant main's\n%s", off, got, mainClickTree)
		}
		f.Type("ac")
		if got := lookTree(tt); got != mainTypedTree {
			t.Fatalf("turned off %v: ac typed shows\n%s\nwant main's\n%s", off, got, mainTypedTree)
		}
	}
}

// SetSearchPanel and the setters from another goroutine while the window
// shows, the drop-down open, show on the next frames, with no race.
func TestLookUpSearchFromGoroutine(t *testing.T) {
	l := searchLookUp()
	tt, _ := lookTester(t, l)
	lookField(tt).Click()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 200 { // alone, so that no other setter's lock orders them with the frames
			l.SetSearchPanel(i%2 == 0)
		}
		for i := range 40 {
			items := custs()
			items[2].Name = fmt.Sprintf("Acorn %d", i)
			l.SetItems(items[:1+i%3])
			l.SetSelected(1 + i%3)
			l.SetSearchPanel(i%4 != 0)
		}
		l.SetItems(func() []cust { items := custs(); items[2].Name = "Acorn 39"; return items }())
		l.SetSearchPanel(true)
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
	f := lookField(tt)
	if v := f.Value(); v != "Acorn 39" {
		t.Fatalf("the field shows %q, want Acorn 39", v)
	}
	if lookOpen(tt) {
		f.Click() // closes
	}
	f.Click()
	if !tt.Has(ByRole(ui.RoleTextField)) {
		t.Fatal("the drop-down has no search box")
	}
	wantRows(t, tt, "", "C01 Acme Hanoi", "C02 Beta Hue", "C03 Acorn 39 Hanoi")
}

// A frame of the field with the panel on, at rest, allocates no more than a
// TextBox's.
func TestLookUpSearchAllocNothingAtRest(t *testing.T) {
	calmOff()
	box := ui.NewTester(content(TextBox().SetText("Acme")), 400, 300)
	look := ui.NewTester(content(searchLookUp().SetSelected(1)), 400, 300)
	for range 10 {
		box.Frame()
		look.Frame()
	}
	want, got := allocrun.Least(30, box.Frame), allocrun.Least(30, look.Frame)
	if got > want {
		t.Fatalf("a frame of the LookUpEdit with its search panel allocates %v, more than a TextBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, a TextBox's %v", got, want)
}
