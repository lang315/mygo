package widget

import (
	"slices"
	"testing"
)

func editableCombo(changed chan string) *ComboBoxWidget {
	c := ComboBox(Option{"a", "Apple"}, Option{"k", "Kale"}).Editable()
	if changed != nil {
		c.OnChange(func() { changed <- c.Text() })
	}
	return c
}

func TestEditableComboTakesTypedTextThatIsNoOption(t *testing.T) {
	changed := make(chan string, 16)
	c := editableCombo(changed)
	tt := tester(t, c)
	focus(tt, 1)
	tt.Type("Zucchini")
	tt.Frame()
	if c.Text() != "Zucchini" || c.Value() != "" {
		t.Fatalf("Text %q, Value %q: want the typed text and no value", c.Text(), c.Value())
	}
	if _, ok := c.Selected(); ok {
		t.Fatal("Selected an option for text matching none")
	}
	if got := received(t, changed); got == "" {
		t.Fatal("OnChange did not see the typing")
	}
}

func TestEditableComboValueIsTheMatchingOptions(t *testing.T) {
	c := editableCombo(nil)
	tt := tester(t, c)
	focus(tt, 1)
	tt.Type("Kale")
	tt.Frame()
	if c.Text() != "Kale" || c.Value() != "k" {
		t.Fatalf("Text %q, Value %q, want Kale and k", c.Text(), c.Value())
	}
	c.SetText("Apple")
	if c.Value() != "a" {
		t.Fatalf("after SetText Value %q, want a", c.Value())
	}
	c.SetText("Apples")
	if c.Value() != "" || c.Text() != "Apples" {
		t.Fatalf("Text %q Value %q", c.Text(), c.Value())
	}
	c.SetValue("k")
	if c.Text() != "Kale" {
		t.Fatalf("SetValue: Text %q, want Kale", c.Text())
	}
	c.Unselect()
	if c.Text() != "" || c.Value() != "" {
		t.Fatalf("Unselect: %q %q", c.Text(), c.Value())
	}
}

func TestEditableComboChoosingAnOptionSetsTheTextAndRunsOnChange(t *testing.T) {
	changed := make(chan string, 16)
	c := editableCombo(changed)
	tt := tester(t, c)
	focus(tt, 1)
	tt.Type("Ka") // opens the popup, filtered
	tt.Frame()
	for received(t, changed) != "Ka" { // the typing
	}
	click(t, tt, "Kale")
	if got := received(t, changed); got != "Kale" {
		t.Fatalf("OnChange after choosing saw %q", got)
	}
	if c.Text() != "Kale" || c.Value() != "k" {
		t.Fatalf("Text %q Value %q", c.Text(), c.Value())
	}
}

func TestEditableComboBindsTextWhenEditableFirstAndValueWhenBoundFirst(t *testing.T) {
	src := newSource("Zucchini")
	c := Bind(editableCombo(nil), src.get, src.set)
	if c.Text() != "Zucchini" {
		t.Fatalf("bound after Editable: Text %q, want the source's", c.Text())
	}
	tt := tester(t, c)
	focus(tt, 1)
	tt.Type("!")
	tt.Frame()
	if got := src.calls(); len(got) == 0 || got[len(got)-1] != c.Text() {
		t.Fatalf("set calls %q, Text %q: want the text written", got, c.Text())
	}

	src2 := newSource("k")
	d := Bind(ComboBox(Option{"a", "Apple"}, Option{"k", "Kale"}), src2.get, src2.set).Editable()
	if d.Text() != "Kale" || d.Value() != "k" {
		t.Fatalf("bound before Editable: Text %q Value %q", d.Text(), d.Value())
	}
	tt2 := tester(t, d)
	focus(tt2, 1)
	d.SetText("")
	tt2.Frame()
	tt2.Type("Ka")
	tt2.Frame()
	click(t, tt2, "Kale") // choosing writes the Value
	if got := src2.calls(); !slices.Contains(got, "k") {
		t.Fatalf("a combo bound before Editable writes Values: %q", got)
	}
}

func TestEditableKeepsTheDropDownWhenNotEditable(t *testing.T) {
	c := ComboBox(Option{"a", "Apple"}).SetValue("a")
	if c.Text() != "Apple" {
		t.Fatalf("Text of a plain combo %q, want the chosen option's", c.Text())
	}
}
