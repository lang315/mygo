package main

import (
	"fmt"
	"image/color"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// Gauge is a UserControl: a control type of its own, drawn with ui.
type Gauge struct {
	*widget.CustomWidget
	value float64
}

// NewGauge makes a gauge at 0%.
func NewGauge() *Gauge {
	g := &Gauge{}
	g.CustomWidget = widget.Custom(func(c *ui.Context) ui.Element {
		return ui.Text(c, fmt.Sprintf("Gauge %.0f%%", g.value*100)).Bold()
	})
	return g
}

// SetValue sets the fraction the gauge shows, from any goroutine.
func (g *Gauge) SetValue(v float64) *Gauge {
	widget.Change(func() { g.value = v })
	return g
}

// components shows what is not a control to place: an ImageList the
// controls share, a control of your own, the access keys of mnemonics and
// a LinkLabel.
func components() widget.Control {
	icons := widget.ImageList(24)
	icons.Add("red", swatch(color.RGBA{230, 80, 60, 255}))
	icons.Add("blue", swatch(color.RGBA{60, 120, 230, 255}))
	type item struct{ id, name, icon string }
	shapes := widget.ListBox(func(i *item) string { return i.id }, func(i *item) string { return i.name }).
		SetItems([]item{{"1", "Tomato", "red"}, {"2", "Sky", "blue"}, {"3", "No icon", ""}}).
		SetImageList(icons, func(i *item) string { return i.icon })

	gauge := NewGauge()
	pct := 0.0
	status := widget.Label("Alt+S, Alt+N")
	return widget.VBox(8).Add(
		widget.Label("ImageList").TextSize(widget.XL).Bold(),
		widget.Label("Two images kept by key in a list of 24 DIPs, shared by the rows of a ListBox (and a TreeView, a menu's icons)").Muted(),
		shapes,
		widget.Separator(),
		widget.Label("A control of your own").TextSize(widget.XL).Bold(),
		widget.HBox(8).Add(gauge, widget.Button("More").OnClick(func() {
			pct = min(pct+0.1, 1)
			gauge.SetValue(pct)
		})),
		widget.Separator(),
		widget.Label("Mnemonics").TextSize(widget.XL).Bold(),
		widget.HBox(8).Add(
			widget.Button("&Save").OnClick(func() { status.SetText("Save clicked") }),
			widget.Button("&Close").OnClick(func() { status.SetText("Close clicked") }),
			status),
		widget.Fields().
			Row("&Name", widget.TextBox().SetPlaceholder("Alt+N focuses this box")).
			Row("&Email", widget.TextBox().SetPlaceholder("Alt+E")),
		widget.Label("&City").SetUseMnemonic(true),
		widget.TextBox().SetPlaceholder("A Label with SetUseMnemonic(true): Alt+C focuses the box after it"),
		widget.Separator(),
		widget.LinkLabel("The docs", "https://github.com/lang315/mygo/blob/main/docs/ui/widget.md"),
	)
}
