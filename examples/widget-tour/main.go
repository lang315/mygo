// Widget-tour shows every widget of package ui/widget, by tab.
package main

import (
	"bytes"
	"cmp"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func main() {
	err := widget.Run("Widget Tour", func(w *widget.Form) {
		w.UseHeaderBar("ui/widget").PackEnd(widget.LinkLabel("Docs", "https://github.com/lang315/mygo/blob/main/docs/ui/widget.md"))
		// The app shell: a main menu, a tool strip above the content and a
		// status strip below it, all changed while the app runs.
		status := widget.StatusLabel("Ready").SetIcon(lucide(lucideIcons[4]))
		done := 0.3
		progress := widget.StatusProgress().SetFraction(done)
		bar := widget.StatusStrip(status, widget.Spring(), progress, widget.Separator(), widget.StatusLabel("ui/widget"))
		closeItem := widget.StripItem("Close").SetShortcut("CmdOrCtrl+W").OnClick(func() { w.Close() })
		showBar := widget.StripCheck("Status bar", true)
		showBar.OnClick(func() { bar.SetVisible(showBar.Checked()) })
		w.MenuStrip(widget.NewMenuStrip(
			widget.StripRole(mygo.RoleAppMenu),
			widget.StripItem("File", closeItem),
			widget.StripRole(mygo.RoleEditMenu),
			widget.StripItem("View", showBar),
		))
		bold := widget.ToolButton("Bold").SetToggle(true)
		bold.OnChange(func() { status.SetText(fmt.Sprintf("Bold: %v", bold.Checked())) })
		w.ToolStrip(widget.ToolStrip(
			widget.ToolButton("Plus").SetIcon(lucide(lucideIcons[0])).SetTooltip("Add one").OnClick(func() {
				done = min(done+0.1, 1) // handlers run one at a time: no lock
				progress.SetFraction(done)
			}),
			bold,
			widget.Separator(),
			widget.DropDownButton("Menu",
				widget.MenuItem{Text: "Disable File > Close", OnClick: func() { closeItem.SetEnabled(!closeItem.Enabled()) }},
				widget.MenuItem{Text: "Say hello", OnClick: func() { status.SetText("Hello") }}),
		))
		w.StatusStrip(bar)
		w.SetContent(tour(w))
	})
	if err != nil {
		log.Fatal(err)
	}
}

func tour(w *widget.Form) widget.Control {
	return widget.TabControl().
		AddPage("Layout", layout()).
		AddPage("Inputs", inputs()).
		AddPage("Choices", choices()).
		AddPage("Containers", containers()).
		AddPage("Dialogs", dialogs(w)).
		AddPage("Extras", extras()).
		AddPage("App shell", shell()).
		AddPage("DataGridView", table()).
		Padding(4).Expand()
}

// shell shows the NotifyIcon; the window's menu, tool strip and status strip
// are the ones around this page.
func shell() widget.Control {
	var icon *widget.NotifyIcon
	state := widget.Label("No tray icon").Muted()
	toggle := widget.CheckBox("Tray icon")
	toggle.OnChange(func() {
		switch {
		case toggle.Checked() && icon == nil:
			n, err := widget.NewNotifyIcon()
			if err != nil {
				state.SetText("No tray icon: " + err.Error())
				return
			}
			icon = n
			n.SetIcon(swatch(color.RGBA{37, 99, 235, 255}), false)
			n.SetText("Widget Tour")
			n.SetContextMenu(widget.MenuItem{Text: "Hide the icon", OnClick: func() { toggle.SetChecked(false); n.SetVisible(false); state.SetText("Hidden") }})
			n.OnClick(func() { state.SetText("The icon was clicked") })
			state.SetText("The tray icon is shown; it has a menu, so a click opens it")
		case icon != nil:
			if err := icon.SetVisible(toggle.Checked()); err != nil {
				toggle.SetChecked(false)
				state.SetText("Could not show the icon: " + err.Error())
				return
			}
			state.SetText(map[bool]string{true: "Shown again", false: "Hidden"}[toggle.Checked()])
		}
	})
	return widget.VBox(8).Add(
		widget.Label("App shell").TextSize(widget.XL).Bold(),
		widget.Label("The menu bar (a MenuStrip), the tool strip above this page and the status strip below it are set on the Form and change while the app runs: try the Plus and Bold buttons, the Menu drop-down and the View menu's Status bar."),
		widget.Separator(),
		toggle, state,
	)
}

func dialogs(w *widget.Form) widget.Control {
	result := widget.Label("Results show here").Muted()
	show := func(format string, a ...any) { result.SetText(fmt.Sprintf(format, a...)) }
	return widget.VBox(8).Add(
		widget.Label("Dialogs").TextSize(widget.XL).Bold(),
		widget.FlowPanel(8).Add(
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
			widget.Button("Color dialog").OnClick(func() {
				cd := widget.ColorDialog().SetColor(ui.Hex("#2563eb"))
				ok := cd.ShowDialog(w)
				show("ColorDialog: %v %v", cd.Color(), ok)
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
				widget.MenuItem{Text: "Copy", Icon: lucide(lucideIcons[0]), OnClick: func() { widget.Clipboard.SetText("Right-click me"); show("Copied") }},
				widget.MenuItem{Text: "Delete…", Icon: lucide(lucideIcons[2]), OnClick: func() { show("Delete: %v", widget.Confirm(w, widget.Warning, "Delete?", "")) }},
			),
		),
	)
}

// customDialog runs a dialog of its own, with a dialog over it.
func customDialog(w *widget.Form) {
	var d *widget.Dialog
	d = widget.NewDialog(w, "New account", "A dialog of your own", 420, 260, widget.VBox(8).Add(
		widget.TextBox().SetPlaceholder("Name"),
		widget.Button("Nested dialog").OnClick(func() {
			widget.Alert(d.Form, widget.Info, "Nested", "A dialog over a dialog.")
		}),
		widget.ButtonBox().Add(widget.Button("Close").Suggested().OnClick(func() { d.Close() })),
	))
	d.Run()
}

func extras() widget.Control {
	theme := widget.ComboBox(
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
		widget.HBox(8).Add(widget.PictureBox(swatch(color.RGBA{230, 80, 60, 255}), 48, 48), widget.PictureBox(swatch(color.RGBA{60, 120, 230, 255}), 96, 48)),
		widget.Label("").Rich(ui.Span{Text: "Rich text: "}, ui.Span{Text: "bold", Weight: 700}, ui.Span{Text: ", "}, ui.Span{Text: "italic", Italic: true}),
		widget.HBox(8).Add(widget.Label("Sectioned combo").VAlign(widget.Center), widget.ComboBox().
			AddSection("Fruits", swatch(color.RGBA{230, 160, 40, 255}), widget.Option{Value: "apple", Text: "Apple"}, widget.Option{Value: "pear", Text: "Pear"}).
			AddSection("Vegetables", swatch(color.RGBA{60, 160, 80, 255}), widget.Option{Value: "kale", Text: "Kale"})),
		clockRow(),
	)
}

// clockRow counts the ticks of a widget.Timer, which a button starts and
// stops.
func clockRow() widget.Control {
	ticks := widget.Label("Ticks: 0")
	n := 0
	timer := widget.Timer(time.Second).OnTick(func() {
		n++
		ticks.SetText(fmt.Sprintf("Ticks: %d", n))
	})
	var toggle *widget.ButtonWidget
	toggle = widget.Button("Start timer").OnClick(func() {
		if timer.Enabled() {
			timer.Stop()
			toggle.SetText("Start timer")
			return
		}
		timer.Start()
		toggle.SetText("Stop timer")
	})
	return widget.HBox(8).Add(toggle, ticks.VAlign(widget.Center))
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

func layout() widget.Control {
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
		widget.Separator(),
		widget.Label("Fields").TextSize(widget.XL).Bold(),
		widget.Fields().
			Row("Name", widget.TextBox().SetPlaceholder("Ada Lovelace")).
			Row("Email address", widget.TextBox().SetPlaceholder("ada@example.com")).
			Row("Country", widget.ComboBox(widget.Option{Value: "vn", Text: "Vietnam"}, widget.Option{Value: "us", Text: "United States"}).SetValue("vn")),
		widget.Fields().Stacked().
			Row("Notes", widget.TextBox().SetPlaceholder("Stacked: the label sits above its control")),
	)
}

func inputs() widget.Control {
	var name *widget.TextBoxWidget
	greeting := widget.Label("Type a name")
	// Bound to a value kept elsewhere: an edit calls set, a program setter does not.
	var stored atomic.Value
	stored.Store("kept elsewhere")
	shown := widget.Label("Stored: kept elsewhere")
	bound := widget.Bind(widget.TextBox(),
		func() string { return stored.Load().(string) },
		func(v string) { stored.Store(v); shown.SetText("Stored: " + v) })
	return widget.VBox(8).Add(
		bound, shown,
		widget.TextBox().SetPlaceholder("Name").AssignTo(&name).OnChange(func() {
			greeting.SetText("Hello, " + name.Text())
		}),
		greeting,
		widget.TextBox().SetPlaceholder("Password").Password(),
		widget.TextBox().SetPlaceholder("At most 8 characters").MaxLength(8),
		widget.TextBox().SetPlaceholder("Search").Icon(searchIcon()),
		widget.MultilineTextBox().SetPlaceholder("Notes").SetSize(0, 100),
		widget.CheckBox("Remember me"),
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

func choices() widget.Control {
	progress := widget.ProgressBar().SetFraction(0.3).SetText("30%")
	spin := widget.NumericUpDown(0, 100, 1).SetValue(30)
	spin.OnChange(func() {
		progress.SetFraction(spin.Value() / 100).SetText(fmt.Sprintf("%d%%", spin.Int()))
	})
	return widget.VBox(8).Add(
		spin, progress,
		widget.RangeSpin(0, 60, 0, 600, 1, "to", "seconds").SetValues(5, 30),
		widget.ComboBox(widget.Option{Value: "vn", Text: "Vietnam"}, widget.Option{Value: "us", Text: "United States"}).SetValue("vn"),
		widget.CheckBoxGroup(true, widget.Option{Value: "a", Text: "Apple"}, widget.Option{Value: "b", Text: "Banana"}),
		widget.RadioGroup(true, widget.Option{Value: "s", Text: "Small"}, widget.Option{Value: "l", Text: "Large"}),
		widget.HBox(8).Add(widget.RadioButton("Red").SetChecked(true), widget.RadioButton("Green"), widget.RadioButton("Blue")),
		widget.ComboBox(widget.Option{Value: "vn", Text: "Vietnam"}, widget.Option{Value: "us", Text: "United States"}).Editable().SetText("Any country"),
		widget.TagSelect(widget.Option{Value: "go", Text: "Go"}, widget.Option{Value: "ui", Text: "UI"}, widget.Option{Value: "gtk", Text: "GTK"}),
		widget.Spinner().Start(),
		widget.DateTimePicker(),
		widget.MonthCalendar(),
	)
}

func containers() widget.Control {
	grid := widget.Grid(8, 8).
		Attach(widget.Label("A"), 0, 0, 1, 1).
		Attach(widget.Label("B"), 1, 0, 1, 1).
		Attach(widget.Label("C spans two"), 0, 1, 2, 1)
	flow := widget.FlowPanel(6)
	for i := range 12 {
		flow.Add(widget.Button(fmt.Sprintf("chip %d", i)))
	}
	return widget.Sidebar().
		Add("frame", "GroupBox and grid", widget.VBox(8).Add(widget.GroupBox("Frame").Add(grid))).
		Add("card", "Card", widget.Card().SetTitle(widget.Label("Card title").Bold()).SetBody(widget.Label("Card body"))).
		Add("flow", "FlowPanel", flow).
		Add("scroll", "ScrollPanel", widget.ScrollPanel(longList())).
		Add("split", "SplitContainer", widget.SplitContainer(
			widget.VBox(4).Add(widget.Label("Files"), widget.Button("Open")),
			widget.SplitContainer(widget.Label("Editor"), widget.Label("Console")).Stacked(),
		).SetSplitterDistance(180))
}

func longList() widget.Control {
	b := widget.VBox(4)
	for i := range 100 {
		b.Add(widget.Label(fmt.Sprintf("Row %d", i)))
	}
	return b
}

type account struct {
	id     int
	name   string
	status string
	score  int
}

// table shows 100,000 accounts; Live changes a hundred of them every
// 100 ms from a goroutine of its own.
func table() widget.Control {
	rows := make([]account, 100_000)
	for i := range rows {
		rows[i] = account{i, fmt.Sprintf("acct%06d", i), "ok", i % 997}
	}
	status := func(a *account) ui.Color {
		if a.status == "failed" {
			return ui.RGB(220, 38, 38)
		}
		return ui.RGB(22, 163, 74)
	}
	var t *widget.DataGridViewWidget[account, int]
	t = widget.DataGridView(func(a *account) int { return a.id },
		widget.Column[account]{Title: "ID", Text: func(a *account) string { return strconv.Itoa(a.id) },
			Compare: func(a, b *account) int { return cmp.Compare(a.id, b.id) }, Width: 90, Align: widget.End, Sortable: true},
		widget.Column[account]{Title: "Name", Text: func(a *account) string { return a.name },
			Width: 180, Sortable: true, Filterable: true},
		widget.Column[account]{Title: "Status", Text: func(a *account) string { return a.status },
			Color: status, Width: 100, Sortable: true, Filterable: true},
		widget.Column[account]{Title: "Score", Text: func(a *account) string { return strconv.Itoa(a.score) },
			Compare: func(a, b *account) int { return cmp.Compare(a.score, b.score) }, Align: widget.End, Sortable: true},
	).SetRows(rows).CheckboxColumn().IndexColumn().BottomPanel().
		ContextMenu(widget.MenuItem{Text: "Delete", OnClick: func() { t.DeleteRows(t.Selected()...) }}).
		OnKey(ui.KeySpace, ui.Shift, func() { t.AutoCheck() }).
		Expand()
	activated := widget.Label("Double-click a row").Muted()
	t.OnActivate(func(id int) { activated.SetText(fmt.Sprintf("Activated %d", id)) })
	var stop chan struct{}
	live := widget.Toggle("Live").OnChange(func() {
		if stop != nil {
			close(stop)
			stop = nil
			return
		}
		stop = make(chan struct{})
		go liveUpdates(t, stop)
	})
	return widget.VBox(8).Add(widget.HBox(8).Add(live, activated)).Pack(t, true, true).Expand()
}

// liveUpdates fails or fixes a hundred accounts every 100 ms until stop
// closes.
func liveUpdates(t *widget.DataGridViewWidget[account, int], stop <-chan struct{}) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for n := 0; ; n++ {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		for i := range 100 {
			t.UpdateRow((n*7919+i*104729)%100_000, func(a *account) {
				a.score = (a.score + 7) % 997
				if a.status == "ok" {
					a.status = "failed"
				} else {
					a.status = "ok"
				}
			})
		}
	}
}

// searchIcon is a magnifying glass drawn in code: a ring and a handle.
func searchIcon() *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	ink := color.RGBA{120, 120, 120, 255}
	for y := range 32 {
		for x := range 32 {
			dx, dy := float64(x-13), float64(y-13)
			d := math.Hypot(dx, dy)
			ring := d > 7.5 && d < 10
			handle := x > 19 && x-y > -1 && x-y < 3 && x < 29
			if ring || handle {
				img.SetRGBA(x, y, ink)
			}
		}
	}
	return ui.NewBitmap(img)
}
