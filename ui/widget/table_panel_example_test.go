package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The snippet of BottomPanel in docs/ui/widget.md, which must compile.
func ExampleDataGridViewWidget_BottomPanel() {
	type Account struct {
		ID   uint
		Name string
	}
	grid := widget.DataGridView(func(a *Account) uint { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }, Filterable: true},
	)

	grid.CheckboxColumn().BottomPanel().
		PanelText(widget.PanelText{Rows: "%d rows", Selected: "%d selected", Search: "Find"}). // no checked count
		ApplyPanel(widget.Style{}.Background(widget.Fixed(ui.RGB(240, 244, 250))).TextSize(widget.SM)).
		SetSearchWidth(320).JoinSearch(true).
		SetRowHeight(28).SetGridLines(true)
}
