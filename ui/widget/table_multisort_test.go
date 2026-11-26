package widget

import (
	"iter"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

type person struct {
	name string
	age  int
}

// peopleTable is a table of people by name: Name sorts as text, Age as a
// number, Note does not sort.
func peopleTable(rows ...person) *DataGridViewWidget[person, string] {
	return DataGridView(func(p *person) string { return p.name },
		Column[person]{Title: "Name", Text: func(p *person) string { return p.name }, Width: 200, Sortable: true},
		IntColumn("Age", 100, func(p *person) int { return p.age }),
		Column[person]{Title: "Note", Text: func(p *person) string { return "note " + p.name }, Width: 150},
	).SetRows(rows).Expand()
}

// bac are the plan's rows: (b,30), (a,30), (c,20).
func bac() []person { return []person{{"b", 30}, {"a", 30}, {"c", 20}} }

func wantOrders(t *testing.T, tb *DataGridViewWidget[person, string], want ...ui.SortOrder) {
	t.Helper()
	if got := tb.SortOrders(); !slices.Equal(got, want) {
		t.Fatalf("SortOrders %+v, want %+v", got, want)
	}
}

func TestSortByTwoColumns(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	inOrder(t, tt, "c", "a", "b")
	wantOrders(t, tb, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
}

func TestSortByDescendingSecond(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name", Descending: true})
	tt := tester(t, tb)
	inOrder(t, tt, "c", "b", "a")
	// The same with a and b swapped in the source, so that Age alone (a
	// stable sort) would give c, a, b.
	tb.SetRows([]person{{"a", 30}, {"b", 30}, {"c", 20}})
	tt.Frame()
	inOrder(t, tt, "c", "b", "a")
}

func TestSortByUnsortableDropped(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Note"}, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Nope"}, ui.SortOrder{Column: "Name"})
	wantOrders(t, tb, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	inOrder(t, tt, "c", "a", "b")
}

func TestSortByNoneClears(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	inOrder(t, tt, "a", "b", "c")
	tb.SortBy()
	tt.Frame()
	inOrder(t, tt, "b", "a", "c")
	if got := tb.SortOrders(); len(got) != 0 {
		t.Fatalf("SortOrders %+v after SortBy()", got)
	}
}

func TestSortStillOneColumn(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.Sort("Name", false)
	wantOrders(t, tb, ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	inOrder(t, tt, "a", "b", "c")
	tb.Sort("Nope", true) // not a sortable column: ignored, as before
	wantOrders(t, tb, ui.SortOrder{Column: "Name"})
	tb.ClearSort()
	tt.Frame()
	inOrder(t, tt, "b", "a", "c")
	if got := tb.SortOrders(); len(got) != 0 {
		t.Fatalf("SortOrders %+v after ClearSort", got)
	}
}

func TestShiftClickAddsWithMultiSort(t *testing.T) {
	shiftClick := func(tt *ui.Tester, s string) {
		t.Helper()
		if err := tt.ClickWith(ui.Shift, s); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	tb := peopleTable(bac()...).SetMultiSort(true)
	tt := tester(t, tb)
	click(t, tt, "Age")
	shiftClick(tt, "Name")
	wantOrders(t, tb, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	inOrder(t, tt, "c", "a", "b")

	tb = peopleTable(bac()...)
	tt = tester(t, tb)
	click(t, tt, "Age")
	shiftClick(tt, "Name")
	wantOrders(t, tb, ui.SortOrder{Column: "Name"})
	inOrder(t, tt, "a", "b", "c")
}

// A click with multi-sort off, then SetMultiSort(true): a Shift-click adds
// after the column the click sorted by.
func TestMultiSortTurnedOnAfterAClick(t *testing.T) {
	tb := peopleTable(bac()...)
	tt := tester(t, tb)
	click(t, tt, "Age")
	tb.SetMultiSort(true)
	tt.Frame()
	if err := tt.ClickWith(ui.Shift, "Name"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	wantOrders(t, tb, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	// And off again: a Shift-click sorts by the column alone.
	tb.SetMultiSort(false)
	tt.Frame()
	if err := tt.ClickWith(ui.Shift, "Age"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	wantOrders(t, tb, ui.SortOrder{Column: "Age", Descending: true})
}

func TestMultiSortStable(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Age"})
	tt := tester(t, tb)
	inOrder(t, tt, "c", "b", "a") // b and a tie: the source's order
	tb.SortBy(ui.SortOrder{Column: "Age", Descending: true})
	tt.Frame()
	inOrder(t, tt, "b", "a", "c") // still b before a
}

func TestMultiSortInsideGroups(t *testing.T) {
	tb := peopleTable(append(bac(), person{"d", 20})...).
		SetGroupBy(func(p *person) string { return strconv.Itoa(p.age) })
	tb.SortBy(ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	// Group 20 (c, d) before group 30 (a, b); by Name alone: a, b, c, d.
	inOrder(t, tt, "c", "d", "a", "b")
	// Age, descending, first: the groups stay in their order (20, then
	// 30), and Name, the second order, sorts inside each.
	tb.SortBy(ui.SortOrder{Column: "Age", Descending: true}, ui.SortOrder{Column: "Name", Descending: true})
	tt.Frame()
	inOrder(t, tt, "d", "c", "b", "a")
}

func TestMultiSortAfterUpdateRow(t *testing.T) {
	tb := peopleTable(bac()...)
	tb.SortBy(ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	tt := tester(t, tb)
	inOrder(t, tt, "c", "a", "b")
	tb.UpdateRow("c", func(p *person) { p.age = 40 })
	tt.Frame()
	inOrder(t, tt, "a", "b", "c")
	tb.UpdateRow("b", func(p *person) { p.age = 10 })
	tt.Frame()
	inOrder(t, tt, "b", "a", "c")
}

// wideTable is a table wider than the tester's 800 DIPs: Name, frozen
// when frozen, then eight columns of 200. Name's cell draws "n" and the
// name, which no row's label (its Text) is. edit, if any, changes the
// columns first.
func wideTable(frozen bool, edit ...func([]Column[person])) *DataGridViewWidget[person, string] {
	cols := []Column[person]{{Title: "Name", Text: func(p *person) string { return p.name }, Width: 150, Frozen: frozen,
		Spans: func(p *person) []ui.Span { return []ui.Span{{Text: "n" + p.name}} }}}
	for i := range 8 {
		n := strconv.Itoa(i)
		cols = append(cols, Column[person]{Title: "C" + n, Text: func(p *person) string { return "c" + n + p.name }, Width: 200})
	}
	for _, fn := range edit {
		fn(cols)
	}
	return DataGridView(func(p *person) string { return p.name }, cols...).SetRows(bac()).Expand()
}

// scrollAcross scrolls the table 150 DIPs sideways and reports where the
// texts were and are.
func scrollAcross(t *testing.T, tt *ui.Tester, texts ...string) (before, after []ui.Rect) {
	t.Helper()
	for _, s := range texts {
		before = append(before, find(t, tt, s))
	}
	r := find(t, tt, "c1b")
	tt.Scroll(r.X, r.Y, 150, 0)
	tt.Frame()
	for _, s := range texts {
		after = append(after, find(t, tt, s))
	}
	return before, after
}

func TestFrozenColumnReachesTable(t *testing.T) {
	tb := wideTable(true)
	tt := tester(t, tb)
	if !tb.columns[0].Frozen {
		t.Fatalf("ui's first column %+v is not Frozen", tb.columns[0])
	}
	before, after := scrollAcross(t, tt, "nb", "Name", "c1b")
	if after[0].X != before[0].X || after[1].X != before[1].X {
		t.Errorf("the frozen column moved: cell %v to %v, header %v to %v", before[0], after[0], before[1], after[1])
	}
	if after[2].X >= before[2].X {
		t.Fatalf("the table did not scroll: c1b at %v, then %v", before[2], after[2])
	}

	// Not frozen, Name scrolls with the others.
	tt = tester(t, wideTable(false))
	before, after = scrollAcross(t, tt, "nb")
	if after[0].X >= before[0].X {
		t.Errorf("Name, not frozen, stayed at %v", before[0])
	}

	// The index column pins with a frozen first column.
	tb = wideTable(true).IndexColumn()
	tt = tester(t, tb)
	if !tb.columns[0].Frozen || !tb.columns[1].Frozen {
		t.Fatalf("ui's columns # %+v and Name %+v, want both Frozen", tb.columns[0], tb.columns[1])
	}
	before, after = scrollAcross(t, tt, "#", "1", "nb", "c1b")
	if after[0].X != before[0].X || after[1].X != before[1].X || after[2].X != before[2].X {
		t.Errorf("# moved from %v to %v, 1 from %v to %v, nb from %v to %v", before[0], after[0], before[1], after[1], before[2], after[2])
	}
	if after[3].X >= before[3].X {
		t.Fatalf("the table with # did not scroll: c1b at %v, then %v", before[3], after[3])
	}
}

// The row of totals follows a frozen column: its cell stays, and a scrolled
// column's cell starts after it.
func TestFrozenColumnSummaryStays(t *testing.T) {
	tt := tester(t, wideTable(true, func(cols []Column[person]) {
		cols[0].Summary = func(iter.Seq[*person]) string { return "total" }
		cols[2].Summary = func(iter.Seq[*person]) string { return "sum1" }
	}))
	tt.Frame()
	name := find(t, tt, "Name")
	before, after := scrollAcross(t, tt, "total", "sum1")
	if after[0].X != before[0].X {
		t.Errorf("the frozen column's total moved from %v to %v", before[0], after[0])
	}
	if after[1].X >= before[1].X {
		t.Errorf("C1's total did not scroll: %v, then %v", before[1], after[1])
	}
	if after[1].X < name.X+100 {
		t.Errorf("C1's total at %v, under the frozen Name at %v", after[1], name)
	}
}

// A frame at rest of a grid sorted by several columns, or with multi-sort
// on, allocates no more than ui's headers do: multi-sort binds nothing that
// allocates, sorting by two columns costs no more than by one, and a second
// arrow (ui's, drawn with multi-sort on) costs what the first does.
func TestMultiSortFrameAllocatesNothingMore(t *testing.T) {
	grid := func(multi bool, orders ...ui.SortOrder) *ui.Tester {
		tb := peopleTable(bac()...).SetMultiSort(multi)
		tb.SortBy(orders...)
		tt := tester(t, tb)
		for range 10 {
			tt.Frame()
		}
		return tt
	}
	age, name := ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name", Descending: true}
	least := func(tt *ui.Tester) float64 { return allocrun.Least(30, tt.Frame) }
	none, one := least(grid(false)), least(grid(false, age))
	oneMulti, twoSingle, twoMulti := least(grid(true, age)), least(grid(false, age, name)), least(grid(true, age, name))
	if oneMulti != one {
		t.Errorf("by one column, a frame allocates %v times with multi-sort, %v without", oneMulti, one)
	}
	if twoSingle != one {
		t.Errorf("without multi-sort, a frame allocates %v times by two columns, %v by one", twoSingle, one)
	}
	if arrow := one - none; twoMulti-oneMulti > arrow {
		t.Errorf("with multi-sort, a second column's header costs %v allocations, the first's %v", twoMulti-oneMulti, arrow)
	}
}

func TestSortByFromGoroutine(t *testing.T) {
	tb := peopleTable(bac()...)
	tt := tester(t, tb)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 50 {
			if i%2 == 0 {
				tb.SortBy(ui.SortOrder{Column: "Name", Descending: true})
			} else {
				tb.SortBy()
			}
			_ = tb.SortOrders()
		}
		tb.SortBy(ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
	}()
	for range 20 {
		tt.Frame()
		_ = tb.SortOrders()
	}
	wg.Wait()
	tt.Frame()
	inOrder(t, tt, "c", "a", "b")
	wantOrders(t, tb, ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name"})
}
