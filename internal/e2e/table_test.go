//go:build !js

package e2e

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	tb := widget.Table(func(r *row) int { return r.id },
		widget.Column[row]{Title: "Name", Width: 200, Sortable: true, Filterable: true,
			Text: func(r *row) string { built.Add(1); return fmt.Sprintf("row%05d", r.id) }},
		widget.Column[row]{Title: "N", Align: widget.End, Sortable: true,
			Text:    func(r *row) string { return strconv.Itoa(r.n) },
			Compare: func(a, b *row) int { return cmp.Compare(a.n, b.n) }},
	).SetRows(rows).CheckboxColumn().BottomPanel()
	w := widget.NewWindow("Table", 500, 400).SetContent(tb)
	w.Show()
	t.Cleanup(func() {
		// Close asks; wait until the window is gone, as Destroy would, so
		// that it takes no input from the tests after this one.
		w.Close()
		eventually(t, "the table's window to close", func() bool { return w.Native() == nil })
	})
	eventually(t, "the table's first frame", func() bool { return built.Load() > 0 })

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			k := 1 + i%9_999 // never row 0, which the click below selects
			tb.UpdateRow(k, func(r *row) { r.n = -r.n })
			tb.DeleteRows(k)
			tb.AddRow(row{k, i})
		}
	}()
	for i := range 30 {
		w.SetSize(500, 400+i%2*500)
		time.Sleep(30 * time.Millisecond)
	}
	before := built.Load()
	eventually(t, "frames while the rows change", func() bool { return built.Load() > before })
	close(stop)
	wg.Wait()
	if n := tb.Len(); n != 10_000 {
		t.Fatalf("Len %d after deleting and adding back each row", n)
	}

	// Row 0 stays first: the rows deleted come back last. Under the
	// header, 32 DIPs tall, at the left of the Name column.
	w.SetSize(500, 400)
	time.Sleep(200 * time.Millisecond)
	if !click(w.Native(), 120, 48) {
		t.Skip("click automation not available on this platform")
	}
	eventually(t, "the click to select row 0", func() bool { return slices.Equal(tb.Selected(), []int{0}) })
}
