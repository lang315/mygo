package widget

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

func acctKeys(rows []acct) []int {
	ks := make([]int, len(rows))
	for i := range rows {
		ks[i] = rows[i].id
	}
	return ks
}

func TestRowSourceInsertRowsPutsRowsAtThePlace(t *testing.T) {
	src := NewRowSource(func(a *acct) int { return a.id })
	src.SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}, {3, "cat", 0}})
	src.InsertRows(0, acct{4, "dan", 0}, acct{5, "eve", 0})
	if got := acctKeys(src.Rows()); !slices.Equal(got, []int{4, 5, 1, 2, 3}) {
		t.Fatalf("after InsertRows(0): %v", got)
	}
	src.InsertRows(3, acct{6, "fay", 0})
	if got := acctKeys(src.Rows()); !slices.Equal(got, []int{4, 5, 1, 6, 2, 3}) {
		t.Fatalf("after InsertRows(3): %v", got)
	}
	// A key held is replaced where it is; a place past the end appends.
	src.InsertRows(0, acct{2, "ben", 0})
	src.InsertRows(99, acct{7, "gus", 0})
	if got := acctKeys(src.Rows()); !slices.Equal(got, []int{4, 5, 1, 6, 2, 3, 7}) {
		t.Fatalf("after a held key and a place past the end: %v", got)
	}
	for _, k := range []int{1, 2, 3, 4, 5, 6, 7} {
		if r, ok := src.Row(k); !ok || r.id != k {
			t.Fatalf("Row(%d) = %+v, %v", k, r, ok)
		}
	}
	if r, _ := src.Row(2); r.name != "ben" {
		t.Fatalf("Row(2) = %+v, not replaced", r)
	}
}

func TestRowSourceRowsIsACopy(t *testing.T) {
	src := NewRowSource(func(a *acct) int { return a.id })
	src.SetRows([]acct{{1, "ann", 0}})
	rows := src.Rows()
	rows[0].name = "changed"
	if r, _ := src.Row(1); r.name != "ann" {
		t.Fatalf("Rows shares the source's slice: %+v", r)
	}
}

func TestRowSourceInsertRowsKeepsTheVersionsOfRowsMoved(t *testing.T) {
	src := NewRowSource(func(a *acct) int { return a.id })
	src.SetRows([]acct{{1, "ann", 0}, {2, "bob", 0}})
	v1, v2 := src.Version(1), src.Version(2)
	src.InsertRows(0, acct{3, "cat", 0})
	if src.Version(1) != v1 || src.Version(2) != v2 || src.Version(3) == 0 {
		t.Fatalf("versions %d %d %d, were %d %d", src.Version(1), src.Version(2), src.Version(3), v1, v2)
	}
}

func TestTableInsertRowsKeepsSelectionAndChecks(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}, acct{2, "bob", 20}, acct{3, "cat", 30}).CheckboxColumn()
	tt := tester(t, tb)
	click(t, tt, "bob")
	tb.SetChecked([]int{3}, true)
	tb.InsertRows(0, acct{4, "dan", 40})
	tt.Frame()
	inOrder(t, tt, "dan", "ann", "bob", "cat")
	if got := tb.Selected(); !slices.Equal(got, []int{2}) {
		t.Fatalf("Selected %v after InsertRows", got)
	}
	if got := tb.Checked(); !slices.Equal(got, []int{3}) {
		t.Fatalf("Checked %v after InsertRows", got)
	}
	click(t, tt, "Name") // sorts the inserted row in with the others
	inOrder(t, tt, "ann", "bob", "cat", "dan")
}

func BenchmarkRowSourceInsertRows(b *testing.B) {
	for _, n := range []int{100_000, 1_000_000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			src := NewRowSource(func(a *acct) int { return a.id })
			src.SetRows(manyAccts(n))
			id := n
			b.ReportAllocs()
			for b.Loop() {
				id++
				src.InsertRows(0, acct{id: id})
			}
		})
		// What putting a row first cost before InsertRows: all the rows
		// again, which also builds every row of the tables again.
		b.Run(fmt.Sprint("SetRows/", n), func(b *testing.B) {
			src := NewRowSource(func(a *acct) int { return a.id })
			src.SetRows(manyAccts(n))
			id := n
			b.ReportAllocs()
			for b.Loop() {
				id++
				src.SetRows(append([]acct{{id: id}}, src.Rows()...))
			}
		})
	}
}

func TestTableSortSetterSortsAndShowsTheArrow(t *testing.T) {
	tb := testTable(acct{1, "cat", 10}, acct{2, "ann", 9}, acct{3, "bob", 100})
	tb.Sort("Name", true)
	tt := tester(t, tb)
	inOrder(t, tt, "cat", "bob", "ann")
	if s := tb.list.Sort; s == nil || *s != (ui.SortOrder{Column: "Name", Descending: true}) {
		t.Fatalf("the header shows %+v", s)
	}
	click(t, tt, "Score")
	tb.Sort("Name", false) // a setter after a click wins
	tt.Frame()
	inOrder(t, tt, "ann", "bob", "cat")
	tb.Sort("Nope", false) // not a sortable column: ignored
	tt.Frame()
	inOrder(t, tt, "ann", "bob", "cat")
}

func TestTableIndexColumnBySourceKeepsNumbersThroughSortAndSearch(t *testing.T) {
	tb := testTable(acct{1, "cat", 40}, acct{2, "ann", 50}, acct{3, "bob", 60}).IndexColumnBySource()
	tt := tester(t, tb)
	inRow := func(num, name string) {
		t.Helper()
		n, row := find(t, tt, num), find(t, tt, name)
		if y := n.Y + n.H/2; y < row.Y || y > row.Y+row.H {
			t.Fatalf("%s at %v, not in the row of %s at %v", num, n, name, row)
		}
	}
	click(t, tt, "Name")
	inOrder(t, tt, "ann", "bob", "cat")
	inRow("1", "cat")
	inRow("2", "ann")
	inRow("3", "bob")
	tb.Search("Name", "b")
	tt.Frame()
	inRow("3", "bob")
	tb.Search("Name", "")
	click(t, tt, "#")
	inOrder(t, tt, "cat", "ann", "bob")
	click(t, tt, "#")
	inOrder(t, tt, "bob", "ann", "cat")
	if s := tb.list.Sort; s == nil || *s != (ui.SortOrder{Column: "#", Descending: true}) {
		t.Fatalf("the header shows %+v", s)
	}
}

func TestTableIndexColumnByPositionDoesNotSort(t *testing.T) {
	tb := testTable(acct{1, "cat", 40}, acct{2, "ann", 50}).IndexColumn()
	tt := tester(t, tb)
	click(t, tt, "#")
	inOrder(t, tt, "cat", "ann")
}

func TestTableOnSelectionChangedFiresOncePerChange(t *testing.T) {
	var n atomic.Int32
	tb := testTable(acct{1, "ann", 10}, acct{2, "bob", 20}, acct{3, "cat", 30}, acct{4, "dan", 40}).
		OnSelectionChanged(func() { n.Add(1) })
	tt := tester(t, tb)
	want := int32(0)
	step := func(name string, changes bool, act func()) {
		t.Helper()
		act()
		tt.Frame()
		tt.Frame()
		flushEvents(t)
		if changes {
			want++
		}
		if got := n.Load(); got != want {
			t.Fatalf("%s: fired %d times, want %d", name, got, want)
		}
	}
	step("click", true, func() { click(t, tt, "bob") })
	step("click again", false, func() { click(t, tt, "bob") })
	step("shift-click", true, func() {
		if err := tt.ClickWith(ui.Shift, "cat"); err != nil {
			t.Fatal(err)
		}
	})
	step("cmd-click", true, func() {
		if err := tt.ClickWith(ui.Cmd, "ann"); err != nil {
			t.Fatal(err)
		}
	})
	step("down key", true, func() { tt.Key(0, ui.KeyDown) })
	step("select all", true, func() { tt.Key(ui.Cmd, ui.KeyA) })
	step("frames", false, func() { tt.Frame() })
	step("delete a row not selected", false, func() { tb.AddRow(acct{5, "eve", 50}) })
	step("delete a selected row", true, func() { tb.DeleteRows(1) })
	step("search hiding selected rows", true, func() { tb.Search("Name", "bob") })
	step("search showing the same", false, func() { tb.Search("Name", "bo") })
	step("search off", true, func() { tb.Search("Name", "") })
}

func TestTableClipboardCopyOff(t *testing.T) {
	tb := testTable(acct{1, "ann", 10}, acct{2, "bob", 20}).SetClipboardCopy(false)
	tt := tester(t, tb)
	click(t, tt, "bob")
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != "" {
		t.Fatalf("copied %q with copy off", got)
	}
	tb.SetClipboardCopy(true)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); !strings.HasPrefix(got, "bob") {
		t.Fatalf("copied %q with copy on", got)
	}
}
