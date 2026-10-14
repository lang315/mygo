// Widget-tour shows every widget of package ui/widget, by tab.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func main() {
	err := widget.Run("Widget Tour", func(w *widget.Window) {
		w.UseHeaderBar("ui/widget").PackEnd(widget.Link("Docs", "https://github.com/lang315/mygo/blob/main/docs/ui/widget.md"))
		w.SetMenuBar(widget.MenuItem{Text: "File", Children: []widget.MenuItem{
			{Text: "Close", OnClick: func() { w.Close() }},
		}})
		w.SetContent(tour(w))
	})
	if err != nil {
		log.Fatal(err)
	}
}

func tour(w *widget.Window) widget.Widget {
	return widget.Notebook().
		AddPage("Layout", layout()).
		AddPage("Inputs", inputs()).
		AddPage("Choices", choices()).
		AddPage("Containers", containers()).
		AddPage("Dialogs", dialogs(w)).
		AddPage("Extras", extras()).
		Padding(4)
}

func dialogs(w *widget.Window) widget.Widget {
	result := widget.Label("Results show here").Muted()
	show := func(format string, a ...any) { result.SetText(fmt.Sprintf(format, a...)) }
	return widget.VBox(8).Add(
		widget.Label("Dialogs").TextSize(widget.XL).Bold(),
		widget.Flow(8).Add(
			widget.Button("Alert").OnClick(func() {
				widget.Alert(w, widget.Info, "Saved", "The file was saved.")
				show("Alert closed")
			}),
			widget.Button("Confirm").OnClick(func() {
				show("Confirm: %v", widget.Confirm(w, widget.Question, "Delete the file?", "It goes for good."))
			}),
			widget.Button("Input text").OnClick(func() {
				s, ok := widget.InputString(w, widget.Question, "Name", "Your name", "")
				show("InputString: %q %v", s, ok)
			}),
			widget.Button("Input number").OnClick(func() {
				n, ok := widget.InputInt(w, widget.Question, "Threads", "How many?", 1, 64, 1, 8)
				show("InputInt: %d %v", n, ok)
			}),
			widget.Button("Input choice").OnClick(func() {
				v, ok := widget.InputSelect(w, widget.Question, "Proxy", "Choose one",
					widget.Option{Value: "http", Text: "HTTP"}, widget.Option{Value: "socks", Text: "SOCKS5"})
				show("InputSelect: %q %v", v, ok)
			}),
			widget.Button("Custom dialog").OnClick(func() {
				customDialog(w)
				show("Custom dialog closed")
			}),
			widget.Button("Open file").OnClick(func() { show("OpenFile: %q", widget.OpenFile(w, "Open a file")) }),
			widget.Button("Save file").OnClick(func() { show("SaveFile: %q", widget.SaveFile(w, "Save", "notes.txt")) }),
			widget.Button("Select folder").OnClick(func() { show("SelectFolder: %q", widget.SelectFolder(w, "Choose a folder", "")) }),
		),
		result,
		widget.Separator(),
		widget.Label("Menus").TextSize(widget.XL).Bold(),
		widget.HBox(8).Add(
			widget.MenuButton("Sort by").SetMenu(
				widget.MenuItem{Text: "Name", OnClick: func() { show("Sort by name") }},
				widget.MenuItem{Text: "Date", Checked: true, OnClick: func() { show("Sort by date") }},
				widget.MenuItem{},
				widget.MenuItem{Text: "More", Children: []widget.MenuItem{{Text: "Size", OnClick: func() { show("Sort by size") }}}},
			),
			widget.Label("Right-click me").Padding(2).Border(1, ui.RGB(160, 160, 160)).Radius(1).ContextMenu(
				widget.MenuItem{Text: "Copy", OnClick: func() { widget.Clipboard.SetText("Right-click me"); show("Copied") }},
				widget.MenuItem{Text: "Delete…", OnClick: func() { show("Delete: %v", widget.Confirm(w, widget.Warning, "Delete?", "")) }},
			),
		),
	)
}

// customDialog runs a dialog of its own, with a dialog over it.
func customDialog(w *widget.Window) {
	var d *widget.Dialog
	d = widget.NewDialog(w, "New account", "A dialog of your own", 420, 260, widget.VBox(8).Add(
		widget.Entry().SetPlaceholder("Name"),
		widget.Button("Nested dialog").OnClick(func() {
			widget.Alert(d.Window, widget.Info, "Nested", "A dialog over a dialog.")
		}),
		widget.ButtonBox().Add(widget.Button("Close").Suggested().OnClick(func() { d.Close() })),
	))
	d.Run()
}

func extras() widget.Widget {
	theme := widget.Combo(
		widget.Option{Value: "system", Text: "System"},
		widget.Option{Value: "light", Text: "Light"},
		widget.Option{Value: "dark", Text: "Dark"},
	).SetValue("system")
	theme.OnChange(func() { widget.SetTheme(widget.Theme(theme.Value())) }) // values are mygo.ThemeSource's
	screen := widget.Label("")
	icons := widget.HBox(12)
	for _, shapes := range lucideIcons {
		icons.Add(widget.Icon(lucide(shapes), 24))
	}
	return widget.VBox(8).Add(
		widget.HBox(8).Add(widget.Label("Theme").VAlign(widget.Center), theme),
		widget.HBox(8).Add(widget.Button("Screen size").OnClick(func() {
			w, h := widget.ScreenSize()
			screen.SetText(fmt.Sprintf("%v x %v", w, h))
		}), screen.VAlign(widget.Center)),
		icons,
		widget.Button("With an icon").SetIcon(lucide(lucideIcons[0])),
		widget.HBox(8).Add(widget.Image(swatch(color.RGBA{230, 80, 60, 255}), 48, 48), widget.Image(swatch(color.RGBA{60, 120, 230, 255}), 96, 48)),
		widget.Label("").Rich(ui.Span{Text: "Rich text: "}, ui.Span{Text: "bold", Weight: 700}, ui.Span{Text: ", "}, ui.Span{Text: "italic", Italic: true}),
		widget.HBox(8).Add(widget.Label("Sectioned combo").VAlign(widget.Center), widget.Combo().
			AddSection("Fruits", swatch(color.RGBA{230, 160, 40, 255}), widget.Option{Value: "apple", Text: "Apple"}, widget.Option{Value: "pear", Text: "Pear"}).
			AddSection("Vegetables", swatch(color.RGBA{60, 160, 80, 255}), widget.Option{Value: "kale", Text: "Kale"})),
	)
}

// lucideIcons are the shapes of some Lucide icons (ISC license): plus,
// minus, x, search, play, square.
var lucideIcons = []string{
	`<path d="M5 12h14"/><path d="M12 5v14"/>`,
	`<path d="M5 12h14"/>`,
	`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`,
	`<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>`,
	`<polygon points="6 3 20 12 6 21 6 3"/>`,
	`<rect width="18" height="18" x="3" y="3" rx="2"/>`,
}

func lucide(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// swatch returns a 16x16 PNG of one color, made at run time so the
// example ships no image files.
func swatch(c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func layout() widget.Widget {
	return widget.VBox(8).Add(
		widget.Label("Boxes").TextSize(widget.XL).Bold(),
		widget.HBox(8).Add(widget.Label("start")).
			Pack(widget.Label("expanded, centered"), true, false).
			PackEnd(widget.Button("end"), false, false),
		widget.Separator(),
		widget.Label("Align(Start)").Align(widget.Start),
		widget.Label("Align(Center)").Align(widget.Center),
		widget.Label("Align(End)").Align(widget.End),
		widget.Label("Muted, italic, mono").Muted().Italic().Mono(),
		widget.Label("Padding(2), border, radius").Padding(2).Border(1, ui.RGB(160, 160, 160)).Radius(1).Align(widget.Start),
		widget.ButtonBox().Add(widget.Button("Cancel"), widget.Button("Delete").Destructive(), widget.Button("Save").Suggested()),
	)
}

func inputs() widget.Widget {
	var name *widget.EntryWidget
	greeting := widget.Label("Type a name")
	return widget.VBox(8).Add(
		widget.Entry().SetPlaceholder("Name").AssignTo(&name).OnChange(func() {
			greeting.SetText("Hello, " + name.Text())
		}),
		greeting,
		widget.Entry().SetPlaceholder("Password").Password(),
		widget.TextView().SetPlaceholder("Notes").SetSize(0, 100),
		widget.Checkbox("Remember me"),
		widget.HBox(8).Add(widget.Switch(), widget.Label("Notifications")),
		widget.Toggle("Bold"),
		widget.Button("Count in the background").OnClick(func() {
			go func() {
				for i := 1; i <= 5; i++ {
					greeting.SetText(fmt.Sprintf("Counted %d", i))
					time.Sleep(300 * time.Millisecond)
				}
			}()
		}),
	)
}

func choices() widget.Widget {
	progress := widget.Progress().SetFraction(0.3).SetText("30%")
	spin := widget.Spin(0, 100, 1).SetValue(30)
	spin.OnChange(func() {
		progress.SetFraction(spin.Value() / 100).SetText(fmt.Sprintf("%d%%", spin.Int()))
	})
	return widget.VBox(8).Add(
		spin, progress,
		widget.RangeSpin(0, 60, 0, 600, 1, "to", "seconds").SetValues(5, 30),
		widget.Combo(widget.Option{Value: "vn", Text: "Vietnam"}, widget.Option{Value: "us", Text: "United States"}).SetValue("vn"),
		widget.CheckboxGroup(true, widget.Option{Value: "a", Text: "Apple"}, widget.Option{Value: "b", Text: "Banana"}),
		widget.RadioGroup(true, widget.Option{Value: "s", Text: "Small"}, widget.Option{Value: "l", Text: "Large"}),
		widget.TagSelect(widget.Option{Value: "go", Text: "Go"}, widget.Option{Value: "ui", Text: "UI"}, widget.Option{Value: "gtk", Text: "GTK"}),
		widget.Spinner().Start(),
	)
}

func containers() widget.Widget {
	grid := widget.Grid(8, 8).
		Attach(widget.Label("A"), 0, 0, 1, 1).
		Attach(widget.Label("B"), 1, 0, 1, 1).
		Attach(widget.Label("C spans two"), 0, 1, 2, 1)
	flow := widget.Flow(6)
	for i := range 12 {
		flow.Add(widget.Button(fmt.Sprintf("chip %d", i)))
	}
	return widget.Sidebar().
		Add("frame", "Frame and grid", widget.VBox(8).Add(widget.Frame("Frame").Add(grid))).
		Add("card", "Card", widget.Card().SetTitle(widget.Label("Card title").Bold()).SetBody(widget.Label("Card body"))).
		Add("flow", "Flow", flow).
		Add("scroll", "Scrolled", widget.Scrolled(longList()))
}

func longList() widget.Widget {
	b := widget.VBox(4)
	for i := range 100 {
		b.Add(widget.Label(fmt.Sprintf("Row %d", i)))
	}
	return b
}
