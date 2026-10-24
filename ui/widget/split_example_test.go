package widget_test

import "github.com/egoist/mygo/ui/widget"

var config = struct{ Set func(string, any) }{func(string, any) {}}

// The snippet of docs/ui/widget/split-container.md, which must compile.
func ExampleSplitContainer() {
	files := widget.VBox(4).Add(widget.Label("files"), widget.Button("Open"))
	editor := widget.SplitContainer(widget.Label("editor"), widget.Label("console")).Stacked()

	var split *widget.SplitContainerWidget
	split = widget.SplitContainer(files, editor).
		SetSplitterDistance(240).
		OnSplitterMoved(func() { config.Set("sidebar", split.SplitterDistance()) })
}
