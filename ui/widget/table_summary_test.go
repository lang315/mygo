package widget

import (
	"fmt"
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

// sumCalls count the runs of the cases' Summary functions.
type sumCalls struct{ status, runs atomic.Int32 }

// statusSummary counts the rows running and failed, as the cases' Status
// column does.
func statusSummary(n *sumCalls) func(iter.Seq[*prof]) string {
	return func(rows iter.Seq[*prof]) string {
		n.status.Add(1)
		running, failed := 0, 0
		for p := range rows {
			switch p.Status {
			case "running":
				running++
			case "failed":
				failed++
			}
		}
		return fmt.Sprintf("%d running · %d failed", running, failed)
	}
}

// sumCols are the cases' columns: Name, which shares the room left and
// searches for the rows of IDs under its keyword; Status and Runs, with
// summaries. With n nil, none has a Summary.
func sumCols(n *sumCalls) []Column[prof] {
	cols := []Column[prof]{
		{Title: "Name", Text: func(p *prof) string { return p.Name }, MinWidth: 120, Filterable: true,
			Filter: func(p *prof, kw string) bool { k, _ := strconv.Atoi(kw); return p.ID < k }},
		{Title: "Status", Text: func(p *prof) string { return p.Status }, Width: 200},
		{Title: "Runs", Text: func(p *prof) string { return strconv.Itoa(p.Runs) }, Width: 100, Align: End},
	}
	if n != nil {
		cols[1].Summary = statusSummary(n)
		cols[2].Summary = func(rows iter.Seq[*prof]) string {
			n.runs.Add(1)
			total := 0
			for p := range rows {
				total += p.Runs
			}
			return strconv.Itoa(total)
		}
	}
	return cols
}

// sumGrid is the cases' grid with summaries.
func sumGrid(n *sumCalls) *DataGridViewWidget[prof, int] { return profGrid(sumCols(n)...) }

// footCells returns the cells under the table, left to right: the footer's.
func footCells(tt *ui.Tester) []ui.TestNode {
	nodes := tt.Nodes()
	var table ui.Rect
	for _, n := range nodes {
		if n.Role == ui.RoleTable {
			table = n.Bounds
		}
	}
	var cells []ui.TestNode
	for _, n := range nodes {
		// Rows the list keeps out of view are cells too, empty and scrolled.
		if n.Role == ui.RoleCell && n.Bounds.Y >= table.Y+table.H && n.Bounds.W > 0 && !n.Scrolls {
			cells = append(cells, n)
		}
	}
	slices.SortFunc(cells, func(a, b ui.TestNode) int { return int(a.Bounds.X - b.Bounds.X) })
	return cells
}

// colHead returns the bounds of the header of the column titled title.
func colHead(t *testing.T, tt *ui.Tester, title string) ui.Rect {
	t.Helper()
	return tcFind(t, tt, ui.RoleColumnHeader, title).Bounds
}

// footUnder returns the footer's cell under the header of title: its left
// edge and its width within 1 DIP of the header's.
func footUnder(t *testing.T, tt *ui.Tester, title string) ui.Rect {
	t.Helper()
	h := colHead(t, tt, title)
	cells := footCells(tt)
	for _, c := range cells {
		if withinDIP(c.Bounds.X, h.X, 1) && withinDIP(c.Bounds.W, h.W, 1) {
			return c.Bounds
		}
	}
	var got []ui.Rect
	for _, c := range cells {
		got = append(got, c.Bounds)
	}
	t.Fatalf("no footer cell under the header %s at %v: cells %v", title, h, got)
	return ui.Rect{}
}

// textsIn returns the texts shown inside r.
func textsIn(tt *ui.Tester, r ui.Rect) []string {
	var in []string
	for _, n := range tt.Nodes() {
		b := n.Bounds
		if n.Role == ui.RoleText && b.W > 0 && b.X >= r.X-1 && b.X+b.W <= r.X+r.W+1 && b.Y >= r.Y-1 && b.Y+b.H <= r.Y+r.H+1 {
			in = append(in, n.Label)
		}
	}
	return in
}

// wantIn fails unless the cell r shows exactly texts.
func wantIn(t *testing.T, tt *ui.Tester, what string, r ui.Rect, texts ...string) {
	t.Helper()
	if got := textsIn(tt, r); !slices.Equal(got, texts) {
		t.Errorf("%s shows %q, want %q", what, got, texts)
	}
}

// footerOf returns g's footer, failing without one.
func footerOf(t *testing.T, g *DataGridViewWidget[prof, int]) *gridFooter {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if g.footer == nil {
		t.Fatal("the grid has no footer")
	}
	return g.footer
}

// builds returns how many frames built f.
func builds(f *gridFooter) int {
	mu.Lock()
	defer mu.Unlock()
	return f.builds
}

// 1. A row of totals under the table, each cell under its header.
func TestGridSummaryRowUnderTheColumns(t *testing.T) {
	var n sumCalls
	tt := profTester(sumGrid(&n))
	if cells := footCells(tt); len(cells) != 3 {
		t.Fatalf("%d cells under the table, want 3: %+v", len(cells), cells)
	}
	wantIn(t, tt, "Name's footer cell", footUnder(t, tt, "Name"))
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "312 running · 7 failed")
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10000")
}

// 2. A row changed in place changes the totals in the next frame.
func TestGridSummaryFollowsUpdateRow(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	g.UpdateRow(0, func(p *prof) { p.Status = "running" })
	tt.Frame()
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "313 running · 7 failed")
}

// 3. Under a search the totals are of the rows it shows.
func TestGridSummaryOfTheSearchedRows(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	g.Search("Name", "50") // rows 0 to 49: running 1 and 33, failed 7
	tt.Frame()
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "2 running · 1 failed")
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "50")
}

// 4. The user drags the Runs column wider: the footer's cell follows within
// two frames.
func TestGridSummaryFollowsAResize(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	f := footerOf(t, g)
	// Runs ends at the window's edge: it is narrowed first, then widened.
	// drag drags Runs' edge by dx and returns how many builds of the footer
	// the release and a frame made.
	drag := func(dx float32) int {
		h := colHead(t, tt, "Runs")
		x, y := h.X+h.W-1, h.Y+h.H/2
		tt.Press(x, y)
		tt.Move(x+dx/3, y)
		tt.Move(x+dx*2/3, y)
		tt.Move(x+dx, y)
		before := builds(f)
		tt.Release(x+dx, y)
		tt.Frame()
		return builds(f) - before
	}
	drag(-60)
	footUnder(t, tt, "Runs")
	h := colHead(t, tt, "Runs")
	b := drag(40)
	if got := colHead(t, tt, "Runs").W; !withinDIP(got, h.W+40, 2) {
		t.Fatalf("Runs' header is %v wide after the drag, want %v", got, h.W+40)
	}
	if b > 3 {
		t.Errorf("the release and a frame built the footer %d times, want 3 at most: one for the release, then the frame and the one it asks for", b)
	}
	footUnder(t, tt, "Runs")
	// One more step, alone: the frame that lays it out asks for the one
	// that moves the footer.
	g.SetColumnVisible("Status", false)
	before := builds(f)
	tt.Frame()
	if b := builds(f) - before; b > 2 {
		t.Errorf("hiding Status built the footer %d times, want 2 at most", b)
	}
	footUnder(t, tt, "Runs")
}

// 5. A frame at rest runs no Summary; one after UpdateRow runs each once.
func TestGridSummaryRunsOnlyWhenTheRowsChange(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	s, r := n.status.Load(), n.runs.Load()
	for range 10 {
		tt.Frame()
	}
	if s2, r2 := n.status.Load(), n.runs.Load(); s2 != s || r2 != r {
		t.Fatalf("10 frames at rest ran Status' Summary %d times and Runs' %d", s2-s, r2-r)
	}
	g.UpdateRow(5, func(p *prof) { p.Runs = 2 })
	tt.Frame()
	if s2, r2 := n.status.Load(), n.runs.Load(); s2 != s+1 || r2 != r+1 {
		t.Fatalf("a frame after UpdateRow ran Status' Summary %d times and Runs' %d, want 1 each", s2-s, r2-r)
	}
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10001")
}

// 6. A column the user moves takes its footer cell along.
func TestGridSummaryFollowsAMovedColumn(t *testing.T) {
	var n sumCalls
	tt := profTester(sumGrid(&n))
	h := colHead(t, tt, "Runs")
	x, y := h.X+h.W/2, h.Y+h.H/2
	tt.Press(x, y)
	for _, d := range []float32{-5, -20, -60, -120, -180, -240} {
		tt.Move(x+d, y)
	}
	tt.Release(x-240, y)
	tt.Frame()
	if r, s := colHead(t, tt, "Runs"), colHead(t, tt, "Status"); r.X >= s.X {
		t.Fatalf("the move did not happen: Runs at %v, Status at %v", r.X, s.X)
	}
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10000")
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "312 running · 7 failed")
}

// 7. A hidden column's footer cell is gone.
func TestGridSummaryHiddenColumn(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	g.SetColumnVisible("Status", false)
	tt.Frame()
	if cells := footCells(tt); len(cells) != 2 {
		t.Fatalf("%d cells under the table with Status hidden, want 2", len(cells))
	}
	if shows(tt, "312 running · 7 failed") {
		t.Fatal("the hidden Status column's total shows")
	}
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10000")
	footUnder(t, tt, "Name")
}

// 10. A grid with no Summary has no footer, and no hook.
func TestGridWithoutSummaryHasNoFooter(t *testing.T) {
	g := profGrid(sumCols(nil)...)
	tt := profTester(g)
	mu.Lock()
	summarize, footer := g.summarize, g.footer
	mu.Unlock()
	if summarize != nil || footer != nil {
		t.Fatal("a grid with no Summary has the footer's hook")
	}
	if cells := footCells(tt); len(cells) != 0 {
		t.Fatalf("cells under the table: %+v", cells)
	}
}

// 11. A frame at rest of a grid with summaries and groups allocates no more
// than the same grid without them, and runs no Summary.
func TestGridSummaryFrameAllocatesNothingMore(t *testing.T) {
	var n sumCalls
	plain := profTester(profGrid(sumCols(nil)...))
	with := profTester(sumGrid(&n).SetGroupBy(func(p *prof) string { return p.Status }))
	for range 10 {
		plain.Frame()
		with.Frame()
	}
	s := n.status.Load()
	a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with summaries and groups, %v without", b, a)
	}
	if got := n.status.Load(); got != s {
		t.Errorf("the frames at rest ran Summary %d times", got-s)
	}
}

// 12. Rows changed from another goroutine while frames run: the totals
// follow, and nothing races (go test -race).
func TestGridSummaryUpdatedFromAGoroutine(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 300 {
			<-tick
			g.UpdateRow(i*32, func(p *prof) { p.Status = "running" }) // idle rows 0, 32, ..., 9568
		}
	}()
	for range 300 {
		tick <- struct{}{}
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "612 running · 7 failed")
}

// Review Focus 1: columns wider than the grid scroll, the footer with them,
// its cells under their headers.
func TestGridSummaryScrollsWithTheColumns(t *testing.T) {
	var n sumCalls
	cols := sumCols(&n)
	cols[0].Width, cols[1].Width, cols[2].Width = 400, 400, 400
	tt := profTester(profGrid(cols...))
	before := colHead(t, tt, "Runs")
	tt.Wheel(450, 300, 300, 0)
	tt.Advance(time.Second) // the wheel glides
	tt.Frame()
	after := colHead(t, tt, "Runs")
	if after.X == before.X && after.W == before.W {
		t.Fatalf("the wheel did not scroll the columns: Runs at %v", after)
	}
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10000")
	footUnder(t, tt, "Status")
}

// Review Focus 2: at text scale 1.5, under each built-in skin, light and
// dark, the footer fills as the header does and its text reaches 4.5:1.
func TestGridSummaryContrastUnderTheSkins(t *testing.T) {
	for _, skin := range []string{"", "Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dark=%v", skin, dark), func(t *testing.T) {
				if skin != "" {
					useSkinNamed(t, skin, false)
				} else {
					calmOff()
				}
				var n sumCalls
				tt := profTester(sumGrid(&n))
				tt.SetDark(dark)
				tt.SetPreferences(ui.Preferences{TextScale: 1.5})
				tt.Advance(time.Second)
				tt.Frame()
				h, c := colHead(t, tt, "Runs"), footUnder(t, tt, "Runs")
				img := tt.Image()
				// Runs is end-aligned: the cells' starts hold no text.
				hb, fb := img.RGBAAt(int(h.X+6), int(h.Y+h.H/2)), img.RGBAAt(int(c.X+6), int(c.Y+c.H/2))
				if hb != fb {
					t.Errorf("the footer fills with %v, the header with %v", fb, hb)
				}
				text := tcFind(t, tt, ui.RoleText, "10000").Bounds
				if got := tcContrast(tt, text, fb); got < 4.5 {
					t.Errorf("the total reaches %.2f:1 on the footer, want 4.5:1", got)
				}
			})
		}
	}
}

// Review Focus 3: a Summary that panics leaves its cell empty and the grid
// drawing, and the panic is reported as a handler's.
func TestGridSummaryPanicIsReported(t *testing.T) {
	var reports atomic.Int32
	OnPanic(func(any, []byte) { reports.Add(1) })
	t.Cleanup(func() { OnPanic(nil) })
	var n sumCalls
	cols := sumCols(&n)
	cols[1].Summary = func(iter.Seq[*prof]) string { panic("boom") }
	tt := NewTester(t, profGrid(cols...), 900, 600)
	tt.Settle()
	if got := reports.Load(); got != 1 {
		t.Fatalf("%d panics reported, want 1", got)
	}
	wantIn(t, tt.Tester, "Status's footer cell", footUnder(t, tt.Tester, "Status"))
	wantIn(t, tt.Tester, "Runs' footer cell", footUnder(t, tt.Tester, "Runs"), "10000")
	if !shows(tt.Tester, "p00000") {
		t.Fatal("the grid's rows do not show")
	}
	for range 5 {
		tt.Frame()
	}
	tt.Settle()
	if got := reports.Load(); got != 1 {
		t.Fatalf("%d panics reported after frames at rest, want 1", got)
	}
}

// Review Focus 4: with no rows each Summary gets an empty sequence, and its
// text shows.
func TestGridSummaryOfNoRows(t *testing.T) {
	var n sumCalls
	g := sumGrid(&n)
	tt := profTester(g)
	g.SetRows(nil)
	tt.Frame()
	wantIn(t, tt, "Status's footer cell", footUnder(t, tt, "Status"), "0 running · 0 failed")
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "0")
}

// Review Focus 5: the index column's footer cell is empty, as wide as its
// header; the check boxes' column has none, and the cells after stay under
// their headers.
func TestGridSummaryIndexAndCheckboxColumns(t *testing.T) {
	var n sumCalls
	tt := profTester(sumGrid(&n).IndexColumn().CheckboxColumn())
	if cells := footCells(tt); len(cells) != 4 {
		t.Fatalf("%d cells under the table, want 4: #, Name, Status, Runs", len(cells))
	}
	wantIn(t, tt, "#'s footer cell", footUnder(t, tt, "#"))
	wantIn(t, tt, "Name's footer cell", footUnder(t, tt, "Name"))
	wantIn(t, tt, "Runs' footer cell", footUnder(t, tt, "Runs"), "10000")
}
