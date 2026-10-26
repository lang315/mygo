package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkListViewDetailsCheckAll1M clicks the header's check-all of a million
// rows, checking them and then unchecking them, without groups and with ten
// (which have headers among the lines).
func BenchmarkListViewDetailsCheckAll1M(b *testing.B) {
	const n = 1_000_000
	for _, groups := range []int{0, 10} {
		b.Run("groups="+strconv.Itoa(groups), func(b *testing.B) {
			titles := make([]string, max(groups, 1))
			for g := range titles {
				titles[g] = "g" + strconv.Itoa(g)
			}
			rows := make([]vitem, n)
			for i := range rows {
				rows[i] = vitem{id: strconv.Itoa(i), name: "Row " + strconv.Itoa(i), kind: titles[i%len(titles)]}
			}
			lv := newView().SetRows(rows).SetView(ViewDetails).CheckBoxes()
			if groups > 0 {
				lv.SetGroupBy(kindOf)
			}
			tt := ui.NewTester(content(lv), 1280, 800)
			for range 3 {
				tt.Frame()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := tt.Click("Check all"); err != nil {
					b.Fatal(err)
				}
				tt.Frame()
			}
			if got := lv.Checked(); len(got) != 0 && len(got) != n {
				b.Fatalf("%d rows checked", len(got))
			}
		})
	}
}

// BenchmarkListViewDetailsGroups1M measures Details of a million rows without
// groups and with them (10 and 1000 groups): a frame at rest and a scrolled
// one, a frame after a row changed in place (its text), and after a row moved
// to the next group.
func BenchmarkListViewDetailsGroups1M(b *testing.B) {
	const n = 1_000_000
	for _, tc := range []struct {
		name   string
		groups int // 0: none
	}{{"none", 0}, {"10-groups", 10}, {"1000-groups", 1000}} {
		setup := func(b *testing.B) (*ListViewWidget[vitem, string], *ui.Tester) {
			// The titles are made once: the benchmark measures the grouping,
			// not the app's function.
			titles := make([]string, max(tc.groups, 1))
			for g := range titles {
				titles[g] = "g" + strconv.Itoa(g)
			}
			rows := make([]vitem, n)
			for i := range rows {
				rows[i] = vitem{id: strconv.Itoa(i), name: "Row " + strconv.Itoa(i), kind: titles[i%len(titles)]}
			}
			lv := newView().SetRows(rows).SetView(ViewDetails)
			if tc.groups > 0 {
				lv.SetGroupBy(kindOf)
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
		if tc.groups > 0 {
			b.Run(tc.name+"/row-moved", func(b *testing.B) {
				lv, tt := setup(b)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; b.Loop(); i++ {
					lv.UpdateRow(strconv.Itoa(i%n), func(v *vitem) {
						if v.kind == "g0" {
							v.kind = "g1"
						} else {
							v.kind = "g0"
						}
					})
					tt.Frame()
				}
			})
		}
	}
}
