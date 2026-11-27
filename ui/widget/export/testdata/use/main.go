// Command use is nouse with Export buttons: TestExportNotLinkedWhenUnused
// builds it to see that its check finds the writers' packages when an app
// imports them.
package main

import (
	"io"
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/export"
)

type row struct {
	id   int
	name string
}

type service struct{}

func (service) Hello() string { return "hello" }

func main() {
	mygo.Bind(service{})
	err := widget.Run("use", func(w *widget.Form) {
		g := widget.DataGridView(func(r *row) int { return r.id },
			widget.Column[row]{Title: "Name", Text: func(r *row) string { return r.name }},
		).SetRows([]row{{1, "a"}})
		save := func(name string, write func(io.Writer, []string, [][]string) error) {
			f, err := os.Create(name)
			if err != nil {
				return
			}
			defer f.Close()
			header, rows := g.ShownCells()
			_ = write(f, header, rows)
		}
		w.SetContent(widget.VBox(8).Add(
			widget.Button("CSV").OnClick(func() { save("rows.csv", export.CSV) }),
			widget.Button("XLSX").OnClick(func() { save("rows.xlsx", export.XLSX) }),
		).Pack(g, true, true))
	})
	if err != nil {
		log.Fatal(err)
	}
}
