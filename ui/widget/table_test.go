package widget

import (
	"cmp"
	"image/color"
	"slices"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

type acct struct {
	id    int
	name  string
	score int
}

// testTable is a table of accounts by id: Name sorts and searches as
// text, Score sorts as a number.
func testTable(rows ...acct) *TableWidget[acct, int] {
	return Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Width: 200, Sortable: true, Filterable: true},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) },
			Compare: func(a, b *acct) int { return cmp.Compare(a.score, b.score) },
			Width:   100, Align: End, Sortable: true},
	).SetRows(rows).Expand()
}

// inOrder fails unless names show from the top down in that order.
func inOrder(t *testing.T, tt *ui.Tester, names ...string) {
	t.Helper()
	for i := 1; i < len(names); i++ {
		if a, b := find(t, tt, names[i-1]), find(t, tt, names[i]); b.Y <= a.Y {
			t.Fatalf("%q (y %v) not above %q (y %v)", names[i-1], a.Y, names[i], b.Y)
		}
	}
}

func TestTableShowsRows(t *testing.T) {
	tt := tester(t, testTable(acct{1, "ann", 30}, acct{2, "bob", 10}))
	for _, s := range []string{"Name", "Score", "ann", "bob", "30", "10"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestTableRowsAppendAndReplace(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{1, "amy", 2}) // the last of a key stays
	if tb.Len() != 1 {
		t.Fatalf("Len %d after SetRows with one key twice", tb.Len())
	}
	tb.AddRow(acct{2, "bob", 3}).AddRow(acct{1, "ada", 4})
	tt := tester(t, tb)
	if tb.Len() != 2 || tt.HasText("amy") || !tt.HasText("ada") {
		t.Fatalf("Len %d, texts %q", tb.Len(), tt.Texts())
	}
	inOrder(t, tt, "ada", "bob") // appended after; replaced in place
	tb.UpdateRow(2, func(a *acct) { a.name = "ben" })
	tt.Frame()
	if !tt.HasText("ben") {
		t.Fatalf("UpdateRow not shown: %q", tt.Texts())
	}
	if r, ok := tb.Row(2); !ok || r.name != "ben" {
		t.Fatalf("Row(2) = %+v, %v", r, ok)
	}
	tb.DeleteRows(1, 99)
	tt.Frame()
	if tb.Len() != 1 || tt.HasText("ada") {
		t.Fatalf("after DeleteRows: Len %d, texts %q", tb.Len(), tt.Texts())
	}
	if _, ok := tb.Row(1); ok {
		t.Fatal("Row(1) after DeleteRows")
	}
	tb.Clear()
	tt.Frame()
	if tb.Len() != 0 || tt.HasText("ben") {
		t.Fatalf("after Clear: Len %d, texts %q", tb.Len(), tt.Texts())
	}
}

func TestTableSortsByHeader(t *testing.T) {
	// As text, the scores would sort 10, 100, 9.
	tt := tester(t, testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100}))
	click(t, tt, "Name")
	inOrder(t, tt, "ann", "bob", "cat")
	click(t, tt, "Name")
	inOrder(t, tt, "cat", "bob", "ann")
	click(t, tt, "Score")
	inOrder(t, tt, "ann", "cat", "bob")
}

func TestTableClearSortShowsRowsAsGiven(t *testing.T) {
	tb := testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100})
	tt := tester(t, tb)
	click(t, tt, "Name")
	inOrder(t, tt, "ann", "bob", "cat")
	tb.ClearSort()
	tt.Frame()
	inOrder(t, tt, "cat", "ann", "bob")
}

func TestTableIndexColumnShowsShownPositions(t *testing.T) {
	// Scores with no 1 or 2 in them: Find matches whole texts, the first.
	// A name finds its row, which ui labels with the first column's text.
	tt := tester(t, testTable(acct{1, "cat", 30}, acct{2, "ann", 40}).IndexColumn())
	click(t, tt, "Name")
	for _, p := range [][2]string{{"1", "ann"}, {"2", "cat"}} {
		n, row := find(t, tt, p[0]), find(t, tt, p[1])
		if y := n.Y + n.H/2; y < row.Y || y > row.Y+row.H {
			t.Fatalf("%s at %v, not in the row of %s at %v", p[0], n, p[1], row)
		}
	}
}

func TestTableCellsShowColorAndImage(t *testing.T) {
	png := testPNG(t, color.White)
	bmp := decodeImage(png)
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Color: func(a *acct) ui.Color { return ui.RGB(200, 0, 0) },
			Image: func(a *acct) *ui.Bitmap { return bmp }},
	).SetRows([]acct{{1, "ann", 0}}).Expand()
	tt := tester(t, tb)
	if !tt.HasText("ann") {
		t.Fatalf("no ann in %q", tt.Texts())
	}
}

func TestTableStaleRowHasZeroKey(t *testing.T) {
	tb := testTable(acct{1, "a", 0}, acct{2, "b", 0}, acct{3, "c", 0})
	tester(t, tb)
	tb.DeleteRows(1, 2, 3) // the view still shows three rows until it syncs
	if k := tb.keyAt(2); k != 0 {
		t.Fatalf("keyAt(2) of a deleted row = %v", k)
	}
	if s := tb.labelAt(2); s != "" {
		t.Fatalf("labelAt(2) of a deleted row = %q", s)
	}
}

func TestTableChecksRows(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn()
	tt := tester(t, tb)
	click(t, tt, "Check bob")
	if got := tb.Checked(); !slices.Equal(got, []int{2}) || !tb.IsChecked(2) {
		t.Fatalf("Checked %v after a click on bob's box", got)
	}
	click(t, tt, "Check all")
	if got := tb.Checked(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Checked %v after Check all", got)
	}
	tb.SetChecked([]int{1, 3}, false)
	tt.Frame()
	if got := tb.Checked(); !slices.Equal(got, []int{2}) {
		t.Fatalf("Checked %v after SetChecked(1, 3, false)", got)
	}
	click(t, tt, "Check all") // some checked: Check all checks the rest
	if got := tb.Checked(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("ui did not show SetChecked: Checked %v", got)
	}
}

func TestTableCheckedFollowsTheShownOrder(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn()
	tb.SetChecked([]int{1, 3}, true)
	tt := tester(t, tb)
	click(t, tt, "Name")
	click(t, tt, "Name") // descending
	if got := tb.Checked(); !slices.Equal(got, []int{3, 1}) {
		t.Fatalf("Checked %v, sorted by Name descending", got)
	}
}

func TestTableSelectsRows(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3})
	tt := tester(t, tb)
	click(t, tt, "cat")
	if err := tt.ClickWith(ui.Cmd, "ann"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := tb.Selected(); !slices.Equal(got, []int{1, 3}) || tb.CountSelected() != 2 {
		t.Fatalf("Selected %v (%d)", got, tb.CountSelected())
	}
}

func TestTableAutoCheckTogglesSelectedByMajority(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn()
	tb.SetChecked([]int{1}, true)
	tt := tester(t, tb)
	click(t, tt, "ann")
	if err := tt.ClickWith(ui.Cmd, "bob"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tb.AutoCheck() // one of two checked: half, so it unchecks both
	if got := tb.Checked(); len(got) != 0 {
		t.Fatalf("Checked %v after AutoCheck with half checked", got)
	}
	tb.AutoCheck() // none checked: it checks both
	if got := tb.Checked(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("Checked %v after AutoCheck with none checked", got)
	}
}

func TestTableDeleteRowsUnchecksAndUnselects(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn()
	tb.SetChecked([]int{1, 2}, true)
	tt := tester(t, tb)
	click(t, tt, "ann")
	tb.DeleteRows(1)
	tt.Frame()
	if tb.IsChecked(1) || tb.CountSelected() != 0 {
		t.Fatalf("deleted row: checked %v, selected %d", tb.IsChecked(1), tb.CountSelected())
	}
	tb.AddRow(acct{1, "ann", 1}) // gota's H3: it came back checked
	if tb.IsChecked(1) || !tb.IsChecked(2) {
		t.Fatalf("checked 1: %v, 2: %v", tb.IsChecked(1), tb.IsChecked(2))
	}
	tb.SetRows([]acct{{2, "bob", 2}, {3, "cat", 3}}) // keeps the check of 2
	if !slices.Equal(tb.Checked(), []int{2}) {
		t.Fatalf("Checked %v after SetRows", tb.Checked())
	}
}

func TestTableSearchKeepsMatchesWhateverTheCase(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "Hannah", 3})
	tt := tester(t, tb)
	tb.Search("Name", "AN")
	tt.Frame()
	if !tt.HasText("ann") || !tt.HasText("Hannah") || tt.HasText("bob") {
		t.Fatalf("search AN: %q", tt.Texts())
	}
}

func TestTableSearchShowsAllForBlankOrUnknown(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2})
	tt := tester(t, tb)
	for _, s := range [][2]string{{"Name", "   "}, {"Score", "1"}, {"Nope", "ann"}} {
		tb.Search("Name", "ann")
		tt.Frame()
		tb.Search(s[0], s[1])
		tt.Frame()
		if !tt.HasText("ann") || !tt.HasText("bob") {
			t.Errorf("Search(%q, %q): %q", s[0], s[1], tt.Texts())
		}
	}
}

func TestTableSearchUsesTheColumnFilter(t *testing.T) {
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Filterable: true,
			Filter: func(a *acct, kw string) bool { return a.name == kw }},
	).SetRows([]acct{{1, "ann", 0}, {2, "anna", 0}}).Expand()
	tt := tester(t, tb)
	tb.Search("Name", "ann")
	tt.Frame()
	if !tt.HasText("ann") || tt.HasText("anna") {
		t.Fatalf("exact filter: %q", tt.Texts())
	}
}

func TestTablePanelSearches(t *testing.T) {
	asked := make(chan [2]string, 2)
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).BottomPanel().
		OnBeforeSearch(func(col, kw string) { asked <- [2]string{col, kw} }).
		SetSearchPlaceholder("Find")
	tt := tester(t, tb)
	// A placeholder is no label: click the keyword, 240 wide, left of the
	// button with the panel's gap of 12 between.
	b := find(t, tt, "Search")
	clickKeyword := func() { tt.ClickAt(b.X-12-120, b.Y+b.H/2); tt.Frame() }
	clickKeyword()
	tt.Type(" bob ")
	click(t, tt, "Search")
	if got := wait(t, asked); got != [2]string{"Name", "bob"} {
		t.Fatalf("OnBeforeSearch got %q", got)
	}
	Update(func() {})
	tt.Frame()
	if tt.HasText("ann") || !tt.HasText("bob") {
		t.Fatalf("after the search button: %q", tt.Texts())
	}
	tb.Search("Name", "") // empties the keyword: its placeholder shows again
	tt.Frame()
	clickKeyword()
	tt.Type("ann")
	tt.Key(0, ui.KeyEnter)
	if got := wait(t, asked); got != [2]string{"Name", "ann"} {
		t.Fatalf("Enter: OnBeforeSearch got %q", got)
	}
	Update(func() {})
	tt.Frame()
	if !tt.HasText("ann") || tt.HasText("bob") {
		t.Fatalf("after Enter: %q", tt.Texts())
	}
}

func TestTablePanelCounts(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "cat", 3}).CheckboxColumn().BottomPanel()
	tt := tester(t, tb)
	click(t, tt, "Check bob")
	click(t, tt, "cat")
	tt.Frame()
	for _, s := range []string{"Rows: 3", "Selected: 1", "Checked: 1"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	tb.PanelText(PanelText{Rows: "Tổng số dòng: %d", Selected: "Đang chọn: %d", Checked: "Đã đánh dấu: %d", Search: "Tìm kiếm"})
	tt.Frame()
	for _, s := range []string{"Tổng số dòng: 3", "Đang chọn: 1", "Đã đánh dấu: 1", "Tìm kiếm"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestTablePanelWithoutFilterableColumns(t *testing.T) {
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }},
	).SetRows([]acct{{1, "ann", 0}}).BottomPanel().SetSearchPlaceholder("Find").Expand()
	tt := tester(t, tb)
	if !tt.HasText("Rows: 1") || tt.HasText("Search") || tt.HasText("Find") || tt.HasText("Checked") {
		t.Fatalf("panel: %q", tt.Texts())
	}
}

func TestTableCheckAllChecksOnlyShownRows(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}, acct{3, "Hannah", 3}).CheckboxColumn()
	tb.SetChecked([]int{2}, true)
	tt := tester(t, tb)
	tb.Search("Name", "an")
	tt.Frame()
	click(t, tt, "Check all")
	if got := tb.Checked(); !slices.Equal(got, []int{1, 3}) {
		t.Fatalf("Checked %v under the search", got)
	}
	tb.Search("Name", "")
	tt.Frame()
	if got := tb.Checked(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("hidden bob lost its check: %v", got)
	}
}
