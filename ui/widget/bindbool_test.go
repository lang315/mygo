package widget

import (
	"slices"
	"testing"
)

var (
	yes = Option{Value: "true", Text: "Có"}
	no  = Option{Value: "false", Text: "Không"}
)

func TestBindBoolShowsTheSourceAndGivesTheComboItsTwoOptions(t *testing.T) {
	src := newSource(true)
	c := BindBool(ComboBox(), yes, no, src.get, src.set)
	if c.Value() != "true" {
		t.Fatalf("Value %q, want the yes option's", c.Value())
	}
	tt := tester(t, c)
	click(t, tt, "Có") // opens the popup
	if !tt.HasText("Không") {
		t.Fatalf("the popup lacks the no option: %q", tt.Texts())
	}
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("binding wrote %v back to the source", got)
	}
	src.change(false)
	Reload(c)
	if c.Value() != "false" {
		t.Fatalf("Value %q after Reload of a false source", c.Value())
	}
}

func TestBindBoolUserChoiceSetsABoolOnce(t *testing.T) {
	src := newSource(true)
	c := BindBool(ComboBox(), yes, no, src.get, src.set)
	tt := tester(t, c)
	click(t, tt, "Có")
	click(t, tt, "Không")
	if got := only(t, src); got {
		t.Fatalf("set got true after choosing Không")
	}
	click(t, tt, "Không") // the trigger now
	click(t, tt, "Có")
	if got := src.calls(); !slices.Equal(got, []bool{false, true}) {
		t.Fatalf("set calls %v, want [false true]", got)
	}
}

func TestBindBoolProgramSetterDoesNotSet(t *testing.T) {
	src := newSource(true)
	c := BindBool(ComboBox(), yes, no, src.get, src.set)
	tt := tester(t, c)
	c.SetValue("false")
	tt.Frame()
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("SetValue called set with %v", got)
	}
	if src.get() != true || c.Value() != "false" {
		t.Fatalf("source %v, control %q", src.get(), c.Value())
	}
}

// set gets what the user chose, as Bind's does, whatever an OnChange or a
// worker then sets on the combo.
func TestBindBoolSetGetsTheUsersChoiceWhenOnChangeSetsTheCombo(t *testing.T) {
	src := newSource(true)
	var c *ComboBoxWidget
	c = ComboBox().OnChange(func() { c.SetValue("true") }) // puts it back
	BindBool(c, yes, no, src.get, src.set)
	tt := tester(t, c)
	click(t, tt, "Có")
	click(t, tt, "Không")
	if got := only(t, src); got {
		t.Fatal("set got true: what OnChange made of the choice, not the user's false")
	}
	if c.Value() != "true" {
		t.Fatalf("Value %q: OnChange's SetValue must still show", c.Value())
	}
}

func TestBindBoolRebindingReplacesTheFirst(t *testing.T) {
	first, second := newSource(true), newSource(false)
	c := BindBool(ComboBox(), yes, no, first.get, first.set)
	BindBool(c, yes, no, second.get, second.set)
	if c.Value() != "false" {
		t.Fatalf("Value %q, want the second source's", c.Value())
	}
	tt := tester(t, c)
	click(t, tt, "Không")
	click(t, tt, "Có")
	only(t, second)
	if got := first.calls(); len(got) != 0 {
		t.Fatalf("the replaced binding was called: %v", got)
	}
}

// An Editable combo bound as text would get "Có", never "true": BindBool
// makes the combo a plain choice of its two options, so picking yes writes
// true.
func TestBindBoolOnAnEditableComboWritesTrueAndFalse(t *testing.T) {
	src := newSource(false)
	c := BindBool(ComboBox().Editable(), yes, no, src.get, src.set)
	if c.Value() != "false" {
		t.Fatalf("Value %q, want the source's no", c.Value())
	}
	tt := tester(t, c)
	click(t, tt, "Không")
	click(t, tt, "Có")
	if got := src.calls(); !slices.Equal(got, []bool{true}) {
		t.Fatalf("set calls %v, want [true]", got)
	}
}

// Editable after BindBool: the typing that is neither option is not a
// choice, so set is not called with false for each keystroke.
func TestBindBoolIgnoresTextThatIsNeitherOption(t *testing.T) {
	src := newSource(true)
	c := BindBool(ComboBox(), yes, no, src.get, src.set).Editable()
	tt := tester(t, c)
	focus(tt, 1)
	c.SetText("")
	tt.Frame()
	tt.Type("C")
	tt.Frame()
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("typing %q called set with %v", "C", got)
	}
	tt.Type("ó")
	tt.Frame()
	if got := src.calls(); !slices.Equal(got, []bool{true}) {
		t.Fatalf("set calls %v after typing Có, want [true]", got)
	}
}

// An option in a section survives SetOptions: choosing it is not "no".
func TestBindBoolIgnoresAnOptionOfASection(t *testing.T) {
	src := newSource(true)
	c := BindBool(ComboBox(), yes, no, src.get, src.set).
		AddSection("Other", nil, Option{Value: "maybe", Text: "Có thể"})
	tt := tester(t, c)
	click(t, tt, "Có")
	click(t, tt, "Có thể")
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("choosing an option that is neither called set with %v", got)
	}
}
