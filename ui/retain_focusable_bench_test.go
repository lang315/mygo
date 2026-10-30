package ui

import (
	"fmt"
	"testing"
)

// BenchmarkListRowsOfControls: a list of rows with a check box, a button and a
// text input (widgets) or a text (text), idle and with the pointer moving.
// Rows with a control that takes the focus were never kept; the figures to read
// are ns/op and built/frame.
func BenchmarkListRowsOfControls(b *testing.B) {
	for _, n := range []int{1000, 100_000} {
		for _, kind := range []string{"text", "widgets", "button-checkbox-link", "radio"} {
			for _, phase := range []string{"idle", "hover"} {
				b.Run(fmt.Sprintf("%d/%s/%s", n, kind, phase), func(b *testing.B) {
					bs := make([]bool, n)
					ss := make([]string, n)
					sel := make([]int, n)
					s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
					tt := NewTester(func(c *Context) {
						Column(c).Fill().Children(func() {
							List(c, s, n, func(i int) {
								Row(c).Padding(4).Gap(8).Children(func() {
									switch kind {
									case "text":
										Textf(c, "row %d", i)
									case "widgets":
										Checkbox(c, &bs[i], "ok")
										Button(c, "go")
										TextInput(c, &ss[i]).Width(120)
									case "radio":
										Radio(c, &sel[i], 1, "a")
										Radio(c, &sel[i], 2, "b")
									default:
										Checkbox(c, &bs[i], "ok")
										Button(c, "go")
										Link(c, "link", "https://example.com")
									}
								})
							}).Grow(1)
						})
					}, 900, 800)
					for range 6 {
						tt.Frame()
					}
					built, kept := tt.rt.rowsBuilt, tt.rt.rowsKept
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; b.Loop(); i++ {
						if phase == "hover" {
							tt.Move(100+float32(i%2), 60)
						}
						tt.Frame()
					}
					b.ReportMetric(float64(tt.rt.rowsBuilt-built)/float64(b.N), "built/frame")
					b.ReportMetric(float64(tt.rt.rowsKept-kept)/float64(b.N), "kept/frame")
				})
			}
		}
	}
}
