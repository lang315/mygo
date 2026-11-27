package widget

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
)

// dragGrid is a table of ann, bob, cat and dan, ids 1 to 4, in that order.
func dragGrid() *DataGridViewWidget[acct, int] {
	return testTable(acct{1, "ann", 30}, acct{2, "bob", 10}, acct{3, "cat", 20}, acct{4, "dan", 40})
}

// names returns the names of rows, in their order.
func names(rows []acct) string {
	s := make([]string, len(rows))
	for i, r := range rows {
		s[i] = r.name
	}
	return fmt.Sprint(s)
}

// dragRow drags the row showing from onto the row showing to, and lets go
// just above its bottom edge, or just below its top: below its middle the rows
// go after it, above it before it.
func dragRow(t *testing.T, tt *Tester, from, to string, after bool) {
	t.Helper()
	dragRowRaw(t, tt, from, to, after)
	tt.Settle()
}

// dragRowRaw is dragRow without settling: it draws the frame that takes the
// drop, and what that posts waits.
func dragRowRaw(t *testing.T, tt *Tester, from, to string, after bool) {
	t.Helper()
	a, b := find(t, tt.Tester, from), find(t, tt.Tester, to)
	ax, ay := a.X+a.W/2, a.Y+a.H/2
	bx, by := b.X+b.W/2, b.Y+1
	if after {
		by = b.Y + b.H - 1
	}
	tt.Press(ax, ay)
	for i := 1; i <= 4; i++ {
		f := float32(i) / 4
		tt.Move(ax+(bx-ax)*f, ay+(by-ay)*f)
		tt.Frame()
	}
	tt.Release(bx, by)
	tt.Frame()
}

// blockEvents holds the event goroutine until release is called, so that
// what is posted meanwhile waits behind it.
func blockEvents() (release func()) {
	hold, started := make(chan struct{}), make(chan struct{})
	Post(func() { close(started); <-hold })
	<-started
	return func() { close(hold) }
}

// Case 14: what changes between the drop and the move it posted: a sort
// started meanwhile, the row the rows go before deleted, or the rows dragged
// deleted, moves nothing and fires nothing.
func TestRowReorderChangedBeforeTheMove(t *testing.T) {
	for _, c := range []struct {
		name   string
		from   string // dragged before cat
		change func(tb *DataGridViewWidget[acct, int])
		want   string
	}{
		{"sorted", "ann", func(tb *DataGridViewWidget[acct, int]) { tb.Sort("Score", false) }, "[ann bob cat dan]"},
		{"anchor deleted", "dan", func(tb *DataGridViewWidget[acct, int]) { tb.DeleteRows(3) }, "[ann bob dan]"},
		{"rows deleted", "ann", func(tb *DataGridViewWidget[acct, int]) { tb.DeleteRows(1) }, "[bob cat dan]"},
	} {
		var fired fires
		tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
		tt := NewTester(t, tb, 800, 600)
		release := blockEvents()
		dragRowRaw(t, tt, c.from, "cat", false)
		c.change(tb)
		release()
		tt.Settle()
		if got := names(tb.Rows()); got != c.want {
			t.Errorf("%s: Rows %s, want %s", c.name, got, c.want)
		}
		if n := fired.get(); n != 0 {
			t.Errorf("%s: OnRowsReordered ran %d times", c.name, n)
		}
	}
}

// Case 7c: rows drag out of a table over a RowSource too.
func TestDragRowsOutOfARowSource(t *testing.T) {
	src := NewRowSource(func(a *acct) int { return a.id })
	src.SetRows([]acct{{1, "ann", 30}, {2, "bob", 10}})
	got := make(chan []*acct, 4)
	tb := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).
		DragRows(func(rows []*acct) any { return rows }).Expand()
	tt := NewTester(t, VBox(8).Add(tb, dropZone(got)), 800, 600)
	dragRow(t, tt, "bob", "dst", false)
	if rows := received(t, got); len(rows) != 1 || rows[0].name != "bob" {
		t.Fatalf("dropped %v, want bob", rows)
	}
}

// fires counts the calls of a handler from any goroutine.
type fires struct{ n atomic.Int32 }

func (c *fires) inc()       { c.n.Add(1) }
func (c *fires) get() int32 { return c.n.Load() }

// Case 1: row 0 dragged below row 2 gives 1, 2, 0; OnRowsReordered once.
func TestRowReorderMoves(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	dragRow(t, tt, "ann", "cat", true)
	if got := names(tb.Rows()); got != "[bob cat ann dan]" {
		t.Fatalf("Rows %s, want [bob cat ann dan]", got)
	}
	inOrder(t, tt.Tester, "bob", "cat", "ann", "dan")
	if n := fired.get(); n != 1 {
		t.Fatalf("OnRowsReordered ran %d times, want 1", n)
	}
}

// Case 1b: the last row dragged above the first goes first.
func TestRowReorderLastToFirst(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	dragRow(t, tt, "dan", "ann", false)
	if got := names(tb.Rows()); got != "[dan ann bob cat]" {
		t.Fatalf("Rows %s, want [dan ann bob cat]", got)
	}
	if n := fired.get(); n != 1 {
		t.Fatalf("OnRowsReordered ran %d times, want 1", n)
	}
}

// Case 2: rows 0 and 2 chosen, dragged to the end, keep their order, and stay
// chosen.
func TestRowReorderSeveralChosen(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	tb.SetSelected(1, 3)
	tt.Settle()
	dragRow(t, tt, "cat", "dan", true)
	if got := names(tb.Rows()); got != "[bob dan ann cat]" {
		t.Fatalf("Rows %s, want [bob dan ann cat]", got)
	}
	if n := fired.get(); n != 1 {
		t.Fatalf("OnRowsReordered ran %d times, want 1", n)
	}
	if got := fmt.Sprint(tb.Selected()); got != "[1 3]" {
		t.Fatalf("Selected %s after the move, want [1 3]", got)
	}
}

// Case 2b: chosen rows dropped where they already are move nothing and fire
// nothing.
func TestRowReorderDroppedInPlaceDoesNotFire(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	tb.SetSelected(1, 2)
	tt.Settle()
	dragRow(t, tt, "ann", "cat", false) // before cat: where they are
	if got := names(tb.Rows()); got != "[ann bob cat dan]" {
		t.Fatalf("Rows %s, want them unmoved", got)
	}
	if n := fired.get(); n != 0 {
		t.Fatalf("OnRowsReordered ran %d times for rows dropped in place", n)
	}
}

// noReorder drags ann below cat and checks that the drag did not start: ui
// was given no Reorder, the rows stay, nothing fires.
func noReorder(t *testing.T, tb *DataGridViewWidget[acct, int], tt *Tester, fired *fires, want string) {
	t.Helper()
	if tb.list.Reorder != nil {
		t.Fatal("ui was given a Reorder")
	}
	dragRow(t, tt, "ann", "cat", true)
	if got := names(tb.Rows()); got != want {
		t.Fatalf("Rows %s, want %s", got, want)
	}
	if n := fired.get(); n != 0 {
		t.Fatalf("OnRowsReordered ran %d times", n)
	}
}

// Case 3: sorted, the rows do not drag.
func TestRowReorderOffWhenSorted(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	tb.Sort("Score", false)
	tt.Settle()
	noReorder(t, tb, tt, &fired, "[ann bob cat dan]")
	// Unsorted again, they do.
	tb.ClearSort()
	tt.Settle()
	if tb.list.Reorder == nil {
		t.Fatal("unsorted, ui has no Reorder")
	}
}

// Case 4: grouped, the rows do not drag.
func TestRowReorderOffWhenGrouped(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tb.SetGroupBy(func(a *acct) string { return "all" })
	tt := NewTester(t, tb, 800, 600)
	noReorder(t, tb, tt, &fired, "[ann bob cat dan]")
}

// Case 5: searched, the rows do not drag.
func TestRowReorderOffWhenFiltered(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tb.Search("Name", "a") // ann, cat, dan
	tt := NewTester(t, tb, 800, 600)
	noReorder(t, tb, tt, &fired, "[ann bob cat dan]")
}

// Case 5b: with a column filter, the rows do not drag; cleared, they do.
func TestRowReorderOffWhenColumnFiltered(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	tb.SetColumnFilter("Name", ColumnFilter{Contains: "a"}) // ann, cat, dan
	tt.Settle()
	noReorder(t, tb, tt, &fired, "[ann bob cat dan]")
	tb.ClearFilters()
	tt.Settle()
	if tb.list.Reorder == nil {
		t.Fatal("filters cleared, ui was given no Reorder")
	}
}

// Case 6: a table over a RowSource never reorders.
func TestRowReorderRowSourceNoOp(t *testing.T) {
	src := NewRowSource(func(a *acct) int { return a.id })
	src.SetRows([]acct{{1, "ann", 30}, {2, "bob", 10}, {3, "cat", 20}, {4, "dan", 40}})
	var fired fires
	tb := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).
		SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb.Expand(), 800, 600)
	noReorder(t, tb, tt, &fired, "[ann bob cat dan]")
	if got := names(src.Rows()); got != "[ann bob cat dan]" {
		t.Fatalf("the source's rows %s", got)
	}
}

// dropZone is a label that takes the rows dropped on it.
func dropZone(got chan<- []*acct) *LabelWidget {
	dst := Label("dst").Padding(16)
	OnDrop(dst, func(rows []*acct) { got <- rows })
	return dst
}

// Case 7: the chosen rows dragged onto an OnDrop[[]*acct] get there, in the
// order shown.
func TestDragRowsOut(t *testing.T) {
	got := make(chan []*acct, 4)
	tb := dragGrid().DragRows(func(rows []*acct) any { return rows })
	tt := NewTester(t, VBox(8).Add(tb, dropZone(got)), 800, 600)
	tb.SetSelected(3, 1)
	tt.Settle()
	dragRow(t, tt, "cat", "dst", false)
	rows := received(t, got)
	if len(rows) != 2 || rows[0].name != "ann" || rows[1].name != "cat" {
		t.Fatalf("dropped %v, want ann and cat", rows)
	}
	if n := names(tb.Rows()); n != "[ann bob cat dan]" {
		t.Fatalf("dragging out moved the rows: %s", n)
	}
}

// Case 7b: sorted, a row dragged out is the one under the pointer.
func TestDragRowsOutSorted(t *testing.T) {
	got := make(chan []*acct, 4)
	tb := dragGrid().DragRows(func(rows []*acct) any { return rows }).Sort("Score", false)
	tt := NewTester(t, VBox(8).Add(tb, dropZone(got)), 800, 600)
	if tb.list.DragRows == nil {
		t.Fatal("sorted, ui has no DragRows")
	}
	dragRow(t, tt, "cat", "dst", false)
	if rows := received(t, got); len(rows) != 1 || rows[0].name != "cat" {
		t.Fatalf("dropped %v, want cat", rows)
	}
}

// Case 8: fn returning nil drags nothing; a nil fn stops dragging out.
func TestDragRowsNilValueNoDrag(t *testing.T) {
	got := make(chan []*acct, 4)
	var asked fires
	tb := dragGrid().DragRows(func(rows []*acct) any { asked.inc(); return nil })
	tt := NewTester(t, VBox(8).Add(tb, dropZone(got)), 800, 600)
	dragRow(t, tt, "ann", "dst", false)
	if asked.get() == 0 {
		t.Fatal("fn was not asked")
	}
	select {
	case rows := <-got:
		t.Fatalf("rows fn left out dropped: %v", rows)
	default:
	}
	tb.DragRows(nil)
	tt.Settle()
	if tb.list.DragRows != nil {
		t.Fatal("DragRows(nil) left ui a DragRows")
	}
}

// Case 8b: DragRows(nil) before the next frame: a drag from the frame ui still
// shows drags nothing, and does not call the fn that went.
func TestDragRowsTurnedOffBeforeTheFrame(t *testing.T) {
	got := make(chan []*acct, 4)
	tb := dragGrid().DragRows(func(rows []*acct) any { return rows })
	tt := NewTester(t, VBox(8).Add(tb, dropZone(got)), 800, 600)
	a, b := find(t, tt.Tester, "ann"), find(t, tt.Tester, "dst")
	tt.Press(a.X+4, a.Y+a.H/2)
	tt.Move(a.X+5, a.Y+a.H/2) // under the threshold: the drag has not started
	tt.Frame()
	tb.DragRows(nil) // no frame: ui keeps the DragRows of the last
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	tt.Release(b.X+b.W/2, b.Y+b.H/2)
	tt.Settle()
	select {
	case rows := <-got:
		t.Fatalf("rows dragged after DragRows(nil): %v", rows)
	default:
	}
}

// Case 9: the later of SetRowReorder(true) and DragRows wins, in either order.
func TestDragRowsAndReorderLaterWins(t *testing.T) {
	out := func(rows []*acct) any { return rows }
	tb := dragGrid().SetRowReorder(true).DragRows(out)
	tt := NewTester(t, tb, 800, 600)
	if tb.list.Reorder != nil || tb.list.DragRows == nil {
		t.Fatal("DragRows after SetRowReorder: want DragRows alone")
	}
	tb.SetRowReorder(true)
	tt.Settle()
	if tb.list.Reorder == nil || tb.list.DragRows != nil {
		t.Fatal("SetRowReorder after DragRows: want Reorder alone")
	}
	dragRow(t, tt, "ann", "cat", true)
	if got := names(tb.Rows()); got != "[bob cat ann dan]" {
		t.Fatalf("Rows %s, want [bob cat ann dan]", got)
	}
	// Turning one off leaves the other off too.
	tb.SetRowReorder(false)
	tt.Settle()
	if tb.list.Reorder != nil || tb.list.DragRows != nil {
		t.Fatal("SetRowReorder(false): want neither")
	}
}

// Case 10: setters that change the order do not run OnRowsReordered.
func TestRowReorderSetterDoesNotFire(t *testing.T) {
	var fired fires
	tb := dragGrid().SetRowReorder(true).OnRowsReordered(fired.inc)
	tt := NewTester(t, tb, 800, 600)
	tb.InsertRows(0, acct{5, "eve", 1})
	tb.DeleteRows(2)
	tb.AddRows([]acct{{6, "fay", 2}})
	tb.SetRows([]acct{{4, "dan", 40}, {1, "ann", 30}})
	tb.Sort("Name", false).ClearSort()
	tt.Settle()
	if n := fired.get(); n != 0 {
		t.Fatalf("setters ran OnRowsReordered %d times", n)
	}
	dragRow(t, tt, "dan", "ann", true)
	if n := fired.get(); n != 1 {
		t.Fatalf("a drag after the setters ran it %d times, want 1", n)
	}
}

// Case 11: the setters from another goroutine while the table shows: no race,
// and the last call shows at the next frame.
func TestDragFromGoroutine(t *testing.T) {
	tb := dragGrid()
	tt := NewTester(t, tb, 800, 600)
	out := func(rows []*acct) any { return rows }
	var wg sync.WaitGroup
	wg.Add(1)
	stop := make(chan struct{})
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			switch i % 3 {
			case 0:
				tb.SetRowReorder(true)
			case 1:
				tb.DragRows(out)
			default:
				tb.OnRowsReordered(func() {})
			}
		}
	}()
	for range 50 {
		tt.Frame()
	}
	close(stop)
	wg.Wait()
	tb.SetRowReorder(true)
	tt.Frame()
	if tb.list.Reorder == nil || tb.list.DragRows != nil {
		t.Fatal("SetRowReorder(true) last: want Reorder alone at the next frame")
	}
	tb.DragRows(out)
	tt.Frame()
	if tb.list.Reorder != nil || tb.list.DragRows == nil {
		t.Fatal("DragRows last: want DragRows alone at the next frame")
	}
}

// Case 12: a table with dragging off builds the tree of one that never had it,
// and allocates no more per frame.
func TestDragOffIsUnchanged(t *testing.T) {
	tb := dragGrid().SetRowReorder(true).DragRows(func(rows []*acct) any { return rows })
	tb.DragRows(nil).SetRowReorder(false)
	tt := NewTester(t, tb, 800, 600)
	if tb.list.Reorder != nil || tb.list.DragRows != nil {
		t.Fatal("off, ui was given row dragging")
	}
	plain := NewTester(t, dragGrid(), 800, 600)
	if got, want := tcTree(tt.Tester), tcTree(plain.Tester); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("off:\n%q\nnever on:\n%q", got, want)
	}
	if a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, tt.Frame); b > a {
		t.Fatalf("a frame at rest allocates %v times turned off, %v never on", b, a)
	}
}

// readGrid is a table of n rows whose Name counts its reads in reads.
func readGrid(n int, reads *fires) *DataGridViewWidget[acct, int] {
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{i + 1, "row " + strconv.Itoa(i+1), i}
	}
	return DataGridView(func(a *acct) int { return a.id },
		Column[acct]{Title: "Name", Text: func(a *acct) string { reads.inc(); return a.name }, Width: 200},
	).SetRows(rows).SetRowHeight(24).Expand()
}

// Case 13: on, a frame at rest allocates at most one more per visible row,
// and one more again (ui keeps no rows of a list that drags them, and makes
// each row's drag source as it builds it): 8 rows shown, at most 9 more. In
// a table of 1000 rows of which the window shows at most 30, the rows built
// at rest stay those shown, and so the allocations, not all the rows.
func TestDragAtRestAllocationBounded(t *testing.T) {
	out := func(rows []*acct) any { return rows }
	for _, c := range []struct {
		rows, shown int // shown: at most this many rows fit in 800×600 at 24 DIPs
	}{{8, 8}, {1000, 30}} {
		var rp, rr, ro fires
		plain := NewTester(t, readGrid(c.rows, &rp), 800, 600)
		reorder := NewTester(t, readGrid(c.rows, &rr).SetRowReorder(true), 800, 600)
		drag := NewTester(t, readGrid(c.rows, &ro).DragRows(out), 800, 600)
		r0, o0 := rr.get(), ro.get()
		for range 10 {
			reorder.Frame()
			drag.Frame()
		}
		// Name is read twice a row built (its text, and the row's label).
		if n := (rr.get() - r0) / 10; n > int32(2*c.shown) {
			t.Errorf("%d rows: reordering reads %d names a frame at rest, want at most %d", c.rows, n, 2*c.shown)
		}
		if n := (ro.get() - o0) / 10; n > int32(2*c.shown) {
			t.Errorf("%d rows: dragging out reads %d names a frame at rest, want at most %d", c.rows, n, 2*c.shown)
		}
		a := allocrun.Least(30, plain.Frame)
		bound := a + float64(c.shown) + 1
		if b := allocrun.Least(30, reorder.Frame); b > bound {
			t.Errorf("%d rows: a frame at rest allocates %v times reordering, want at most %v (%v without)", c.rows, b, bound, a)
		}
		if b := allocrun.Least(30, drag.Frame); b > bound {
			t.Errorf("%d rows: a frame at rest allocates %v times dragging out, want at most %v (%v without)", c.rows, b, bound, a)
		}
	}
}
