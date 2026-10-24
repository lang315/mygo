package widget_test

import (
	"github.com/egoist/mygo/ui/widget"
)

type node struct{ Icon string }
type file struct{ Kind string }

var folder16, folder32, file16, file32 []byte

func key(n *node) string   { return n.Icon }
func text(n *node) string  { return n.Icon }
func fkey(f *file) string  { return f.Kind }
func ftext(f *file) string { return f.Kind }

// The snippet of docs/ui/widget/imagelist.md, which must compile.
func ExampleImageList() {
	icons := widget.ImageList(16)
	icons.Add("folder", folder16, folder32)
	icons.Add("file", file16, file32)

	tree := widget.TreeView(key, text).SetImage(widget.ImagesBy(icons, func(n *node) string { return n.Icon }))
	list := widget.ListBox(fkey, ftext).SetImage(widget.ImagesBy(icons, func(f *file) string { return f.Kind }))
	menu := widget.MenuItem{Text: "Open", IconPNG: icons.PNG("folder")}
	_, _, _ = tree, list, menu
}
