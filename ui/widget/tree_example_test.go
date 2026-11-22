package widget_test

import "github.com/egoist/mygo/ui/widget"

// The snippets of docs/ui/widget/treeview.md, which must compile.
func ExampleTreeView() {
	status := widget.Label("")
	listDir := func(path string) []entry { return nil }

	tree := widget.TreeView(
		func(e *entry) string { return e.path },
		func(e *entry) string { return e.name },
	).CheckBoxes().
		SetRoots([]entry{{"/home", "Home", true}, {"/tmp", "Temp", true}}).
		SetHasChildren("/home", true).
		SetHasChildren("/tmp", true)

	tree.OnExpand(func(path string) {
		go func() {
			kids := listDir(path) // the entries of a directory, read from the disk
			tree.SetChildren(path, kids)
			for _, k := range kids {
				if k.dir {
					tree.SetHasChildren(k.path, true)
				}
			}
		}()
	}).OnSelect(func(path string, ok bool) {
		if ok {
			status.SetText(path)
		}
	})

	tree.OnCheck(func(k string, on bool) {
		tree.SetChecked(tree.Children(k), on)
	})
}

// The tree in columns of docs/ui/widget/treeview.md, "Columns".
func ExampleTreeViewWidget_SetColumns() {
	type file struct{ path, name, size, kind string }

	tree := widget.TreeView(
		func(f *file) string { return f.path },
		func(f *file) string { return f.name },
	).SetColumns("Name",
		widget.Column[file]{Title: "Size", Text: func(f *file) string { return f.size }, Width: 80, Align: widget.End},
		widget.Column[file]{Title: "Kind", Text: func(f *file) string { return f.kind }, Width: 100},
	).SetRoots([]file{{"/src", "src", "", "Folder"}, {"/README", "README", "3 KB", "Text"}})
	_ = tree
}

type entry struct {
	path, name string
	dir        bool
}
