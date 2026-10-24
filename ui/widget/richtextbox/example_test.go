package richtextbox_test

import (
	"github.com/egoist/mygo/ui/richtext"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/richtextbox"
)

func save(string) {}

// The snippet of docs/ui/widget/richtextbox.md, which must compile.
func ExampleNew() {
	var notes *richtextbox.Box
	notes = richtextbox.New().
		SetMarkup("Hello <b>world</b>").
		SetLabel("Notes").
		OnChange(func() { save(notes.Markdown()) })
	bold := widget.Button("B").OnClick(func() { notes.Edit(func(d *richtext.Document) { d.ToggleBold() }) })
	_ = bold
}
