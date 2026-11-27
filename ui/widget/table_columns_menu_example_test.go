package widget_test

import "github.com/egoist/mygo/ui/widget"

// The snippet of BestFit, SetColumnChooser and SetCellHints in
// docs/ui/widget.md, which must compile.
func ExampleDataGridViewWidget_BestFit() {
	type Profile struct {
		ID                    uint
		Name, Note, LastError string
	}
	key := func(p *Profile) uint { return p.ID }
	name := widget.Column[Profile]{Title: "Name", Text: func(p *Profile) string { return p.Name }, Width: 40}
	note := widget.Column[Profile]{Title: "Note", Text: func(p *Profile) string { return p.Note }, Width: 160,
		Hint: func(p *Profile) string { return p.LastError }} // the cell's tooltip; "" for none
	t := widget.DataGridView(key, name, note).
		SetColumnChooser(true). // a right-click on the header: the columns, Best fit, Clear sorting
		SetCellHints(true)      // a Text cut off shows whole as a tooltip
	t.BestFit("Name") // at the next frame; BestFit() fits every column shown
}
