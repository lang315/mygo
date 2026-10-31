package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkListViewRows measures a frame of a list view of 1000 and of 100,000
// rows in each view, at rest and with the pointer moving over the rows: the
// rows an icon view kept are not built.
func BenchmarkListViewRows(b *testing.B) {
	for _, n := range []int{1000, 100_000} {
		for _, m := range allModes {
			for _, phase := range []string{"idle", "hover"} {
				b.Run(fmt.Sprintf("%d/%s/%s", n, m.name, phase), func(b *testing.B) {
					r := newRowsView(n, m.mode)
					tt := ui.NewTester(content(r.lv), 1280, 800)
					for range 4 {
						tt.Frame()
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; b.Loop(); i++ {
						if phase == "hover" {
							tt.Move(200+float32(i%2), 200)
						}
						tt.Frame()
					}
				})
			}
		}
	}
}
