package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// entry is a folder or a file of the TreeView's tour.
type entry struct {
	path, name string
	dir        bool
}

// folderLoad is the wait for a folder's children: a disk, or a server. A test
// gives its own, to hold the Loading row for as long as it checks it.
var folderLoad = func() { time.Sleep(400 * time.Millisecond) }

// treeView shows a TreeView of folders whose children load, a moment late,
// when the user opens them; its check boxes count in the label below it.
func treeView() widget.Control {
	status := widget.Label("Open a folder, choose a node, check some")
	var tree *widget.TreeViewWidget[entry, string]
	tree = widget.TreeView(
		func(e *entry) string { return e.path },
		func(e *entry) string { return e.name },
	).CheckBoxes().SetLoadingText("Loading…")
	roots := []entry{{"/documents", "Documents", true}, {"/pictures", "Pictures", true}, {"/music", "Music", true}}
	tree.SetRoots(roots)
	for _, r := range roots {
		tree.SetHasChildren(r.path, true)
	}
	tree.OnExpand(func(path string) {
		go func() {
			folderLoad()
			kids := []entry{{path + "/archive", "Archive", true}, {path + "/notes.txt", "notes.txt", false}, {path + "/todo.txt", "todo.txt", false}}
			if strings.Count(path, "/") > 2 {
				kids = kids[1:] // no deeper folders
			}
			tree.SetChildren(path, kids)
			for _, k := range kids {
				if k.dir {
					tree.SetHasChildren(k.path, true)
				}
			}
		}()
	}).OnSelect(func(path string, ok bool) {
		if ok {
			status.SetText("Chosen " + path)
		}
	}).OnCheck(func(path string, on bool) {
		status.SetText(fmt.Sprintf("%d checked", len(tree.Checked())))
	})
	return widget.VBox(8).Add(tree, status)
}
