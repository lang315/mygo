package widget

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"sync"
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

func TestTableActivateGivesTheKey(t *testing.T) {
	got := make(chan int, 1)
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).OnActivate(func(k int) { got <- k })
	tt := tester(t, tb)
	click(t, tt, "bob")
	tt.Key(0, ui.KeyEnter)
	if k := wait(t, got); k != 2 {
		t.Fatalf("activated %d", k)
	}
}

func TestTableOnKeyRunsOnTheEventGoroutine(t *testing.T) {
	got := make(chan []int, 1)
	var tb *TableWidget[acct, int]
	tb = testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).
		OnKey(ui.KeyDelete, 0, func() {
			if onEvents() {
				got <- tb.Selected()
			}
		})
	tt := tester(t, tb)
	click(t, tt, "bob")
	tt.Key(0, ui.KeyDelete)
	if ks := wait(t, got); !slices.Equal(ks, []int{2}) {
		t.Fatalf("Delete with %v", ks)
	}
}

func TestTableRowMenuActsOnSelected(t *testing.T) {
	done := make(chan bool, 1)
	var tb *TableWidget[acct, int]
	tb = testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).
		ContextMenu(MenuItem{Text: "Delete", OnClick: func() { tb.DeleteRows(tb.Selected()...); done <- true }})
	tt := tester(t, tb)
	if err := tt.RightClick("bob"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Delete"); err != nil {
		t.Fatal(err)
	}
	wait(t, done)
	tt.Frame()
	if tt.HasText("bob") || !tt.HasText("ann") {
		t.Fatalf("after Delete: %q", tt.Texts())
	}
}

func TestTableCopiesSelectedRowsInShownOrder(t *testing.T) {
	tb := testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100})
	tt := tester(t, tb)
	click(t, tt, "Name")
	click(t, tt, "cat")
	if err := tt.ClickWith(ui.Cmd, "ann"); err != nil {
		t.Fatal(err)
	}
	tt.Command("copy")
	if got := tt.Clipboard(); got != "ann\t9\ncat\t10" {
		t.Fatalf("copied %q", got)
	}
}

func TestTableSetColumnsLayoutOrdersColumns(t *testing.T) {
	tt := tester(t, testTable(acct{1, "ann", 1}).
		SetColumnsLayout(ui.TableLayout{Order: []string{"Score", "Name"}}))
	if score, name := find(t, tt, "Score"), find(t, tt, "Name"); score.X >= name.X {
		t.Fatalf("Score at %v, Name at %v: the layout's order not shown", score, name)
	}
}

func TestTableColumnsChangeGivesTheLayout(t *testing.T) {
	got := make(chan ui.TableLayout, 64)
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "A", Text: func(a *acct) string { return a.name }, Width: 100},
		Column[acct]{Title: "B", Text: func(a *acct) string { return a.name }, Width: 100},
	).SetRows([]acct{{1, "ann", 0}}).OnColumnsChange(func(l ui.TableLayout) { got <- l }).Expand()
	tt := tester(t, tb)
	// Both headers start their text at the same inset, so B's starts 100
	// DIPs after A's; the table starts at the left of the tester, so A's
	// right edge is at 100.
	a, b := find(t, tt, "A"), find(t, tt, "B")
	if b.X-a.X != 100 || a.X > 16 {
		t.Fatalf("headers A %v, B %v: not 100 apart from the left", a, b)
	}
	y := a.Y + a.H/2
	tt.Press(98, y)
	tt.Move(128, y)
	tt.Release(128, y)
	var l ui.TableLayout
	for l.Widths["A"] != 130 {
		l = wait(t, got) // fails after 2 s without a layout of A at 130
	}
}

// manyAccts returns n accounts: acct000000, acct000001, ..., scores
// counting down.
func manyAccts(n int) []acct {
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{i, fmt.Sprintf("acct%06d", i), n - i}
	}
	return rows
}

func TestTable100kRows(t *testing.T) {
	tb := testTable(manyAccts(100_000)...).CheckboxColumn().BottomPanel()
	tt := tester(t, tb)
	if !tt.HasText("Rows: 100000") {
		t.Fatalf("no row count in %q", tt.Texts())
	}
	click(t, tt, "Score") // ascending: the last account first
	if !tt.HasText("acct099999") {
		t.Fatalf("sorted by Score: %q", tt.Texts())
	}
	tb.Search("Name", "acct00001") // acct000010..acct000019
	tt.Frame()
	click(t, tt, "Check all")
	got := tb.Checked()
	want := []int{19, 18, 17, 16, 15, 14, 13, 12, 11, 10}
	if !slices.Equal(got, want) {
		t.Fatalf("Checked %v, want %v", got, want)
	}
	tb.DeleteRows(got...)
	tt.Frame()
	if tb.Len() != 100_000-10 || len(tb.Checked()) != 0 || tb.IsChecked(15) {
		t.Fatalf("after DeleteRows: Len %d, Checked %v", tb.Len(), tb.Checked())
	}
}

func TestTableIsSafeFromManyGoroutines(t *testing.T) {
	tb := testTable(manyAccts(1000)...).CheckboxColumn().BottomPanel()
	tt := tester(t, tb)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				k := 1000 + g*1000 + i
				tb.AddRow(acct{k, "new", i})
				tb.UpdateRow(i, func(a *acct) { a.score++ })
				tb.SetChecked([]int{k, i}, i%2 == 0)
				_ = tb.Checked()
				_ = tb.Selected()
				if i%50 == 0 {
					tb.DeleteRows(k)
					tb.Search("Name", "new")
					tb.Search("Name", "")
				}
			}
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for {
		select {
		case <-done:
			tt.Frame()
			if n := tb.Len(); n != 1000+8*200-8*4 {
				t.Fatalf("Len %d", n)
			}
			return
		default:
			tt.Frame()
		}
	}
}

// BenchmarkTableLiveUpdates changes 1000 rows of a table sorted by Score,
// then draws a frame, as account-manager's BenchmarkLiveUpdates.
func BenchmarkTableLiveUpdates(b *testing.B) {
	for _, n := range []int{100_000, 1_000_000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			tb := testTable(manyAccts(n)...).CheckboxColumn().IndexColumn()
			tt := ui.NewTester(content(tb), 1280, 800)
			tt.Frame()
			if err := tt.Click("Score"); err != nil {
				b.Fatal(err)
			}
			tt.Frame()
			b.ResetTimer()
			for i := range b.N {
				for j := range 1000 {
					tb.UpdateRow((i*7919+j*104729)%n, func(a *acct) { a.score += 3 })
				}
				tt.Frame()
			}
		})
	}
}

func TestTableInputLandsOnTheRowsOfTheFrameShown(t *testing.T) {
	tb := testTable(acct{1, "a", 0}, acct{2, "b", 0}, acct{3, "c", 0})
	tt := tester(t, tb)
	click(t, tt, "Name")
	click(t, tt, "Name") // descending: c, b, a
	check := func(when string) {
		t.Helper()
		if k, s, c := tb.keyAt(0), tb.labelAt(0), tb.copyRows([]int{0}); k != 3 || s != "c" || c != "c\t0" {
			t.Fatalf("%s: row 0 of the frame shown is %v, %q, %q", when, k, s, c)
		}
	}
	tb.DeleteRows(3) // the frame shown still has c at 0
	check("after a delete")
	tb.SetChecked([]int{1}, true).Checked() // syncs the view, as the frame shown is not
	check("after a getter synced")
	tt.Frame()
	if k := tb.keyAt(0); k != 2 {
		t.Fatalf("keyAt(0) after a frame = %v", k)
	}
}

func TestTableCommitDropsKeysOfRowsGone(t *testing.T) {
	tb := testTable(acct{1, "a", 0}, acct{2, "b", 0}).CheckboxColumn()
	tb.SetChecked([]int{1, 2}, true)
	tester(t, tb)
	tb.DeleteRows(1) // before ui applies input to the frame shown
	tb.commit(&tb.checked)
	if tb.IsChecked(1) || !tb.IsChecked(2) {
		t.Fatalf("checked 1: %v, 2: %v after ui's commit", tb.IsChecked(1), tb.IsChecked(2))
	}
}

func TestTableAutoCheckActsOnShownRows(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn()
	tt := tester(t, tb)
	click(t, tt, "ann")
	if err := tt.ClickWith(ui.Cmd, "bob"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tb.Search("Name", "ann")
	tb.AutoCheck()
	if !tb.IsChecked(1) || tb.IsChecked(2) {
		t.Fatalf("checked ann: %v, hidden bob: %v", tb.IsChecked(1), tb.IsChecked(2))
	}
}

// ui builds the rows a taller window shows late, while it lays the frame
// out, without the package's lock: a goroutine changing the rows meanwhile
// must not race them.
func TestTableRowsBuiltWhileLayingOutAreSafe(t *testing.T) {
	tb := testTable(manyAccts(2000)...)
	tt := tester(t, tb)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			tb.DeleteRows(i % 2000)
			tb.AddRow(acct{i % 2000, "back", i})
		}
	}()
	for i := range 40 {
		tt.SetSize(800, 300+i%2*2000)
		tt.Frame()
	}
	close(stop)
	<-done
}

func TestTableFitsItsWindowWithoutExpand(t *testing.T) {
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }},
	).SetRows(manyAccts(1000)).BottomPanel()
	tt := tester(t, tb) // 600 DIPs tall
	if r := find(t, tt, "Rows: 1000"); r.Y+r.H > 600 {
		t.Fatalf("panel at %v, below the window", r)
	}
}

// A selected row's text takes the selection's color, not its column's,
// as GTK and AppKit draw it: red text on the accent would not read.
func TestTableSelectedRowDropsTheColumnColor(t *testing.T) {
	tb := Table(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name },
			Color: func(a *acct) ui.Color { return ui.RGB(220, 0, 0) }},
	).SetRows([]acct{{1, "annnnnnn", 0}}).Expand()
	tt := tester(t, tb)
	red := func() int {
		r, n := find(t, tt, "annnnnnn"), 0
		img := tt.Image()
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X); x < int(r.X+r.W); x++ {
				if c := img.RGBAAt(x, y); c.R > 150 && c.G < 80 && c.B < 80 {
					n++
				}
			}
		}
		return n
	}
	if red() == 0 {
		t.Fatal("no red text before the row is selected")
	}
	click(t, tt, "annnnnnn")
	if n := red(); n > 0 {
		t.Fatalf("%d red pixels in the selected row's text", n)
	}
}
