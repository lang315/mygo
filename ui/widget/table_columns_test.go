package widget

import (
	"cmp"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

// threeColumns is a table of Name, Score and Note, with the columns hidden
// that hidden names.
func threeColumns(n int, hidden ...string) *DataGridViewWidget[acct, int] {
	hide := func(title string) bool {
		for _, h := range hidden {
			if h == title {
				return true
			}
		}
		return false
	}
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{i + 1, fmt.Sprintf("name%03d", i+1), (i * 37) % 101}
	}
	return DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Width: 200, Sortable: true, Filterable: true, Hidden: hide("Name")},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) },
			Compare: func(a, b *acct) int { return cmp.Compare(a.score, b.score) },
			Width:   100, Align: End, Sortable: true, Hidden: hide("Score")},
		Column[acct]{Title: "Note", Text: func(a *acct) string { return fmt.Sprint("note", a.id) },
			Width: 150, Filterable: true, Hidden: hide("Note")},
	).SetRows(rows).Expand()
}

func headers(tt *ui.Tester) (titles []string) {
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleColumnHeader {
			titles = append(titles, n.Label)
		}
	}
	return
}

func sameTitles(a, b []string) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

func TestColumnHiddenFromTheStart(t *testing.T) {
	tb := threeColumns(3, "Score")
	tt := tester(t, tb)
	if got := headers(tt); !sameTitles(got, []string{"Name", "Note"}) {
		t.Fatalf("headers %q", got)
	}
	if !tt.HasText("name001") || !tt.HasText("note1") || tt.HasText("Score") || tt.HasText("37") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if tb.ColumnVisible("Score") || !tb.ColumnVisible("Name") || tb.ColumnVisible("nope") {
		t.Fatal("ColumnVisible disagrees with Hidden")
	}
}

func TestColumnSetVisibleHidesAndShowsInPlace(t *testing.T) {
	tb := threeColumns(3)
	tt := tester(t, tb)
	tb.SetColumnVisible("Score", false)
	settleTable(tt)
	if got := headers(tt); !sameTitles(got, []string{"Name", "Note"}) || tt.HasText("37") {
		t.Fatalf("headers %q, texts %q", got, tt.Texts())
	}
	tb.SetColumnVisible("Score", true)
	settleTable(tt)
	if got := headers(tt); !sameTitles(got, []string{"Name", "Score", "Note"}) || !tt.HasText("37") {
		t.Fatalf("headers %q, texts %q", got, tt.Texts())
	}
	tb.SetColumnVisible("nope", false).SetColumnVisible("Name", true) // no such column; no change
	settleTable(tt)
	if got := headers(tt); len(got) != 3 {
		t.Fatalf("headers %q", got)
	}
}

// The cells of the columns that stay move to where the hidden one was; rows
// the table keeps are built again.
func TestColumnHiddenCellsMoveForKeptRows(t *testing.T) {
	tb := threeColumns(200)
	tt := tester(t, tb)
	settleTable(tt)
	name, note := cellX(t, tt, "name001"), cellX(t, tt, "note1")
	if note-name != 300 {
		t.Fatalf("note1 at %v, name001 at %v: not 300 apart", note, name)
	}
	tb.SetColumnVisible("Score", false)
	settleTable(tt)
	if tt.HasText("Score") {
		t.Fatal("the header stayed")
	}
	if got := cellX(t, tt, "note1"); got-name != 200 {
		t.Fatalf("note1 %v from name001, want 200 (Name's width)", got-name)
	}
	tb.SetColumnVisible("Score", true)
	settleTable(tt)
	if got := cellX(t, tt, "note1"); got != note {
		t.Fatalf("after showing it again note1 is at %v, want %v", got, note)
	}
}

// A table sorted by a column keeps its order when the column hides, and shows
// the sort again when it comes back.
func TestColumnHiddenSortStays(t *testing.T) {
	tb := testTableOf3()
	tt := tester(t, tb)
	click(t, tt, "Score") // ascending: 3, 10, 30
	inOrder(t, tt, "cat", "bob", "ann")
	tb.SetColumnVisible("Score", false)
	settleTable(tt)
	inOrder(t, tt, "cat", "bob", "ann")
	tb.AddRow(acct{4, "dan", 5}) // goes in by the sort
	settleTable(tt)
	inOrder(t, tt, "cat", "dan", "bob", "ann")
	tb.SetColumnVisible("Score", true)
	settleTable(tt)
	inOrder(t, tt, "cat", "dan", "bob", "ann")
	click(t, tt, "Score") // the sort was still ascending: this one descends
	inOrder(t, tt, "ann", "bob", "dan", "cat")
}

func testTableOf3() *DataGridViewWidget[acct, int] {
	return testTable(acct{1, "ann", 30}, acct{2, "bob", 10}, acct{3, "cat", 3})
}

func TestColumnHiddenKeepsTheLayoutOfTheOthers(t *testing.T) {
	tb := threeColumns(3).SetColumnsLayout(ui.TableLayout{
		Order: []string{"Note", "Score", "Name"}, Widths: map[string]float32{"Score": 240}})
	tt := tester(t, tb)
	order := func() []string { return headers(tt) }
	if !sameTitles(order(), []string{"Note", "Score", "Name"}) {
		t.Fatalf("headers %q", order())
	}
	gap := find(t, tt, "Name").X - find(t, tt, "Score").X
	tb.SetColumnVisible("Score", false)
	settleTable(tt)
	if !sameTitles(order(), []string{"Note", "Name"}) {
		t.Fatalf("headers %q", order())
	}
	tb.SetColumnVisible("Score", true)
	settleTable(tt)
	if !sameTitles(order(), []string{"Note", "Score", "Name"}) || find(t, tt, "Name").X-find(t, tt, "Score").X != gap {
		t.Fatalf("headers %q; Score's width %v, was %v", order(), find(t, tt, "Name").X-find(t, tt, "Score").X, gap)
	}
}

func TestColumnHiddenIndexColumnStaysFirst(t *testing.T) {
	tb := threeColumns(3).IndexColumn()
	tt := tester(t, tb)
	tb.SetColumnVisible("Name", false)
	settleTable(tt)
	if got := headers(tt); !sameTitles(got, []string{"#", "Score", "Note"}) {
		t.Fatalf("headers %q", got)
	}
	if !tt.HasText("note3") || tt.HasText("name003") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestColumnHiddenAssistiveTechnologySeesTheShownOnes(t *testing.T) {
	tb := threeColumns(2)
	tt := tester(t, tb)
	cells := func() (n int) {
		for _, x := range tt.Nodes() {
			if x.Role == ui.RoleCell {
				n++
			}
		}
		return
	}
	if cells() != 6 {
		t.Fatalf("%d cells of 2 rows of 3 columns", cells())
	}
	tb.SetColumnVisible("Note", false)
	settleTable(tt)
	named := false
	for _, n := range tt.Nodes() {
		named = named || n.Role == ui.RoleRow && n.Label == "Name Score"
	}
	if !named {
		t.Errorf("the table's header row is not named by the columns shown")
	}
	if cells() != 4 || len(headers(tt)) != 2 {
		t.Fatalf("%d cells, %d headers of 2 columns", cells(), len(headers(tt)))
	}
}

func TestColumnHiddenRowIsNamedByItsFirstShownColumn(t *testing.T) {
	tb := threeColumns(2)
	tt := tester(t, tb)
	tb.SetColumnVisible("Name", false)
	settleTable(tt)
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleRow && n.Label == "name001" {
			t.Fatalf("a row is still named by the hidden column")
		}
	}
	found := false
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleRow && n.Label == "0" {
			found = true
		}
	}
	if !found {
		t.Fatal("no row named by Score (0)")
	}
}

func TestColumnHiddenIsLeftOutOfACopy(t *testing.T) {
	tb := threeColumns(3, "Score")
	tt := tester(t, tb)
	click(t, tt, "name001")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "name001\tnote1" {
		t.Fatalf("copied %q", got)
	}
}

func TestColumnHiddenSearchStillFindsByIt(t *testing.T) {
	tb := threeColumns(20, "Note").BottomPanel()
	tt := tester(t, tb)
	tb.Search("Note", "note7")
	settleTable(tt)
	if !tt.HasText("name007") || tt.HasText("name008") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestColumnEveryOneHidden(t *testing.T) {
	tb := threeColumns(3)
	tt := tester(t, tb)
	tb.SetColumnVisible("Name", false).SetColumnVisible("Score", false).SetColumnVisible("Note", false)
	settleTable(tt)
	if got := headers(tt); len(got) != 0 {
		t.Fatalf("headers %q", got)
	}
	tb.SetColumnVisible("Score", true)
	settleTable(tt)
	if got := headers(tt); !sameTitles(got, []string{"Score"}) || !tt.HasText("37") {
		t.Fatalf("headers %q, texts %q", got, tt.Texts())
	}
}

func settleTable(tt *ui.Tester) {
	tt.Frame()
	tt.Frame()
}

// cellX is the left edge of the cell showing text.
func cellX(t *testing.T, tt *ui.Tester, text string) float32 {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleText && n.Label == text {
			return n.Bounds.X
		}
	}
	t.Fatalf("no cell %q", text)
	return 0
}

// Rows the table kept from the last frame are built again when a column
// hides or shows, and show the columns shown now.
func TestColumnHiddenKeptRowsAreBuiltAgain(t *testing.T) {
	var calls atomic.Int64
	rows := make([]acct, 50)
	for i := range rows {
		rows[i] = acct{i + 1, fmt.Sprintf("name%03d", i+1), i}
	}
	tb := DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "A", Text: func(a *acct) string { calls.Add(1); return a.name }, Width: 100},
		Column[acct]{Title: "B", Text: func(a *acct) string { return fmt.Sprint("b", a.score) }, Width: 100},
	).SetRows(rows).Expand()
	tt := tester(t, tb)
	for range 3 {
		tt.Frame()
	}
	calls.Store(0)
	tt.Frame()
	if calls.Load() != 0 {
		t.Fatalf("%d cells built in an idle frame: the rows are not kept, so the test shows nothing", calls.Load())
	}
	tb.SetColumnVisible("B", false)
	settleTable(tt)
	if calls.Load() == 0 || tt.HasText("b1") || !tt.HasText("name001") {
		t.Fatalf("%d cells built, texts %q", calls.Load(), tt.Texts())
	}
	tb.SetColumnVisible("A", false).SetColumnVisible("B", true)
	settleTable(tt)
	if tt.HasText("name001") || !tt.HasText("b0") {
		t.Fatalf("texts %q", tt.Texts())
	}
}
