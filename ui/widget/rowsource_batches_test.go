package widget

import (
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

func accts(n int) []acct {
	rows := make([]acct, n)
	for i := range rows {
		rows[i] = acct{i + 1, "n" + strconv.Itoa(i+1), i}
	}
	return rows
}

// progressLog collects the calls of a progress func, which run on the event
// goroutine.
type progressLog struct {
	mu    sync.Mutex
	calls [][2]int
}

func (p *progressLog) add(done, total int) {
	p.mu.Lock()
	p.calls = append(p.calls, [2]int{done, total})
	p.mu.Unlock()
}

func (p *progressLog) get() [][2]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.calls)
}

func TestAddRowsInBatchesReportsEachBatchAndEndsAtTheTotal(t *testing.T) {
	src := NewRowSource(acctKey)
	var log progressLog
	src.AddRowsInBatches(accts(10), 4, log.add)
	wait.Until(t, "the last report", func() bool { c := log.get(); return len(c) == 3 })
	want := [][2]int{{4, 10}, {8, 10}, {10, 10}}
	if got := log.get(); !slices.Equal(got, want) {
		t.Fatalf("progress %v, want %v", got, want)
	}
	if src.Len() != 10 {
		t.Fatalf("%d rows held, want 10", src.Len())
	}
}

func TestAddRowsInBatchesShowsTheRowsInTheTable(t *testing.T) {
	src := NewRowSource(acctKey)
	tb := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200})
	tt := tester(t, tb)
	done := make(chan struct{})
	src.AddRowsInBatches(accts(5), 2, func(d, total int) {
		if d == total {
			close(done)
		}
	})
	received(t, done)
	tt.Frame()
	for _, s := range []string{"n1", "n5"} {
		if !tt.HasText(s) {
			t.Errorf("the table lacks %q: %q", s, tt.Texts())
		}
	}
}

// A batch bigger than the rows, or a batch of zero, is one batch.
func TestAddRowsInBatchesWithAnyBatchSizeEnds(t *testing.T) {
	for _, batch := range []int{0, -1, 100} {
		src := NewRowSource(acctKey)
		done := make(chan [2]int, 4)
		src.AddRowsInBatches(accts(3), batch, func(d, total int) { done <- [2]int{d, total} })
		if got := received(t, done); got != [2]int{3, 3} {
			t.Errorf("batch %d: progress %v, want {3 3}", batch, got)
		}
	}
}

func TestAddRowsInBatchesOfNoRowsReportsOnce(t *testing.T) {
	src := NewRowSource(acctKey)
	done := make(chan [2]int, 4)
	src.AddRowsInBatches(nil, 4, func(d, total int) { done <- [2]int{d, total} })
	if got := received(t, done); got != [2]int{0, 0} {
		t.Errorf("progress %v, want {0 0}", got)
	}
}

// Cancel stops after the batch in hand, waits for the loader to exit, and
// drops the reports posted but not yet run: none runs once cancel returned.
func TestAddRowsInBatchesCancelDropsPostedReportsAndWaitsForTheLoader(t *testing.T) {
	// Hold the event goroutine, so the reports of the batches done queue.
	hold, held := make(chan struct{}), make(chan struct{})
	Post(func() { close(held); <-hold })
	received(t, held)
	reached, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	src := NewRowSource(func(a *acct) int {
		if calls.Add(1) == 3 { // inside the second batch of two
			close(reached)
			<-release
		}
		return a.id
	})
	var log progressLog
	cancel := src.AddRowsInBatches(accts(10), 2, log.add)
	received(t, reached)
	close(release)
	cancel() // returns once the loader has ended
	n := calls.Load()
	if src.Len() != 4 || n != 4 {
		t.Fatalf("after cancel: %d rows held, %d key calls, want the 2 batches in hand (4, 4)", src.Len(), n)
	}
	close(hold)
	idle()
	if got := log.get(); len(got) != 0 {
		t.Fatalf("progress %v ran after cancel returned", got)
	}
	if calls.Load() != n {
		t.Fatalf("the loader went on after cancel returned")
	}
}

// Cancel from the progress func, on the event goroutine, waits for the loader
// without waiting on itself, and SetRows(nil) after it leaves no row.
func TestAddRowsInBatchesCancelFromProgressThenSetRows(t *testing.T) {
	src := NewRowSource(acctKey)
	cancelled := make(chan func(), 1) // the cancel func reaches progress by this
	done := make(chan int)
	cancel := src.AddRowsInBatches(accts(2000), 1, func(d, total int) {
		if d == 1 {
			(<-cancelled)()
			src.SetRows(nil)
			done <- src.Len()
		}
	})
	cancelled <- cancel
	if n := received(t, done); n != 0 {
		t.Fatalf("%d rows after cancel and SetRows(nil)", n)
	}
	idle()
	if src.Len() != 0 {
		t.Fatalf("%d rows came back after SetRows(nil)", src.Len())
	}
}

// progress runs on the event goroutine, and the rows are added in order.
func TestAddRowsInBatchesReportsOnTheEventGoroutineAndKeepsTheOrder(t *testing.T) {
	src := NewRowSource(acctKey)
	var off atomic.Bool
	done := make(chan struct{})
	src.AddRowsInBatches(accts(25), 4, func(d, total int) {
		if !onEvents() {
			off.Store(true)
		}
		if d == total {
			close(done)
		}
	})
	received(t, done)
	if off.Load() {
		t.Error("progress ran off the event goroutine")
	}
	mu.Lock()
	defer mu.Unlock()
	for i, r := range src.rows {
		if r.id != i+1 {
			t.Fatalf("row %d has id %d: batches out of order", i, r.id)
		}
	}
	if len(src.rows) != 25 {
		t.Fatalf("%d rows", len(src.rows))
	}
}

// A panic in the key func ends the load and is reported as a handler's is; it
// does not end the program, and the source stays usable.
func TestAddRowsInBatchesReportsAPanicOfTheKeyFunc(t *testing.T) {
	got := reports(t)
	src := NewRowSource(func(a *acct) int {
		if a.id == 5 {
			panic("bad row")
		}
		return a.id
	})
	cancel := src.AddRowsInBatches(accts(10), 2, nil)
	if r := received(t, got); r.v != "bad row" {
		t.Fatalf("reported %v, want bad row", r.v)
	}
	cancel() // the loader has ended: no wait
	src.AddRows([]acct{{99, "z", 0}})
	if src.Len() != 5 { // rows 1-4 and 99: row 5 and what followed never came
		t.Fatalf("%d rows held, want 5", src.Len())
	}
}

// The source grows once at the start, so appending a batch reallocates no
// more: a table that merges new rows into its view is not made to sort them
// all again.
func TestAddRowsInBatchesGrowsTheSourceOnce(t *testing.T) {
	src := NewRowSource(acctKey)
	done := make(chan struct{})
	var first, last *acct
	src.AddRowsInBatches(accts(1000), 10, func(d, total int) {
		mu.Lock()
		if d == 10 {
			first = &src.rows[0]
		}
		if d == total {
			last = &src.rows[0]
			close(done)
		}
		mu.Unlock()
	})
	received(t, done)
	if first != last {
		t.Fatal("the rows moved to a new array between batches")
	}
}
