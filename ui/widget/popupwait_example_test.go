package widget_test

import (
	"strings"

	"github.com/egoist/mygo/ui/widget"
)

// The snippets of docs/ui/widget/popupedit.md, which must compile.
func ExamplePopupEdit() {
	cols := widget.CheckBoxGroup(false,
		widget.Option{Value: "name", Text: "Name"},
		widget.Option{Value: "city", Text: "City"},
	)
	var edit *widget.PopupEditWidget
	edit = widget.PopupEdit(cols).SetText("No columns").OnClosed(func() {
		edit.SetText(strings.Join(cols.Values(), ", "))
	})
	_ = edit
}

// The snippets of docs/ui/widget/progresspanel.md, which must compile.
func ExampleProgressPanel() {
	panel := widget.ProgressPanel("Loading").SetDescription("Reading the customers")
	_ = panel
}

func load() {}

func ExampleForm_ShowWait() {
	w := widget.NewForm("Customers", 800, 600)
	go func() {
		w.ShowWait("Loading", "Reading the customers")
		defer w.CloseWait()
		load()
	}()
}
