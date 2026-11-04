package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of OnSelectionChanged in docs/ui/widget.md, which must compile.
func ExampleDataGridViewWidget_OnSelectionChanged() {
	type Account struct {
		ID   uint
		Name string
	}
	status := widget.Label("")

	grid := widget.DataGridView(func(a *Account) uint { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }, Sortable: true},
	).IndexColumnBySource().Sort("Name", false)
	grid.OnSelectionChanged(func() { status.SetText(fmt.Sprint(len(grid.Selected()), " selected")) })
	grid.InsertRows(0, Account{ID: 7, Name: "newest"}) // first in the source: number 1
}
