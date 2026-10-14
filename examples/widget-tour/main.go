// Widget-tour shows every widget of package ui/widget, by tab.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func main() {
	err := widget.Run("Widget Tour", func(w *widget.Window) {
		w.UseHeaderBar("ui/widget").PackEnd(widget.Link("Docs", "https://github.com/lang315/mygo/blob/main/docs/ui/widget.md"))
		w.SetContent(tour())
	})
	if err != nil {
		log.Fatal(err)
	}
}

func tour() widget.Widget {
	return widget.Notebook().
		AddPage("Layout", layout()).
		AddPage("Inputs", inputs()).
		AddPage("Choices", choices()).
		AddPage("Containers", containers()).
		Padding(4)
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
