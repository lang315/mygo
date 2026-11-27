package widget

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// wideName is a text more than 300 DIPs wide in the theme's font.
var wideName = strings.Repeat("W", 30)

// colsTable is a table of a narrow Name, 40 DIPs, and a Score, 100, with a
// row whose name is wideName.
func colsTable() *DataGridViewWidget[acct, int] {
	return DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 40, Sortable: true},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return strings.Repeat("9", a.score) }, Width: 100, Align: End},
	).SetRows([]acct{{1, "ann", 3}, {2, wideName, 20}}).Expand()
}

// menuTester shows w with its menus drawn in the window, which assistive
// technology reads with their checks and the items disabled.
func menuTester(t *testing.T, w Control) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		content(w)(c)
	}, 800, 600)
	tt.Frame()
	return tt
}

// menuItem returns the node of the menu item labeled label, failing the test
// without one.
func menuItem(t *testing.T, tt *ui.Tester, label string) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		switch n.Role {
		case ui.RoleMenuItem, ui.RoleMenuItemCheckBox, ui.RoleMenuItemRadio:
			if n.Label == label {
				return n
			}
		}
	}
	t.Fatalf("no menu item %q in %q", label, tt.Menu())
	return ui.TestNode{}
}

// widths returns the widths ui gives the table's columns, as the user left
// them. On the tester's goroutine, between frames.
func widths(tb *DataGridViewWidget[acct, int]) map[string]float32 {
	mu.Lock()
	defer mu.Unlock()
	return tb.list.Columns.Widths
}

// textCount counts the texts of the last frame that are s.
func textCount(tt *ui.Tester, s string) int {
	n := 0
	for _, x := range tt.Texts() {
		if x == s {
			n++
		}
	}
	return n
}

// rest rests the pointer on the cell of the text s until a tooltip would
// show.
func rest(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	var r ui.Rect // the narrowest element named s: the cell's, not its row's
	for _, n := range tt.Nodes() {
		if n.Label == s && n.Bounds.W > 0 && (r.W == 0 || n.Bounds.W < r.W) {
			r = n.Bounds
		}
	}
	if r.W <= 0 {
		t.Fatalf("no cell of %q", s)
	}
	tt.Move(r.X+min(r.W/2, 20), r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
}

// leave moves the pointer off the table until no tooltip shows.
func leave(tt *ui.Tester) {
	tt.Move(790, 590)
	tt.Advance(2 * time.Second)
	tt.Frame()
}

// tipShows rests the pointer on the text s, after leaving the table, and
// reports whether a tooltip reading tip shows: one text tip more than the
// frame showed before (a row's text is its name too).
func tipShows(t *testing.T, tt *ui.Tester, s, tip string) bool {
	t.Helper()
	leave(tt)
	before := textCount(tt, tip)
	rest(t, tt, s)
	return textCount(tt, tip) > before
}

// textWidth measures s whole, as a table's cell draws it.
func textWidth(t *testing.T, s string) float32 {
	t.Helper()
	var w float32
	tt := ui.NewTester(func(c *ui.Context) { w, _ = c.MeasureText(0, ui.Span{Text: s}) }, 400, 100)
	tt.Frame()
	if w <= 0 {
		t.Fatalf("%q measured %v", s, w)
	}
	return w
}

func TestBestFitWidensToContent(t *testing.T) {
	got := make(chan ui.TableLayout, 16)
	tb := colsTable().OnColumnsChange(func(l ui.TableLayout) { got <- l })
	tt := tester(t, tb)
	tb.BestFit("Name")
	tt.Frame()
	tt.Frame()
	l := received(t, got)
	text := textWidth(t, wideName)
	if text < 300 {
		t.Fatalf("the name measures %v DIPs, not 300", text)
	}
	if w := l.Widths["Name"]; w < text+20-0.01 || w < 320 {
		t.Fatalf("Name fitted to %v for a text of %v and 20 DIPs of padding", w, text)
	}
	if w := widths(tb)["Score"]; w != 0 {
		t.Errorf("Score, not named, fitted to %v", w)
	}
	// Done: the frames after it fit nothing more.
	tb.SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 40}})
	tt.Frame()
	tt.Frame()
	if w := widths(tb)["Name"]; w != 40 {
		t.Errorf("Name fitted again to %v without BestFit", w)
	}
}

// A measure of the view (a window sized to its content) leaves BestFit to
// the window.
func TestBestFitNotTakenByAMeasure(t *testing.T) {
	got := make(chan ui.TableLayout, 16)
	tb := colsTable().OnColumnsChange(func(l ui.TableLayout) { got <- l })
	view := content(tb)
	tb.BestFit("Name")
	ui.View(view).MeasureHeight(800, 1)
	if w := widths(tb)["Name"]; w != 0 {
		t.Fatalf("the measure fitted Name to %v", w)
	}
	tt := ui.NewTester(view, 800, 600)
	tt.Frame()
	tt.Frame()
	if l := received(t, got); l.Widths["Name"] < 320 {
		t.Fatalf("the window's OnColumnsChange gave Name %v", l.Widths["Name"])
	}
}

func TestBestFitAllVisible(t *testing.T) {
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "A", Text: func(a *acct) string { return a.name }, Width: 30},
		Column[acct]{Title: "B", Text: func(a *acct) string { return a.name }, Width: 30, Hidden: true},
		Column[acct]{Title: "C", Text: func(a *acct) string { return a.name + a.name }, Width: 30},
	).SetRows([]acct{{1, wideName, 0}}).Expand()
	tt := tester(t, tb)
	tb.BestFit()
	tt.Frame()
	tt.Frame()
	w := widths(tb)
	if w["A"] < 320 || w["C"] < 2*w["A"]-30 {
		t.Fatalf("widths %v: A and C not fitted to their texts", w)
	}
	if _, ok := w["B"]; ok {
		t.Errorf("the hidden B fitted: %v", w)
	}
}

func TestBestFitSameAsDoubleClick(t *testing.T) {
	fitted := colsTable()
	tf := tester(t, fitted)
	fitted.BestFit("Name")
	tf.Frame()
	tf.Frame()
	clicked := colsTable()
	tc := tester(t, clicked)
	head := find(t, tc, "Name")
	// Name is 40 wide from the table's left, at the tester's.
	tc.ClickAt(38, head.Y+head.H/2)
	tc.ClickAt(38, head.Y+head.H/2)
	tc.Frame()
	a, b := widths(fitted)["Name"], widths(clicked)["Name"]
	if a != b || a < 320 {
		t.Fatalf("BestFit gave %v, a double click on the edge %v", a, b)
	}
}

func TestColumnChooserLists(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}).SetColumnChooser(true)
	tt := menuTester(t, tb)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.Menu(), []string{"Name", "Score", "-", "Best fit", "Best fit (all columns)"}; !slices.Equal(got, want) {
		t.Fatalf("the menu %q, want %q", got, want)
	}
	for _, title := range []string{"Name", "Score"} {
		if n := menuItem(t, tt, title); n.Role != ui.RoleMenuItemCheckBox || !n.Checked || n.Disabled {
			t.Errorf("%s's item: %+v, want checked and enabled", title, n)
		}
	}
	if err := tt.ChooseMenuItem("Score"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	if tb.ColumnVisible("Score") || tt.HasText("Score") || !tt.HasText("Name") {
		t.Fatalf("after unchecking Score: visible %v, texts %q", tb.ColumnVisible("Score"), tt.Texts())
	}
	// Checked again, it shows again.
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if n := menuItem(t, tt, "Score"); n.Role != ui.RoleMenuItemCheckBox || n.Checked {
		t.Errorf("the hidden Score's item: %+v, want a check box not checked", n)
	}
	if err := tt.ChooseMenuItem("Score"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	if !tb.ColumnVisible("Score") || !tt.HasText("Score") {
		t.Fatalf("after checking Score: visible %v", tb.ColumnVisible("Score"))
	}
}

func TestColumnChooserKeepsLastColumn(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}).SetColumnChooser(true).SetColumnVisible("Score", false)
	tt := menuTester(t, tb)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if n := menuItem(t, tt, "Name"); !n.Disabled || !n.Checked {
		t.Fatalf("the last column's item: %+v, want checked and disabled", n)
	}
	if n := menuItem(t, tt, "Score"); n.Disabled {
		t.Fatalf("the hidden Score's item is disabled: %+v", n)
	}
	if err := tt.ChooseMenuItem("Name"); err == nil {
		t.Fatal("the last column's item could be chosen")
	}
	drained(t)
	tt.Frame()
	if !tb.ColumnVisible("Name") {
		t.Fatal("the last column was hidden")
	}
}

func TestColumnChooserBestFitItems(t *testing.T) {
	tb := colsTable().SetColumnChooser(true)
	tt := menuTester(t, tb)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Best fit"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	tt.Frame()
	if w := widths(tb); w["Name"] < 320 || w["Score"] != 0 {
		t.Fatalf("Best fit on Name: widths %v", w)
	}
	// Past the columns, no column to fit.
	head := find(t, tt, "Name")
	tt.RightClickAt(780, head.Y+head.H/2)
	tt.Frame()
	if n := menuItem(t, tt, "Best fit"); !n.Disabled {
		t.Fatalf("Best fit past the columns: %+v", n)
	}
	if err := tt.ChooseMenuItem("Best fit (all columns)"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	tt.Frame()
	if w, nines := widths(tb)["Score"], textWidth(t, strings.Repeat("9", 20)); w < nines+20-0.01 || w < 120 {
		t.Fatalf("Best fit (all columns) left Score at %v for 20 nines of %v", w, nines)
	}
}

func TestColumnChooserBestFitIndexColumn(t *testing.T) {
	tb := colsTable().IndexColumn().SetColumnChooser(true)
	tt := menuTester(t, tb)
	if err := tt.RightClick("#"); err != nil {
		t.Fatal(err)
	}
	if n := menuItem(t, tt, "Best fit"); !n.Disabled {
		t.Fatalf("Best fit on the index column: %+v", n)
	}
	tt.CloseMenu()
	// Score is ui's third column, after # and Name.
	if err := tt.RightClick("Score"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Best fit"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	tt.Frame()
	if w := widths(tb); w["Score"] < 120 || w["Name"] != 0 {
		t.Fatalf("Best fit on Score, with the index column: widths %v", w)
	}
}

// The column right-clicked hidden from code while the menu shows: choosing
// Best fit then fits no other column, and does not fail.
func TestColumnChooserColumnHiddenWhileOpen(t *testing.T) {
	tb := colsTable().SetColumnChooser(true)
	tt := menuTester(t, tb)
	if err := tt.RightClick("Score"); err != nil {
		t.Fatal(err)
	}
	tb.SetColumnVisible("Score", false)
	tt.Frame()
	if err := tt.ChooseMenuItem("Best fit"); err == nil {
		drained(t)
		tt.Frame()
		tt.Frame()
	}
	if w := widths(tb)["Name"]; w != 0 {
		t.Fatalf("Best fit on the hidden Score fitted Name to %v", w)
	}
}

func TestColumnChooserClearSorting(t *testing.T) {
	tb := testTable(acct{1, "bob", 1}, acct{2, "ann", 2}).SetColumnChooser(true)
	tt := menuTester(t, tb)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(tt.Menu(), "Clear sorting") {
		t.Fatalf("Clear sorting unsorted: %q", tt.Menu())
	}
	tt.CloseMenu()
	tb.Sort("Name", false)
	tt.Frame()
	inOrder(t, tt, "ann", "bob")
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Clear sorting"); err != nil {
		t.Fatalf("%v in %q", err, tt.Menu())
	}
	drained(t)
	tt.Frame()
	if got := tb.SortOrders(); len(got) != 0 {
		t.Fatalf("after Clear sorting: %v", got)
	}
	inOrder(t, tt, "bob", "ann")
}

func TestColumnChooserOffNoMenu(t *testing.T) {
	tb := testTable(acct{1, "ann", 1})
	tt := menuTester(t, tb)
	if err := tt.RightClick("Name"); err == nil {
		tt.Frame()
	}
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu without the chooser: %q", m)
	}
	// Turned on, then off again.
	tb.SetColumnChooser(true).SetColumnChooser(false)
	tt.Frame()
	if err := tt.RightClick("Name"); err == nil {
		tt.Frame()
	}
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu with the chooser turned off: %q", m)
	}
}

// A Filterable column's header with both the filter drop-downs and the
// column chooser: a right-click opens the chooser, Best fit makes room for
// the title and its funnel, and a click on the funnel opens the drop-down.
func TestColumnChooserWithFilterDropDowns(t *testing.T) {
	cols := fdocCols()
	cols[0].Width = 30 // Name: narrower than its title and funnel
	g := DataGridView(func(d *fdoc) int { return d.id }, cols...).SetRows(fdocs()).Expand().
		SetFilterDropDowns(true).SetColumnChooser(true)
	tt := menuTester(t, g)
	head := find(t, tt, "Name")
	tt.RightClickAt(head.X+3, head.Y+head.H/2) // the funnel covers the title's middle, 30 DIPs wide
	tt.Frame()
	if got, want := tt.Menu(), []string{"Name", "Kind", "Size", "-", "Best fit", "Best fit (all columns)"}; !slices.Equal(got, want) {
		t.Fatalf("the menu %q, want %q", got, want)
	}
	if tt.HasText("Select all") {
		t.Fatal("a right-click on the header opened the filter's drop-down")
	}
	if err := tt.ChooseMenuItem("Best fit"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	tt.Frame()
	var funnel ui.Rect
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleButton && n.Label == "Filter Name" {
			funnel = n.Bounds
		}
	}
	if funnel.W <= 0 {
		t.Fatal("no funnel on Name")
	}
	mu.Lock()
	w := g.list.Columns.Widths["Name"]
	mu.Unlock()
	if title := textWidth(t, "Name"); w < title+funnel.W+20-0.01 {
		t.Fatalf("Best fit gave Name %v: a title of %v, a funnel of %v and 20 DIPs of padding", w, title, funnel.W)
	}
	if funnel.X+funnel.W > w+0.5 { // Name is the first column, at the tester's left
		t.Fatalf("the funnel ends at %v, past Name's %v", funnel.X+funnel.W, w)
	}
	tt.ClickAt(funnel.X+funnel.W/2, funnel.Y+funnel.H/2)
	tt.Frame()
	if !tt.HasText("Select all") {
		t.Fatalf("a click on the funnel opened no drop-down: %q", tt.Texts())
	}
	if m := tt.Menu(); m != nil {
		t.Fatalf("a click on the funnel opened the chooser: %q", m)
	}
}

// hintCalls counts the runs of hintTable's Hint.
var hintCalls atomic.Int32

// hintTable is a table of Name, 80 DIPs, holding "ann" whole and a long name
// cut off, and Note, whose Hint is "note of" the name, 300 wide.
func hintTable() *DataGridViewWidget[acct, int] {
	return DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 80},
		Column[acct]{Title: "Note", Text: func(a *acct) string { return "n " + a.name }, Width: 300,
			Hint: func(a *acct) string {
				hintCalls.Add(1)
				if a.id == 3 {
					return ""
				}
				return "note of " + a.name
			}},
	).SetRows([]acct{{1, "ann", 0}, {2, "a name much too long for its column", 0}, {3, "cy", 0}}).Expand()
}

func TestCellHint(t *testing.T) {
	tb := hintTable()
	tt := tester(t, tb)
	if !tipShows(t, tt, "n ann", "note of ann") {
		t.Fatalf("no tooltip of the Hint: %q", tt.Texts())
	}
	// Shown, the tooltip asks for no more frames: the cell is not built again.
	n := hintCalls.Load()
	tt.Advance(time.Second)
	if got := hintCalls.Load() - n; got != 0 {
		t.Fatalf("the cell was built %d times more while its tooltip showed", got)
	}
	leave(tt)
	if tt.HasText("note of ann") {
		t.Fatal("the tooltip stayed after the pointer left")
	}
	// Back on the same cell, the tooltip shows again.
	if !tipShows(t, tt, "n ann", "note of ann") {
		t.Fatal("no tooltip as the pointer came back")
	}
	// An empty Hint: no tooltip.
	rest(t, tt, "n cy")
	if tt.HasText("note of") {
		t.Fatalf("a tooltip for an empty Hint: %q", tt.Texts())
	}
	// Without SetCellHints, a cell cut off has no tooltip.
	long := "a name much too long for its column"
	if tipShows(t, tt, long, long) {
		t.Fatal("the cut-off name has a tooltip without SetCellHints")
	}
	// Nor one of a column with a Hint, which is empty for it.
	tb.SetRows([]acct{{3, long, 0}}).SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Note": 60}})
	tt.Frame()
	if tipShows(t, tt, "n "+long, "n "+long) {
		t.Fatal("a cut-off note whose Hint is empty has a tooltip without SetCellHints")
	}
}

func TestCellHintsTruncatedOnly(t *testing.T) {
	long := "a name much too long for its column"
	tb := hintTable().SetCellHints(true)
	tt := tester(t, tb)
	if !tipShows(t, tt, long, long) {
		t.Fatalf("no tooltip of the cut-off name: %q", tt.Texts())
	}
	if tipShows(t, tt, "ann", "ann") {
		t.Fatal("a name that fits has a tooltip")
	}
	// The Hint wins over the full text; the Note of row 2 is cut off too
	// once the column is narrow.
	tb.SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Note": 60}})
	tt.Frame()
	if tipShows(t, tt, "n "+long, "n "+long) || !tt.HasText("note of "+long) {
		t.Fatalf("the Hint did not win: %q", tt.Texts())
	}
	// A Hint that is empty leaves the full text to SetCellHints.
	tb.SetRows([]acct{{3, long, 0}})
	tt.Frame()
	if !tipShows(t, tt, "n "+long, "n "+long) {
		t.Fatalf("with an empty Hint, no tooltip of the cut-off note: %q", tt.Texts())
	}
	// Turned off, no tooltip, though the row was built with one.
	tb.SetCellHints(false)
	tt.Frame()
	if tipShows(t, tt, long, long) {
		t.Fatal("a tooltip after SetCellHints(false)")
	}
}

// The tip is the cell's: the pointer on the blank part of a wide cell, past
// its short text, shows it.
func TestCellHintOnTheWholeCell(t *testing.T) {
	tt := tester(t, hintTable())
	var r ui.Rect // the text "n ann", at the start of its 300-wide cell
	for _, n := range tt.Nodes() {
		if n.Label == "n ann" && (r.W == 0 || n.Bounds.W < r.W) {
			r = n.Bounds
		}
	}
	if r.W <= 0 || r.W > 100 {
		t.Fatalf("the text n ann at %v", r)
	}
	leave(tt)
	tt.Move(r.X+r.W+150, r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("note of ann") {
		t.Fatalf("no tooltip on the blank part of the cell: %q", tt.Texts())
	}
}

// A badge's cell and a formatted row's cell show their column's Hint.
func TestCellHintOnBadgeAndFormat(t *testing.T) {
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 120,
			Hint: func(a *acct) string { return "name of " + a.name }},
		Column[acct]{Title: "State", Text: func(a *acct) string { return "ok" }, Width: 120,
			Badge: func(*acct) Status { return StatusSuccess },
			Hint:  func(a *acct) string { return "state of " + a.name }},
	).SetRows([]acct{{1, "ann", 0}}).Expand().
		SetRowFormat(func(*acct) CellFormat { return CellFormat{Bold: true} })
	tt := tester(t, tb)
	if !tipShows(t, tt, "ok", "state of ann") {
		t.Errorf("no Hint on a badge's cell: %q", tt.Texts())
	}
	if !tipShows(t, tt, "ann", "name of ann") {
		t.Errorf("no Hint on a formatted row's cell: %q", tt.Texts())
	}
	// A badge, which its pill draws, takes no full text from SetCellHints;
	// a formatted cell does, and the # cell, which takes the row's format,
	// has none.
	long := "a state much too long for its column"
	cut := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 60},
		Column[acct]{Title: "State", Text: func(a *acct) string { return long }, Width: 60,
			Badge: func(*acct) Status { return StatusSuccess }},
	).SetRows([]acct{{1, long, 0}}).Expand().SetCellHints(true).IndexColumn().
		SetRowFormat(func(*acct) CellFormat { return CellFormat{Bold: true} })
	tc := tester(t, cut)
	if !tipShows(t, tc, long, long) {
		t.Error("no full text of a formatted cell cut off")
	}
	leave(tc)
	before := textCount(tc, long)
	var r ui.Rect // the badge, the last element named long
	for _, n := range tc.Nodes() {
		if n.Label == long {
			r = n.Bounds
		}
	}
	tc.Move(r.X+10, r.Y+r.H/2)
	tc.Advance(time.Second)
	tc.Frame()
	if textCount(tc, long) > before {
		t.Error("a badge cut off shows its full text as a tooltip")
	}
}

// Turned on once the rows are built and kept, the hints reach them.
func TestCellHintsTurnedOnLate(t *testing.T) {
	long := "a name much too long for its column"
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 80},
	).SetRows([]acct{{1, long, 0}}).Expand()
	tt := tester(t, tb)
	if tipShows(t, tt, long, long) {
		t.Fatal("a tooltip before SetCellHints")
	}
	tb.SetCellHints(true)
	tt.Frame()
	if !tipShows(t, tt, long, long) {
		t.Fatal("no tooltip on a row built before SetCellHints(true)")
	}
}

// The column's width the user gave, its least width and the grid line
// decide whether a text is cut off.
func TestCellHintsWidthBounds(t *testing.T) {
	text := "abcdefgh"
	w := textWidth(t, text)
	grid := func(width float32, lines bool) *DataGridViewWidget[acct, int] {
		return DataGridView(func(a *acct) int { return a.id },
			Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: width},
			Column[acct]{Title: "Other", Text: func(a *acct) string { return "x" }, Width: 50},
		).SetRows([]acct{{1, text, 0}}).Expand().SetCellHints(true).SetGridLines(lines)
	}
	// Just wide enough: the text and the padding of 10 on each side.
	if tt := tester(t, grid(w+20+0.4, false)); tipShows(t, tt, text, text) {
		t.Error("a text that just fits has a tooltip")
	}
	if tt := tester(t, grid(w+20-1, false)); !tipShows(t, tt, text, text) {
		t.Error("a text 1 DIP too wide has no tooltip")
	}
	// The grid line takes 1 DIP of the cell.
	if tt := tester(t, grid(w+20+0.4, true)); !tipShows(t, tt, text, text) {
		t.Error("with grid lines, a text 0.6 DIP too wide has no tooltip")
	}
	// The width the user gave wins over the column's.
	tb := grid(w+20+0.4, false).SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": w}})
	if tt := tester(t, tb); !tipShows(t, tt, text, text) {
		t.Error("cut off by the width the user gave, no tooltip")
	}
	// A Width under MinWidth: the column is as wide as MinWidth.
	least := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 30, MinWidth: w + 20 + 0.4},
		Column[acct]{Title: "Other", Text: func(a *acct) string { return "x" }, Width: 50},
	).SetRows([]acct{{1, text, 0}}).Expand().SetCellHints(true)
	if tt := tester(t, least); tipShows(t, tt, text, text) {
		t.Error("a text that fits the column's MinWidth has a tooltip")
	}
}

// A column sharing the room left is cut off by its header's width: a window
// narrowed cuts its text, and widened again shows it whole.
func TestCellHintsSharedColumn(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("long ", 16))
	// Name has 800-100 DIPs, less its padding and the index column's 56, in
	// the 800-wide window, and 400-100 in the 400-wide one.
	if w := textWidth(t, long); w < 320 || w > 600 {
		t.Fatalf("the name measures %v", w)
	}
	for _, numbered := range []bool{false, true} {
		t.Run(fmt.Sprint("numbered=", numbered), func(t *testing.T) { sharedColumnHints(t, long, numbered) })
	}
}

func sharedColumnHints(t *testing.T, long string, numbered bool) {
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return "1" }, Width: 100},
	).SetRows([]acct{{1, long, 0}}).Expand().SetCellHints(true)
	if numbered {
		tb.IndexColumn()
	}
	tt := tester(t, tb)
	tt.Frame()
	if tipShows(t, tt, long, long) {
		t.Fatal("whole in an 800-wide window, the name has a tooltip")
	}
	tt.SetSize(400, 600)
	tt.Frame()
	tt.Frame()
	if !tipShows(t, tt, long, long) {
		t.Fatal("cut off in a 400-wide window, the name has no tooltip")
	}
	tt.SetSize(800, 600)
	tt.Frame()
	tt.Frame()
	if tipShows(t, tt, long, long) {
		t.Fatal("whole again, the name has a tooltip")
	}
}

// An image before the text takes room from it.
func TestCellHintsImage(t *testing.T) {
	text := "abcdefgh"
	w := textWidth(t, text)
	img := decodeImage(testPNG(t, color.White))
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: w + 20 + 10,
			Image: func(*acct) *ui.Bitmap { return img }},
		Column[acct]{Title: "Other", Text: func(a *acct) string { return "x" }, Width: 50},
	).SetRows([]acct{{1, text, 0}}).Expand().SetCellHints(true)
	tt := tester(t, tb)
	if !tipShows(t, tt, text, text) {
		t.Fatal("a text the image cuts off has no tooltip")
	}
}

// The setters, called from another goroutine while frames run, show at the
// next frame, on rows built before them, and nothing races (go test -race).
func TestColumnsFromGoroutine(t *testing.T) {
	long := "a name much too long for its column"
	tb := hintTable()
	tt := menuTester(t, tb)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 60 {
			<-tick
			switch i {
			case 10:
				tb.SetCellHints(true)
			case 20:
				tb.SetColumnChooser(true)
			case 30:
				tb.BestFit("Name")
			}
		}
	}()
	for range 60 {
		tick <- struct{}{}
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	tt.Frame()
	if w := widths(tb)["Name"]; w < 200 {
		t.Errorf("BestFit from a goroutine: Name %v", w)
	}
	// Narrow again, the long name is cut off and hints.
	tb.SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 80}})
	tt.Frame()
	if !tipShows(t, tt, long, long) {
		t.Errorf("SetCellHints from a goroutine: no tooltip of the cut-off name")
	}
	leave(tt)
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if m := tt.Menu(); !slices.Contains(m, "Best fit") {
		t.Errorf("SetColumnChooser from a goroutine: the menu %q", m)
	}
}

// With the three off, a grid builds what one never told of them does: the
// same pixels, the same nodes, no header menu.
func TestColumnsOffIsUnchanged(t *testing.T) {
	plain := tester(t, hintTable())
	off := hintTable().SetCellHints(true).SetColumnChooser(true)
	to := tester(t, off)
	off.SetCellHints(false).SetColumnChooser(false)
	to.Frame()
	plain.Frame()
	mu.Lock()
	menu := off.list.HeaderMenu
	mu.Unlock()
	if menu != nil {
		t.Fatal("ui has a header menu with the chooser off")
	}
	a, b := plain.Image(), to.Image()
	if !slices.Equal(a.Pix, b.Pix) {
		t.Fatal("the frames differ with the features off")
	}
	// Off, a cell does not read the pointer: moving over the rows builds none.
	var texts atomic.Int32
	counted := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { texts.Add(1); return a.name }, Width: 80},
	).SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}}).Expand()
	tc := tester(t, counted)
	n := texts.Load()
	rest(t, tc, "ann")
	rest(t, tc, "bob")
	if got := texts.Load() - n; got != 0 {
		t.Errorf("the pointer moving over the rows built cells %d times with the features off", got)
	}
	na, nb := plain.Nodes(), to.Nodes()
	if len(na) != len(nb) {
		t.Fatalf("%d nodes, %d with the features off", len(na), len(nb))
	}
	for i := range na {
		na[i].ID, nb[i].ID = 0, 0
		if na[i] != nb[i] {
			t.Fatalf("node %d: %+v, with the features off %+v", i, na[i], nb[i])
		}
	}
}

// A frame at rest with the three on allocates no more than one without.
func TestColumnsAllocNothingAtRest(t *testing.T) {
	plain := tester(t, testTable(manyAccts(200)...))
	tb := testTable(manyAccts(200)...).SetCellHints(true).SetColumnChooser(true).
		SetColumnsLayout(ui.TableLayout{Widths: map[string]float32{"Name": 40}})
	with := tester(t, tb)
	tb.BestFit("Score")
	for range 5 {
		plain.Frame()
		with.Frame()
	}
	a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with the features on, %v without", b, a)
	}
	// A column sharing the room left, whose header's width the hints read.
	shared := func() *DataGridViewWidget[acct, int] {
		return DataGridView(func(a *acct) int { return a.id },
			Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }},
			Column[acct]{Title: "Score", Text: func(a *acct) string { return "1" }, Width: 100},
		).SetRows(manyAccts(200)).Expand()
	}
	plain, with = tester(t, shared()), tester(t, shared().SetCellHints(true))
	for range 5 {
		plain.Frame()
		with.Frame()
	}
	a, b = allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with hints on a shared column, %v without", b, a)
	}
}
