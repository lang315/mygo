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

// BenchmarkTableFrame100k builds an idle frame of a table of 100,000 rows
// sorted by a TableView, with checkboxes.
func BenchmarkTableFrame100k(b *testing.B) {
	type account struct {
		id, status int
		name       string
	}
	rows := make([]account, 100_000)
	for i := range rows {
		rows[i] = account{id: i, status: i % 7, name: fmt.Sprintf("acc%06d", i)}
	}
	view := ui.NewTableView(map[string]func(a, b *account) int{
		"Status": func(a, b *account) int { return cmp.Compare(a.status, b.status) },
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
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tt.Frame()
	}
}
