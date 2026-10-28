package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
)

type dtRow struct {
	ID         int
	Name, Mail string
}

func dtRows() []dtRow {
	return []dtRow{{1, "charlie", "c@x.io"}, {2, "alpha", "a@x.io"}, {3, "bravo", "b@y.io"}}
}

func dtState() *DataTableState[dtRow, int] {
	return &DataTableState[dtRow, int]{
		Key: func(x *dtRow) int { return x.ID },
		Columns: []DataColumn[dtRow]{
			{
				TableColumn: TableColumn{Title: "Name", Width: 150},
				Text:        func(x *dtRow) string { return x.Name },
				Compare:     func(a, b *dtRow) int { return strings.Compare(a.Name, b.Name) },
			},
			{TableColumn: TableColumn{Title: "Mail", Width: 200}, Text: func(x *dtRow) string { return x.Mail }},
		},
	}
}

// dtShown returns the IDs of the rows s shows, in order.
func dtShown(s *DataTableState[dtRow, int]) []int {
	ids := make([]int, s.Len())
	for i := range ids {
		ids[i] = s.At(i).ID
	}
	return ids
}

func dtTester(s *DataTableState[dtRow, int], rows *[]dtRow) *Tester {
	return NewTester(func(c *Context) { DataTable(c, s, *rows).Grow(1) }, 600, 400)
}

// dtClickLast clicks the last element showing text: the search bar, above the
// table, can show the same text (the select shows a column's title, the box
// and the button both read "Search"), and the header or the button comes last.
func dtClickLast(t *testing.T, tt *Tester, title string) {
	t.Helper()
	var r Rect
	found := false
	for _, n := range tt.rt.labels {
		if n.text == title {
			r, found = n.r, true
		}
	}
	if !found {
		t.Fatalf("no element %q: %q", title, tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
}

// dtPanic returns what fn panics with, or "".
func dtPanic(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	fn()
	return ""
}

func TestDataTableShowsTheRowsInTheirOrder(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	if got := dtShown(s); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("rows shown: %v, want 1 2 3", got)
	}
	for _, text := range []string{"Name", "Mail", "charlie", "a@x.io"} {
		if !tt.HasText(text) {
			t.Errorf("%q is not shown: %q", text, tt.Texts())
		}
	}
}

func TestDataTableSortsByAHeaderClick(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	dtClickLast(t, tt, "Name")
	if want := (SortOrder{Column: "Name"}); s.Sort != want {
		t.Fatalf("a click on Name: %+v, want %+v", s.Sort, want)
	}
	if got := dtShown(s); !slices.Equal(got, []int{2, 3, 1}) {
		t.Fatalf("sorted by Name: %v, want 2 3 1", got)
	}
	dtClickLast(t, tt, "Name")
	if got := dtShown(s); !slices.Equal(got, []int{1, 3, 2}) {
		t.Fatalf("sorted by Name, descending: %v, want 1 3 2", got)
	}
	// Mail has no Compare: its header does not sort.
	before := s.Sort
	dtClickLast(t, tt, "Mail")
	if s.Sort != before {
		t.Fatalf("a click on Mail, which has no Compare, sorted by %+v", s.Sort)
	}
}

func TestDataTableKeepsChecksAndChoiceByKeyAcrossASort(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	s.Checked.Add(2)
	tt.Click("charlie")
	dtClickLast(t, tt, "Name")
	if !s.Checked.Has(2) || s.Checked.Len() != 1 {
		t.Errorf("checked after a sort: %d keys, has 2: %v", s.Checked.Len(), s.Checked.Has(2))
	}
	if cur := s.Current(); cur == nil || cur.Name != "charlie" {
		t.Errorf("the row chosen last after a sort: %+v, want charlie", cur)
	}
	if !s.Chosen.Has(1) {
		t.Errorf("chosen after a sort: has 1: %v", s.Chosen.Has(1))
	}
}

func TestDataTableSeesChangedAndAppendedRows(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Sort = SortOrder{Column: "Name"}
	tt := dtTester(s, &rows)
	if got := dtShown(s); !slices.Equal(got, []int{2, 3, 1}) {
		t.Fatalf("rows shown: %v, want 2 3 1", got)
	}
	// An appended row is placed by the next frame, without a call.
	rows = append(rows, dtRow{4, "bravissimo", "b@z.io"})
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{2, 4, 3, 1}) {
		t.Fatalf("after an append: %v, want 2 4 3 1", got)
	}
	// A changed row is placed again once the app says so.
	rows[1].Name = "zulu"
	s.Changed(1)
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{4, 3, 1, 2}) {
		t.Fatalf("after Changed(1): %v, want 4 3 1 2", got)
	}
	// Removed rows need Reset.
	rows = rows[:2]
	s.Reset()
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("after Reset: %v, want 1 2", got)
	}
}

func TestDataTableChecksOfRemovedRowsAreKept(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	s.Checked.Add(1)
	s.Checked.Add(3)
	rows = rows[:0]
	s.Reset()
	tt.Frame()
	if s.Checked.Len() != 2 || !s.Checked.Has(1) || !s.Checked.Has(3) {
		t.Fatalf("checked after the rows left: %d keys, want 2 (1 and 3)", s.Checked.Len())
	}
}

func TestDataTableMisuseHasClearPanics(t *testing.T) {
	rows := dtRows()
	noKey := dtState()
	noKey.Key = nil
	noText := dtState()
	noText.Columns[0].Text = nil
	none := dtState()
	none.Columns = nil
	for _, tc := range []struct {
		name string
		s    *DataTableState[dtRow, int]
		want string
	}{
		{"no Key", noKey, "Key is nil"},
		{"no Text", noText, `column "Name" has no Text`},
		{"no columns", none, "needs a column"},
	} {
		if got := dtPanic(func() { dtTester(tc.s, &rows) }); !strings.Contains(got, tc.want) {
			t.Errorf("%s: panic %q, want it to say %q", tc.name, got, tc.want)
		}
	}
	s := dtState()
	tt := dtTester(s, &rows)
	s.Columns = s.Columns[:1]
	if got := dtPanic(tt.Frame); !strings.Contains(got, "columns changed after the first frame") {
		t.Errorf("fewer columns: panic %q", got)
	}
}

func TestDataTableSearchMatches(t *testing.T) {
	rows := append(dtRows(), dtRow{4, "Alpine", "ALP@x.io"})
	for _, tc := range []struct {
		name            string
		column, keyword string
		want            []int
	}{
		{"substring", "Name", "alp", []int{2, 4}},
		{"case aside", "Name", "ALPHA", []int{2}},
		{"another column", "Mail", "x.io", []int{1, 2, 4}},
		{"empty keeps all", "Name", "", []int{1, 2, 3, 4}},
		{"spaces only keep all", "Name", "   ", []int{1, 2, 3, 4}},
		{"list of values is exact", "Name", "alpha|bravo", []int{2, 3}},
		{"list values are trimmed", "Name", " alpha | |bravo ", []int{2, 3}},
		{"list is not a substring", "Name", "alp|zzz", nil},
		{"list of nothing keeps all", "Name", "|", []int{1, 2, 3, 4}},
		{"nothing found", "Name", "zzz", nil},
	} {
		s := dtState()
		r := slices.Clone(rows)
		tt := dtTester(s, &r)
		s.SetSearch(tc.column, tc.keyword)
		tt.Frame()
		if got := dtShown(s); !slices.Equal(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("%s: search %q in %s shows %v, want %v", tc.name, tc.keyword, tc.column, got, tc.want)
		}
	}
}

func TestDataTableSearchKeepsChecksAndNoSearchColumnsOut(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Columns[1].NoSearch = true
	tt := dtTester(s, &rows)
	s.Checked.Add(2)
	s.SetSearch("Name", "charlie")
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{1}) {
		t.Fatalf("rows shown: %v, want 1", got)
	}
	if !s.Checked.Has(2) {
		t.Error("a row the search hides lost its check")
	}
	if got := dtPanic(func() { s.SetSearch("Mail", "x") }); !strings.Contains(got, `no searchable column "Mail"`) {
		t.Errorf("a search in a NoSearch column: panic %q", got)
	}
}

func TestDataTableFilterAndItsKey(t *testing.T) {
	s, rows := dtState(), dtRows()
	only, calls := 0, 0
	s.Filter = func(x *dtRow) bool { calls++; return only == 0 || x.ID == only }
	s.FilterKey = only
	tt := dtTester(s, &rows)
	if got := dtShown(s); len(got) != 3 {
		t.Fatalf("rows shown: %v, want all", got)
	}
	// The filter changed, the key did not: the rows are not filtered again.
	only = 2
	calls = 0
	tt.Frame()
	if got := dtShown(s); len(got) != 3 || calls != 0 {
		t.Fatalf("same FilterKey: rows %v, %d calls, want all and no call", got, calls)
	}
	s.FilterKey = only
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{2}) {
		t.Fatalf("new FilterKey: rows %v, want 2", got)
	}
	// Together with the search, and a filter that keeps nothing.
	s.SetSearch("Name", "charlie")
	tt.Frame()
	if got := dtShown(s); len(got) != 0 {
		t.Fatalf("filter and search: rows %v, want none", got)
	}
	if !tt.HasText("Search") {
		t.Error("the search bar is gone with the rows")
	}
}

func TestDataTableAnUnchangedSearchLeavesTheViewAlone(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	s.SetSearch("Name", "alpha")
	tt.Frame()
	// The rows are searched through their Text; the cells read only the row shown.
	read := map[string]int{}
	text := s.Columns[0].Text
	s.Columns[0].Text = func(x *dtRow) string { read[x.Name]++; return text(x) }
	s.SetSearch("Name", "alpha")
	tt.Frame()
	if read["charlie"] != 0 || read["bravo"] != 0 {
		t.Errorf("the same search again went through the rows again: %v", read)
	}
	if column, keyword := s.Search(); column != "Name" || keyword != "alpha" {
		t.Errorf("Search: %q %q", column, keyword)
	}
	// An empty keyword keeps the column out of the search.
	s.SetSearch("Mail", "")
	if column, keyword := s.Search(); column != "" || keyword != "" {
		t.Errorf("Search of nothing: %q %q", column, keyword)
	}
}

func TestDataTableSearchBox(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	tt.Click("Search") // the box, left of the button
	tt.Type("alp")
	if got := dtShown(s); len(got) != 3 {
		t.Fatalf("typing searched at once: rows %v, want all", got)
	}
	tt.Key(0, KeyEnter)
	if got := dtShown(s); !slices.Equal(got, []int{2}) {
		t.Fatalf("Enter: rows %v, want 2", got)
	}
	// The box keeps the focus: replace its text, and the button applies it.
	tt.Key(Cmd, KeyA)
	tt.Type("bra")
	if got := dtShown(s); !slices.Equal(got, []int{2}) {
		t.Fatalf("typing searched at once: rows %v, want 2", got)
	}
	dtClickLast(t, tt, "Search") // the button, right of the box
	if got := dtShown(s); !slices.Equal(got, []int{3}) {
		t.Fatalf("the button: rows %v, want 3", got)
	}
	// Clearing the box shows all again, without Enter.
	tt.Click("Search") // the box again: the button took the focus
	tt.Key(Cmd, KeyA)
	tt.Key(0, KeyBackspace)
	if got := dtShown(s); len(got) != 3 {
		t.Fatalf("a cleared box: rows %v, want all", got)
	}
}

func TestDataTableChoosingAnotherColumnSearchesIt(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	tt.Click("Search")
	tt.Type("x.io")
	tt.Key(0, KeyEnter)
	if got := dtShown(s); len(got) != 0 {
		t.Fatalf("x.io in Name: rows %v, want none", got)
	}
	tt.Click("Name")           // the select, above the header
	dtClickLast(t, tt, "Mail") // its list, drawn after the table
	if got := dtShown(s); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("x.io in Mail: rows %v, want 1 2", got)
	}
}

func TestDataTableIndexColumn(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Index = true
	tt := dtTester(s, &rows)
	for _, text := range []string{"#", "1", "2", "3"} {
		if !tt.HasText(text) {
			t.Errorf("%q is not shown: %q", text, tt.Texts())
		}
	}
	// The numbers are positions: a search leaves 1 alone, whichever row is left.
	s.SetSearch("Name", "bravo")
	tt.Frame()
	if !tt.HasText("1") || tt.HasText("2") || tt.HasText("3") {
		t.Errorf("one row shown: %q, want only 1", tt.Texts())
	}
	// The column does not sort, and the other columns still do.
	before := s.Sort
	dtClickLast(t, tt, "#")
	if s.Sort != before {
		t.Errorf("a click on #: sorted by %+v", s.Sort)
	}
	dtClickLast(t, tt, "Name")
	if s.Sort != (SortOrder{Column: "Name"}) {
		t.Errorf("a click on Name with a # column: %+v", s.Sort)
	}
}

func TestDataTableWithoutIndexHasNoNumbers(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	if tt.HasText("#") || tt.HasText("1") {
		t.Errorf("a # column without Index: %q", tt.Texts())
	}
}

func TestDataTableIndexOfNoRows(t *testing.T) {
	s, rows := dtState(), []dtRow(nil)
	s.Index = true
	tt := dtTester(s, &rows)
	if !tt.HasText("#") {
		t.Errorf("the # column of no rows: %q", tt.Texts())
	}
}

func TestDataTableCellsKeepTheirColumnsWithIndex(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Index = true
	built := ""
	s.Columns[1].Cell = func(c *Context, x *dtRow) { built += x.Mail + ";"; Text(c, "<"+x.Mail+">") }
	tt := dtTester(s, &rows)
	if !tt.HasText("<c@x.io>") || !tt.HasText("charlie") {
		t.Errorf("cells with a # column: %q (Cell built %q)", tt.Texts(), built)
	}
}

func TestDataTableIndexColumnFitsTheDigitsOfTheLastRow(t *testing.T) {
	nameX := func(n int) float32 {
		s := dtState()
		s.Index = true
		rows := make([]dtRow, n)
		for i := range rows {
			rows[i] = dtRow{ID: i, Name: fmt.Sprintf("n%d", i), Mail: "m"}
		}
		tt := dtTester(s, &rows)
		var x float32
		for _, l := range tt.rt.labels {
			if l.text == "Mail" { // the header of the column right of Name
				x = l.r.X
			}
		}
		return x
	}
	two, ninetyNine, hundred := nameX(5), nameX(99), nameX(100)
	if two != ninetyNine {
		t.Errorf("columns move between 5 and 99 rows: %v, %v", two, ninetyNine)
	}
	if hundred <= ninetyNine {
		t.Errorf("a third digit does not widen #: %v, then %v", ninetyNine, hundred)
	}
}

func TestDataTableCopy(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Index = true
	tt := dtTester(s, &rows)
	// By default: the Text of the columns, without #, a line a row, in the
	// order shown.
	tt.Click("alpha")
	tt.ClickWith(Cmd, "charlie")
	tt.Key(Cmd, KeyC)
	if got, want := tt.Clipboard(), "charlie\tc@x.io\nalpha\ta@x.io"; got != want {
		// charlie is row 1 and alpha row 2 while the rows are in the app's order.
		t.Fatalf("default Copy: %q, want %q", got, want)
	}
	// The app's Copy gets the rows chosen, pointing into its slice.
	var got []*dtRow
	s.Copy = func(rows []*dtRow) string {
		got = rows
		return "mine"
	}
	tt.Key(Cmd, KeyC)
	if tt.Clipboard() != "mine" || len(got) != 2 || got[0] != &rows[0] || got[1] != &rows[1] {
		t.Fatalf("app Copy: clipboard %q, rows %v", tt.Clipboard(), got)
	}
}

func TestDataTableRowMenu(t *testing.T) {
	s, rows := dtState(), dtRows()
	var opened [][]string
	s.RowMenu = func(m *Menu, picked []*dtRow) {
		if m.Item("Open").Chosen() {
			names := make([]string, len(picked))
			for i, x := range picked {
				names[i] = x.Name
			}
			opened = append(opened, names)
		}
	}
	tt := dtTester(s, &rows)
	tt.Click("alpha")
	tt.ClickWith(Cmd, "bravo")
	if err := tt.RightClick("bravo"); err != nil {
		t.Fatal(err)
	}
	if m := tt.Menu(); !slices.Equal(m, []string{"Open"}) {
		t.Fatalf("menu: %q", m)
	}
	if err := tt.ChooseMenuItem("Open"); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"alpha", "bravo"}}; !slices.EqualFunc(opened, want, slices.Equal) {
		t.Fatalf("Open got %v, want %v (the rows chosen, in the order shown)", opened, want)
	}
	// Without a RowMenu, a right click opens no menu.
	s.RowMenu = nil
	tt.Frame()
	tt.RightClick("alpha") // an error is fine: no menu opens
	if m := tt.Menu(); m != nil {
		t.Errorf("a menu without RowMenu: %q", m)
	}
}

func TestDataTableCopyFollowsTheSort(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Sort = SortOrder{Column: "Name"}
	tt := dtTester(s, &rows)
	tt.Click("charlie")
	tt.ClickWith(Cmd, "alpha")
	tt.Key(Cmd, KeyC)
	// charlie comes first in the app's slice, alpha first in the order shown.
	if got, want := tt.Clipboard(), "alpha\ta@x.io\ncharlie\tc@x.io"; got != want {
		t.Fatalf("Copy after a sort: %q, want %q", got, want)
	}
}

// BenchmarkDataTable1M is a frame of a DataTable of a million rows, against
// a Table the app wires itself with a TableView, with the same columns: the
// layer's overhead is the search row, whose three widgets are painted at each
// frame (without it the two are as fast), and, with the # column, one more
// cell a row. The Sync that finds nothing changed costs nothing to see.
func BenchmarkDataTable1M(b *testing.B) {
	rows := make([]dtRow, 1_000_000)
	for i := range rows {
		rows[i] = dtRow{i, fmt.Sprintf("name %d", i), "x@x.io"}
	}
	for _, index := range []bool{false, true} {
		b.Run(fmt.Sprintf("plain-index=%v", index), func(b *testing.B) {
			var list ListState
			var checked Selection[int]
			cursor := -1
			view := NewTableView(map[string]func(a, b *dtRow) int{})
			// Name sorts, as the DataTable's does: a sortable header costs a frame.
			cols := []TableColumn{{Title: "Name", Width: 150, Sortable: true}, {Title: "Mail", Width: 200}}
			var order SortOrder
			list.Sort = &order
			if index {
				cols = append([]TableColumn{{Title: "#", Fixed: true, Width: 40}}, cols...)
			}
			list.Selected, list.Checked = &cursor, &checked
			list.Key = func(row int) any { return rows[view.Row(row)].ID }
			list.Label = func(row int) string { return rows[view.Row(row)].Name }
			tt := NewTester(func(c *Context) {
				view.Sync(rows)
				list.Version = view.Version()
				Table(c, &list, cols, view.Len(), func(row, col int) {
					x := &rows[view.Row(row)]
					if index {
						if col == 0 {
							Text(c, strconv.Itoa(row+1)).SingleLine()
							return
						}
						col--
					}
					if col == 0 {
						Text(c, x.Name).SingleLine()
					} else {
						Text(c, x.Mail).SingleLine()
					}
				}).Grow(1)
			}, 600, 400)
			tt.Move(400, 300)
			b.ReportAllocs()
			for b.Loop() {
				tt.Scroll(400, 300, 0, 97)
			}
		})
		b.Run(fmt.Sprintf("datatable-index=%v", index), func(b *testing.B) {
			s := dtState()
			s.Index = index
			tt := dtTester(s, &rows)
			tt.Move(400, 300)
			b.ReportAllocs()
			for b.Loop() {
				tt.Scroll(400, 300, 0, 97)
			}
		})
	}
}

func TestDataTableSearchesTheColumnOfTheSelectNotAnotherOfTheSameTitle(t *testing.T) {
	s, rows := dtState(), dtRows()
	// The first "Name" is not searchable; the second, the mails, is.
	s.Columns[0].NoSearch = true
	s.Columns[1].Title, s.Columns[1].ID = "Name", "mail"
	tt := dtTester(s, &rows)
	s.SetSearch("Name", "x.io")
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("search in the searchable Name: rows %v, want 1 2", got)
	}
}

func TestDataTableSearchSurvivesARenamedColumn(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	s.Columns[0].Title = "Nom" // a language change relabels the columns
	tt.Click("Search")
	tt.Type("alp")
	if got := dtPanic(func() { tt.Key(0, KeyEnter) }); got != "" {
		t.Fatalf("a search after a column was renamed panics: %s", got)
	}
	if got := dtShown(s); !slices.Equal(got, []int{2}) {
		t.Fatalf("rows %v, want 2", got)
	}
}

func TestDataTableCheckboxesAreChecksOfKeys(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	if err := tt.Click("Check alpha"); err != nil {
		t.Fatal(err)
	}
	if !s.Checked.Has(2) || s.Checked.Len() != 1 {
		t.Fatalf("a click on a checkbox: %d keys, has 2: %v", s.Checked.Len(), s.Checked.Has(2))
	}
	s.SetSearch("Name", "charlie")
	tt.Frame()
	if !s.Checked.Has(2) {
		t.Error("the check of a row the search hides is gone")
	}
}

func TestDataTableSearchBarSpeaksTheLanguageOfTheApp(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	tt.SetLocale("zh-Hans")
	tt.Frame()
	if !tt.HasText("搜索") || tt.HasText("Column") || !slices.Contains(tt.Texts(), "列") {
		t.Errorf("the search bar in Chinese: %q", tt.Texts())
	}
}

func TestDataTableNoSearchBarHidesOnlyTheBar(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.NoSearchBar = true
	tt := dtTester(s, &rows)
	if tt.HasText("Search") || tt.HasText("Column") {
		t.Errorf("the bar is shown with NoSearchBar: %q", tt.Texts())
	}
	// The app's own bar drives the table through SetSearch and Search.
	s.SetSearch("Name", "alp")
	tt.Frame()
	if got := dtShown(s); !slices.Equal(got, []int{2}) {
		t.Fatalf("SetSearch with NoSearchBar: rows %v, want 2", got)
	}
	if column, keyword := s.Search(); column != "Name" || keyword != "alp" {
		t.Errorf("Search: %q %q", column, keyword)
	}
}

func TestDataTableWithoutSearchableColumnsHasNoBar(t *testing.T) {
	s, rows := dtState(), dtRows()
	for i := range s.Columns {
		s.Columns[i].NoSearch = true
	}
	tt := dtTester(s, &rows)
	if tt.HasText("Search") || tt.HasText("Column") {
		t.Errorf("a bar with nothing to search: %q", tt.Texts())
	}
	if got := dtShown(s); len(got) != 3 {
		t.Errorf("rows %v, want all", got)
	}
	if got := dtPanic(func() { s.SetSearch("Name", "x") }); !strings.Contains(got, `no searchable column "Name"`) {
		t.Errorf("SetSearch with no searchable column: panic %q", got)
	}
}

func TestDataTableVersionChangesWithTheRowsShown(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	v := s.Version()
	tt.Frame()
	if s.Version() != v {
		t.Errorf("a frame with nothing changed moved the version: %d, was %d", s.Version(), v)
	}
	dtClickLast(t, tt, "Name")
	if s.Version() == v {
		t.Error("a sort did not move the version")
	}
	v = s.Version()
	s.SetSearch("Name", "alpha")
	tt.Frame()
	if s.Version() == v {
		t.Error("a search did not move the version")
	}
}

func TestDataTableAColumnTitledHashWorksWithIndex(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.Index = true
	s.Columns[1].Title = "#"
	var tt *Tester
	if got := dtPanic(func() { tt = dtTester(s, &rows) }); got != "" {
		t.Fatalf("a column titled # with Index panics: %s", got)
	}
	for _, text := range []string{"1", "2", "3", "c@x.io"} {
		if !tt.HasText(text) {
			t.Errorf("%q is not shown: %q", text, tt.Texts())
		}
	}
}

func TestDataTableAnUncomparableFilterKeyPanicsAtOnce(t *testing.T) {
	// A struct holding a slice in an interface compares at run time only.
	for _, key := range []any{[]int{1}, map[string]int{}, func() {}, struct{ v any }{[]int{1}}} {
		s, rows := dtState(), dtRows()
		s.FilterKey = key
		got := dtPanic(func() { dtTester(s, &rows) })
		want := fmt.Sprintf("ui: DataTable FilterKey %T is not comparable", key)
		if got != want {
			t.Errorf("FilterKey %T: panic %q, want %q", key, got, want)
		}
	}
}

// The search finds what strings.ToLower finds, on the texts lowered as on
// the keyword, whatever the letters: Vietnamese, İ, invalid UTF-8.
func TestDataMatcherFoldsAsToLower(t *testing.T) {
	cells := []string{"Thành công", "THÀNH CÔNG", "Đang chạy", "ĐANG", "İstanbul", "i̇stanbul", "a\xffb", "A\xffB", "abc", "ABC", "", "Ǆ"}
	kws := []string{"thành", "THÀNH công", "đang", "İ", "i̇", "\xff", "a\xffb", "b", "ǆ", "Ǆ", "ǅ"}
	text := func(x *dtRow) string { return x.Name }
	for _, cell := range cells {
		for _, kw := range kws {
			want := strings.Contains(strings.ToLower(cell), strings.ToLower(kw))
			if got := dataMatcher(text, kw)(&dtRow{Name: cell}); got != want {
				t.Errorf("dataMatcher(%q) on %q = %v, want %v", kw, cell, got, want)
			}
		}
	}
}

// The search goes through every row: an allocation a row is a million at a
// million rows.
func TestDataMatcherDoesNotAllocatePerRow(t *testing.T) {
	x := &dtRow{Name: "Thành công"}
	text := func(x *dtRow) string { return x.Name }
	for _, kw := range []string{"thành", "Thành công|Lỗi"} {
		keep := dataMatcher(text, kw)
		if !keep(x) {
			t.Fatalf("%q does not keep the row searched for", kw)
		}
		if n := allocrun.Least(100, func() { keep(x) }); n != 0 {
			t.Errorf("%q: %v allocations a row", kw, n)
		}
	}
}

func TestDataTableColumnMatch(t *testing.T) {
	s, rows := dtState(), dtRows()
	var got []string
	s.Columns = append(s.Columns, DataColumn[dtRow]{
		TableColumn: TableColumn{Title: "ID", Width: 60},
		Text:        func(x *dtRow) string { return strconv.Itoa(x.ID) },
		Match: func(x *dtRow, keyword string) bool {
			got = append(got, keyword)
			n, err := strconv.Atoi(strings.TrimPrefix(keyword, ">="))
			return err == nil && x.ID >= n
		},
	})
	tt := dtTester(s, &rows)
	s.SetSearch("ID", " >=2 ")
	tt.Frame()
	if ids := dtShown(s); !slices.Equal(ids, []int{2, 3}) {
		t.Fatalf("Match >=2: rows %v, want 2 3", ids)
	}
	if got[0] != ">=2" {
		t.Errorf("Match got %q, want the keyword trimmed", got[0])
	}
	// Match replaces the | list too.
	got = nil
	s.SetSearch("ID", "1|2")
	tt.Frame()
	if len(got) == 0 || got[0] != "1|2" {
		t.Errorf("Match got %q for 1|2, want the raw keyword", got)
	}
	// Columns without Match search as before.
	s.SetSearch("Name", "alp")
	tt.Frame()
	if ids := dtShown(s); !slices.Equal(ids, []int{2}) {
		t.Fatalf("Name alp: rows %v, want 2", ids)
	}
}

func dtSortState() *DataTableState[dtRow, int] {
	s := dtState()
	s.Columns[1].Compare = func(a, b *dtRow) int { return strings.Compare(a.Mail, b.Mail) }
	s.MultiSort = true
	return s
}

func TestDataTableMultiSort(t *testing.T) {
	rows := []dtRow{{1, "bravo", "b@x.io"}, {2, "alpha", "z@x.io"}, {3, "bravo", "a@x.io"}}
	s := dtSortState()
	tt := dtTester(s, &rows)
	dtClickLast(t, tt, "Name")
	tt.ClickWith(Cmd, "Mail") // the header: the select shows Name
	if want := []SortOrder{{Column: "Name"}, {Column: "Mail"}}; !slices.Equal(s.SortBy, want) {
		t.Fatalf("SortBy %+v, want %+v", s.SortBy, want)
	}
	if ids := dtShown(s); !slices.Equal(ids, []int{2, 3, 1}) {
		t.Fatalf("by Name then Mail: rows %v, want 2 3 1", ids)
	}
	if s.Sort != (SortOrder{}) {
		t.Errorf("Sort changed with MultiSort: %+v", s.Sort)
	}
	dtClickLast(t, tt, "Mail")
	if want := []SortOrder{{Column: "Mail"}}; !slices.Equal(s.SortBy, want) {
		t.Fatalf("a click: SortBy %+v, want %+v", s.SortBy, want)
	}
}

func TestDataTableMultiSortByAColumnWithoutCompare(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.MultiSort, s.SortBy = true, []SortOrder{{Column: "Mail"}}
	dtTester(s, &rows)
	if ids := dtShown(s); !slices.Equal(ids, []int{1, 2, 3}) {
		t.Fatalf("by Mail, which has no Compare: rows %v, want 1 2 3", ids)
	}
}

func TestDataTableMultiSortCannotChangeAfterTheFirstFrame(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	s.MultiSort = true
	if msg := dtPanic(tt.Frame); !strings.Contains(msg, "MultiSort") {
		t.Fatalf("panic %q, want one naming MultiSort", msg)
	}
}

func TestDataTableSearchDelay(t *testing.T) {
	s, rows := dtState(), dtRows()
	s.SearchDelay = 200 * time.Millisecond
	tt := dtTester(s, &rows)
	tt.Click("Search") // the box
	tt.Type("alp")
	if ids := dtShown(s); len(ids) != 3 {
		t.Fatalf("at once: rows %v, want all", ids)
	}
	tt.Advance(300 * time.Millisecond)
	if ids := dtShown(s); !slices.Equal(ids, []int{2}) {
		t.Fatalf("after the delay: rows %v, want 2", ids)
	}
	// Enter still applies at once.
	tt.Key(Cmd, KeyA)
	tt.Type("bra")
	tt.Key(0, KeyEnter)
	if ids := dtShown(s); !slices.Equal(ids, []int{3}) {
		t.Fatalf("Enter: rows %v, want 3", ids)
	}
	// Nothing pending after it: a later frame changes nothing.
	if tt.FramePending() {
		t.Fatal("after Enter, the search still asks for a frame")
	}
	tt.Advance(300 * time.Millisecond)
	if ids := dtShown(s); !slices.Equal(ids, []int{3}) {
		t.Fatalf("after Enter and the delay: rows %v, want 3", ids)
	}
}

func TestDataTableFooter(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	if tt.HasText("3 rows") {
		t.Fatal("a footer without Footer")
	}
	s.Footer = true
	tt.Frame()
	if !tt.HasText("3 rows") {
		t.Fatalf("footer: %q", tt.Texts())
	}
	s.Checked.Add(1)
	s.Checked.Add(2)
	tt.Frame()
	if !tt.HasText("3 rows · 2 checked") {
		t.Fatalf("2 checked: %q", tt.Texts())
	}
	// A search hiding a checked row: checked counts every checked key.
	s.SetSearch("Name", "alp")
	tt.Frame()
	if !tt.HasText("1 row · 2 checked") {
		t.Fatalf("1 row shown: %q", tt.Texts())
	}
}
