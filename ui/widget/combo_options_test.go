package widget

import (
	"slices"
	"testing"
)

// AddOption appends to the options outside any section, after a
// RemoveOption too, and keeps the chosen value; the Tester picks the
// appended option.
func TestComboAddOptionAppends(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboBoxWidget
	ComboBox().AssignTo(&c).
		SetOptions(Option{"a", "A"}, Option{"b", "B"}).
		AddSection("More", nil, Option{"m", "M"}).
		SetValue("b").
		RemoveOption("a").
		AddOption(Option{"c", "C"}).
		OnChange(func() { changed <- c.Value() })
	if got, want := c.Options(), []Option{{"b", "B"}, {"c", "C"}}; !slices.Equal(got, want) {
		t.Fatalf("Options %v, want %v", got, want)
	}
	if c.Value() != "b" {
		t.Fatalf("AddOption unchose b: Value %q", c.Value())
	}
	if n := c.SectionLen("More"); n != 1 {
		t.Fatalf("SectionLen(More) %d, want 1", n)
	}
	tt := tester(t, c)
	click(t, tt, "B") // open
	click(t, tt, "C")
	if v := received(t, changed); v != "c" {
		t.Fatalf("OnChange saw %q, want c", v)
	}
}

// Options returns a copy: changing it leaves the combo as it is.
func TestComboOptionsIsACopy(t *testing.T) {
	c := ComboBox(Option{"a", "A"}, Option{"b", "B"})
	got := c.Options()
	got[0] = Option{"x", "X"}
	_ = append(got[:1], Option{"y", "Y"})
	if got, want := c.Options(), []Option{{"a", "A"}, {"b", "B"}}; !slices.Equal(got, want) {
		t.Fatalf("Options %v after changing a copy, want %v", got, want)
	}
}
