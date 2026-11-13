package skin

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

type galleryRow struct{ ID, Name, State string }

type galleryNode struct{ ID string }

// gallery returns one form with every control of package widget that draws,
// in its states: the test's own checklist of what a skin must reach. extra
// goes at the top of its content. Not in it: pictures, web views and the
// color dialog (bitmap content), the message boxes' icons and dialogs
// (windows of their own; Task 5 brings their icons onto the skin), and
// timers (nothing to draw).
func gallery(extra ...widget.Control) *widget.Form {
	f := widget.NewForm("Gallery", 1100, 1500)
	f.UseHeaderBar("Every control").PackEnd(widget.Button("Header action"))
	f.ToolStrip(widget.ToolStrip(widget.ToolButton("Cut"), widget.ToolButton("Copy"), widget.Separator(), widget.ToolButton("Paste")))
	f.StatusStrip(widget.StatusStrip(widget.StatusLabel("Ready"), widget.Spring(), widget.StatusProgress()))

	rows := []galleryRow{{"1", "Ada", "Running"}, {"2", "Grace", "Done"}, {"3", "Edsger", "Failed"}}
	cols := []widget.Column[galleryRow]{
		{Title: "Name", Text: func(r *galleryRow) string { return r.Name }},
		{Title: "State", Text: func(r *galleryRow) string { return r.State }},
	}
	grid := widget.DataGridView(func(r *galleryRow) string { return r.ID }, cols...).SetRows(rows).SetSize(0, 140)
	list := widget.ListView(func(r *galleryRow) string { return r.ID }, cols...).SetRows(rows).SetSize(0, 120)
	tree := widget.TreeView(func(n *galleryNode) string { return n.ID }, func(n *galleryNode) string { return n.ID }).
		SetRoots([]galleryNode{{"Root"}, {"Other"}}).SetHasChildren("Root", true).SetSize(0, 90)
	listBox := widget.ListBox(func(r *galleryRow) string { return r.ID }, func(r *galleryRow) string { return r.Name }).
		SetItems(rows).SetSize(0, 90)
	checked := widget.CheckedListBox(func(r *galleryRow) string { return r.ID }, func(r *galleryRow) string { return r.Name }).
		SetItems(rows).SetSize(0, 90)
	checked.SetChecked([]string{"2"}, true)

	chart := widget.Chart(widget.LineChart).SetLabels("Mon", "Tue", "Wed").
		SetSeries(widget.ChartSeries{Name: "A", Values: []float64{1, 3, 2}}, widget.ChartSeries{Name: "B", Values: []float64{2, 1, 3}})
	chart.SetAccessibleName("chart")
	chart.SetSize(0, 160)

	errorBox := widget.TextBox()
	errors := widget.ErrorProvider()
	errors.SetError(errorBox, "Required")

	tabs := widget.TabControl().AddPage("First", widget.Label("First page")).AddPage("Second", widget.Label("Second page"))
	sidebar := widget.Sidebar()
	sidebar.Add("mail", "Mail", widget.Label("Mail page"))
	sidebar.Add("notes", "Notes", widget.Label("Notes page"))
	sidebar.SetSize(0, 120)
	fields := widget.Fields()
	fields.Section("Section")
	fields.Row("Field", widget.TextBox())

	progress := widget.ProgressBar()
	progress.SetFraction(0.4)
	calendar := widget.MonthCalendar()
	calendar.SetValue(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	radio := widget.RadioButton("Radio on")
	radio.SetChecked(true)
	box := widget.CheckBox("Checked")
	box.SetChecked(true)
	sw := widget.Switch()
	sw.SetOn(true)
	toggle := widget.Toggle("Toggle on")
	toggle.SetActive(true)
	track := widget.TrackBar(0, 10)
	track.SetValue(4)

	group := widget.GroupBox("Group box")
	group.Add(widget.Label("Inside a group"))
	card := widget.Card().SetTitle(widget.Label("Card title")).SetBody(widget.Label("Card body"))
	flow := widget.FlowPanel(4)
	flow.Add(widget.Label("Flow one"), widget.Label("Flow two"))

	left := widget.VBox(8).Add(
		widget.MenuBar(widget.MenuItem{Text: "File"}, widget.MenuItem{Text: "Edit"}),
		widget.HBox(8).Add(widget.Button("Plain"), widget.Button("Disabled").SetDisabled(true),
			widget.DropDownButton("Drop", widget.MenuItem{Text: "One"}), widget.MenuButton("Menu"), widget.LinkLabel("Link", "https://example.com")),
		widget.HBox(8).Add(toggle, widget.Toggle("Toggle off"), sw, widget.Switch()),
		widget.HBox(8).Add(box, widget.CheckBox("Unchecked"), widget.CheckBox("Off").SetDisabled(true), radio, widget.RadioButton("Radio off")),
		widget.CheckBoxGroup(true, widget.Option{Value: "a", Text: "Alpha"}, widget.Option{Value: "b", Text: "Beta"}),
		widget.RadioGroup(true, widget.Option{Value: "a", Text: "One"}, widget.Option{Value: "b", Text: "Two"}),
		widget.TextBox().Focus(),
		widget.TextBox().SetDisabled(true),
		widget.MaskedTextBox("000-000"),
		widget.MultilineTextBox().SetSize(0, 60),
		widget.ComboBox(widget.Option{Value: "a", Text: "Alpha"}, widget.Option{Value: "b", Text: "Beta"}),
		widget.ComboBox(widget.Option{Value: "a", Text: "Alpha"}).Editable(),
		widget.TagSelect(widget.Option{Value: "a", Text: "Alpha"}, widget.Option{Value: "b", Text: "Beta"}).SetValues("a"),
		widget.HBox(8).Add(widget.NumericUpDown(0, 10, 1), widget.DomainUpDown("North", "South")),
		widget.DateTimePicker(),
		widget.RangeSpin(0, 10, 0, 10, 1, "to", "days"),
		track,
		progress,
		widget.Spinner(),
		errorBox, // its error glyph beside it
		widget.Separator(),
		widget.Label("A label"),
		fields,
	)
	right := widget.VBox(8).Add(
		grid, list, tree,
		widget.HBox(8).Add(listBox, checked),
		tabs, sidebar, group, card, flow,
		widget.SplitContainer(widget.Label("Split left"), widget.Label("Split right")).SetSize(0, 60),
		widget.ScrollPanel(widget.Label("Scrolled")).SetSize(0, 40),
		calendar,
		chart,
	)
	content := widget.VBox(8).Add(extra...)
	content.Add(widget.HBox(16).Add(left, right))
	f.SetContent(content)
	return f
}

// galleryStates puts the gallery's controls in the states a still frame can
// show: a chosen grid row, and the pointer over the plain button.
func galleryStates(tt *widget.Tester) {
	tt.Find(widget.ByRole(ui.RoleRow).Label("Grace").Nth(0)).Click() // the grid's; the list view's is next
	r := tt.Find(widget.ByLabel("Plain")).Bounds()
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
}
