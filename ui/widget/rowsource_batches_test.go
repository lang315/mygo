package widget

import (
	"slices"
	"strconv"
	"sync"
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

// Cancel stops after the batch in hand: the rows of the batches after it are
// not added, and the progress func is not called again.
func TestAddRowsInBatchesCanBeCancelled(t *testing.T) {
	var cancel func()
	reached := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	src := NewRowSource(func(a *acct) int {
		calls++
		if calls == 3 { // inside the second batch of two
			close(reached)
			<-release
		}
		return a.id
	})
	var log progressLog
	cancel = src.AddRowsInBatches(accts(10), 2, log.add)
	received(t, reached)
	cancel()
	close(release)
	wait.Until(t, "the load to stop", func() bool { return src.Len() == 4 })
	// The load is over: one more Post after it tells that no report is left.
	flushed := make(chan struct{})
	Post(func() { close(flushed) })
	received(t, flushed)
	// A report posted before the cancel may or may not have run; none past
	// the batch in hand does, and none says the load is complete.
	for _, c := range log.get() {
		if c[0] > 2 {
			t.Fatalf("progress %v after cancel", log.get())
		}
	}
	if src.Len() != 4 {
		t.Fatalf("%d rows held after cancel, want the 2 batches in hand", src.Len())
	}
}
