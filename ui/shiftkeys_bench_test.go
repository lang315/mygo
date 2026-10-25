package ui

import (
	"runtime"
	"testing"
)

// heapMB returns the live heap in MB after a collection.
func heapMB() float64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1 << 20)
}

// extendCases are what the benchmarks of End on a million rows press: Shift
// and Cmd+Shift from the first row, and from the first row with every row
// chosen (Cmd+A first), where every row Cmd+Shift reaches is chosen already
// and is kept as an exception, the worst case of the extension's memory.
var extendCases = []struct {
	name string
	mods Modifiers
	all  bool
}{
	{"shift", Shift, false},
	{"cmd-shift", Cmd | Shift, false},
	{"all-shift", Shift, true},
	{"all-cmd-shift", Cmd | Shift, true},
}

// BenchmarkListExtendToEnd1M measures End with Shift or Cmd+Shift from the
// first row of a list of a million rows (the work of one key, with the choice
// put back to the first row outside the timed part), and the heap the list
// holds right after the first key (heap-MB-after, the rows chosen and the
// exceptions) and after the choice is put back to the first row
// (heap-MB-kept), both over the heap with nothing chosen.
func BenchmarkListExtendToEnd1M(b *testing.B) {
	for _, tc := range extendCases {
		b.Run(tc.name, func(b *testing.B) {
			var sel Selection[int]
			s := ListState{Selection: &sel}
			tt := coreNewTester(func(c *context) {
				coreList(c, &s, 1_000_000, func(i int) { coreTextf(c, "Row %d", i).Height(30) }).Grow(1)
			}, 300, 400)
			tt.Click("Row 1")
			tt.Key(0, KeyHome)
			nothing := heapMB() // nothing chosen but the first row
			if tc.all {
				tt.Key(Cmd, KeyA)
			}
			b.ReportAllocs()
			first := true
			for b.Loop() {
				tt.Key(tc.mods, KeyEnd)
				if sel.Len() != 1_000_000 {
					b.Fatalf("chose %d rows", sel.Len())
				}
				b.StopTimer()
				if first {
					b.ReportMetric(heapMB()-nothing, "heap-MB-after")
					first = false
				}
				tt.Key(0, KeyHome)
				if tc.all {
					tt.Key(Cmd, KeyA)
				}
				b.StartTimer()
			}
			tt.Key(0, KeyHome)
			b.ReportMetric(heapMB()-nothing, "heap-MB-kept")
		})
	}
}

// BenchmarkGridViewExtendToEnd1M is the same for a grid of a million items.
func BenchmarkGridViewExtendToEnd1M(b *testing.B) {
	for _, tc := range extendCases {
		b.Run(tc.name, func(b *testing.B) {
			var sel Selection[int]
			cur := -1
			s := GridState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }}
			tt := coreNewTester(func(c *context) {
				coreGridView(c, &s, 1_000_000, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
			}, 448, 300)
			tt.Click("Item 1")
			tt.Key(0, KeyHome)
			nothing := heapMB() // nothing chosen but the first row
			if tc.all {
				tt.Key(Cmd, KeyA)
			}
			b.ReportAllocs()
			first := true
			for b.Loop() {
				tt.Key(tc.mods, KeyEnd)
				if sel.Len() != 1_000_000 {
					b.Fatalf("chose %d items", sel.Len())
				}
				b.StopTimer()
				if first {
					b.ReportMetric(heapMB()-nothing, "heap-MB-after")
					first = false
				}
				tt.Key(0, KeyHome)
				if tc.all {
					tt.Key(Cmd, KeyA)
				}
				b.StartTimer()
			}
			tt.Key(0, KeyHome)
			b.ReportMetric(heapMB()-nothing, "heap-MB-kept")
		})
	}
}
