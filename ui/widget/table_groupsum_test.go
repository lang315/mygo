package widget

import (
	"fmt"
	"hash/fnv"
	"iter"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// gsRow is a row of the group totals' cases: a file of a kind and a size.
type gsRow struct {
	ID         int
	Name, Kind string
	Size       int
}

// gsRows are the cases' rows: Doc 10+20+30 = 60, Sheet 5+7 = 12, Image 100.
func gsRows() []gsRow {
	return []gsRow{
		{1, "a.txt", "Doc", 10}, {2, "b.xls", "Sheet", 5}, {3, "c.txt", "Doc", 20},
		{4, "d.png", "Image", 100}, {5, "e.xls", "Sheet", 7}, {6, "f.txt", "Doc", 30},
	}
}

// gsCols are the cases' columns: Name, Kind and Size, with no GroupSummary.
func gsCols() []Column[gsRow] {
	return []Column[gsRow]{
		{Title: "Name", Text: func(r *gsRow) string { return r.Name }, Width: 200, Sortable: true, Filterable: true},
		{Title: "Kind", Text: func(r *gsRow) string { return r.Kind }, Width: 120},
		IntColumn("Size", 100, func(r *gsRow) int { return r.Size }),
	}
}

// gsCalls count the runs of the cases' GroupSummary functions.
type gsCalls struct{ size, kind atomic.Int32 }

// gsSumCols are gsCols with totals: Size's sum, and Kind's count of files.
func gsSumCols(n *gsCalls) []Column[gsRow] {
	cols := gsCols()
	cols[1].GroupSummary = func(rows iter.Seq[*gsRow]) string {
		n.kind.Add(1)
		c := 0
		for range rows {
			c++
		}
		return strconv.Itoa(c) + " files"
	}
	cols[2].GroupSummary = func(rows iter.Seq[*gsRow]) string {
		n.size.Add(1)
		total := 0
		for r := range rows {
			total += r.Size
		}
		return strconv.Itoa(total)
	}
	return cols
}

// gsGrid is the cases' grid of cols, holding gsRows, grouped by Kind.
func gsGrid(cols ...Column[gsRow]) *DataGridViewWidget[gsRow, int] {
	return DataGridView(func(r *gsRow) int { return r.ID }, cols...).SetRows(gsRows()).
		SetGroupBy(func(r *gsRow) string { return r.Kind }).Expand()
}

// nodesPrint is a hash of the roles, labels, values and states of the nodes
// of the last frame, and their count: what a grid builds, whatever the OS's
// fonts measure.
func nodesPrint(tt *ui.Tester) string {
	h := fnv.New64a()
	nodes := tt.Nodes()
	for _, n := range nodes {
		fmt.Fprintf(h, "%d|%s|%s|%t%t%t%t|", n.Role, n.Label, n.Value, n.Disabled, n.Checked, n.Selected, n.Expanded)
	}
	return strconv.Itoa(len(nodes)) + ":" + strconv.FormatUint(h.Sum64(), 16)
}

// mainPrint is nodesPrint of gsGrid(gsCols()...) in profTester, as main
// built it before GroupSummary (origin/main 1789a5c6c).
const mainPrint = "53:156ca18c5dce3755"

// headLine returns the bounds of the text of the group header head, as
// "Doc (3)".
func headLine(t *testing.T, tt *ui.Tester, head string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleText && n.Label == head && n.Bounds.W > 0 {
			return n.Bounds
		}
	}
	t.Fatalf("no group header %q: %q", head, tt.Texts())
	return ui.Rect{}
}

// groupTotals returns the texts on the line of the group header head that lie
// under the header of the column title, but the header's own text.
func groupTotals(t *testing.T, tt *ui.Tester, head, title string) []ui.TestNode {
	t.Helper()
	line, col := headLine(t, tt, head), colHead(t, tt, title)
	cy := line.Y + line.H/2
	var in []ui.TestNode
	for _, n := range tt.Nodes() {
		b := n.Bounds
		if n.Role != ui.RoleText || b.W <= 0 || n.Label == head {
			continue
		}
		if my := b.Y + b.H/2; my-cy > 2 || cy-my > 2 {
			continue
		}
		if b.X >= col.X-1 && b.X+b.W <= col.X+col.W+1 {
			in = append(in, n)
		}
	}
	return in
}

// wantTotal fails unless the line of the group header head shows exactly
// texts under the column title.
func wantTotal(t *testing.T, tt *ui.Tester, head, title string, texts ...string) {
	t.Helper()
	var got []string
	for _, n := range groupTotals(t, tt, head, title) {
		got = append(got, n.Label)
	}
	if !slices.Equal(got, texts) {
		t.Errorf("%s's line shows %q under %s, want %q", head, got, title, texts)
	}
}

// groupSumsOf returns g's group totals' state, failing without it.
func groupSumsOf(t *testing.T, g *DataGridViewWidget[gsRow, int]) *gridGroupSums {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if g.groupSums == nil {
		t.Fatal("the grid has no group totals")
	}
	return g.groupSums
}

// Grouped by Kind, each group's header line shows its own totals under
// their columns, Size's at the end of its cell (End), and nothing under
// Name but the header's own text.
func TestGroupSummaryInHeader(t *testing.T) {
	var n gsCalls
	tt := profTester(gsGrid(gsSumCols(&n)...))
	for _, c := range []struct{ head, size, kind string }{
		{"Doc (3)", "60", "3 files"}, {"Image (1)", "100", "1 files"}, {"Sheet (2)", "12", "2 files"},
	} {
		wantTotal(t, tt, c.head, "Size", c.size)
		wantTotal(t, tt, c.head, "Kind", c.kind)
		wantTotal(t, tt, c.head, "Name")
	}
	wantAligned(t, tt, "60", "10", true)        // Size is End: the total ends where a cell's text does
	wantAligned(t, tt, "3 files", "Doc", false) // Kind is Start: it starts where a cell's text does
}

// A header line painted elsewhere than the totals were placed by, the
// header's cells where they were, asks for the frame that places them, and
// the line's start is what that frame places them from.
func TestGroupSummaryLineMovedAsksAFrame(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	gs := groupSumsOf(t, g)
	mu.Lock()
	asked, x := gs.asked, gs.xs[0]
	// Out of a frame, where the header's cells cannot be read: a moved line
	// answers before they are (a line in place, which reads them, is
	// TestGroupSummaryNotInFrameAtRest's).
	gs.checkPlaces(0, ui.Rect{X: x + 5, W: 100})
	moved, at := gs.asked, gs.xs[0]
	gs.xs[0] = x // as the frame's paint will find it
	mu.Unlock()
	if moved != asked+1 || at != x+5 {
		t.Errorf("a line painted 5 DIPs on asked for %d frames and keeps start %v, want 1 and %v", moved-asked, at, x+5)
	}
	if !tt.FramePending() {
		t.Error("a line painted 5 DIPs on asked the window for no frame")
	}
	tt.Frame()
	wantAligned(t, tt, "60", "10", true)
}

// gsTextAt returns the bounds of the first text label shown whole (a row
// half under the pinned header shows a sliver of its texts).
func gsTextAt(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleText && n.Label == label && n.Bounds.W > 0 && n.Bounds.H > 10 {
			return n.Bounds
		}
	}
	t.Fatalf("no text %q: %q", label, tt.Texts())
	return ui.Rect{}
}

// wantAligned fails unless the total's text ends (end) or starts within 1 DIP
// of where the text cell of a row of its column does: under it, with the
// cells' padding.
func wantAligned(t *testing.T, tt *ui.Tester, total, cell string, end bool) {
	t.Helper()
	a, b := gsTextAt(t, tt, total), gsTextAt(t, tt, cell)
	got, want := a.X, b.X
	if end {
		got, want = a.X+a.W, b.X+b.W
	}
	if !withinDIP(got, want, 1) {
		t.Errorf("the total %q is at %v, the cell %q at %v: want them aligned (end %v)", total, a, cell, b, end)
	}
}

// Moving a row to another group updates both groups' totals.
func TestGroupSummaryAfterUpdateRow(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	g.UpdateRow(1, func(r *gsRow) { r.Kind = "Sheet" }) // a.txt, 10
	tt.Frame()
	wantTotal(t, tt, "Doc (2)", "Size", "50")
	wantTotal(t, tt, "Sheet (3)", "Size", "22")
	wantTotal(t, tt, "Image (1)", "Size", "100")
	// A size changed in place, its group alone moving.
	g.UpdateRow(4, func(r *gsRow) { r.Size = 1 })
	tt.Frame()
	wantTotal(t, tt, "Image (1)", "Size", "1")
}

// A frame at rest runs no GroupSummary; a frame after UpdateRow runs each
// column's once for each group.
func TestGroupSummaryNotInFrameAtRest(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	if got := n.size.Load(); got != 3 {
		t.Fatalf("the first frame ran Size's GroupSummary %d times, want 3: one a group", got)
	}
	for range 10 {
		tt.Frame()
	}
	if got := n.size.Load(); got != 3 {
		t.Fatalf("10 frames at rest ran Size's GroupSummary %d more times", got-3)
	}
	g.UpdateRow(5, func(r *gsRow) { r.Size = 8 })
	tt.Frame()
	if got := n.size.Load(); got != 6 {
		t.Fatalf("a frame after UpdateRow ran Size's GroupSummary %d times, want 3", got-3)
	}
	wantTotal(t, tt, "Sheet (2)", "Size", "13")
	// At rest, the header lines ask for no frame (their places settled).
	gs := groupSumsOf(t, g)
	mu.Lock()
	asked := gs.asked
	mu.Unlock()
	for range 5 {
		tt.Frame()
	}
	mu.Lock()
	defer mu.Unlock()
	if gs.asked != asked {
		t.Fatalf("5 frames at rest asked for %d more frames", gs.asked-asked)
	}
}

// With no grouping, no total shows and no GroupSummary runs; grouping
// again shows them.
func TestGroupSummaryWithoutGroups(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...).SetGroupBy(nil)
	tt := profTester(g)
	for range 5 {
		tt.Frame()
	}
	if s, k := n.size.Load(), n.kind.Load(); s != 0 || k != 0 {
		t.Fatalf("an ungrouped grid ran Size's GroupSummary %d times and Kind's %d", s, k)
	}
	for _, s := range []string{"60", "12", "3 files", "1 files"} { // no cell shows these
		if shows(tt, s) {
			t.Fatalf("an ungrouped grid shows the total %q", s)
		}
	}
	g.SetGroupBy(func(r *gsRow) string { return r.Kind })
	tt.Frame()
	tt.Frame()
	wantTotal(t, tt, "Doc (3)", "Size", "60")
}

// Rows changed from another goroutine while frames run: the totals follow,
// and nothing races (go test -race).
func TestGroupSummaryFromGoroutine(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 100 {
			<-tick
			g.UpdateRow(1+i%6, func(r *gsRow) { r.Size++ })
		}
	}()
	for range 100 {
		tick <- struct{}{}
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	tt.Frame()
	// Each row grew by 100/6: rows 1 to 4 by 17, rows 5 and 6 by 16.
	wantTotal(t, tt, "Doc (3)", "Size", "110")   // 27 + 37 + 46
	wantTotal(t, tt, "Sheet (2)", "Size", "45")  // 22 + 23
	wantTotal(t, tt, "Image (1)", "Size", "117") // 117
}

// A grid with no GroupSummary builds main's tree and has no hook; an
// ungrouped grid with GroupSummary builds what one without it does.
func TestGroupSummaryOffIsUnchanged(t *testing.T) {
	plain := gsGrid(gsCols()...)
	tt := profTester(plain)
	if got := nodesPrint(tt); got != mainPrint {
		t.Errorf("a grouped grid with no GroupSummary builds %s, main %s", got, mainPrint)
	}
	mu.Lock()
	hook := plain.groupSums
	mu.Unlock()
	if hook != nil {
		t.Fatal("a grid with no GroupSummary has the group totals' hook")
	}
	var n gsCalls
	a, b := profTester(gsGrid(gsCols()...).SetGroupBy(nil)), profTester(gsGrid(gsSumCols(&n)...).SetGroupBy(nil))
	na, nb := a.Nodes(), b.Nodes()
	if len(na) != len(nb) {
		t.Fatalf("%d nodes ungrouped with GroupSummary, %d without", len(nb), len(na))
	}
	for i := range na {
		x, y := na[i], nb[i]
		x.ID, y.ID = 0, 0
		if x != y {
			t.Fatalf("node %d: %+v with GroupSummary, %+v without", i, y, x)
		}
	}
}

// A frame at rest of a grouped grid with totals allocates no more than the
// same grid without them.
func TestGroupSummaryAllocNothingAtRest(t *testing.T) {
	var n gsCalls
	plain, with := profTester(gsGrid(gsCols()...)), profTester(gsGrid(gsSumCols(&n)...))
	for range 10 {
		plain.Frame()
		with.Frame()
	}
	a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	t.Logf("a frame at rest allocates %v times with group totals, %v without", b, a)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with group totals, %v without", b, a)
	}
}

// The totals are of the rows the column filters and the search keep, as the
// groups are.
func TestGroupSummaryOfFilteredRows(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	g.SetColumnFilter("Name", ColumnFilter{Values: []string{"a.txt", "b.xls", "d.png", "f.txt"}})
	tt.Frame()
	wantTotal(t, tt, "Doc (2)", "Size", "40")
	wantTotal(t, tt, "Sheet (1)", "Size", "5")
	g.ClearFilters().Search("Name", "xls")
	tt.Frame()
	wantTotal(t, tt, "Sheet (2)", "Size", "12")
	if shows(tt, "Doc (3)") || shows(tt, "60") {
		t.Fatal("the search's hidden group shows")
	}
}

// A hidden column's totals are gone; the others stay under their headers.
func TestGroupSummaryHiddenColumn(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	g.SetColumnVisible("Kind", false)
	tt.Frame()
	if shows(tt, "3 files") {
		t.Fatal("the hidden Kind column's total shows")
	}
	wantTotal(t, tt, "Doc (3)", "Size", "60")
	wantAligned(t, tt, "60", "10", true)
}

// A column the user moves takes its totals along.
func TestGroupSummaryFollowsAMovedColumn(t *testing.T) {
	var n gsCalls
	tt := profTester(gsGrid(gsSumCols(&n)...))
	h := colHead(t, tt, "Size")
	x, y := h.X+h.W/2, h.Y+h.H/2
	tt.Press(x, y)
	for _, d := range []float32{-5, -20, -60, -120, -180, -240} {
		tt.Move(x+d, y)
	}
	tt.Release(x-240, y)
	tt.Frame()
	if s, k := colHead(t, tt, "Size"), colHead(t, tt, "Kind"); s.X >= k.X {
		t.Fatalf("the move did not happen: Size at %v, Kind at %v", s.X, k.X)
	}
	wantTotal(t, tt, "Doc (3)", "Size", "60")
	wantTotal(t, tt, "Doc (3)", "Kind", "3 files")
	wantAligned(t, tt, "60", "10", true)
	wantAligned(t, tt, "3 files", "Doc", false)
}

// The user drags the Size column wider: its total follows in the frame
// after the release.
func TestGroupSummaryFollowsAResize(t *testing.T) {
	var n gsCalls
	tt := profTester(gsGrid(gsSumCols(&n)...))
	h := colHead(t, tt, "Size")
	x, y := h.X+h.W-1, h.Y+h.H/2
	tt.Press(x, y)
	for _, d := range []float32{10, 20, 30, 40} {
		tt.Move(x+d, y)
	}
	tt.Release(x+40, y)
	tt.Frame()
	if got := colHead(t, tt, "Size").W; !withinDIP(got, h.W+40, 2) {
		t.Fatalf("Size's header is %v wide after the drag, want %v", got, h.W+40)
	}
	wantAligned(t, tt, "60", "10", true)
}

// The window grows, and with it the Name column, which shares the room
// left: the totals after it move in the next frame, the lines' start where
// it was.
func TestGroupSummaryFollowsAWindowResize(t *testing.T) {
	var n gsCalls
	cols := gsSumCols(&n)
	cols[0].Width = 0
	tt := profTester(gsGrid(cols...))
	before := colHead(t, tt, "Size")
	tt.SetSize(1100, 600)
	tt.Frame()
	if after := colHead(t, tt, "Size"); after.X == before.X {
		t.Fatalf("Size did not move with the window: at %v", after)
	}
	wantAligned(t, tt, "60", "10", true)
	wantAligned(t, tt, "3 files", "Doc", false)
}

// Columns wider than the grid scroll sideways, the totals with them; a
// Frozen first column's stay pinned under it.
func TestGroupSummaryScrollsWithTheColumns(t *testing.T) {
	var n gsCalls
	cols := gsSumCols(&n)
	cols[0].Width, cols[1].Width, cols[2].Width = 400, 400, 400
	cols[1].Frozen, cols[0].Frozen = true, true
	cols[0], cols[1] = cols[1], cols[0] // Kind first, pinned
	tt := profTester(gsGrid(cols...))
	before := colHead(t, tt, "Size")
	tt.Wheel(450, 300, 300, 0)
	tt.Advance(time.Second) // the wheel glides
	tt.Frame()
	tt.Frame()
	if after := colHead(t, tt, "Size"); after.X == before.X {
		t.Fatalf("the wheel did not scroll the columns: Size at %v", after)
	}
	// The header's title scrolled out with the line's start: the totals are
	// found by their texts, which no cell shows.
	wantUnder(t, tt, "60", "Size")
	wantUnder(t, tt, "3 files", "Kind")
	wantAligned(t, tt, "60", "10", true)
	wantAligned(t, tt, "3 files", "Doc", false)
}

// wantUnder fails unless a text label shows within the header of the column
// title, left and right.
func wantUnder(t *testing.T, tt *ui.Tester, label, title string) {
	t.Helper()
	col := colHead(t, tt, title)
	var at []ui.Rect
	for _, n := range tt.Nodes() {
		if b := n.Bounds; n.Role == ui.RoleText && n.Label == label && b.W > 0 {
			if b.X >= col.X-1 && b.X+b.W <= col.X+col.W+1 {
				return
			}
			at = append(at, b)
		}
	}
	t.Errorf("%q is not under %s at %v: it shows at %v", label, title, col, at)
}

// The index and check box columns move the totals right, still under their
// headers.
func TestGroupSummaryIndexAndCheckboxColumns(t *testing.T) {
	var n gsCalls
	tt := profTester(gsGrid(gsSumCols(&n)...).IndexColumn().CheckboxColumn())
	tt.Frame()
	wantTotal(t, tt, "Doc (3)", "Size", "60")
	wantTotal(t, tt, "Doc (3)", "Kind", "3 files")
	wantTotal(t, tt, "Doc (3)", "#")
	wantAligned(t, tt, "60", "10", true)
	wantAligned(t, tt, "3 files", "Doc", false)
}

// A GroupSummary that panics leaves its texts empty and the grid drawing,
// and the panic is reported once, as a handler's.
func TestGroupSummaryPanicIsReported(t *testing.T) {
	var reports atomic.Int32
	OnPanic(func(any, []byte) { reports.Add(1) })
	t.Cleanup(func() { OnPanic(nil) })
	var n gsCalls
	cols := gsSumCols(&n)
	cols[1].GroupSummary = func(iter.Seq[*gsRow]) string { panic("boom") }
	tt := NewTester(t, gsGrid(cols...), 900, 600)
	tt.Settle()
	if got := reports.Load(); got != 3 {
		t.Fatalf("%d panics reported, want 3: one a group", got)
	}
	wantTotal(t, tt.Tester, "Doc (3)", "Kind")
	wantTotal(t, tt.Tester, "Doc (3)", "Size", "60")
	for range 5 {
		tt.Frame()
	}
	tt.Settle()
	if got := reports.Load(); got != 3 {
		t.Fatalf("%d panics reported after frames at rest, want 3", got)
	}
}

// A GroupSummary that stops early, at a group's first row, shows it, and
// nothing panics.
func TestGroupSummaryStopsEarly(t *testing.T) {
	var reports atomic.Int32
	OnPanic(func(any, []byte) { reports.Add(1) })
	t.Cleanup(func() { OnPanic(nil) })
	cols := gsCols()
	cols[2].GroupSummary = func(rows iter.Seq[*gsRow]) string {
		for r := range rows {
			return "first " + r.Name
		}
		return ""
	}
	tt := NewTester(t, gsGrid(cols...), 900, 600)
	tt.Settle()
	wantTotal(t, tt.Tester, "Doc (3)", "Size", "first a.txt")
	wantTotal(t, tt.Tester, "Sheet (2)", "Size", "first b.xls")
	if got := reports.Load(); got != 0 {
		t.Fatalf("%d panics reported", got)
	}
}

// runsTotal is a GroupSummary of the profiles' runs: their sum.
func runsTotal(rows iter.Seq[*prof]) string {
	total := 0
	for p := range rows {
		total += p.Runs
	}
	return strconv.Itoa(total)
}

// pinnedGrid is the 10,000 profiles grouped by status, Runs totalled per
// group, with cols' widths w (0 keeps sumCols').
func pinnedGrid(w float32) *DataGridViewWidget[prof, int] {
	cols := sumCols(nil)
	cols[2].GroupSummary = runsTotal
	if w > 0 {
		for i := range cols {
			cols[i].Width = w
		}
	}
	return profGrid(cols...).SetGroupBy(func(p *prof) string { return p.Status })
}

// wantSettled fails unless frames at rest ask for no more frames.
func wantSettled(t *testing.T, tt *ui.Tester, g *DataGridViewWidget[prof, int], what string) {
	t.Helper()
	mu.Lock()
	gs := g.groupSums
	asked := gs.asked
	mu.Unlock()
	for range 5 {
		tt.Frame()
	}
	mu.Lock()
	after := gs.asked
	mu.Unlock()
	if after != asked || tt.FramePending() {
		t.Errorf("%s: 5 frames at rest asked for %d more frames (pending %v)", what, after-asked, tt.FramePending())
	}
}

// Scrolled down, the "failed" header pinned at the top and "idle"'s in the
// rows: both lines' totals sit under Runs, at the end of a cell's text,
// and frames at rest ask for nothing; so too further down, and scrolled
// sideways with wide columns.
func TestGroupSummaryPinnedHeader(t *testing.T) {
	g := pinnedGrid(0)
	tt := profTester(g)
	tt.Wheel(450, 300, 0, 150)
	tt.Advance(time.Second)
	tt.Frame()
	for _, total := range []string{"7", "9681"} {
		wantUnder(t, tt, total, "Runs")
		wantAligned(t, tt, total, "1", true)
	}
	wantSettled(t, tt, g, "failed pinned, idle in the rows")
	tt.Wheel(450, 300, 0, 5000)
	tt.Advance(time.Second)
	tt.Frame()
	wantUnder(t, tt, "9681", "Runs")
	wantAligned(t, tt, "9681", "1", true)
	wantSettled(t, tt, g, "idle pinned")

	g = pinnedGrid(400)
	tt = profTester(g)
	tt.Wheel(450, 300, 0, 150)
	tt.Advance(time.Second)
	tt.Frame()
	tt.Wheel(450, 300, 300, 0)
	tt.Advance(time.Second)
	tt.Frame()
	for _, total := range []string{"7", "9681"} {
		wantUnder(t, tt, total, "Runs")
		wantAligned(t, tt, total, "1", true)
	}
	wantSettled(t, tt, g, "wide columns scrolled sideways, failed pinned")
}

// Group lines scrolled into view for the first time are placed where the
// lines painted before start: no frame is asked for to move them.
func TestGroupSummaryNewLineInPlace(t *testing.T) {
	cols := sumCols(nil)
	cols[2].GroupSummary = runsTotal
	g := profGrid(cols...).SetGroupBy(func(p *prof) string { return fmt.Sprintf("g%03d", p.ID/20) }) // 500 groups of 20
	tt := profTester(g)
	wantSettled(t, tt, g, "at first")
	mu.Lock()
	asked, lines := g.groupSums.asked, len(g.groupSums.checks)
	mu.Unlock()
	tt.Wheel(450, 300, 0, 3000)
	tt.Advance(time.Second)
	tt.Frame()
	mu.Lock()
	asked2, lines2 := g.groupSums.asked, len(g.groupSums.checks)
	mu.Unlock()
	if lines2 <= lines {
		t.Fatalf("the scroll built no new group line: %d lines, %d before", lines2, lines)
	}
	if asked2 != asked {
		t.Errorf("new group lines asked for %d frames to move", asked2-asked)
	}
	wantAligned(t, tt, "20", "1", true)
}

// While frames scroll, a goroutine moves the failed rows to idle and back:
// the failed group goes and comes back, and its totals follow (go test
// -race).
func TestGroupSummaryPinnedFromGoroutine(t *testing.T) {
	g := pinnedGrid(0)
	tt := profTester(g)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 40 {
			<-tick
			status := "idle"
			if i%2 == 1 {
				status = "failed"
			}
			for r := 7; r < 7000; r += 1000 {
				g.UpdateRow(r, func(p *prof) { p.Status = status })
			}
		}
	}()
	for i := range 40 {
		tick <- struct{}{}
		tt.Wheel(450, 300, 0, float32(10+i%3*40))
		tt.Advance(time.Second)
		tt.Frame()
	}
	wg.Wait()
	tt.Wheel(450, 300, 0, -100000)
	tt.Advance(time.Second)
	tt.Frame()
	tt.Frame()
	wantTotal(t, tt, "failed (7)", "Runs", "7")
	wantTotal(t, tt, "idle (9681)", "Runs", "9681")
	wantAligned(t, tt, "7", "1", true)
}

// Two grids on one RowSource: a row changed through one moves the other's
// totals too.
func TestGroupSummarySharedRowSource(t *testing.T) {
	src := NewRowSource(func(r *gsRow) int { return r.ID })
	src.SetRows(gsRows())
	var n gsCalls
	by := func(r *gsRow) string { return r.Kind }
	a := DataGridViewOf(src, gsSumCols(&n)...).SetGroupBy(by).Expand()
	b := DataGridViewOf(src, gsSumCols(&n)...).SetGroupBy(by).Expand()
	ta, tb := profTester(a), profTester(b)
	a.UpdateRow(4, func(r *gsRow) { r.Size = 1 })
	ta.Frame()
	tb.Frame()
	wantTotal(t, ta, "Image (1)", "Size", "1")
	wantTotal(t, tb, "Image (1)", "Size", "1")
}

// Under a column filter, a row changed in place moves out of its group and
// into another, both totals following; one the filter then hides leaves.
func TestGroupSummaryUpdateRowUnderFilter(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	g.SetColumnFilter("Name", ColumnFilter{Values: []string{"a.txt", "b.xls", "c.txt", "e.xls"}})
	tt.Frame()
	wantTotal(t, tt, "Doc (2)", "Size", "30")
	wantTotal(t, tt, "Sheet (2)", "Size", "12")
	g.UpdateRow(1, func(r *gsRow) { r.Kind = "Sheet" }) // a.txt, 10
	tt.Frame()
	wantTotal(t, tt, "Doc (1)", "Size", "20")
	wantTotal(t, tt, "Sheet (3)", "Size", "22")
	g.UpdateRow(3, func(r *gsRow) { r.Name = "c.doc" }) // the filter hides it: Doc empties
	tt.Frame()
	if shows(tt, "Doc (1)") || shows(tt, "Doc (0)") {
		t.Fatalf("the emptied Doc group still shows: %q", tt.Texts())
	}
	wantTotal(t, tt, "Sheet (3)", "Size", "22")
}

// A column the chooser hides takes its totals away; the others stay under
// their headers in the next frame.
func TestGroupSummaryColumnChooserHides(t *testing.T) {
	var n gsCalls
	tt := menuTester(t, gsGrid(gsSumCols(&n)...).SetColumnChooser(true))
	if err := tt.RightClick("Name"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Kind"); err != nil {
		t.Fatal(err)
	}
	drained(t)
	tt.Frame()
	if shows(tt, "Kind") || shows(tt, "3 files") {
		t.Fatalf("Kind or its total still shows: %q", tt.Texts())
	}
	wantTotal(t, tt, "Doc (3)", "Size", "60")
	wantAligned(t, tt, "60", "10", true)
}

// BestFit narrows Kind: Size moves left, its total with it, in one frame.
func TestGroupSummaryBestFit(t *testing.T) {
	var n gsCalls
	g := gsGrid(gsSumCols(&n)...)
	tt := profTester(g)
	before := colHead(t, tt, "Kind")
	g.BestFit("Kind")
	tt.Frame()
	if after := colHead(t, tt, "Kind"); after.W == before.W {
		t.Fatalf("BestFit left Kind %v wide", after.W)
	}
	wantTotal(t, tt, "Doc (3)", "Size", "60")
	wantTotal(t, tt, "Doc (3)", "Kind", "3 files")
	wantAligned(t, tt, "60", "10", true)
	wantAligned(t, tt, "3 files", "Doc", false)
	wantSettledG(t, tt, g.groupSums, "after BestFit")
}

// Frozen Name and Status, Status totalled per group, and a wide Runs:
// scrolled down (failed pinned, idle in the rows) and sideways, Status'
// totals stay under its pinned header and Runs' under its scrolled one, on
// both lines, settled.
func TestGroupSummaryFrozenPinnedSideways(t *testing.T) {
	cols := sumCols(nil)
	cols[0].Width, cols[0].Frozen = 200, true
	cols[1].Width, cols[1].Frozen = 200, true
	cols[1].Summary = nil
	cols[1].GroupSummary = func(rows iter.Seq[*prof]) string {
		c := 0
		for range rows {
			c++
		}
		return strconv.Itoa(c) + " profiles"
	}
	cols[2].Width, cols[2].GroupSummary = 500, runsTotal
	cols = append(cols, Column[prof]{Title: "Note", Text: func(p *prof) string { return p.Name }, Width: 600}) // room to scroll
	g := profGrid(cols...).SetGroupBy(func(p *prof) string { return p.Status })
	tt := profTester(g)
	status := colHead(t, tt, "Status")
	tt.Wheel(450, 300, 0, 150)
	tt.Advance(time.Second)
	tt.Frame()
	tt.Wheel(450, 300, 300, 0)
	tt.Advance(time.Second)
	tt.Frame()
	if h := colHead(t, tt, "Status"); h.X != status.X {
		t.Fatalf("the frozen Status moved from %v to %v", status.X, h.X)
	}
	if h := colHead(t, tt, "Runs"); h.X >= 401 {
		t.Fatalf("the wheel did not scroll Runs: at %v", h.X)
	}
	for _, c := range []struct{ total, cell string }{{"7 profiles", "failed"}, {"9681 profiles", "idle"}} {
		wantUnder(t, tt, c.total, "Status")
		wantAligned(t, tt, c.total, c.cell, false)
	}
	for _, total := range []string{"7", "9681"} {
		wantUnder(t, tt, total, "Runs")
		wantAligned(t, tt, total, "1", true)
	}
	wantSettledG(t, tt, g.groupSums, "frozen, pinned, sideways")
}

// wantSettledG fails unless frames at rest ask for no more frames.
func wantSettledG(t *testing.T, tt *ui.Tester, gs *gridGroupSums, what string) {
	t.Helper()
	mu.Lock()
	asked := gs.asked
	mu.Unlock()
	for range 5 {
		tt.Frame()
	}
	mu.Lock()
	after := gs.asked
	mu.Unlock()
	if after != asked || tt.FramePending() {
		t.Errorf("%s: 5 frames at rest asked for %d more frames (pending %v)", what, after-asked, tt.FramePending())
	}
}

// An empty text shows nothing on the line, and builds no cell; nor do the
// columns without a GroupSummary.
func TestGroupSummaryEmptyText(t *testing.T) {
	cols := gsCols()
	cols[2].GroupSummary = func(iter.Seq[*gsRow]) string { return "" }
	g := gsGrid(cols...)
	tt := profTester(g)
	wantTotal(t, tt, "Doc (3)", "Size")
	gs := groupSumsOf(t, g)
	mu.Lock()
	defer mu.Unlock()
	if gs.cells != 0 {
		t.Fatalf("%d cells built for empty totals", gs.cells)
	}
}

// A ListView's Details groups show the totals too, as its grid's.
func TestGroupSummaryListViewDetails(t *testing.T) {
	var n gsCalls
	lv := ListView(func(r *gsRow) int { return r.ID }, gsSumCols(&n)...).SetRows(gsRows()).
		SetGroupBy(func(r *gsRow) string { return r.Kind })
	tt := profTester(lv)
	tt.Frame()
	wantTotal(t, tt, "Doc (3)", "Size", "60")
}
