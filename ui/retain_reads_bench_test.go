package ui

import "testing"

// BenchmarkListRowsReadingTheLayout: a list of rows that ask InView and Bounds
// of their own element, idle. With the answers recorded and compared the rows
// are kept once the layout settles; built in every frame they are not.
func BenchmarkListRowsReadingTheLayout(b *testing.B) {
	for _, name := range []string{"InView", "Bounds", "none"} {
		b.Run(name, func(b *testing.B) {
			s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
			tt := NewTester(func(c *Context) {
				Column(c).Fill().Children(func() {
					List(c, s, 100_000, func(i int) {
						r := Row(c).Padding(4)
						r.Children(func() {
							switch name {
							case "InView":
								Textf(c, "row %d %v", i, r.InView())
							case "Bounds":
								Textf(c, "row %d %v", i, r.Bounds().W)
							default:
								Textf(c, "row %d", i)
							}
						})
					}).Grow(1)
				})
			}, 600, 800)
			for range 6 {
				tt.Frame()
			}
			built := tt.rt.rowsBuilt
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.Frame()
			}
			b.ReportMetric(float64(tt.rt.rowsBuilt-built)/float64(b.N), "rows-built/frame")
		})
	}
}
