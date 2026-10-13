package main

import (
	"flag"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var e2eRows = flag.Int("e2e.rows", 100_000, "how many accounts TestLiveColumns shows")

// raceBuild is set under -race, whose frames are too slow to time.
var raceBuild bool

// liveColumns are the columns a stress run changes.
var liveColumns = []int{colStatus, colFriends, colGroups, colNote, colIP, colLastRun}

// TestLiveColumns drives the app as a busy run does, 10,000 changes a
// second to the columns of liveColumns, while it is sorted by each column
// both ways, searched by a column changing, and moved to another folder.
// After every frame the rows shown must be those sorted and filtered from
// scratch, their cells must show the rows' values, and the version must
// change if and only if the order did. Run it at a million rows with
// -args -e2e.rows=1000000.
func TestLiveColumns(t *testing.T) {
	n := *e2eRows
	a := newApp(n)
	// A third of the accounts in a second folder, for a switch to show.
	other := a.s.addFolder()
	var ids []int
	for i := 0; i < n; i += 3 {
		ids = append(ids, a.s.accounts[i].ID)
	}
	a.s.move(ids, other.ID)
	e := &live{t: t, tt: newTester(t, a), a: a, compare: compare(),
		rnd: rand.New(rand.NewPCG(1, 2)), r: &runner{done: make(chan struct{})},
		every: max(1, n/100_000)}
	e.tt.SetSize(3200, 800) // every column shown, for its header to click and its cells to read
	e.tt.Frame()
	e.check(true)

	e.ticks(10) // not sorted
	for _, col := range append(slices.Clone(liveColumns), colName) {
		title := columns[col].Title
		for _, desc := range []bool{false, true} {
			e.tt.do(e.tt.Click(title))
			if want := (ui.SortOrder{Column: title, Descending: desc}); a.sort != want {
				t.Fatalf("a click on %s sorted by %+v, want %+v", title, a.sort, want)
			}
			e.check(true)
			e.ticks(10)
		}
	}

	// Sorted by a column changing, searched by one changing: the rows the
	// search keeps change at every tick.
	e.tt.do(e.tt.Click(columns[colFriends].Title))
	a.column = columns[colStatus].Title // the select's popup is the select's own test's
	e.tt.do(e.tt.Click("Nhập từ khoá (UID1|UID2|…)"))
	e.tt.Type("thành công")
	e.tt.do(e.tt.Click("Tìm kiếm"))
	if a.filtered.keyword != "thành công" {
		t.Fatalf("the search typed is not shown: %+v", a.filtered)
	}
	e.check(true)
	e.ticks(10)
	a.keyword = ""
	e.tt.do(e.tt.Click("Tìm kiếm"))
	e.check(true)
	e.ticks(10)

	for _, f := range []string{other.Name, a.s.folders[0].Name} {
		a.folder = f
		a.showFolder()
		e.tt.Frame()
		e.check(true)
		e.ticks(10)
	}

	p50, p95, worst := percentile(e.took, 50), percentile(e.took, 95), percentile(e.took, 100)
	t.Logf("%d rows, %d frames of 1000 changes: apply and frame p50=%s p95=%s max=%s", n, len(e.took), ms(p50), ms(p95), ms(worst))
	if !raceBuild && p95 > 50*time.Millisecond {
		t.Errorf("p95 of a frame of 1000 changes is %s, want under 50ms", ms(p95))
	}
}

// live drives an app with a stress run's changes, a batch a frame, and
// checks what each frame shows.
type live struct {
	t       *testing.T
	tt      *tester
	a       *app
	compare map[string]func(a, b *account) int
	rnd     *rand.Rand
	r       *runner
	every   int // the ticks between two checks of the order: each sorts all rows
	order   []int
	version uint64
	took    []time.Duration // applying each batch and building its frame
}

// ticks applies n batches of 1000 changes, a frame each, as a stress run
// of 10,000 a second does ten times a second.
func (e *live) ticks(n int) {
	e.t.Helper()
	for i := range n {
		now := time.Now()
		for range 1000 {
			e.r.push(change(e.rnd, e.a.s.nextID, now))
		}
		start := time.Now()
		e.a.apply(e.r)
		e.tt.Frame()
		e.took = append(e.took, time.Since(start))
		e.check(i%e.every == e.every-1 || i == n-1)
	}
}

// check fails unless the version changed if and only if the order shown
// did, and the first rows show in that order with their values; and,
// when full, unless the order is the rows the search keeps, sorted from
// scratch.
func (e *live) check(full bool) {
	e.t.Helper()
	a := e.a
	got := make([]int, a.view.Len())
	for i := range got {
		got[i] = a.view.Row(i)
	}
	if full {
		if want := e.want(); !slices.Equal(got, want) {
			i := 0
			for i < min(len(got), len(want)) && got[i] == want[i] {
				i++
			}
			e.t.Fatalf("sort %+v, search %+v: %d rows, want %d; they differ from row %d", a.sort, a.filtered, len(got), len(want), i)
		}
	}
	if changed := !slices.Equal(got, e.order); changed != (a.view.Version() != e.version) {
		e.t.Fatalf("sort %+v: the order changed %v, the version changed %v", a.sort, changed, !changed)
	}
	e.order, e.version = got, a.view.Version()
	// A row shows its check box, labeled by the UID, then its cells.
	texts, at := e.tt.Texts(), -1
	for row := range min(5, len(got)) {
		x := &a.s.accounts[got[row]]
		i := slices.Index(texts, "Check "+x.UID)
		if i <= at || i+len(columns) >= len(texts) {
			e.t.Fatalf("sort %+v: row %d, %s, not shown after row %d", a.sort, row, x.UID, row-1)
		}
		for col := range columns {
			if s := cellText(x, col); texts[i+1+col] != s {
				e.t.Fatalf("sort %+v: row %d's %s shows %q, want %q", a.sort, row, columns[col].Title, texts[i+1+col], s)
			}
		}
		at = i
	}
}

// want is the order of the rows the search keeps, sorted from scratch:
// rows equal keep the slice's order, as the view's do.
func (e *live) want() []int {
	a := e.a
	keep := keep(a.filtered)
	var rows []int
	for i := range a.s.accounts {
		if keep(&a.s.accounts[i]) {
			rows = append(rows, i)
		}
	}
	if f := e.compare[a.sort.Column]; f != nil {
		slices.SortStableFunc(rows, func(i, j int) int {
			if a.sort.Descending {
				return f(&a.s.accounts[j], &a.s.accounts[i])
			}
			return f(&a.s.accounts[i], &a.s.accounts[j])
		})
	}
	return rows
}
