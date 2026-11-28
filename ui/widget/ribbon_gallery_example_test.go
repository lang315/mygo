package widget_test

import "github.com/egoist/mygo/ui/widget"

// The gallery of docs/ui/widget/ribbon.md: styles in a row of four, previewed
// as the pointer or the keyboard moves over them, applied when picked.
func ExampleRibbonWidget_AddGallery() {
	ribbon := widget.Ribbon().AddTab("Home").AddGroup("Styles")
	styles := ribbon.AddGallery([]widget.GalleryItem{
		{Text: "Normal"}, {Text: "Heading 1"}, {Text: "Heading 2"}, {Text: "Title"}, {Text: "Quote"},
	}, 4)
	styles.SetAccessibleName("Styles") // its name, and its key tip's letter
	styles.SetSelected(0)
	styles.OnHover(func(i int) { println("preview", i) }) // -1: back to the style applied
	styles.OnSelect(func(i int) { println("apply", i) })
}
