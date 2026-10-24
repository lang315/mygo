package widget_test

import "github.com/egoist/mygo/ui/widget"

type Task struct{ Done bool }

func hint(over bool) string {
	if over {
		return "Drop to finish"
	}
	return "Done"
}

func open([]string) {}

// The snippet of docs/ui/widget/drag-and-drop.md, which must compile.
func ExampleOnDrop() {
	task := &Task{}
	card := widget.Label("Task 1").Padding(8).DragSource(task)
	done := widget.Label("Done").Padding(12)
	widget.OnDragOver(done, func(over bool, t *Task) { done.SetText(hint(over)) })
	widget.OnDrop(done, func(t *Task) { t.Done = true })

	zone := widget.Label("Drop files here").Padding(24)
	zone.OnDropFiles(func(paths []string) { open(paths) })
	_ = card
}
