package skin

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

type sizeRow struct{ ID, Name string }

type sizeItem struct{ ID string }

// sizeForm holds one of each widget TestCalmSizesWidgets measures.
func sizeForm() widget.Control {
	grid := widget.DataGridView(func(r *sizeRow) string { return r.ID },
		widget.Column[sizeRow]{Title: "Name", Text: func(r *sizeRow) string { return r.Name }},
	).SetRows([]sizeRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}).SetSize(0, 180)
	tree := widget.TreeView(func(n *sizeItem) string { return n.ID }, func(n *sizeItem) string { return n.ID }).
		SetRoots([]sizeItem{{"Leaf"}}).SetSize(0, 80)
	return widget.VBox(8).Add(
		widget.Button("OK"),
		widget.TextBox(),
		widget.ComboBox(widget.Option{Value: "a", Text: "A"}),
		widget.NumericUpDown(0, 10, 1),
		widget.DateTimePicker(),
		widget.CheckBox(""),
		grid,
		tree,
	)
}

// widgetSizes measures the heights of the widgets of sizeForm.
func widgetSizes(t *testing.T, tt *widget.Tester) map[string]float32 {
	t.Helper()
	got := map[string]float32{}
	first := func(name string, role ui.Role, label string, any bool) ui.TestNode {
		t.Helper()
		for _, n := range tt.Nodes() {
			if n.Role == role && (any || n.Label == label) {
				return n
			}
		}
		t.Fatalf("no %s (%v %q)", name, role, label)
		return ui.TestNode{}
	}
	got["Button"] = first("Button", ui.RoleButton, "OK", false).Bounds.H
	got["TextBox"] = first("TextBox", ui.RoleTextField, "", true).Bounds.H
	got["ComboBox"] = first("ComboBox", ui.RolePopUpButton, "", false).Bounds.H
	// NumericUpDown's field is no node of its own: its arrows fill it, inside
	// its border of 1.
	up, down := first("Increase", ui.RoleButton, "Increase", false).Bounds, first("Decrease", ui.RoleButton, "Decrease", false).Bounds
	got["NumericUpDown"] = down.Y + down.H - up.Y + 2
	for _, n := range tt.Nodes() {
		if n.Role == ui.RolePopUpButton && n.Label != "" { // the combo box's is unnamed
			got["DateTimePicker date"] = n.Bounds.H
		}
	}
	got["DateTimePicker time"] = first("time", ui.RoleGroup, "", true).Bounds.H
	// Without a label, the check box is its box.
	box := first("CheckBox", ui.RoleCheckBox, "", true).Bounds
	got["CheckBox box"] = box.H // its row stretches across the box: the width is the form's
	got["DataGridView header"] = first("header", ui.RoleColumnHeader, "Name", false).Bounds.H
	got["DataGridView row"] = first("row", ui.RoleRow, "Grace", false).Bounds.H
	got["TreeView row"] = first("tree row", ui.RoleTreeItem, "Leaf", false).Bounds.H
	return got
}

// After Use(Calm) the widgets take Calm's metrics, and after Use(nil) their
// plain sizes back. Plain, on macOS at f9ec6f97: 29.31 high for the
// controls, a check box of 16, rows and header 32.
func TestCalmSizesWidgets(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	tt := widget.NewTester(t, sizeForm(), 600, 700)
	tt.Frame()
	plain := widgetSizes(t, tt)
	Use(Calm)
	tt.Frame()
	got := widgetSizes(t, tt)
	for name, want := range map[string]float32{
		"Button": 36, "TextBox": 36, "ComboBox": 36, "NumericUpDown": 36, "DateTimePicker date": 36, "DateTimePicker time": 36,
		"CheckBox box": 15, "DataGridView header": 30, "DataGridView row": 30, "TreeView row": 30,
	} {
		if !same(got[name], want) {
			t.Errorf("with Calm, %s is %v, want %v", name, got[name], want)
		}
	}
	Use(nil)
	tt.Frame()
	for name, h := range widgetSizes(t, tt) {
		if !same(h, plain[name]) {
			t.Errorf("after Use(nil), %s is %v, want its plain %v", name, h, plain[name])
		}
	}
	for name, h := range plain {
		if same(h, got[name]) {
			t.Errorf("%s is %v plain and with Calm: the test does not see the skin", name, h)
		}
	}
}

// same reports whether a and b are the same size, to rounding.
func same(a, b float32) bool { return a-b < 0.01 && b-a < 0.01 }

// An app's own size wins over the skin's metrics in either direction, and a
// MinHeight the app sets still applies.
func TestWidgetSetSizeWinsOverMetrics(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	Use(Calm)
	for _, c := range []struct {
		name string
		b    *widget.ButtonWidget
		want float32
	}{
		{"SetSize(0, 24)", widget.Button("OK").SetSize(0, 24), 24},
		{"SetSize(0, 50)", widget.Button("OK").SetSize(0, 50), 50},
		{"SetSize(0, 24) and MinHeight(40)", widget.Button("OK").SetSize(0, 24).Apply(widget.NewStyle().MinHeight(40)), 40},
		{"none", widget.Button("OK"), 36},
	} {
		tt := widget.NewTester(t, widget.VBox(0).Add(c.b), 300, 200)
		tt.Frame()
		if h := tt.Find(widget.ByLabel("OK")).Bounds().H; !same(h, c.want) {
			t.Errorf("%s with Calm: the button is %v, want %v", c.name, h, c.want)
		}
	}
	// Widgets of several parts: SetSize sizes their root; the parts inside,
	// logged for the review, are within it (an editable combo's input keeps
	// its padding, as before Metrics). DateTimePicker has its own test.
	for _, c := range []struct {
		name string
		w    widget.Control
		part widget.Query
	}{
		{"NumericUpDown", widget.NumericUpDown(0, 10, 1).SetSize(0, 24), widget.ByLabel("Increase")},
		{"editable ComboBox", widget.ComboBox(widget.Option{Value: "a", Text: "A"}).Editable().SetSize(0, 24), widget.ByRole(ui.RoleComboBox)},
	} {
		tt := widget.NewTester(t, widget.VBox(0).Add(c.w, widget.Label("below")), 300, 200)
		tt.Frame()
		below := tt.Find(widget.ByLabel("below")).Bounds().Y
		part := tt.Find(c.part).Bounds()
		t.Logf("%s, SetSize(0, 24) with Calm: the next control starts at %v; a part inside is %v high", c.name, below, part.H)
	}
}

// A DateTimePicker's own height wins over the skin's metrics: its date and
// time fields fill it, inside its bounds. Without a skin they keep their
// size, as before Metrics.
func TestDateTimePickerSetSizeWinsOverMetrics(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	// fields returns the heights of a picker's date and time fields, with the
	// app's height h (none when 0), and where the control below it starts.
	fields := func(h float32) (date, tm ui.Rect, below float32) {
		p := widget.DateTimePicker()
		if h > 0 {
			p.SetSize(0, h)
		}
		tt := widget.NewTester(t, widget.VBox(0).Add(p, widget.Label("below")), 300, 200)
		tt.Frame()
		for _, n := range tt.Nodes() {
			if n.Role == ui.RolePopUpButton {
				date = n.Bounds
			}
		}
		return date, tt.Find(widget.ByRole(ui.RoleGroup)).Bounds(), tt.Find(widget.ByLabel("below")).Bounds().Y
	}
	pd, pt, _ := fields(0)
	d, tm, _ := fields(24)
	if !same(d.H, pd.H) || !same(tm.H, pt.H) {
		t.Errorf("without a skin, SetSize(0, 24): the fields are %v and %v high, want %v and %v as without SetSize", d.H, tm.H, pd.H, pt.H)
	}
	Use(Calm)
	d, tm, below := fields(24)
	for name, r := range map[string]ui.Rect{"date": d, "time": tm} {
		if !same(r.H, 24) || r.Y < 0 || r.Y+r.H > below+0.01 {
			t.Errorf("with Calm, SetSize(0, 24): the %s field is %v, want 24 high inside the picker (next control at %v)", name, r, below)
		}
	}
}

// Without a skin, an app's height below a control's own least height keeps
// that least height, as before Metrics: a MultilineTextBox's 80.
func TestSetSizeWithoutSkinKeepsTheLeastHeight(t *testing.T) {
	tt := widget.NewTester(t, widget.VBox(0).Add(widget.MultilineTextBox().SetSize(0, 40).SetAccessibleName("notes")), 300, 200)
	tt.Frame()
	if h := tt.Find(widget.ByLabel("notes")).Bounds().H; !same(h, 80) {
		t.Errorf("a MultilineTextBox with SetSize(0, 40) is %v high, want its least, 80", h)
	}
}
