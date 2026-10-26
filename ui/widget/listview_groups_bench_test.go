package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkListViewGroups100k and BenchmarkListViewGroups1M measure the icon
// view of 100,000 and 1,000,000 rows without groups and with them (10 and 1000
// groups): a frame at rest, a scrolled one, and a frame after a row changed.
func BenchmarkListViewGroups100k(b *testing.B) { benchListViewGroups(b, 100_000) }
func BenchmarkListViewGroups1M(b *testing.B)   { benchListViewGroups(b, 1_000_000) }

func benchListViewGroups(b *testing.B, n int) {
	rows := make([]vitem, n)
	for i := range rows {
		rows[i] = vitem{id: strconv.Itoa(i), name: "Row " + strconv.Itoa(i), kind: "k" + strconv.Itoa(i%1000)}
	}
	for _, tc := range []struct {
		name   string
		groups int // 0: none
	}{{"none", 0}, {"10-groups", 10}, {"1000-groups", 1000}} {
		setup := func(b *testing.B) (*ListViewWidget[vitem, string], *ui.Tester) {
			lv := newView().SetRows(rows).SetView(ViewLargeIcon)
			if tc.groups > 0 {
				// The titles are made once: the benchmark measures the
				// grouping, not the app's function.
				titles := make([]string, tc.groups)
				for g := range titles {
					titles[g] = "g" + strconv.Itoa(g)
				}
				lv.SetGroupBy(func(v *vitem) string {
					i, _ := strconv.Atoi(v.id)
					return titles[i%tc.groups]
				})
			}
			tt := ui.NewTester(content(lv), 1280, 800)
			for range 3 {
				tt.Frame()
			}
			return lv, tt
		}
		b.Run(tc.name+"/rest", func(b *testing.B) {
			_, tt := setup(b)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.Frame()
			}
		})
		b.Run(tc.name+"/scroll", func(b *testing.B) {
			_, tt := setup(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				dy := float32(120)
				if i%2 == 1 {
					dy = -120
				}
				tt.Wheel(400, 300, 0, dy)
				tt.Frame()
			}
		})
		b.Run(tc.name+"/row-changed", func(b *testing.B) {
			lv, tt := setup(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				lv.UpdateRow(strconv.Itoa(i%n), func(v *vitem) { v.name += "." })
				tt.Frame()
			}
		})
	}
}
