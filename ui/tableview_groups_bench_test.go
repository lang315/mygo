package ui

import (
	"strconv"
	"testing"
)

// groupBenchSizes are how many groups the benchmarks of groups make: none is
// the view without SetGroupBy.
var groupBenchSizes = []int{0, 10, 1000, 100_000}

// groupBenchView makes a view of a million rows sorted by column a, in groups
// groups when groups > 0. The titles are made once, so the benchmarks measure
// the grouping, not the titles' allocation.
func groupBenchView(groups int) (*TableView[tvRow], []tvRow) {
	const n = 1_000_000
	rows := make([]tvRow, n)
	for i := range rows {
		rows[i] = tvRow{a: i * 7919 % n, b: i}
	}
	v := NewTableView(tvCompare)
	if groups > 0 {
		titles := make([]string, groups)
		for i := range titles {
			titles[i] = "group " + strconv.Itoa(i)
		}
		v.SetGroupBy(func(x *tvRow) string { return titles[x.b%groups] })
	}
	return v, rows
}

// BenchmarkTableViewGroups1M places a million rows again (a new sort, Reset),
// and syncs after one and a thousand rows changed, and with none, in a view
// of 0 (no SetGroupBy), 10, 1000 and 100,000 groups. A row changed moves to
// another place, and, in a grouped view, to the next group.
func BenchmarkTableViewGroups1M(b *testing.B) {
	by := SortOrder{Column: "a"}
	for _, g := range groupBenchSizes {
		name := "groups=" + strconv.Itoa(g)
		b.Run(name+"/rebuild", func(b *testing.B) {
			v, rows := groupBenchView(g)
			v.Sync(rows, by)
			b.ReportAllocs()
			for b.Loop() {
				v.Reset()
				v.Sync(rows, by)
			}
		})
		for _, k := range []int{1, 1000} {
			b.Run(name+"/changed="+strconv.Itoa(k), func(b *testing.B) {
				v, rows := groupBenchView(g)
				v.Sync(rows, by)
				seq := 0
				b.ReportAllocs()
				for b.Loop() {
					for j := range k {
						i := (seq*31 + j*977) % len(rows)
						rows[i].a = (rows[i].a*13 + 7) % len(rows)
						rows[i].b++
						v.Changed(i)
					}
					seq++
					v.Sync(rows, by)
				}
			})
		}
		b.Run(name+"/idle", func(b *testing.B) {
			v, rows := groupBenchView(g)
			v.Sync(rows, by)
			b.ReportAllocs()
			for b.Loop() {
				v.Sync(rows, by)
			}
		})
	}
}
