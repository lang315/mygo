package main

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// listViewTour shows one ListView in the four views, which four buttons
// switch between; what is chosen and checked stays as the view changes. Two
// more buttons group the icon views by kind, or not.
func listViewTour() widget.Control {
	type file struct{ path, name, kind string }
	files := []file{
		{"/a", "Annual report", "Document"}, {"/b", "Budget", "Sheet"}, {"/c", "Cover", "Image"},
		{"/d", "Design notes", "Document"}, {"/e", "Expenses", "Sheet"}, {"/f", "Logo", "Image"},
		{"/g", "Minutes", "Document"}, {"/h", "Roadmap", "Sheet"}, {"/i", "Team photo", "Image"},
	}
	status := widget.Label("Choose files; Space checks them; switch the view")
	view := widget.ListView(
		func(f *file) string { return f.path },
		widget.Column[file]{Title: "Name", Text: func(f *file) string { return f.name }, Width: 240, Sortable: true, Filterable: true},
		widget.Column[file]{Title: "Kind", Text: func(f *file) string { return f.kind }, Width: 120, Sortable: true},
	).SetRows(files).CheckBoxes()
	view.OnSelect(func(ks []string) { status.SetText(fmt.Sprintf("%d chosen: %v", len(ks), ks)) })
	view.OnCheck(func(string, bool) { status.SetText(fmt.Sprintf("%d checked: %v", len(view.Checked()), view.Checked())) })

	mode := func(text string, m widget.ViewMode) widget.Control {
		return widget.Button(text).OnClick(func() { view.SetView(m) })
	}
	return widget.VBox(8).Add(
		widget.HBox(8).Add(
			mode("Details", widget.ViewDetails), mode("Large icons", widget.ViewLargeIcon),
			mode("Small icons", widget.ViewSmallIcon), mode("List", widget.ViewList),
			widget.Button("Group by kind").OnClick(func() { view.SetGroupBy(func(f *file) string { return f.kind }) }),
			widget.Button("No groups").OnClick(func() { view.SetGroupBy(nil) })),
		view, status,
	)
}
