package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// The studio's window, in DIPs.
const (
	formWidth  = 1100
	formHeight = 900
)

// palettes are the palettes of the palette combo box, by the value it holds.
var palettes = map[string]*skin.Palette{
	"cham": skin.Cham, "indigo": skin.Indigo, "ink": skin.Ink, "teal": skin.Teal,
}

// newStudio makes the window: a header bar, a tool strip with three combo
// boxes that set the skin, its palette and the appearance at once, a status
// strip, and one tab per family of controls.
func newStudio() *widget.Form {
	f := widget.NewForm("Skin studio", formWidth, formHeight)
	f.UseHeaderBar("Every control in every skin").PackEnd(widget.Button("Header action"))
	f.ToolStrip(widget.ToolStrip(switches()...))
	f.StatusStrip(widget.StatusStrip(widget.StatusLabel("Ready"), widget.Spring(), widget.StatusProgress().SetFraction(0.4)))
	f.SetContent(widget.TabControl().
		AddPage("Buttons and toggles", buttons()).
		AddPage("Text and choices", text()).
		AddPage("Numbers and dates", numbers()).
		AddPage("Grid", gridPage()).
		AddPage("Lists and trees", lists()).
		AddPage("Containers", containers()).
		AddPage("Chart and progress", chart()).
		AddPage("Dialogs and messages", dialogs(f)).
		Padding(4).Expand())
	return f
}

// switches returns the tool strip's combo boxes: skin (None, Calm), palette
// and appearance. A choice applies at once; the palette waits, kept, while no
// skin is on.
func switches() []widget.Control {
	useSkin, usePalette := "none", "cham"
	apply := func() {
		if useSkin == "none" {
			skin.Use(nil)
			return
		}
		skin.Use(skin.Calm.WithPalette(palettes[usePalette]))
	}
	choose := func(name string, set *string, opts ...widget.Option) *widget.ComboBoxWidget {
		c := widget.ComboBox(opts...).SetValue(*set).SetAccessibleName(name)
		c.OnChange(func() { *set = c.Value(); apply() })
		return c
	}
	mode := widget.ComboBox(
		widget.Option{Value: string(widget.ThemeSystem), Text: "System"},
		widget.Option{Value: string(widget.ThemeLight), Text: "Light"},
		widget.Option{Value: string(widget.ThemeDark), Text: "Dark"},
	).SetValue(string(widget.ThemeSystem)).SetAccessibleName("Mode")
	mode.OnChange(func() { widget.SetTheme(widget.Theme(mode.Value())) })
	return []widget.Control{
		choose("Skin", &useSkin, widget.Option{Value: "none", Text: "None"}, widget.Option{Value: "calm", Text: "Calm"}),
		choose("Palette", &usePalette,
			widget.Option{Value: "cham", Text: skin.Cham.Name()}, widget.Option{Value: "indigo", Text: skin.Indigo.Name()},
			widget.Option{Value: "ink", Text: skin.Ink.Name()}, widget.Option{Value: "teal", Text: skin.Teal.Name()}),
		mode,
	}
}

func option(pairs ...string) []widget.Option {
	var out []widget.Option
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, widget.Option{Value: pairs[i], Text: pairs[i+1]})
	}
	return out
}

// title is a section's heading.
func title(s string) widget.Control { return widget.Label(s).TextSize(widget.LG).Bold() }

func buttons() widget.Control {
	sw := widget.Switch()
	sw.SetOn(true)
	toggle := widget.Toggle("Toggle on")
	toggle.SetActive(true)
	box := widget.CheckBox("Checked")
	box.SetChecked(true)
	radio := widget.RadioButton("Radio on")
	radio.SetChecked(true)
	return widget.VBox(12).Add(
		title("Buttons"),
		widget.HBox(8).Add(
			widget.Button("Plain"), widget.Button("Suggested").Suggested(), widget.Button("Destructive").Destructive(),
			widget.Button("Disabled").SetDisabled(true),
			widget.DropDownButton("Drop down", widget.MenuItem{Text: "One"}, widget.MenuItem{Text: "Two"}),
			widget.MenuButton("Menu").SetMenu(widget.MenuItem{Text: "Name"}, widget.MenuItem{Text: "Date", Checked: true}),
			widget.LinkLabel("Link", "https://github.com/lang315/mygo"),
		),
		widget.HBox(8).Add(widget.Button("Đăng nhập lại tài khoản đã khoá"), widget.Button("Xuất danh sách ra Excel")),
		title("Toggles"),
		widget.HBox(12).Add(toggle, widget.Toggle("Toggle off"), sw, widget.Switch(), widget.Switch().SetDisabled(true)),
		widget.HBox(12).Add(box, widget.CheckBox("Unchecked"), widget.CheckBox("Disabled").SetDisabled(true),
			radio, widget.RadioButton("Radio off")),
		widget.HBox(32).Add(
			widget.CheckBoxGroup(true, option("a", "Alpha", "b", "Beta", "c", "Gamma")...),
			widget.RadioGroup(true, option("a", "One", "b", "Two", "c", "Three")...),
		),
	)
}

func text() widget.Control {
	invalid := widget.TextBox().SetAccessibleName("Required field")
	errors := widget.ErrorProvider()
	errors.SetError(invalid, "Required")
	fields := widget.Fields().SettingsRows()
	fields.Section("Settings rows")
	fields.Row("Name", widget.TextBox().SetPlaceholder("Ada Lovelace"))
	fields.Row("Country", widget.ComboBox(option("vn", "Vietnam", "us", "United States")...).SetValue("vn"))
	return widget.VBox(8).Add(
		title("Text"),
		widget.TextBox().SetPlaceholder("A text box"),
		widget.TextBox().SetPlaceholder("Password").Password(),
		widget.TextBox().SetText("Disabled").SetDisabled(true),
		invalid,
		widget.MaskedTextBox("000-000"),
		widget.MultilineTextBox().SetPlaceholder("Notes").SetSize(0, 80),
		title("Choices"),
		widget.ComboBox(option("a", "Alpha", "b", "Beta")...).SetValue("a"),
		widget.ComboBox(option("a", "Alpha", "b", "Beta")...).Editable().SetText("Editable"),
		widget.TagSelect(option("go", "Go", "ui", "UI", "gtk", "GTK")...).SetValues("go"),
		fields,
	)
}

func numbers() widget.Control {
	calendar := widget.MonthCalendar()
	calendar.SetValue(time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))
	track := widget.TrackBar(0, 10)
	track.SetValue(4)
	return widget.VBox(8).Add(
		title("Numbers"),
		widget.HBox(8).Add(widget.NumericUpDown(0, 100, 1).SetValue(30), widget.DomainUpDown("North", "South", "East")),
		widget.RangeSpin(0, 60, 0, 600, 1, "to", "seconds").SetValues(5, 30),
		track,
		title("Dates"),
		widget.DateTimePicker(),
		calendar,
	)
}

// account is a row of the grid.
type account struct {
	ID           int
	Name, Status string
}

// accounts returns 40 accounts with a status each.
func accounts() []account {
	status := []string{"Running", "Done", "Failed", "Waiting"}
	rows := make([]account, 40)
	for i := range rows {
		rows[i] = account{ID: i + 1, Name: fmt.Sprintf("Tài khoản %02d", i+1), Status: status[i%len(status)]}
	}
	return rows
}

func columns() []widget.Column[account] {
	return []widget.Column[account]{
		{Title: "ID", Text: func(a *account) string { return fmt.Sprint(a.ID) }, Width: 80, Align: widget.End},
		{Title: "Name", Text: func(a *account) string { return a.Name }, Width: 260, Sortable: true},
		{Title: "Status", Text: func(a *account) string { return a.Status }, Width: 140, Sortable: true},
	}
}

func gridPage() widget.Control {
	grid := widget.DataGridView(func(a *account) int { return a.ID }, columns()...).SetRows(accounts()).CheckboxColumn()
	return widget.VBox(8).Pack(grid, true, true).Expand()
}

func lists() widget.Control {
	rows := accounts()[:6]
	id := func(a *account) int { return a.ID }
	name := func(a *account) string { return a.Name }
	listBox := widget.ListBox(id, name).SetItems(rows).SetSize(0, 120)
	checked := widget.CheckedListBox(id, name).SetItems(rows).SetSize(0, 120)
	checked.SetChecked([]int{2}, true)
	details := widget.ListView(id, columns()...).SetRows(rows).CheckBoxes().SetSize(0, 170)
	icons := widget.ListView(id, columns()...).SetRows(rows).SetView(widget.ViewLargeIcon).SetSize(0, 170)
	type node struct{ path, name string }
	tree := widget.TreeView(func(n *node) string { return n.path }, func(n *node) string { return n.name }).
		SetRoots([]node{{"/docs", "Documents"}, {"/pics", "Pictures"}})
	tree.SetHasChildren("/docs", true)
	tree.SetChildren("/docs", []node{{"/docs/a", "Annual report"}, {"/docs/b", "Budget"}})
	tree.SetSize(0, 170)
	return widget.VBox(8).Add(
		widget.HBox(8).Add(listBox, checked),
		widget.HBox(8).Add(details, icons),
		tree,
	)
}

func containers() widget.Control {
	group := widget.GroupBox("Group box")
	group.Add(widget.VBox(4).Add(widget.Label("Inside a group"), widget.CheckBox("A check box")))
	card := widget.Card().SetTitle(widget.Label("Card title").Bold()).SetBody(widget.Label("Card body"))
	flow := widget.FlowPanel(6)
	for i := range 6 {
		flow.Add(widget.Button(fmt.Sprintf("Chip %d", i+1)))
	}
	sidebar := widget.Sidebar()
	sidebar.Add("mail", "Mail", widget.Label("Mail page"))
	sidebar.Add("notes", "Notes", widget.Label("Notes page"))
	sidebar.SetSize(0, 110)
	nested := widget.TabControl().AddPage("First", widget.Label("First page")).AddPage("Second", widget.Label("Second page"))
	return widget.VBox(8).Add(
		widget.MenuBar(
			widget.MenuItem{Text: "File", Children: []widget.MenuItem{{Text: "New"}, {Text: "Open"}}},
			widget.MenuItem{Text: "Edit", Children: []widget.MenuItem{{Text: "Copy"}, {Text: "Paste"}}},
		),
		widget.HBox(8).Add(group, card),
		flow,
		widget.SplitContainer(widget.Label("Split left"), widget.Label("Split right")).SetSize(0, 70),
		widget.ScrollPanel(widget.VBox(4).Add(widget.Label("Scroll panel one"), widget.Label("Scroll panel two"),
			widget.Label("Scroll panel three"), widget.Label("Scroll panel four"))).SetSize(0, 70),
		widget.HBox(8).Add(sidebar, nested),
	)
}

func chart() widget.Control {
	line := widget.Chart(widget.LineChart).SetTitle("Line").SetLabels("Mon", "Tue", "Wed").
		SetSeries(widget.ChartSeries{Name: "A", Values: []float64{1, 3, 2}}, widget.ChartSeries{Name: "B", Values: []float64{2, 1, 3}})
	line.SetSize(0, 200)
	bars := widget.Chart(widget.BarChart).SetTitle("Bars").SetLabels("Jan", "Feb", "Mar").
		SetSeries(widget.ChartSeries{Name: "2026", Values: []float64{10, 20, 30}})
	bars.SetSize(0, 200)
	pie := widget.Chart(widget.PieChart).SetTitle("Pie").SetLabels("Win", "Mac", "Linux").
		SetSeries(widget.ChartSeries{Values: []float64{60, 30, 10}})
	pie.SetSize(0, 200)
	return widget.VBox(12).Add(
		widget.HBox(12).Add(line, bars, pie),
		widget.ProgressBar().SetFraction(0.4).SetText("40%"),
		widget.ProgressBar().SetFraction(1),
		widget.Spinner().Start(),
	)
}

func dialogs(w *widget.Form) widget.Control {
	result := widget.Label("Results show here").Muted()
	show := func(format string, a ...any) { result.SetText(fmt.Sprintf(format, a...)) }
	return widget.VBox(8).Add(
		title("Message boxes"),
		widget.FlowPanel(8).Add(
			widget.Button("Info").OnClick(func() { widget.Alert(w, widget.Info, "Saved", "The file was saved."); show("Info closed") }),
			widget.Button("Warning").OnClick(func() { widget.Alert(w, widget.Warning, "Careful", "The disk is almost full."); show("Warning closed") }),
			widget.Button("Error").OnClick(func() { widget.Alert(w, widget.Error, "Failed", "The file could not be saved."); show("Error closed") }),
			widget.Button("Question").OnClick(func() {
				show("Question: %v", widget.Confirm(w, widget.Question, "Delete the file?", "It goes for good."))
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
				v, ok := widget.InputSelect(w, widget.Question, "Proxy", "Choose one", option("http", "HTTP", "socks", "SOCKS5")...)
				show("InputSelect: %q %v", v, ok)
			}),
			widget.Button("Dialog").OnClick(func() { customDialog(w); show("Dialog closed") }),
		),
		result,
	)
}

// customDialog runs a dialog of its own.
func customDialog(w *widget.Form) {
	var d *widget.Dialog
	ok := widget.Button("Save").Suggested().OnClick(func() { d.Close() })
	cancel := widget.Button("Cancel").OnClick(func() { d.Close() })
	d = widget.NewDialog(w, "New account", "A dialog of your own", 420, 260, widget.VBox(8).Add(
		widget.TextBox().SetPlaceholder("Name"),
		widget.ComboBox(option("a", "Alpha", "b", "Beta")...),
		widget.ButtonBox().Add(cancel, ok),
	))
	d.SetAcceptButton(ok)
	d.SetCancelButton(cancel)
	d.Run()
}
