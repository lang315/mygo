package ui_test

import (
	"cmp"
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
)

func BenchmarkValueFrame(b *testing.B) {
	labels := make([]string, 1000)
	for i := range labels {
		labels[i] = fmt.Sprintf("Item %d", i)
	}
	tt := ui.NewTester(func(f *ui.Context) {
		ui.Column(f).Fill().Children(func() {
			for _, label := range labels {
				ui.Text(f, label).FontSize(12).Padding(1, 2)
			}
		})
	}, 300, 600)
	for range 10 {
		tt.Frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tt.Frame()
	}
}

func TestValueFrameRetainedHeap(t *testing.T) {
	labels := make([]string, 1000)
	for i := range labels {
		labels[i] = fmt.Sprintf("Item %d", i)
	}
	view := func(f *ui.Context) {
		ui.Column(f).Fill().Children(func() {
			for _, label := range labels {
				ui.Text(f, label).FontSize(12).Padding(1, 2)
			}
		})
	}
	warm := ui.NewTester(view, 300, 600)
	for range 10 {
		warm.Frame()
	}
	runtime.KeepAlive(warm)
	warm = nil
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	tt := ui.NewTester(view, 300, 600)
	for range 10 {
		tt.Frame()
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("retained Go heap for the warmed 1000-label view: %d bytes", int64(after.HeapAlloc)-int64(before.HeapAlloc))
	runtime.KeepAlive(tt)
}

func BenchmarkValueList(b *testing.B) {
	var list ui.ListState
	tt := ui.NewTester(func(f *ui.Context) {
		ui.List(f.Key("items"), &list, 1_000_000).Grow(1).
			Rows(func(row ui.ListRow) { ui.Textf(row.Context, "Row %d", row.Index).Height(24) })
	}, 300, 400)
	for range 10 {
		tt.Frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tt.Frame()
	}
}

// tableAccount is a row of the tables of BenchmarkTableFrame100k.
type tableAccount struct {
	id, status int
	name       string
}

// tableFrame100k builds the table of BenchmarkTableFrame100k, sorted by a
// TableView, with checkboxes, and warms it with 10 frames.
func tableFrame100k(b *testing.B) (*ui.Tester, []tableAccount, *ui.TableView[tableAccount]) {
	b.Helper()
	rows := make([]tableAccount, 100_000)
	for i := range rows {
		rows[i] = tableAccount{id: i, status: i % 7, name: fmt.Sprintf("acc%06d", i)}
	}
	view := ui.NewTableView(map[string]func(a, b *tableAccount) int{
		"Status": func(a, b *tableAccount) int { return cmp.Compare(a.status, b.status) },
	})
	var list ui.ListState
	var checked ui.Selection[int]
	sort := ui.SortOrder{Column: "Status"}
	cols := []ui.TableColumn{{Title: "Name"}, {Title: "Status", Width: 80, Sortable: true}}
	tt := ui.NewTester(func(c *ui.Context) {
		view.Sync(rows, sort)
		list.Sort, list.Checked, list.Version = &sort, &checked, view.Version()
		list.Key = func(i int) any { return rows[view.Row(i)].id }
		ui.Table(c, &list, cols, view.Len(), func(r, col int) {
			a := &rows[view.Row(r)]
			if col == 0 {
				ui.Text(c, a.name)
			} else {
				ui.Textf(c, "%d", a.status)
			}
		}).Grow(1)
	}, 400, 600)
	for range 10 {
		tt.Frame()
	}
	return tt, rows, view
}

// BenchmarkTableFrame100k builds an idle frame of a table of 100,000 rows
// sorted by a TableView, with checkboxes.
func BenchmarkTableFrame100k(b *testing.B) {
	tt, _, _ := tableFrame100k(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tt.Frame()
	}
}

// BenchmarkTableFrame100kFrames measures, on the table of
// BenchmarkTableFrame100k, the frames users feel:
//   - idle: nothing changed;
//   - hover: the pointer moved from a row to the next;
//   - update: a visible row's text changed;
//   - scroll1px and scrollhalf: a scroll by a whole pixel or by half of one,
//     down and up in turn.
func BenchmarkTableFrame100kFrames(b *testing.B) {
	b.Run("idle", func(b *testing.B) {
		tt, _, _ := tableFrame100k(b)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			tt.Frame()
		}
	})
	b.Run("hover", func(b *testing.B) {
		tt, _, _ := tableFrame100k(b)
		ys := [2]float32{120, 160} // two rows below the header
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			tt.Move(150, ys[i%2])
			i++
		}
	})
	b.Run("update", func(b *testing.B) {
		tt, rows, view := tableFrame100k(b)
		r := view.Row(5) // a row in view
		names := [2]string{rows[r].name, rows[r].name + "x"}
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			rows[r].name = names[i%2]
			tt.Frame()
			i++
		}
	})
	for _, s := range []struct {
		name string
		dy   float32
	}{{"scroll1px", 1}, {"scrollhalf", 0.5}} {
		b.Run(s.name, func(b *testing.B) {
			tt, _, _ := tableFrame100k(b)
			tt.Scroll(150, 300, 0, 2000) // away from the top, so that both ways scroll
			b.ReportAllocs()
			b.ResetTimer()
			i := 0
			for b.Loop() {
				dy := s.dy
				if i%2 == 1 {
					dy = -dy
				}
				tt.Scroll(150, 300, 0, dy)
				i++
			}
		})
	}
}
