package export_test

import (
	"os"

	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/export"
)

type Account struct {
	ID   int
	Name string
}

// The snippet of docs/ui/widget/export.md, which must compile.
func ExampleXLSX() {
	var w *widget.Form // the window, as widget.Run gives it
	grid := widget.DataGridView(func(a *Account) int { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }})
	widget.Button("Export").OnClick(func() {
		path := widget.SaveFile(w, "Export", "accounts.xlsx",
			widget.Filter{Name: "Excel workbook", Extensions: []string{"xlsx"}})
		if path == "" {
			return // canceled
		}
		f, err := os.Create(path)
		if err != nil {
			widget.Alert(w, widget.Error, "Export", err.Error())
			return
		}
		header, rows := grid.ShownCells()
		err = export.XLSX(f, header, rows)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			widget.Alert(w, widget.Error, "Export", err.Error())
		}
	})
}

// The snippet of the DataGridView part of docs/ui/widget.md.
func ExampleCSV() {
	t := widget.DataGridView(func(a *Account) int { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }})
	f, _ := os.Create("accounts.csv")
	defer f.Close()
	header, rows := t.ShownCells() // the columns and the rows as shown
	err := export.CSV(f, header, rows)
	_ = err
}
