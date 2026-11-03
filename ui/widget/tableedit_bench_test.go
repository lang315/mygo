package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkTableEdit measures a table of 100k rows with an editable column:
// idle as it is (it must cost what a table with nothing to edit costs), idle
// with a cell being edited, and a character typed in the editor.
func BenchmarkTableEdit(b *testing.B) {
	const n = 100_000
	setup := func(b *testing.B, editable bool) (*DataGridViewWidget[acct, int], *ui.Tester) {
		cols := []Column[acct]{
			{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true, Filterable: true},
			{Title: "Score", Text: func(a *acct) string { return strconv.Itoa(a.score) }, Width: 100},
		}
		if editable {
			cols[0] = cols[0].Editable(CellEditor[acct]{Set: func(a *acct, s string) { a.name = s }})
		}
		tb := DataGridView(acctKey, cols...).SetRows(manyAccts(n)).Expand()
		tt := ui.NewTester(content(tb), 1280, 800)
		for range 3 {
			tt.Frame()
		}
		return tb, tt
	}
	b.Run("idlePlain", func(b *testing.B) {
		_, tt := setup(b, false)
		b.ReportAllocs()
		for b.Loop() {
			tt.Frame()
		}
	})
	b.Run("idleEditable", func(b *testing.B) {
		_, tt := setup(b, true)
		b.ReportAllocs()
		for b.Loop() {
			tt.Frame()
		}
	})
	b.Run("idleEditing", func(b *testing.B) {
		tb, tt := setup(b, true)
		tb.BeginEdit(0, "Name")
		for range 3 {
			tt.Frame()
		}
		b.ReportAllocs()
		for b.Loop() {
			tt.Frame()
		}
	})
	b.Run("typing", func(b *testing.B) {
		tb, tt := setup(b, true)
		tb.BeginEdit(0, "Name")
		for range 3 {
			tt.Frame()
		}
		b.ReportAllocs()
		for b.Loop() {
			tt.Type("a")
			tt.Frame()
		}
	})
}
