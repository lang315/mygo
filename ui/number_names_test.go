package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
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

// Two elements named by each other: nameOf recurses until the stack overflows
// (fatal, not a panic a test can recover).
func TestNamedByCycleDoesNotOverflowTheStack(t *testing.T) {
	a, b := "", ""
	tt := NewTester(func(c *Context) {
		var x, y Element
		Column(c).Children(func() {
			x = Row(c).Children(func() { TextInput(c, &a) })
			y = Row(c).Children(func() { TextInput(c, &b) })
		})
		x.NamedBy(y)
		y.NamedBy(x)
	}, 400, 200)
	tt.Frame()
}

// A row whose input is named by a label inside the same row can be kept, as
// the row without NamedBy is: NamedBy's link has no keepLinks.
func TestNamedByRowsAreKept(t *testing.T) {
	const n = 30
	vals := make([]string, n)
	s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return 1 }}
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			List(c, s, n, func(i int) {
				Row(c).Padding(4).Gap(8).Children(func() {
					l := Textf(c, "name %d", i)
					TextInput(c, &vals[i]).Width(140).NamedBy(l)
				})
			}).Grow(1)
		})
	}, 500, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for range 6 {
		tt.Frame()
	}
	if tt.rt.rowsKept == 0 {
		t.Fatalf("no row kept, %d built (78 kept without NamedBy)", tt.rt.rowsBuilt)
	}
}

// A field inside a field keeps its own label for its input.
func TestNestedFieldsNameTheirInputs(t *testing.T) {
	a, b := "", ""
	tt := NewTester(func(c *Context) {
		Field(c, "Address", func() {
			Field(c, "Street", func() { TextInput(c, &a) })
			Field(c, "City", func() { TextInput(c, &b) })
		})
	}, 400, 300)
	tt.Frame()
	var got []string
	for _, n := range tt.Nodes() {
		if n.Role == RoleTextField {
			got = append(got, n.Label)
		}
	}
	if !slices.Equal(got, []string{"Street", "City"}) {
		t.Fatalf("inputs %q, want [Street City]", got)
	}
}
