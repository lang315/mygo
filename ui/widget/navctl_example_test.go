package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

type customers struct{}

func (customers) Filter(string) {}

// The snippets of docs/ui/widget/searchbox.md, which must compile.
func ExampleSearchBox() {
	var list customers
	var search *widget.SearchBoxWidget
	search = widget.SearchBox().SetPlaceholder("Find a customer").
		OnChange(func() { list.Filter(search.Text()) })
	_ = search
}

// The snippets of docs/ui/widget/breadcrumb.md, which must compile.
func ExampleBreadCrumb() {
	path := []string{"C:", "Users", "ada"}
	var crumb *widget.BreadCrumbWidget
	crumb = widget.BreadCrumb(path...).OnSelect(func(i int) {
		path = path[:i+1]
		crumb.SetItems(path...)
	})
}

// The snippets of docs/ui/widget/accordion.md, which must compile.
func ExampleAccordion() {
	settings := widget.Accordion().
		AddSection("General", widget.TextBox().SetPlaceholder("Name"), widget.CheckBox("Start at login")).
		AddSection("Privacy", widget.CheckBox("Send usage data")).
		SetOpen(0)
	settings.OnChange(func() { _ = settings.Open() })
}

// The snippets of docs/ui/widget/coloredit.md, which must compile.
func ExampleColorEdit() {
	var tint *widget.ColorEditWidget
	tint = widget.ColorEdit().SetColor(ui.Hex("#3b82f6")).OnChange(func() { _ = tint.Color() })
}

// The snippets of docs/ui/widget/textbox-suggestions.md, which must compile.
func ExampleTextBoxWidget_SetSuggestions() {
	city := widget.TextBox().SetSuggestions("Hanoi", "Hue", "Da Nang")
	recent := widget.TextBox().SetMRU(10) // remembers the last 10 entries
	_, _ = city, recent
}
