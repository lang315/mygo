package widget_test

import "github.com/egoist/mygo/ui/widget"

// The snippet of row dragging in docs/ui/widget.md, which must compile.
func ExampleDataGridViewWidget_SetRowReorder() {
	type Job struct {
		ID     int
		Name   string
		Status string
	}
	key := func(j *Job) int { return j.ID }
	name := widget.Column[Job]{Title: "Name", Text: func(j *Job) string { return j.Name }}
	status := widget.Column[Job]{Title: "Status", Text: func(j *Job) string { return j.Status }}
	save := func([]Job) {}
	archive := func([]*Job) {}

	queue := widget.DataGridView(key, name, status).SetRowReorder(true)
	queue.OnRowsReordered(func() { save(queue.Rows()) })
	done := widget.DataGridView(key, name, status).
		DragRows(func(rows []*Job) any { return rows })
	bin := widget.Label("Drop jobs here").Padding(24)
	widget.OnDrop(bin, func(rows []*Job) { archive(rows) })
	_ = done
}
