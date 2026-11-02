package ui

import (
	"fmt"
	"testing"
)

// BenchmarkWideTableIdle is the frame of a table of 12 columns whose rows are
// kept (RowVersion), beside what an app changes in every frame: a label set
// from another goroutine (Post) and a progress bar. The variants:
//   - idle: nothing changes;
//   - label: the label changes in every frame;
//   - label-hover: the label changes and the pointer moves over it;
//   - hover-table: the pointer moves from a row to the next.
func BenchmarkWideTableIdle(b *testing.B) {
	for _, v := range []string{"idle", "label", "label-hover", "hover-table"} {
		b.Run(v, func(b *testing.B) {
			const rows, cols = 400, 12
			var list ListState
			list.Key = func(i int) any { return i }
			list.RowVersion = func(int) uint64 { return 1 }
			columns := make([]TableColumn, cols)
			for i := range columns {
				columns[i] = TableColumn{Title: fmt.Sprint("C", i), Width: 70}
			}
			n := 0
			tt := NewTester(func(c *Context) {
				Column(c).Fill().Children(func() {
					Textf(c, "label %d", n)
					Progress(c, float64(n%100)/100)
					Table(c, &list, columns, rows, func(r, col int) {
						Textf(c, "r%d c%d", r, col)
					}).Grow(1)
				})
			}, 900, 600)
			tt.SkipRaster(true)
			for range 10 {
				tt.Frame()
			}
			b.ReportAllocs()
			b.ResetTimer()
			i := 0
			for b.Loop() {
				switch v {
				case "label":
					n++
				case "label-hover":
					n++
					tt.Move(20+float32(i%2), 12)
				case "hover-table":
					tt.Move(100, 120+float32(i%2)*40)
				}
				tt.Frame()
				i++
			}
		})
	}
}
