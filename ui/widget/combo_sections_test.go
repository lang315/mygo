package widget

import (
	"image/color"
	"strings"
	"testing"

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
	tt := tester(t, c)
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
