package widget_test

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The snippets of docs/ui/widget/menubar.md, which must compile.
func ExampleMenuBar() {
	var openFile, openRecent, toggleWrap, deleteAll func()
	var wrap bool
	editor := widget.MultilineTextBox()
	f := widget.NewForm("Notes", 800, 600)
	f.UseHeaderBar("") // the title bar the app draws
	bar := widget.MenuBar(
		widget.MenuItem{Text: "&File", Children: []widget.MenuItem{
			{Text: "&Open…", OnClick: openFile, Shortcut: widget.Shortcut{Mods: ui.Ctrl, Key: ui.KeyO}},
			{Text: "Open &recent", Children: []widget.MenuItem{{Text: "notes.txt", OnClick: openRecent}}},
			{},
			{Text: "E&xit", OnClick: f.Close},
		}},
		widget.MenuItem{Text: "&View", Children: []widget.MenuItem{
			{Text: "&Word wrap", Checked: wrap, OnClick: toggleWrap},
			{Text: "&Delete all notes", Destructive: true, OnClick: deleteAll},
		}},
	)
	f.SetContent(widget.VBox(0).Add(bar, editor))
	f.Shortcut(ui.Ctrl, ui.KeyO, openFile) // the shortcut an item shows answers nothing itself
}
