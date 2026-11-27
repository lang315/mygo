// Command nouse is a small app with a DataGridView that does not import
// ui/widget/export: TestExportNotLinkedWhenUnused builds it and looks for
// the writers' packages in its symbols. It binds a service, as an app with
// a web page does, so that a build without mygo_static keeps every exported
// method of the program (ShownCells too).
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/widget"
)

type row struct {
	id   int
	name string
}

type service struct{}

func (service) Hello() string { return "hello" }

func main() {
	mygo.Bind(service{})
	err := widget.Run("nouse", func(w *widget.Form) {
		w.SetContent(widget.DataGridView(func(r *row) int { return r.id },
			widget.Column[row]{Title: "Name", Text: func(r *row) string { return r.name }},
		).SetRows([]row{{1, "a"}}))
	})
	if err != nil {
		log.Fatal(err)
	}
}
