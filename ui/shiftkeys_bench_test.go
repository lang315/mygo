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

// BenchmarkListExtendToEnd1M measures Shift+End and Cmd+Shift+End from the
// first row of a list of a million rows (the work of one key, with the
// choice put back to the first row outside the timed part), and the heap the
// list holds afterwards.
func BenchmarkListExtendToEnd1M(b *testing.B) {
	for _, tc := range []struct {
		name string
		mods Modifiers
	}{{"shift", Shift}, {"cmd-shift", Cmd | Shift}} {
		b.Run(tc.name, func(b *testing.B) {
			var sel Selection[int]
			s := ListState{Selection: &sel}
			tt := coreNewTester(func(c *context) {
				coreList(c, &s, 1_000_000, func(i int) { coreTextf(c, "Row %d", i).Height(30) }).Grow(1)
			}, 300, 400)
			tt.Click("Row 1")
			tt.Key(0, KeyHome)
			before := heapMB()
			b.ReportAllocs()
			for b.Loop() {
				tt.Key(tc.mods, KeyEnd)
				if sel.Len() != 1_000_000 {
					b.Fatalf("chose %d rows", sel.Len())
				}
				b.StopTimer()
				tt.Key(0, KeyHome)
				b.StartTimer()
			}
			b.ReportMetric(heapMB()-before, "heap-MB-kept")
		})
	}
}

// BenchmarkGridViewExtendToEnd1M is the same for a grid of a million items.
func BenchmarkGridViewExtendToEnd1M(b *testing.B) {
	for _, tc := range []struct {
		name string
		mods Modifiers
	}{{"shift", Shift}, {"cmd-shift", Cmd | Shift}} {
		b.Run(tc.name, func(b *testing.B) {
			var sel Selection[int]
			cur := -1
			s := GridState{Selected: &cur, Selection: &sel, Key: func(i int) any { return i }}
			tt := coreNewTester(func(c *context) {
				coreGridView(c, &s, 1_000_000, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
			}, 448, 300)
			tt.Click("Item 1")
			tt.Key(0, KeyHome)
			before := heapMB()
			b.ReportAllocs()
			for b.Loop() {
				tt.Key(tc.mods, KeyEnd)
				if sel.Len() != 1_000_000 {
					b.Fatalf("chose %d items", sel.Len())
				}
				b.StopTimer()
				tt.Key(0, KeyHome)
				b.StartTimer()
			}
			b.ReportMetric(heapMB()-before, "heap-MB-kept")
		})
	}
}
