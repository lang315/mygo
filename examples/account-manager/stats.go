package main

import (
	"fmt"
	"runtime/metrics"
	"slices"
	"time"
)

// stats measures the app for a log line a second: how often frames come
// and how long its view takes to build them, how many changes reach the
// rows, and what the runtime holds. MYGO_FRAME_STATS splits a frame into
// its parts besides.
type stats struct {
	began, since time.Time // when measuring began, and this second
	last         time.Time // when the last frame began
	intervals    []time.Duration
	builds       []time.Duration
	updates      int // changes applied this second
	total        int // and since measuring began
	batches      int
	applyMax     time.Duration
	cycles       uint64 // GC cycles ended when this second began
}

func newStats(now time.Time) *stats {
	_, cycles, _ := readRuntime()
	return &stats{began: now, since: now, cycles: cycles}
}

// frame notes a frame built from start to end.
func (s *stats) frame(start, end time.Time) {
	if !s.last.IsZero() {
		s.intervals = append(s.intervals, start.Sub(s.last))
	}
	s.last = start
	s.builds = append(s.builds, end.Sub(start))
}

// applied notes a batch of n changes applied to the rows in took.
func (s *stats) applied(n int, took time.Duration) {
	s.updates += n
	s.total += n
	s.batches++
	s.applyMax = max(s.applyMax, took)
}

// flush returns the line of the second ended at now, and starts the next,
// or "" before a second passed. Its fields are name=value, the sort
// quoted, for a column's title has spaces.
func (s *stats) flush(now time.Time, a *app) string {
	d := now.Sub(s.since)
	if d < time.Second {
		return ""
	}
	order := "-"
	if a.table.Sort.Column != "" {
		order = a.table.Sort.Column + "↑"
		if a.table.Sort.Descending {
			order = a.table.Sort.Column + "↓"
		}
	}
	heap, cycles, goroutines := readRuntime()
	line := fmt.Sprintf("stats t=%.0fs rows=%d shown=%d sort=%q updates=%.0f/s batches=%d apply_max=%s frames=%d interval_p50=%s p95=%s max=%s build_p50=%s p95=%s max=%s heap=%dMB gc=%d goroutines=%d",
		now.Sub(s.began).Seconds(), len(a.s.accounts), a.table.Len(), order,
		float64(s.updates)/d.Seconds(), s.batches, ms(s.applyMax), len(s.builds),
		ms(percentile(s.intervals, 50)), ms(percentile(s.intervals, 95)), ms(percentile(s.intervals, 100)),
		ms(percentile(s.builds, 50)), ms(percentile(s.builds, 95)), ms(percentile(s.builds, 100)),
		heap>>20, cycles-s.cycles, goroutines)
	s.since, s.cycles = now, cycles
	s.intervals, s.builds = s.intervals[:0], s.builds[:0]
	s.updates, s.batches, s.applyMax = 0, 0, 0
	return line
}

// percentile is the nearest-rank p-th percentile of d, 0 for none.
func percentile(d []time.Duration, p int) time.Duration {
	if len(d) == 0 {
		return 0
	}
	sorted := slices.Clone(d)
	slices.Sort(sorted)
	return sorted[max(0, (p*len(sorted)+99)/100-1)]
}

func ms(d time.Duration) string { return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond)) }

// runtimeSamples are the runtime's counters a line reads.
var runtimeSamples = []metrics.Sample{
	{Name: "/memory/classes/heap/objects:bytes"},
	{Name: "/gc/cycles/total:gc-cycles"},
	{Name: "/sched/goroutines:goroutines"},
}

// readRuntime returns the bytes of the objects on the heap, the GC cycles
// ended, and the goroutines.
func readRuntime() (heap, cycles, goroutines uint64) {
	metrics.Read(runtimeSamples)
	return runtimeSamples[0].Value.Uint64(), runtimeSamples[1].Value.Uint64(), runtimeSamples[2].Value.Uint64()
}
