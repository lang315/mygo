package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The snippet of docs/ui/widget/listview.md, which must compile.
func ExampleListView() {
	status := widget.Label("")
	open := func(path string) {}
	type file struct{ path, name, kind string }
	var files []file
	icons := map[string]*ui.Bitmap{}
	viewMode := widget.ComboBox()

	view := widget.ListView(
		func(f *file) string { return f.path },
		widget.Column[file]{Title: "Name", Text: func(f *file) string { return f.name }, Width: 240, Sortable: true, Filterable: true},
		widget.Column[file]{Title: "Kind", Text: func(f *file) string { return f.kind }, Width: 100},
	).SetRows(files).CheckBoxes().
		SetImage(func(f *file) *ui.Bitmap { return icons[f.kind] })

	view.OnSelect(func(selected []string) {
		status.SetText(fmt.Sprint(len(selected), " chosen"))
	}).OnActivate(func(path string) {
		open(path)
	})

	viewMode.OnChange(func() { view.SetView(widget.ViewLargeIcon) })

	view.Details().Search("Name", "report") // filters every view
}

// The snippet of the Groups section of docs/ui/widget/listview.md, which must
// compile.
func ExampleListViewWidget_SetGroupBy() {
	type file struct{ path, name, kind string }
	view := widget.ListView(
		func(f *file) string { return f.path },
		widget.Column[file]{Title: "Name", Text: func(f *file) string { return f.name }},
	)

	view.SetGroupBy(func(f *file) string { return f.kind }).
		SetGroupOrder("Document", "Image")

	for _, g := range view.Groups() {
		fmt.Println(g.Title, g.Count) // "Document 3", then "Image 3", then the others
	}
}
