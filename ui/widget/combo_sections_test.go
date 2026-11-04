package widget

import (
	"image/color"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// AddSection of a title the combo has adds to that section: one header.
func TestComboAddSectionAddsToTheSectionOfItsTitle(t *testing.T) {
	c := ComboBox().
		AddSection("Fruits", nil, Option{"apple", "Apple"}).
		AddSection("Vegetables", nil, Option{"kale", "Kale"}).
		AddSection("Fruits", testPNG(t, color.RGBA{200, 0, 0, 255}), Option{"pear", "Pear"}).
		SetValue("apple")
	if n := c.SectionLen("Fruits"); n != 2 {
		t.Fatalf("SectionLen(Fruits) %d, want 2", n)
	}
	tt := tester(t, c)
	click(t, tt, "Apple")
	if n := strings.Count(strings.Join(tt.Texts(), "\n"), "Fruits"); n != 1 {
		t.Fatalf("%d Fruits headers in %q", n, tt.Texts())
	}
	// Pear is in Fruits, before Vegetables.
	if p, v := find(t, tt, "Pear"), find(t, tt, "Vegetables"); p.Y > v.Y {
		t.Fatalf("Pear at %v, below Vegetables at %v", p, v)
	}
}

// A section's title is a header that chooses nothing, unless
// SetSectionValue makes it an option of its own.
func TestComboSelectableSectionTitle(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboBoxWidget
	ComboBox().AssignTo(&c).
		AddSection("Fruits", nil, Option{"apple", "Apple"}, Option{"pear", "Pear"}).
		SetSectionValue("Fruits", "fruit").
		SetValue("apple").
		OnChange(func() { changed <- c.Value() })
	tt := tester(t, c)
	click(t, tt, "Apple")
	click(t, tt, "Fruits")
	if v := received(t, changed); v != "fruit" {
		t.Fatalf("OnChange saw %q, want fruit", v)
	}
	if o, ok := c.Selected(); !ok || o != (Option{"fruit", "Fruits"}) || c.Text() != "Fruits" {
		t.Fatalf("Selected %v %v, Text %q", o, ok, c.Text())
	}
	// The keys reach it: Up from Apple.
	click(t, tt, "Fruits") // open
	click(t, tt, "Apple")
	received(t, changed)
	click(t, tt, "Apple") // open
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v := received(t, changed); v != "fruit" {
		t.Fatalf("Up, Enter from Apple chose %q, want fruit", v)
	}
	// SetValue reaches it, and "" makes it a header again, unchoosing it.
	c.SetValue("pear").SetValue("fruit")
	if c.Value() != "fruit" {
		t.Fatalf("SetValue(fruit) gave %q", c.Value())
	}
	c.SetSectionValue("Fruits", "")
	if c.Value() != "" {
		t.Fatalf("a header again, still chosen: %q", c.Value())
	}
}

// AddSubmenu adds an option that opens a nested list of options.
func TestComboSubmenu(t *testing.T) {
	changed := make(chan string, 4)
	var c *ComboBoxWidget
	ComboBox(Option{"none", "None"}).AssignTo(&c).
		AddSubmenu("Fruits", nil, Option{"apple", "Apple"}, Option{"pear", "Pear"}).
		AddSection("Vegetables", nil, Option{"kale", "Kale"}).
		SetValue("none").
		OnChange(func() { changed <- c.Value() })
	tt := tester(t, HBox(0).Add(c)) // as wide as its text: its submenu has room beside it
	click(t, tt, "None")
	if !tt.HasText("Fruits") || !tt.HasText("Kale") || tt.HasText("Pear") {
		t.Fatalf("closed submenu: %q", tt.Texts())
	}
	click(t, tt, "Fruits")
	if !tt.HasText("Pear") {
		t.Fatalf("submenu did not open: %q", tt.Texts())
	}
	click(t, tt, "Pear")
	if v := received(t, changed); v != "pear" {
		t.Fatalf("OnChange saw %q, want pear", v)
	}
	if tt.HasText("Kale") || c.Text() != "Pear" {
		t.Fatalf("after choosing: texts %q, Text %q", tt.Texts(), c.Text())
	}
	// The keys: Down to Fruits, Right opens it, Enter takes its first.
	click(t, tt, "Pear") // open
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v := received(t, changed); v != "apple" {
		t.Fatalf("Home, Down, Right, Enter chose %q, want apple", v)
	}
	if n := c.SectionLen("Fruits"); n != 2 {
		t.Fatalf("SectionLen(Fruits) %d", n)
	}
	// Assistive technology sees a menu button and its radio items.
	click(t, tt, "Apple")
	click(t, tt, "Fruits")
	var radios, checked int
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleMenuItemRadio {
			radios++
			if n.Checked {
				checked++
			}
		}
	}
	if radios != 4 || checked != 1 {
		t.Fatalf("%d radio items, %d checked; want 4 and 1", radios, checked)
	}
}

// An Editable combo shows a submenu's options as a section's, which typing
// filters.
func TestEditableComboShowsASubmenusOptionsInline(t *testing.T) {
	c := ComboBox().AddSubmenu("Fruits", nil, Option{"apple", "Apple"}, Option{"pear", "Pear"}).Editable()
	tt := tester(t, c)
	tt.Key(0, ui.KeyTab)
	tt.Type("pe")
	tt.Frame()
	if !tt.HasText("Pear") || tt.HasText("Apple") {
		t.Fatalf("typed pe: %q", tt.Texts())
	}
}

// A combo with a submenu takes Enter and Escape before the form's buttons,
// as a plain combo does.
func TestSubmenuComboTakesEnterAndEscapeBeforeTheFormsButtons(t *testing.T) {
	ok, cancels := 0, 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	cancel := Button("Cancel").OnClick(func() { cancels++ })
	combo := ComboBox(Option{"a", "Apple"}).AddSubmenu("More", nil, Option{"b", "Banana"}).SetValue("a")
	f := keysForm(VBox(4).Add(combo, okBtn, cancel)).SetAcceptButton(okBtn).SetCancelButton(cancel)
	tt := TestForm(t, f)
	tt.Find(ByLabel("Apple")).Focus().Key(0, ui.KeyEnter)
	tt.Settle()
	if ok != 0 || !tt.Has(ByLabel("More")) {
		t.Fatalf("Enter on the closed combo: OK clicked %d times; menu open %v", ok, tt.Has(ByLabel("More")))
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if cancels != 0 || tt.Has(ByLabel("More")) {
		t.Fatalf("Escape on the open combo: Cancel clicked %d times; menu open %v", cancels, tt.Has(ByLabel("More")))
	}
}

// Hiding a combo closes its menu, which does not show again with it.
func TestHidingASubmenuComboClosesItsMenu(t *testing.T) {
	c := ComboBox(Option{"a", "Apple"}).AddSubmenu("More", nil, Option{"b", "Banana"}).SetValue("a")
	tt := tester(t, VBox(4).Add(c))
	click(t, tt, "Apple")
	if !tt.HasText("More") {
		t.Fatalf("menu not open: %q", tt.Texts())
	}
	c.SetVisible(false)
	tt.Frame()
	c.SetVisible(true)
	tt.Frame()
	tt.Frame()
	if tt.HasText("More") {
		t.Fatalf("the menu showed again with the combo: %q", tt.Texts())
	}
}

// menuWidth is the width of the outermost menu shown, 0 for none.
func menuWidth(tt *ui.Tester) float32 {
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleMenu {
			return n.Bounds.W
		}
	}
	return 0
}

// The menu of a combo with a submenu is as wide as the combo, as a plain
// combo's drop-down is; Up opens it too.
func TestSubmenuComboMenuIsAsWideAsTheComboAndUpOpensIt(t *testing.T) {
	c := ComboBox(Option{"a", "Apple"}).AddSubmenu("More", nil, Option{"b", "Banana"}).SetValue("a")
	tt := tester(t, c)
	trigger := find(t, tt, "Apple")
	click(t, tt, "Apple")
	if w := menuWidth(tt); w < trigger.W-0.5 {
		t.Fatalf("menu %v wide, the combo %v", w, trigger.W)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.HasText("More") {
		t.Fatalf("Escape left the menu open: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("More") {
		t.Fatalf("Up did not open the menu: %q", tt.Texts())
	}
}

// The menu closes when the combo goes out of sight in any way: a box
// around it hidden, its TabControl page left, its dialog shown again.
func TestSubmenuComboMenuClosesOutOfSight(t *testing.T) {
	combo := func() *ComboBoxWidget {
		return ComboBox(Option{"a", "Apple"}).AddSubmenu("More", nil, Option{"b", "Banana"}).SetValue("a")
	}
	t.Run("box hidden", func(t *testing.T) {
		box := VBox(4).Add(combo())
		tt := tester(t, VBox(4).Add(box))
		click(t, tt, "Apple")
		box.SetVisible(false)
		tt.Frame()
		box.SetVisible(true)
		tt.Frame()
		tt.Frame()
		if !tt.HasText("Apple") || tt.HasText("More") {
			t.Fatalf("the menu showed again with the box: %q", tt.Texts())
		}
	})
	t.Run("page left", func(t *testing.T) {
		nb := TabControl().AddPage("One", combo()).AddPage("Two", Label("page two"))
		tt := tester(t, nb)
		click(t, tt, "Apple")
		nb.SetPage(1)
		tt.Frame()
		nb.SetPage(0)
		tt.Frame()
		tt.Frame()
		if !tt.HasText("Apple") || tt.HasText("More") {
			t.Fatalf("the menu showed again with the page: %q", tt.Texts())
		}
	})
	t.Run("dialog shown again", func(t *testing.T) {
		d := NewDialog(nil, "Pick", "", 400, 300, combo())
		dt := TestDialog(t, d)
		ran := make(chan struct{}, 2)
		run := func() { go func() { d.Run(); ran <- struct{}{} }() }
		run()
		dt.Find(ByLabel("Apple")).Click()
		if !dt.Has(ByLabel("More")) {
			t.Fatalf("menu not open: %q", dt.Texts())
		}
		d.Close()
		wait.For(t, "the first Run", ran)
		run()
		wait.Until(t, "the second Run", func() bool { return !dt.Closed() })
		dt.Settle()
		if dt.Has(ByLabel("More")) {
			t.Fatalf("the menu showed again with the dialog: %q", dt.Texts())
		}
		d.Close()
		wait.For(t, "the second Run", ran)
	})
}

// An option whose value the combo has is left out: the first stays.
func TestComboAddSectionLeavesOutValuesItHas(t *testing.T) {
	c := ComboBox(Option{"none", "None"}).
		AddSection("Fruits", nil, Option{"apple", "Apple"}).
		AddSection("Fruits", nil, Option{"apple", "Apple again"}, Option{"pear", "Pear"}, Option{"pear", "Pear again"}).
		AddSubmenu("More", nil, Option{"none", "None again"}).
		SetValue("none")
	if n, m := c.SectionLen("Fruits"), c.SectionLen("More"); n != 2 || m != 0 {
		t.Fatalf("SectionLen Fruits %d, More %d; want 2 and 0", n, m)
	}
	tt := tester(t, HBox(0).Add(c))
	click(t, tt, "None")
	for _, s := range []string{"Apple again", "Pear again", "None again"} {
		if tt.HasText(s) {
			t.Fatalf("%q shown: %q", s, tt.Texts())
		}
	}
	click(t, tt, "Apple")
	if c.Value() != "apple" || c.Text() != "Apple" {
		t.Fatalf("chose %q %q", c.Value(), c.Text())
	}
}
