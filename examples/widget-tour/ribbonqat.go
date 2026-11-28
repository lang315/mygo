package main

import "github.com/egoist/mygo/ui/widget"

// ribbonQAT shows a ribbon with a Quick Access Toolbar, which draws in the
// tour's header bar while this page shows, a gallery of styles, which a
// label previews and shows the pick of, and a contextual tab, Format of
// Picture Tools, that a check box shows.
func ribbonQAT() widget.Control {
	saved := widget.Label("Quick Access: Save, Undo")
	save := widget.ToolButton("Save").SetIcon(lucide(lucideIcons[0]))
	undo := widget.ToolButton("Undo").SetIcon(lucide(lucideIcons[1]))
	var ribbon *widget.RibbonWidget
	ribbon = widget.Ribbon().
		AddTab("Home").AddGroup("Clipboard").
		AddLarge(widget.ToolButton("Paste").SetIcon(lucide(lucideIcons[0]))).
		AddSmall(widget.ToolButton("Cut"), widget.ToolButton("Copy")).
		AddGroup("Styles")
	style := widget.Label("Style: Normal")
	styles := []widget.GalleryItem{{Text: "Normal"}, {Text: "Heading 1"}, {Text: "Heading 2"}, {Text: "Title"}, {Text: "Quote"}, {Text: "Code"}}
	gallery := ribbon.AddGallery(styles, 3)
	gallery.SetAccessibleName("Styles").SetSelected(0)
	gallery.OnHover(func(i int) { // a live preview, back to the style picked at -1
		if i < 0 {
			i = gallery.Selected()
		}
		if i >= 0 {
			style.SetText("Style: " + styles[i].Text)
		}
	})
	gallery.OnSelect(func(i int) { style.SetText("Style: " + styles[i].Text) })
	ribbon.AddContextualTab("Picture Tools", "Format").AddGroup("Arrange").
		AddSmall(widget.ToolButton("Rotate"), widget.ToolButton("Crop")).
		AddQuickAccess(save, undo)
	ribbon.OnQuickAccessChange(func() {
		names := ribbon.QuickAccess()
		text := "Quick Access:"
		for i, n := range names {
			if i > 0 {
				text += ","
			}
			text += " " + n
		}
		saved.SetText(text)
	})
	var picture *widget.CheckBoxWidget
	picture = widget.CheckBox("Picture selected").OnChange(func() {
		ribbon.SetCategoryVisible("Picture Tools", picture.Checked())
	})
	return widget.VBox(4).Add(ribbon, widget.HBox(8).Add(picture, saved, style))
}
