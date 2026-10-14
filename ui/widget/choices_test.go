package widget

import (
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSpinTypedValue(t *testing.T) {
	s := Spin(0, 100, 1).SetValue(5)
	tt := tester(t, s)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	tt.Type("0")
	tt.Frame()
	if s.Int() != 50 || s.Value() != 50 {
		t.Fatalf("Value %v Int %d, want 50", s.Value(), s.Int())
	}
}

func TestRangeSpinValues(t *testing.T) {
	r := RangeSpin(0, 10, 0, 100, 1, "to", "seconds").SetValues(2, 30)
	tt := tester(t, r)
	for _, s := range []string{"to", "seconds"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	if lo, hi := r.Ints(); lo != 2 || hi != 30 {
		t.Fatalf("Ints %d %d", lo, hi)
	}
}

func TestComboValueAndText(t *testing.T) {
	c := Combo(Option{"vn", "Vietnam"}, Option{"us", "United States"})
	c.SetValue("us")
	tt := tester(t, c)
	if !tt.HasText("United States") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if o, ok := c.Selected(); !ok || o.Value != "us" {
		t.Fatalf("Selected %v %v", o, ok)
	}
	c.Unselect()
	if _, ok := c.Selected(); ok || c.Value() != "" {
		t.Fatal("Unselect left a choice")
	}
	c.SetOptions(Option{"x", "X"})
	c.SetValue("x")
	tt.Frame()
	if !tt.HasText("X") {
		t.Fatalf("texts after SetOptions %q", tt.Texts())
	}
}

func TestTagSelect(t *testing.T) {
	g := TagSelect(Option{"a", "Alpha"}, Option{"b", "Beta"})
	tt := tester(t, g)
	click(t, tt, "Beta")
	if got := g.Values(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Values %v", got)
	}
}

func TestProgressAndSpinner(t *testing.T) {
	p := Progress().SetFraction(0.5).SetText("Half")
	s := Spinner()
	tt := tester(t, VBox(0).Add(p, s))
	if !tt.HasText("Half") {
		t.Fatalf("texts %q", tt.Texts())
	}
	s.Start()
	tt.Frame()
	s.Stop()
	tt.Frame()
}

func TestComboChoosesByValueWithEqualTexts(t *testing.T) {
	c := Combo(Option{"a", "Same"}, Option{"b", "Same"}).SetValue("b")
	tester(t, c)
	if c.Value() != "b" {
		t.Fatalf("Value %q, want b", c.Value())
	}
}

func TestComboSections(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboWidget
	Combo().AssignTo(&c).
		AddSection("Fruits", nil, Option{"apple", "Apple"}, Option{"pear", "Pear"}).
		AddSection("Vegetables", testPNG(t, color.RGBA{0, 128, 0, 255}), Option{"kale", "Kale"}).
		SetValue("apple").
		OnChange(func() { changed <- c.Value() })
	tt := tester(t, c)
	click(t, tt, "Apple") // the trigger shows the choice
	if !tt.HasText("Fruits") || !tt.HasText("Vegetables") {
		t.Fatalf("no section headers in %q", tt.Texts())
	}
	click(t, tt, "Fruits") // a header chooses nothing
	if c.Value() != "apple" {
		t.Fatalf("a click on a header chose %q", c.Value())
	}
	click(t, tt, "Kale")
	if v := wait(t, changed); v != "kale" {
		t.Fatalf("OnChange saw %q", v)
	}
	if n := c.SectionLen("Fruits"); n != 2 {
		t.Fatalf("SectionLen(Fruits) %d", n)
	}
}

func TestComboRemoveAndRename(t *testing.T) {
	c := Combo(Option{"x", "X"}).AddSection("S", nil, Option{"y", "Y"}).SetValue("y")
	tt := tester(t, c)
	c.SetOptionText("y", "Why")
	tt.Frame()
	if !tt.HasText("Why") {
		t.Fatalf("renamed option not shown: %q", tt.Texts())
	}
	c.RemoveSection("S")
	if c.Value() != "" || c.SectionLen("S") != 0 {
		t.Fatalf("after RemoveSection: value %q", c.Value())
	}
	c.SetValue("x").RemoveOption("x")
	if _, ok := c.Selected(); ok {
		t.Fatal("removed option still selected")
	}
}

func TestComboEditsLeaveTheCallersOptions(t *testing.T) {
	opts := []Option{{"x", "X"}, {"y", "Y"}}
	sec := []Option{{"z", "Z"}, {"w", "W"}}
	more := []Option{{"v", "V"}, {"u", "U"}}
	c := Combo(opts...).AddSection("S", nil, sec...).SetOptions(more...)
	c.SetOptionText("v", "Vee").SetOptionText("z", "Zed").RemoveOption("u").RemoveOption("z")
	if opts[0].Text != "X" || sec[0] != (Option{"z", "Z"}) || sec[1] != (Option{"w", "W"}) ||
		more[0].Text != "V" || more[1] != (Option{"u", "U"}) {
		t.Fatalf("caller's options changed: %v %v %v", opts, sec, more)
	}
}
