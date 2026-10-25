package ui

import "testing"

// headerEvery returns a Header func: every k-th item heads a group.
func headerEvery(k int) func(int) bool { return func(i int) bool { return i%k == 0 } }

// BenchmarkGridViewHeadersRescan1M measures what the grid does when it has to
// ask Header for every item of a million: the number of items changed, or
// ReloadHeaders was called. The first scan allocates the headers' places; the
// ones after it reuse them.
func BenchmarkGridViewHeadersRescan1M(b *testing.B) {
	for _, tc := range []struct {
		name  string
		every int
	}{{"1000-headers", 1000}, {"100k-headers", 10}} {
		b.Run(tc.name, func(b *testing.B) {
			s := &GridState{Header: headerEvery(tc.every)}
			s.prepare(1_000_000, 8)
			b.ReportAllocs()
			for b.Loop() {
				s.ReloadHeaders()
				s.prepare(1_000_000, 8)
			}
			if got := len(s.layout.heads); got != 1_000_000/tc.every {
				b.Fatalf("found %d headers", got)
			}
		})
	}
}

// BenchmarkGridViewHeaders1M scrolls a grid of a million items a frame a step,
// as BenchmarkGridViewChecked1M does: without Header, and with a header every
// 1000 items and every 10 (the frame asks Header for each header it knows, to
// see that it is one still).
func BenchmarkGridViewHeaders1M(b *testing.B) {
	for _, tc := range []struct {
		name  string
		every int
	}{{"none", 0}, {"every-1000", 1000}, {"every-10", 10}} {
		b.Run(tc.name, func(b *testing.B) {
			var s GridState
			if tc.every > 0 {
				s.Header = headerEvery(tc.every)
			}
			tt := coreNewTester(func(c *context) {
				coreGridView(c, &s, 1_000_000, 100, 80, func(i int) {
					coreTextf(c, "Item %d", i)
				}).Grow(1)
			}, 980, 720)
			tt.Move(400, 300)
			b.ReportAllocs()
			for b.Loop() {
				tt.Scroll(400, 300, 0, 97)
			}
		})
	}
}
