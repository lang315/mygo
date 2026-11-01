package widget

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// roleLabels are the (role, label) of the nodes of the tree that are not the
// label's own text.
func fieldNodes(tt *ui.Tester) (out []string) {
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleText {
			continue
		}
		out = append(out, n.Label)
	}
	return
}

// The label of a row names its control for assistive technology.
func TestFieldsNameTheirControls(t *testing.T) {
	opts := []Option{{Value: "a", Text: "Alpha"}, {Value: "b", Text: "Beta"}}
	for _, stacked := range []bool{false, true} {
		for _, c := range []struct {
			name string
			w    Control
			role ui.Role
		}{
			{"TextBox", TextBox(), ui.RoleTextField},
			{"MultilineTextBox", MultilineTextBox(), ui.RoleTextField},
			{"MaskedTextBox", MaskedTextBox("00-00"), ui.RoleTextField},
			{"NumericUpDown", NumericUpDown(0, 10, 1), ui.RoleTextField},
			{"ComboBox", ComboBox(opts...), ui.RolePopUpButton},
			{"Switch", Switch(), ui.RoleSwitch},
			{"TrackBar", TrackBar(0, 10), ui.RoleSlider},
			{"RadioGroup", RadioGroup(false, opts...), ui.RoleRadioGroup},
			{"DateTimePicker", DateTimePicker().SetValue(time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)), ui.RolePopUpButton},
		} {
			name := c.name
			if stacked {
				name += "/stacked"
			}
			t.Run(name, func(t *testing.T) {
				f := Fields().Row("Subject", c.w)
				if stacked {
					f.Stacked()
				}
				tt := tester(t, f)
				found := false
				for _, n := range tt.Nodes() {
					if n.Role == c.role && n.Label == "Subject" {
						found = true
					}
				}
				if !found {
					t.Fatalf("no %d node named Subject among %q", c.role, fieldNodes(tt))
				}
			})
		}
	}
}

func TestFieldsNameTheButtonsOfANumber(t *testing.T) {
	tt := tester(t, Fields().Row("Copies", NumericUpDown(0, 10, 1)))
	if got := fieldNodes(tt); !slices.Contains(got, "Copies Increase") || !slices.Contains(got, "Copies Decrease") {
		t.Fatalf("nodes %q", got)
	}
}

// A control that has a name of its own keeps it.
func TestFieldsKeepTheNameOfAControl(t *testing.T) {
	tt := tester(t, Fields().
		Row("Subject", NumericUpDown(0, 10, 1).SetAccessibleName("Quantity")).
		Row("Remember", CheckBox("Keep me signed in")))
	got := fieldNodes(tt)
	if !slices.Contains(got, "Quantity") || slices.Contains(got, "Subject") {
		t.Errorf("a named number: nodes %q", got)
	}
	if !slices.Contains(got, "Keep me signed in") || slices.Contains(got, "Remember") {
		t.Errorf("a check box with a text: nodes %q", got)
	}
}

// Showing a row's control again, or adding rows later, names them as well.
func TestFieldsNameRowsAddedLater(t *testing.T) {
	f := Fields().Row("One", TextBox())
	tt := tester(t, f)
	f.Row("Two", TextBox())
	tt.Frame()
	tt.Frame()
	got := fieldNodes(tt)
	if !slices.Contains(got, "One") || !slices.Contains(got, "Two") {
		t.Fatalf("nodes %q", got)
	}
}

// A row whose label is hidden has no name to give, and still builds.
func TestFieldsHiddenLabelNamesNothing(t *testing.T) {
	f := Fields().Row("Hidden", TextBox())
	f.Label(0).SetVisible(false)
	tt := tester(t, f)
	for _, n := range tt.Nodes() {
		if n.Label == "Hidden" {
			t.Fatalf("a hidden label named %v", n)
		}
	}
}

// A Fields form in a row of another: the inner rows' labels name their
// controls; the outer label must not take the first one's name.
func TestFieldsNestedKeepTheInnerNames(t *testing.T) {
	tt := tester(t, Fields().Row("Address", Fields().Row("Street", TextBox()).Row("City", TextBox())))
	var got []string
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTextField {
			got = append(got, n.Label)
		}
	}
	if len(got) != 2 || got[0] != "Street" || got[1] != "City" {
		t.Fatalf("text fields %q, want [Street City]", got)
	}
}

// Controls that are several (a group of check boxes, a range of two numbers, a
// tag field) are a group named by the row's label, as ui.Field makes it.
func TestFieldsNameGroupsOfControls(t *testing.T) {
	opts := []Option{{Value: "a", Text: "Alpha"}, {Value: "b", Text: "Beta"}}
	for _, c := range []struct {
		name string
		w    Control
	}{
		{"CheckBoxGroup", CheckBoxGroup(true, opts...)},
		{"RangeSpin", RangeSpin(1, 9, 1, 9, 1, "to", "")},
		{"TagSelect", TagSelect(opts...)},
	} {
		t.Run(c.name, func(t *testing.T) {
			tt := tester(t, Fields().Row("Choices", c.w))
			for _, n := range tt.Nodes() {
				if n.Label == "Choices" && n.Role != ui.RoleText {
					return
				}
			}
			var got []string
			for _, n := range tt.Nodes() {
				got = append(got, fmt.Sprintf("%d:%q", n.Role, n.Label))
			}
			t.Fatalf("no node named Choices but its text: %v", got)
		})
	}
}
