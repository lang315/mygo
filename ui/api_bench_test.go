package ui_test

import (
	"cmp"
	"fmt"
	"runtime"
	"testing"
	"time"

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

// tableRowVersions, when set (api_bench_rowversion_test.go), gives the table
// of tableFrame100k a RowVersion while tableVersioned is, and returns the
// function that changes the version of a row; tableBump is that function for
// the table last built. The sub-benchmarks of BenchmarkTableFrame100kPhases
// named -rv run so.
var (
	tableRowVersions func(list *ui.ListState, rows []tableAccount, view *ui.TableView[tableAccount]) (bump func(r int))
	tableVersioned   bool
	tableBump        func(r int)
)

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
	tableBump = nil
	if tableVersioned && tableRowVersions != nil {
		tableBump = tableRowVersions(&list, rows, view)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		view.Sync(rows, sort)
		list.Sort, list.Checked, list.Version = &sort, &checked, view.Version()
		// What an app returns: the id boxed in every call. An integer of 256 or
		// more allocates, so the frames far from the top make about 25
		// allocations more than the frames at the top (the ids there are small):
		// that is the app's Key, not the cost of the library.
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

// tableFrameVariant is a frame users feel on the table of
// BenchmarkTableFrame100k: setup makes what the frames need, and step does
// the i-th frame's input and runs it.
type tableFrameVariant struct {
	name  string
	setup func(tt *ui.Tester, rows []tableAccount, view *ui.TableView[tableAccount]) (step func(i int))
}

// tableFrameVariants are the frames:
//   - idle: nothing changed;
//   - idle-scrolled: nothing changed, at an offset away from the top with the
//     pointer outside the window (an app idles at any offset);
//   - hover: the pointer moved from a row to the next;
//   - update: a visible row's text changed;
//   - scroll1px: a scroll by a whole pixel, down and up in turn;
//   - scrollhalf: a scroll by half a pixel, down twice and up twice. A
//     Table's offset snaps to device pixels, so half a pixel moves its
//     rows on every second frame only.
var tableFrameVariants = []tableFrameVariant{
	{"idle", func(tt *ui.Tester, _ []tableAccount, _ *ui.TableView[tableAccount]) func(int) {
		return func(int) { tt.Frame() }
	}},
	{"idle-scrolled", func(tt *ui.Tester, _ []tableAccount, _ *ui.TableView[tableAccount]) func(int) {
		tt.Scroll(150, 300, 0, 2000)
		tt.Move(-50, -50) // no row is hovered
		tt.Frame()
		tt.Frame()
		return func(int) { tt.Frame() }
	}},
	{"hover", func(tt *ui.Tester, _ []tableAccount, _ *ui.TableView[tableAccount]) func(int) {
		ys := [2]float32{120, 160} // two rows below the header
		return func(i int) { tt.Move(150, ys[i%2]) }
	}},
	{"update", func(tt *ui.Tester, rows []tableAccount, view *ui.TableView[tableAccount]) func(int) {
		r := view.Row(5) // a row in view
		names := [2]string{rows[r].name, rows[r].name + "x"}
		return func(i int) {
			rows[r].name = names[i%2]
			if tableBump != nil {
				tableBump(r)
			}
			tt.Frame()
		}
	}},
	{"scroll1px", func(tt *ui.Tester, _ []tableAccount, _ *ui.TableView[tableAccount]) func(int) {
		tt.Scroll(150, 300, 0, 2000) // away from the top, so that both ways scroll
		dys := [2]float32{1, -1}
		return func(i int) { tt.Scroll(150, 300, 0, dys[i%2]) }
	}},
	{"scrollhalf", func(tt *ui.Tester, _ []tableAccount, _ *ui.TableView[tableAccount]) func(int) {
		tt.Scroll(150, 300, 0, 2000)
		dys := [4]float32{0.5, 0.5, -0.5, -0.5}
		return func(i int) { tt.Scroll(150, 300, 0, dys[i%4]) }
	}},
}

// BenchmarkTableFrame100kFrames measures, on the table of
// BenchmarkTableFrame100k, the frames users feel (tableFrameVariants).
// The Tester draws each frame with the CPU rasterizer that stands in for the
// GPU, which is most of a frame that moves every row:
// BenchmarkTableFrame100kPhases tells the parts of a frame.
func BenchmarkTableFrame100kFrames(b *testing.B) {
	for _, v := range tableFrameVariants {
		b.Run(v.name, func(b *testing.B) {
			tt, rows, view := tableFrame100k(b)
			step := v.setup(tt, rows, view)
			b.ReportAllocs()
			b.ResetTimer()
			i := 0
			for b.Loop() {
				step(i)
				i++
			}
		})
	}
}

// BenchmarkTableFrame100kPhases runs the frames of
// BenchmarkTableFrame100kFrames and reports how long each part of a frame
// took, averaged over the frames: build-µs/frame, layout-µs/frame,
// paint-µs/frame and present-µs/frame. The variants ending in -app skip the
// harness's rasterizer (Tester.SkipRaster), so that they measure the work an
// app does itself: their present is empty. Reading the parts costs time of
// its own (about 31 B and a few hundred ns a frame): compare these frames
// with each other, and not with BenchmarkTableFrame100kFrames. The variants
// ending in -rv (-rv-app) run with a RowVersion on the table, which keeps the
// rows that did not change; the "update" frame changes the version of its row.
func BenchmarkTableFrame100kPhases(b *testing.B) {
	versions := []bool{false}
	if tableRowVersions != nil {
		versions = append(versions, true)
	}
	defer func() { tableVersioned = false }()
	for _, versioned := range versions {
		for _, skip := range []bool{false, true} {
			for _, v := range tableFrameVariants {
				name := v.name
				if versioned {
					name += "-rv"
				}
				if skip {
					name += "-app"
				}
				b.Run(name, func(b *testing.B) {
					tableVersioned = versioned
					tt, rows, view := tableFrame100k(b)
					step := v.setup(tt, rows, view)
					tt.SkipRaster(skip)
					ui.MeasureFrames(tt)
					var laps [4]time.Duration
					b.ReportAllocs()
					b.ResetTimer()
					i := 0
					for b.Loop() {
						step(i)
						l := ui.FrameLaps(tt)
						for p := range laps {
							laps[p] += l[p]
						}
						i++
					}
					b.StopTimer()
					for p, part := range [4]string{"build", "layout", "paint", "present"} {
						b.ReportMetric(float64(laps[p].Nanoseconds())/float64(i)/1000, part+"-µs/frame")
					}
				})
			}
		}
	}
}
