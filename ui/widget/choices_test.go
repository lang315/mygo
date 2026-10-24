package widget

import (
	"fmt"
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSpinTypedValue(t *testing.T) {
	s := NumericUpDown(0, 100, 1).SetValue(5)
	tt := tester(t, s)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	tt.Type("0")
	tt.Frame()
	if s.Int() != 50 || s.Value() != 50 {
		t.Fatalf("Value %v Int %d, want 50", s.Value(), s.Int())
	}
}

// Focus gives a spin's text field the keyboard, as InputInt does: typing
// goes in, and OnFocusOut waits for the focus to leave the field.
func TestSpinFocus(t *testing.T) {
	left := make(chan struct{}, 1)
	s := NumericUpDown(0, 100, 1).SetValue(5)
	s.OnFocusOut(func() { left <- struct{}{} })
	tt := tester(t, VBox(0).Add(s, TextBox()))
	s.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Type("0")
	tt.Frame()
	if s.Int() != 50 {
		t.Fatalf("typed 0 after 5 in the focused spin: %v", s.Value())
	}
	select {
	case <-left:
		t.Fatal("OnFocusOut ran while the spin's field had the focus")
	default:
	}
	focus(tt, 1) // past the minus, to the plus button
	focus(tt, 1)
	focus(tt, 1) // the entry
	received(t, left)
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
	c := ComboBox(Option{"vn", "Vietnam"}, Option{"us", "United States"})
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
	p := ProgressBar().SetFraction(0.5).SetText("Half")
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
	c := ComboBox(Option{"a", "Same"}, Option{"b", "Same"}).SetValue("b")
	tester(t, c)
	if c.Value() != "b" {
		t.Fatalf("Value %q, want b", c.Value())
	}
}

func TestComboSections(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboBoxWidget
	ComboBox().AssignTo(&c).
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
	if v := received(t, changed); v != "kale" {
		t.Fatalf("OnChange saw %q", v)
	}
	if n := c.SectionLen("Fruits"); n != 2 {
		t.Fatalf("SectionLen(Fruits) %d", n)
	}
}

func TestComboRemoveAndRename(t *testing.T) {
	c := ComboBox(Option{"x", "X"}).AddSection("S", nil, Option{"y", "Y"}).SetValue("y")
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
	c := ComboBox(opts...).AddSection("S", nil, sec...).SetOptions(more...)
	c.SetOptionText("v", "Vee").SetOptionText("z", "Zed").RemoveOption("u").RemoveOption("z")
	if opts[0].Text != "X" || sec[0] != (Option{"z", "Z"}) || sec[1] != (Option{"w", "W"}) ||
		more[0].Text != "V" || more[1] != (Option{"u", "U"}) {
		t.Fatalf("caller's options changed: %v %v %v", opts, sec, more)
	}
}

// V6: a popup of 5 sections of 20 options stays inside the window and
// scrolls: End brings the last option of the last section into view, and
// the wheel scrolls back to the first section's header. (Choosing it is
// ui's TestSelectLongPopupScrolls.)
func TestComboLongSectionedPopupScrolls(t *testing.T) {
	c := ComboBox()
	for s := range 5 {
		var opts []Option
		for i := range 20 {
			v := fmt.Sprintf("s%di%02d", s, i)
			opts = append(opts, Option{v, v})
		}
		c.AddSection(fmt.Sprintf("Section %d", s), nil, opts...)
	}
	c.SetValue("s0i00")
	tt := tester(t, c)
	click(t, tt, "s0i00")
	if !tt.HasText("Section 0") {
		t.Fatalf("popup not open: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	r := find(t, tt, "s4i19") // its part in view
	if r.H <= 0 || r.Y+r.H > 600 {
		t.Fatalf("End: the last option shows at %+v, not inside the 600-tall window", r)
	}
	tt.Scroll(r.X, r.Y, 0, -8000) // up, past the top
	tt.Frame()
	if h := find(t, tt, "Section 0"); h.H <= 0 {
		t.Fatalf("the wheel did not scroll back to the first header: %+v", h)
	}
}

// V4: popups close on Escape or an outside click without OnChange; no
// widget binds such an open flag (a ComboBox's is inside ui.SelectBase), so
// closing one leaves the value alone and reports nothing.
func TestComboClosedWithoutChoosingKeepsItsValue(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboBoxWidget
	ComboBox(Option{"a", "Alpha"}, Option{"b", "Beta"}).AssignTo(&c).SetValue("a").
		OnChange(func() { changed <- c.Value() })
	tt := tester(t, c)
	for i, dismiss := range []func(){
		func() { tt.Key(0, ui.KeyEscape) },
		func() { tt.ClickAt(700, 500) }, // off the popup
	} {
		click(t, tt, "Alpha")
		tt.Key(0, ui.KeyDown) // the highlight on Beta, not chosen
		tt.Frame()
		if !tt.HasText("Beta") {
			t.Fatalf("popup not open: %q", tt.Texts())
		}
		dismiss()
		tt.Frame()
		if tt.HasText("Beta") || c.Value() != "a" {
			t.Fatalf("after closing %d: texts %q, value %q", i, tt.Texts(), c.Value())
		}
	}
	done := make(chan struct{})
	Post(func() { close(done) }) // after any handler posted before it
	received(t, done)
	select {
	case v := <-changed:
		t.Fatalf("OnChange ran with %q", v)
	default:
	}
}

// A click on a section header chooses nothing and keeps the keyboard on the
// combo: the arrows still move among its options.
func TestComboHeaderClickKeepsArrows(t *testing.T) {
	c := ComboBox().AddSection("Fruit", nil, Option{"a", "Apple"}, Option{"b", "Banana"}, Option{"c", "Cherry"}).
		SetValue("a")
	tt := tester(t, c)
	click(t, tt, "Apple")
	click(t, tt, "Fruit")
	if !tt.HasText("Banana") {
		t.Fatalf("the header click closed the popup: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyDown) // from Apple, the highlight
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v := c.Value(); v != "c" {
		t.Fatalf("Down, Down, Enter after a header click chose %q, want c", v)
	}
}

func TestComboSectionsAndTextsChangeWhileShown(t *testing.T) {
	c := ComboBox().AddSection("Fruits", nil, Option{"apple", "Apple"}).SetValue("apple")
	_, w := showFrame(t, c)
	redrawn := recordInvalidations(t)
	c.SetOptionText("apple", "Green apple")
	wantRedrawn(t, redrawn(), w)
	c.AddSection("Vegetables", testPNG(t, color.RGBA{0, 128, 0, 255}), Option{"kale", "Kale"})
	wantRedrawn(t, redrawn(), w)
	c.RemoveSection("Fruits")
	wantRedrawn(t, redrawn(), w)
	c.SetOptions(Option{"x", "X"})
	wantRedrawn(t, redrawn(), w)

	// And what a frame draws follows: the trigger's text, the popup's sections.
	d := ComboBox().AddSection("Fruits", nil, Option{"apple", "Apple"}).SetValue("apple")
	tt := tester(t, d)
	d.SetOptionText("apple", "Green apple")
	tt.Frame()
	if !tt.HasText("Green apple") {
		t.Fatalf("trigger after SetOptionText: %q", tt.Texts())
	}
	click(t, tt, "Green apple") // open
	d.AddSection("Vegetables", nil, Option{"kale", "Kale"})
	tt.Frame()
	if !tt.HasText("Vegetables") || !tt.HasText("Kale") {
		t.Fatalf("popup after AddSection: %q", tt.Texts())
	}
	d.RemoveSection("Fruits")
	tt.Frame()
	if tt.HasText("Fruits") {
		t.Fatalf("popup after RemoveSection: %q", tt.Texts())
	}
}
