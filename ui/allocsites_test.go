package ui

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"testing"
)

// allocsPerRun is testing.AllocsPerRun for tests that fail above limit.
// AllocsPerRun counts the mallocs of every goroutine and of the runtime, so
// an earlier test's timer, finalizer or goroutine, or the runtime starting a
// thread (a loaded runner does), allocating meanwhile fails a test whose own
// code is steady. Such mallocs only add to a count: what f allocates in
// every run is in each count. So allocsPerRun returns the least of up to
// allocTries counts, and is over limit only if each one is. When it is, the
// test log says where mallocs come from in one more run of f, with every
// goroutine alive.
func allocsPerRun(t *testing.T, runs int, limit float64, f func()) float64 {
	t.Helper()
	n := testing.AllocsPerRun(runs, f)
	for try := 1; n > limit && try < allocTries; try++ {
		n = min(n, testing.AllocsPerRun(runs, f))
	}
	if n > limit {
		t.Logf("%.0f mallocs per run of %d over the limit %.0f; %s", n, runs, limit, allocReport(runs, f))
	}
	return n
}

// allocTries is how many counts allocsPerRun takes at most.
const allocTries = 5

// allocSite is where mallocs came from.
type allocSite struct{ objects, bytes int64 }

// memSites returns the mallocs by their call sites, as of the last
// garbage collection, which it makes.
func memSites() map[string]allocSite {
	runtime.GC()
	recs := make([]runtime.MemProfileRecord, 1024)
	for {
		n, ok := runtime.MemProfile(recs, true)
		if ok {
			recs = recs[:n]
			break
		}
		recs = make([]runtime.MemProfileRecord, n+256)
	}
	sites := map[string]allocSite{}
	for _, r := range recs {
		var sb strings.Builder
		frames := runtime.CallersFrames(r.Stack())
		for i := 0; i < 8; i++ {
			fr, more := frames.Next()
			fmt.Fprintf(&sb, "%s:%d <- ", fr.Function, fr.Line)
			if !more {
				break
			}
		}
		s := sites[sb.String()]
		s.objects += r.AllocObjects
		s.bytes += r.AllocBytes
		sites[sb.String()] = s
	}
	return sites
}

// allocReport runs f runs times with every malloc sampled, and describes
// the sites that allocated, with the goroutines alive.
func allocReport(runs int, f func()) string {
	old := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() { runtime.MemProfileRate = old }()
	before := memSites()
	n := testing.AllocsPerRun(runs, f)
	after := memSites()
	type diff struct {
		site    string
		objects int64
	}
	var diffs []diff
	for site, a := range after {
		if strings.Contains(site, "ui.memSites") || strings.Contains(site, "ui.allocReport") {
			continue // the report's own
		}
		if d := a.objects - before[site].objects; d > 0 {
			diffs = append(diffs, diff{site, d})
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].objects > diffs[j].objects })
	var sb strings.Builder
	fmt.Fprintf(&sb, "the second run made %.0f mallocs per run; sites (objects: stack):\n", n)
	for i, d := range diffs {
		if i == 10 {
			break
		}
		fmt.Fprintf(&sb, "  %d: %s\n", d.objects, d.site)
	}
	var gs bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&gs, 1)
	fmt.Fprintf(&sb, "goroutines:\n%s", gs.String())
	return sb.String()
}

var allocSink []byte

// The report names the code that allocated.
func TestAllocReportNamesTheSites(t *testing.T) {
	r := allocReport(5, func() { allocSink = make([]byte, 64) })
	if !strings.Contains(r, "allocsites_test.go") || !strings.Contains(r, "goroutine profile") {
		t.Fatalf("report has neither the allocating site nor the goroutines:\n%.2000s", r)
	}
}

// Mallocs of other goroutines that stop (a runtime table growing, a timer, a
// test's leftover) add to a count and never take from it: allocsPerRun is
// the least of its tries, so it is not failed by them.
func TestAllocsPerRunIgnoresMallocsThatStop(t *testing.T) {
	calls := 0
	noisy := func() {
		calls++
		if calls <= 25 { // the first try: its warm-up call and its 20 runs
			for range 4 {
				allocSink = make([]byte, 64)
			}
		}
	}
	if n := allocsPerRun(t, 20, 2, noisy); n > 2 {
		t.Fatalf("%.0f mallocs per run for a function that allocated in its first try only", n)
	}
}

// A function that allocates in every try is still over the limit.
func TestAllocsPerRunKeepsSteadyMallocs(t *testing.T) {
	if n := allocsPerRun(t, 20, 2, func() {
		allocSink = make([]byte, 64)
		allocSink = make([]byte, 64)
		allocSink = make([]byte, 64)
	}); n < 3 {
		t.Fatalf("%.0f mallocs per run for a function that makes 3 in every run, want at least 3", n)
	}
}
