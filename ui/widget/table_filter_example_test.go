package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of the column filters in docs/ui/widget.md, which must compile.
func ExampleDataGridViewWidget_SetColumnFilter() {
	type Doc struct {
		ID         int
		Name, Kind string
	}
	key := func(d *Doc) int { return d.ID }
	name := widget.Column[Doc]{Title: "Name", Text: func(d *Doc) string { return d.Name }, Sortable: true, Filterable: true}
	kind := widget.Column[Doc]{Title: "Kind", Text: func(d *Doc) string { return d.Kind }, Filterable: true}
	save := func(f widget.ColumnFilter) { fmt.Println(f.Values) }

	var t *widget.DataGridViewWidget[Doc, int]
	t = widget.DataGridView(key, name, kind).
		SetFilterDropDowns(true). // a funnel in each Filterable header
		SetAutoFilterRow(true).   // a field under each Filterable title
		SetFilterPanel(true).     // the filters as text under the table, while any is set
		SetColumnFilter("Kind", widget.ColumnFilter{Values: []string{"Doc", "Sheet"}}).
		OnFilterChange(func() { save(t.ColumnFilter("Kind")) }) // the user's changes only
}
