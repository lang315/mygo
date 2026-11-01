package widget

import (
	"github.com/egoist/mygo/ui"
	"sync/atomic"
	"testing"
	"time"
)

// A setter does not run the OnChange of the control it sets: OnChange is for
// what the user does (docs/ui/widget.md, "Handlers"). Each control shows it.
func TestSettersDoNotRunOnChange(t *testing.T) {
	var n atomic.Int64
	count := func() { n.Add(1) }
	cases := []struct {
		name string
		c    Control
		set  func()
	}{}
	add := func(name string, c Control, set func()) {
		cases = append(cases, struct {
			name string
			c    Control
			set  func()
		}{name, c, set})
	}
	tb := TextBox().OnChange(count)
	add("TextBox.SetText", tb, func() { tb.SetText("x") })
	ml := MultilineTextBox().OnChange(count)
	add("MultilineTextBox.SetText", ml, func() { ml.SetText("x") })
	cb := CheckBox("c").OnChange(count)
	add("CheckBox.SetChecked", cb, func() { cb.SetChecked(true) })
	sw := Switch().OnChange(count)
	add("Switch.SetOn", sw, func() { sw.SetOn(true) })
	r1, r2 := RadioButton("a").OnChange(count), RadioButton("b").OnChange(count)
	add("RadioButton.SetChecked", VBox(2).Add(r1, r2), func() { r2.SetChecked(true) })
	co := ComboBox(Option{Value: "a", Text: "A"}, Option{Value: "b", Text: "B"}).OnChange(count)
	add("ComboBox.SetValue", co, func() { co.SetValue("b") })
	nu := NumericUpDown(0, 10, 1).OnChange(count)
	add("NumericUpDown.SetValue", nu, func() { nu.SetValue(4) })
	rs := RangeSpin(0, 10, 0, 10, 1, "to", "").OnChange(count)
	add("RangeSpin.SetValues", rs, func() { rs.SetValues(2, 5) })
	tr := TrackBar(0, 10).OnChange(count)
	add("TrackBar.SetValue", tr, func() { tr.SetValue(4) })
	dt := DateTimePicker().OnChange(count)
	add("DateTimePicker.SetValue", dt, func() { dt.SetValue(time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)) })
	cg := CheckBoxGroup(true, Option{Value: "a", Text: "A"}, Option{Value: "b", Text: "B"}).OnChange(count)
	add("CheckBoxGroup.SetValues", cg, func() { cg.SetValues("b") })
	rg := RadioGroup(true, Option{Value: "a", Text: "A"}, Option{Value: "b", Text: "B"}).OnChange(count)
	add("RadioGroup.SetValue", rg, func() { rg.SetValue("b") })
	mk := MaskedTextBox("00-00").OnChange(count)
	add("MaskedTextBox.SetText", mk, func() { mk.SetText("1234") })
	di := DomainUpDown("a", "b", "c").OnChange(count)
	add("DomainUpDown.SetSelectedIndex", di, func() { di.SetSelectedIndex(2) })
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n.Store(0)
			tt := tester(t, c.c)
			tt.Frame()
			tt.Frame()
			flushEvents(t)
			n.Store(0)
			c.set()
			tt.Frame()
			tt.Frame()
			flushEvents(t)
			if got := n.Load(); got != 0 {
				t.Fatalf("OnChange ran %d times for a setter", got)
			}
		})
	}
}

// The text of a field is not among a tester's Texts; its node holds it.
func TestTesterTextsLeaveOutFieldValues(t *testing.T) {
	tt := tester(t, VBox(2).Add(TextBox().SetText("in a field"), Label("a label"), MultilineTextBox().SetText("in an area")))
	if !tt.HasText("a label") || tt.HasText("in a field") || tt.HasText("in an area") {
		t.Fatalf("texts %q", tt.Texts())
	}
	tw := NewTester(t, TextBox().SetText("in a field"), 300, 100)
	if got := tw.Find(ByRole(ui.RoleTextField)).Value(); got != "in a field" {
		t.Fatalf("value %q", got)
	}
}

// ByLabel counts a control once, though Nodes lists a sidebar entry and the
// text it shows, named alike; ByRole(RoleText) still finds the text.
func TestTesterByLabelCountsASidebarEntryOnce(t *testing.T) {
	var picked string
	sb := Sidebar().Add("a", "Mail", Label("inbox")).Add("b", "Cal", Label("today"))
	tt := NewTester(t, sb, 600, 300)
	same := 0
	for _, n := range tt.Nodes() {
		if n.Label == "Mail" {
			same++
		}
	}
	if same != 2 {
		t.Fatalf("%d nodes named Mail; the test needs the entry and its text", same)
	}
	if n := len(tt.FindAll(ByLabel("Mail"))); n != 1 {
		t.Fatalf("%d matches of the entry Mail", n)
	}
	tt.Find(ByLabel("Cal")).Click()
	picked = sb.Selected()
	if picked != "b" {
		t.Fatalf("selected %q after a click on Cal", picked)
	}
	if n := len(tt.FindAll(ByRole(ui.RoleText).Label("Cal"))); n != 1 {
		t.Fatalf("%d texts Cal", n)
	}
}
