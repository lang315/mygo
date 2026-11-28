package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// navigation shows a ribbon with a Quick Access Toolbar and a contextual
// tab, a search box filtering a list, a breadcrumb path, an accordion, a color edit, and text boxes with suggestions and a history.
func navigation() widget.Control {
	cities := []string{"Hanoi", "Hue", "Da Nang", "Ho Chi Minh City", "Can Tho"}
	found := widget.Label(strings.Join(cities, ", "))
	var search *widget.SearchBoxWidget
	search = widget.SearchBox().SetPlaceholder("Find a city").OnChange(func() {
		var hits []string
		for _, c := range cities {
			if strings.Contains(strings.ToLower(c), strings.ToLower(search.Text())) {
				hits = append(hits, c)
			}
		}
		found.SetText(strings.Join(hits, ", "))
	})

	path := []string{"C:", "Users", "ada", "Documents"}
	where := widget.Label("At " + strings.Join(path, "/"))
	var crumb *widget.BreadCrumbWidget
	crumb = widget.BreadCrumb(path...).OnSelect(func(i int) {
		path = path[:i+1]
		crumb.SetItems(path...)
		where.SetText("At " + strings.Join(path, "/"))
	})

	opened := widget.Label("Open section: General")
	var acc *widget.AccordionWidget
	acc = widget.Accordion().
		AddSection("General", widget.TextBox().SetPlaceholder("Name"), widget.CheckBox("Start at login")).
		AddSection("Privacy", widget.CheckBox("Send usage data")).
		AddSection("Advanced", widget.TextBox().SetPlaceholder("Proxy")).
		SetOpen(0)
	acc.OnChange(func() { opened.SetText(fmt.Sprintf("Open section: %d", acc.Open())) })

	tint := widget.Label("Tint #3b82f6")
	var color *widget.ColorEditWidget
	color = widget.ColorEdit().SetColor(ui.Hex("#3b82f6")).OnChange(func() {
		c := color.Color()
		tint.SetText(fmt.Sprintf("Tint #%02x%02x%02x", c.R, c.G, c.B))
	})

	return widget.VBox(8).Add(
		ribbonQAT(),
		search, found,
		crumb, where,
		acc, opened,
		widget.HBox(8).Add(color, tint),
		widget.TextBox().SetPlaceholder("City").SetAccessibleName("City").SetSuggestions(cities...),
		widget.TextBox().SetPlaceholder("Recent searches").SetMRU(5),
	)
}
