package ui

import (
	"slices"
	"testing"
)

func numberNodes(tt *Tester) (fields, buttons []string) {
	for _, n := range tt.Nodes() {
		switch n.Role {
		case RoleTextField:
			fields = append(fields, n.Label)
		case RoleButton:
			buttons = append(buttons, n.Label)
		}
	}
	return
}

// A NumberInput's Label names its field and its buttons, so that assistive
// technology tells two of them apart.
func TestNumberInputLabelNamesTheFieldAndTheButtons(t *testing.T) {
	a, b := 3.0, 4.0
	tt := NewTester(func(c *Context) {
		Row(c).Children(func() {
			NumberInput(c, &a, 0, 10, 1).Label("From")
			NumberInput(c, &b, 0, 10, 1).Label("To")
		})
	}, 600, 100)
	tt.Frame()
	fields, buttons := numberNodes(tt)
	if !slices.Equal(fields, []string{"From", "To"}) {
		t.Errorf("fields %q", fields)
	}
	if want := []string{"From Decrease", "From Increase", "To Decrease", "To Increase"}; !slices.Equal(buttons, want) {
		t.Errorf("buttons %q, want %q", buttons, want)
	}
}

func TestNumberInputWithoutALabelKeepsItsButtonNames(t *testing.T) {
	v := 3.0
	tt := NewTester(func(c *Context) { NumberInput(c, &v, 0, 10, 1) }, 300, 100)
	tt.Frame()
	fields, buttons := numberNodes(tt)
	if !slices.Equal(fields, []string{""}) || !slices.Equal(buttons, []string{"Decrease", "Increase"}) {
		t.Errorf("fields %q, buttons %q", fields, buttons)
	}
}

func TestNumberInputNamedButtonStillSteps(t *testing.T) {
	v := 3.0
	tt := NewTester(func(c *Context) { NumberInput(c, &v, 0, 10, 1).Label("Copies") }, 300, 100)
	tt.Frame()
	for _, n := range tt.Nodes() {
		if n.Label == "Copies Increase" {
			tt.ClickAt(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
		}
	}
	tt.Frame()
	if v != 4 {
		t.Fatalf("value %v after a click on Copies Increase", v)
	}
}

// NamedBy names the control inside an element by a label of a form of one's
// own; a control with a name of its own keeps it.
func TestNamedByNamesTheControlAndTheButtonsOfANumber(t *testing.T) {
	v, w := 3.0, 4.0
	var on bool
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			l1 := Text(c, "Copies")
			NumberInput(c, &v, 0, 10, 1).NamedBy(l1)
			l2 := Text(c, "Other")
			NumberInput(c, &w, 0, 10, 1).Label("Own").NamedBy(l2)
			l3 := Text(c, "Remember")
			Checkbox(c, &on, "Keep me signed in").NamedBy(l3)
		})
	}, 400, 300)
	tt.Frame()
	fields, buttons := numberNodes(tt)
	if !slices.Equal(fields, []string{"Copies", "Own"}) {
		t.Errorf("fields %q: the second number keeps its own Label", fields)
	}
	if !slices.Contains(buttons, "Copies Increase") || !slices.Contains(buttons, "Copies Decrease") {
		t.Errorf("buttons %q", buttons)
	}
	for _, n := range tt.Nodes() {
		if n.Label == "Remember" && n.Role != RoleText {
			t.Errorf("a check box with a text took the label's name: %v", n)
		}
	}
}
