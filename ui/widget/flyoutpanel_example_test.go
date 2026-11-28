package widget_test

import "github.com/egoist/mygo/ui/widget"

// The FlyoutPanel of docs/ui/widget/flyoutpanel.md: a settings panel sliding
// in over a document from its end, opened by a button, read as it closes.
func ExampleFlyoutPanel() {
	doc := widget.MultilineTextBox().Expand()
	wrap := widget.CheckBox("Wrap lines")
	fly := widget.FlyoutPanel(doc, widget.VBox(8).Add(widget.Label("View").Bold(), wrap))
	fly.SetEdge(widget.FlyoutRight)
	fly.OnClosed(func() { _ = wrap.Checked() })
	widget.VBox(8).Add(widget.Button("View…").OnClick(func() { fly.ShowPopup() }), fly)
}
