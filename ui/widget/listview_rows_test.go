package widget

import (
	"bytes"
	"fmt"
	"image/color"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

// rowsView is a list view of n rows whose first column counts the calls of its
// Text: a row kept is not built, so it asks nothing.
type rowsView struct {
	lv    *ListViewWidget[vitem, string]
	calls atomic.Int64
}

func newRowsView(n int, mode ViewMode) *rowsView {
	rows := make([]vitem, n)
	for i := range rows {
		rows[i] = vitem{id: strconv.Itoa(i), name: "Row " + strconv.Itoa(i), kind: "k" + strconv.Itoa(i%7)}
	}
	r := &rowsView{}
	r.lv = ListView(func(v *vitem) string { return v.id },
		Column[vitem]{Title: "Name", Text: func(v *vitem) string { r.calls.Add(1); return v.name }, Width: 200},
		Column[vitem]{Title: "Kind", Text: func(v *vitem) string { return v.kind }, Width: 100}).SetRows(rows).SetView(mode)
	return r
}

func (r *rowsView) tester() *ui.Tester {
	tt := ui.NewTester(content(r.lv), 800, 500)
	for range 4 {
		tt.Frame()
	}
	return tt
}

// idleCalls returns how many times the first column was asked for in a frame at
// rest.
func (r *rowsView) idleCalls(tt *ui.Tester) float64 {
	before := r.calls.Load()
	for range 5 {
		tt.Frame()
	}
	return float64(r.calls.Load()-before) / 5
}

// same fails when the picture of tt is not the picture a list view built afresh
// in the same state draws: what a kept row shows is what a built row does.
func sameAsFresh(t *testing.T, what string, tt *ui.Tester, n int, mode ViewMode, setup func(lv *ListViewWidget[vitem, string])) {
	t.Helper()
	f := newRowsView(n, mode)
	setup(f.lv)
	ft := f.tester()
	tt.Frame()
	tt.Frame()
	if a, b := tt.Image(), ft.Image(); !bytes.Equal(a.Pix, b.Pix) {
		t.Fatalf("%s: the list view kept does not draw what one built afresh does", what)
	}
}

// The rows of every view are kept at rest: nothing asks the app for the text of
// a row it showed.
func TestListViewKeepsItsRowsInEveryView(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			r := newRowsView(2000, m.mode)
			tt := r.tester()
			if got := r.idleCalls(tt); got != 0 {
				t.Errorf("%s: %.1f calls of the first column in a frame at rest, want none", m.name, got)
			}
		})
	}
}

// What a row shows follows the widget: the text of a row changed in place, a
// row added, the view, the checks, the choice, the icon size and the groups.
func TestListViewKeptRowsShowWhatChanged(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			const n = 200
			r := newRowsView(n, m.mode)
			r.lv.CheckBoxes()
			tt := r.tester()
			var steps []func(lv *ListViewWidget[vitem, string])
			do := func(what string, f func(lv *ListViewWidget[vitem, string])) {
				t.Helper()
				steps = append(steps, f)
				f(r.lv)
				tt.Frame()
				tt.Frame()
				sameAsFresh(t, what, tt, n, m.mode, func(lv *ListViewWidget[vitem, string]) {
					lv.CheckBoxes()
					for _, s := range steps {
						s(lv)
					}
				})
			}
			do("updated", func(lv *ListViewWidget[vitem, string]) {
				lv.UpdateRow("3", func(v *vitem) { v.name = "Renamed" })
			})
			if !tt.HasText("Renamed") {
				t.Errorf("%s: the text of a row changed in place is not shown", m.name)
			}
			do("checked", func(lv *ListViewWidget[vitem, string]) { lv.SetChecked([]string{"1", "4"}, true) })
			do("chosen", func(lv *ListViewWidget[vitem, string]) { lv.SetSelected("2", "5") })
			do("unchecked", func(lv *ListViewWidget[vitem, string]) { lv.SetChecked([]string{"1"}, false) })
			do("icon size", func(lv *ListViewWidget[vitem, string]) { lv.SetIconSize(72) })
			il := ImageList(16)
			if err := il.Add("doc", pngOf(t, 16, color.RGBA{0, 0, 255, 255})); err != nil {
				t.Fatal(err)
			}
			do("an image", func(lv *ListViewWidget[vitem, string]) {
				lv.SetImage(func(*vitem) *ui.Bitmap { return il.Image("doc") })
			})
			do("grouped", func(lv *ListViewWidget[vitem, string]) {
				lv.SetGroupBy(func(v *vitem) string { return v.kind })
			})
			do("a row moved to another group", func(lv *ListViewWidget[vitem, string]) {
				lv.UpdateRow("6", func(v *vitem) { v.kind = "k0" })
			})
			do("a row added", func(lv *ListViewWidget[vitem, string]) {
				lv.AddRow(vitem{id: "new", name: "New one", kind: "k1"})
			})
			do("the view switched", func(lv *ListViewWidget[vitem, string]) {
				if m.mode == ViewList {
					lv.SetView(ViewLargeIcon)
				} else {
					lv.SetView(ViewList)
				}
			})
			do("groups off", func(lv *ListViewWidget[vitem, string]) { lv.SetGroupBy(nil) })
			if got := r.idleCalls(tt); got != 0 {
				t.Errorf("%s: %.1f calls in a frame at rest after the changes", m.name, got)
			}
		})
	}
}

// A row changed in place by the app's own slice is not seen: the widget holds
// a copy of the rows (SetRows), and says what changed through UpdateRow, as a
// DataGridView does.
func TestListViewRowsAreChangedByTheWidget(t *testing.T) {
	rows := []vitem{{"a", "Apple", "pome"}, {"b", "Banana", "berry"}}
	lv := ListView(func(v *vitem) string { return v.id },
		Column[vitem]{Title: "Name", Text: func(v *vitem) string { return v.name }, Width: 200}).SetRows(rows).SetView(ViewLargeIcon)
	tt := ui.NewTester(content(lv), 400, 300)
	for range 3 {
		tt.Frame()
	}
	rows[0].name = "Mutated"
	for range 3 {
		tt.Frame()
	}
	if tt.HasText("Mutated") || !tt.HasText("Apple") {
		t.Errorf("a change of the app's own slice reached the view: %q", fmt.Sprint(tt.Texts()))
	}
}
