package allocrun

import "testing"

var sink []byte

func alloc(n int) {
	for range n {
		sink = make([]byte, 64)
	}
}

// Mallocs of other goroutines and of the runtime (a thread it starts) stop
// and only add to a count: the least of the counts is the function's own.
func TestLeastIgnoresMallocsThatStop(t *testing.T) {
	calls := 0
	noisy := func() {
		calls++
		if calls <= 21 { // the first count: its warm-up call and its 20 runs
			alloc(4)
		}
	}
	if n := Least(20, noisy); n != 0 {
		t.Fatalf("%v mallocs per run for a function that allocated in its first count only", n)
	}
}

// What a function allocates in every run is in every count.
func TestLeastKeepsSteadyMallocs(t *testing.T) {
	if n := Least(20, func() { alloc(3) }); n != 3 {
		t.Fatalf("%v mallocs per run for a function that makes 3 in every run, want 3", n)
	}
}

// A function that allocates nothing is run once for each of its runs and the
// warm-up, as testing.AllocsPerRun does: no count is repeated.
func TestLeastCountsOnceWhatAllocatesNothing(t *testing.T) {
	calls := 0
	if n := Least(20, func() { calls++ }); n != 0 {
		t.Fatalf("%v mallocs per run for a function that allocates nothing", n)
	}
	if calls != 21 {
		t.Fatalf("the function ran %d times, want 21 (a warm-up and 20 runs)", calls)
	}
}
