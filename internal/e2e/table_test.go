//go:build !js

package e2e

import (
	"cmp"
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

// TestWidgetTable runs ui/widget's table in a window: a goroutine changes,
// deletes and adds rows while the window grows and shrinks, which has ui
// build rows while it lays frames out, without the widget lock; then a
// click selects the row under it, and the rows add up.
func TestWidgetTable(t *testing.T) {
	type row struct{ id, n int }
	rows := make([]row, 10_000)
	for i := range rows {
		rows[i] = row{i, i}
	}
	var built atomic.Int64 // cells built, which frames do
	tb := widget.DataGridView(func(r *row) int { return r.id },
		widget.Column[row]{Title: "Name", Width: 200, Sortable: true, Filterable: true,
			Text: func(r *row) string { built.Add(1); return fmt.Sprintf("row%05d", r.id) }},
		widget.Column[row]{Title: "N", Align: widget.End, Sortable: true,
			Text:    func(r *row) string { return strconv.Itoa(r.n) },
			Compare: func(a, b *row) int { return cmp.Compare(a.n, b.n) }},
	).SetRows(rows).CheckboxColumn().BottomPanel()
	w := widget.NewForm("Table", 500, 240).SetContent(tb)
	w.Show()
	eventually(t, "the table's first frame", func() bool { return built.Load() > 0 })
	t.Cleanup(w.Native().Destroy)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			k := 1 + i%9_999                             // never row 0, which the click below selects
			tb.UpdateRow(k, func(r *row) { r.n = -r.n }) // a write in place, as well
			tb.DeleteRows(k)
			tb.AddRow(row{k, i})
			runtime.Gosched() // let the frames take the table's lock between rounds
		}
	})
	// Heights inside Xvfb's 640x480 screen: past it, then clicked, a window
	// broke the input method of the next test's window under Xvfb, which
	// has no window manager. It ends 240 tall.
	for i := range 30 {
		_, was := w.Native().Size()
		w.SetSize(500, 240+(i+1)%2*180)
		eventually(t, "the window to resize", func() bool { _, h := w.Native().Size(); return h != was })
	}
	before := built.Load()
	eventually(t, "frames while the rows change", func() bool { return built.Load() > before })
	close(stop)
	wg.Wait()
	if n := tb.Len(); n != 10_000 {
		t.Fatalf("Len %d after deleting and adding back each row", n)
	}

	// Row 0 stays first: the rows deleted come back last. Under the
	// header, 32 DIPs tall, in the Name column, past the check boxes.
	if !click(w.Native(), 120, 48) {
		t.Skip("click automation not available on this platform")
	}
	eventually(t, "the click to select row 0", func() bool { return slices.Equal(tb.Selected(), []int{0}) })
}
