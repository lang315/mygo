package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestToolStripDocksAtTheTop(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content")).ToolStrip(ToolStrip(ToolButton("Save")))
	tt := formTester(t, f)
	save, content := find(t, tt, "Save"), find(t, tt, "content")
	if save.Y >= content.Y {
		t.Fatalf("the tool strip's Save at %v is not above the content at %v", save, content)
	}
	if save.Y > 20 {
		t.Fatalf("Save at y %v: not at the top", save.Y)
	}
}

func TestToolStripIsBelowTheHeaderBarAndAboveTheContent(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content")).ToolStrip(ToolStrip(ToolButton("Save")))
	f.UseHeaderBar("sub").PackEnd(Label("hdr"))
	tt := formTester(t, f)
	hdr, save, content := find(t, tt, "hdr"), find(t, tt, "Save"), find(t, tt, "content")
	if !(hdr.Y < save.Y && save.Y < content.Y) {
		t.Fatalf("order from the top: header %v, tool strip %v, content %v", hdr.Y, save.Y, content.Y)
	}
}

func TestToolStripWithTheStatusStripKeepsTheContentBetween(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content")).
		ToolStrip(ToolStrip(ToolButton("Save"))).StatusStrip(StatusStrip(StatusLabel("Ready")))
	tt := formTester(t, f)
	save, content, ready := find(t, tt, "Save"), find(t, tt, "content"), find(t, tt, "Ready")
	if !(save.Y < content.Y && content.Y < ready.Y) {
		t.Fatalf("order from the top: tools %v, content %v, status %v", save.Y, content.Y, ready.Y)
	}
}

func TestToolButtonClickIsPostedToTheEventGoroutine(t *testing.T) {
	ran := make(chan bool, 1)
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(ToolButton("Save").OnClick(func() { ran <- onEvents() })))
	tt := formTester(t, f)
	click(t, tt, "Save")
	if !received(t, ran) {
		t.Fatal("OnClick did not run on the event goroutine")
	}
}

func TestToolButtonToggles(t *testing.T) {
	changed := make(chan bool, 4)
	var b *ToolButtonWidget
	b = ToolButton("Bold").SetToggle(true).OnChange(func() { changed <- b.Checked() })
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(b))
	tt := formTester(t, f)
	click(t, tt, "Bold")
	if !received(t, changed) || !b.Checked() {
		t.Fatal("clicking did not check the toggle")
	}
	b.SetChecked(false)
	tt.Frame()
	if b.Checked() {
		t.Fatal("SetChecked(false) did not stick")
	}
	Update(func() {}) // OnChange runs for the user's click only
	if len(changed) != 0 {
		t.Fatal("a program setter ran OnChange")
	}
}

func TestToolButtonChangesAtRunTimeAndRedraws(t *testing.T) {
	b := ToolButton("Save")
	f := NewForm("Shell", 300, 200).SetContent(Label("c")).ToolStrip(ToolStrip(b))
	w := showFormFrame(t, f)
	redrawn := recordInvalidations(t)
	b.SetText("Saving")
	wantRedrawn(t, redrawn(), w)
	tt := formTester(t, NewForm("Shell", 300, 200).ToolStrip(ToolStrip(b)))
	if !tt.HasText("Saving") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestToolButtonIconOnlyShowsNoText(t *testing.T) {
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(ToolButton("").SetIcon(chevronDown).SetTooltip("Next"), ToolButton("Text")))
	tt := formTester(t, f)
	if !tt.HasText("Text") {
		t.Fatalf("texts %q", tt.Texts())
	}
	for _, s := range tt.Texts() {
		if s == "Next" {
			t.Fatal("the tooltip shows without hovering")
		}
	}
}

func TestToolStripSeparatorAndSpring(t *testing.T) {
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(ToolButton("one"), Separator(), ToolButton("two"), Spring(), ToolButton("end")))
	tt := formTester(t, f)
	one, two, end := find(t, tt, "one"), find(t, tt, "two"), find(t, tt, "end")
	found, y, border := false, int(one.Y+one.H/2), ui.LightTheme().Border
	for x := int(one.X + one.W); x < int(two.X); x++ {
		if near(tt.Image().RGBAAt(x, y), border) {
			found = true
		}
	}
	if !found {
		t.Fatal("no separator line between one and two")
	}
	if e := end.X + end.W; e < 770 {
		t.Fatalf("end ends at x %v of 800: the spring did not push it", e)
	}
}

func TestToolStripHoldsAComboBoxWithSections(t *testing.T) {
	combo := ComboBox().AddSection("Fruits", nil, Option{"apple", "Apple"}, Option{"pear", "Pear"}).SetValue("apple")
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(combo))
	tt := formTester(t, f)
	click(t, tt, "Apple") // opens the popup
	if !tt.HasText("Fruits") || !tt.HasText("Pear") {
		t.Fatalf("the popup lacks its section or option: %q", tt.Texts())
	}
	click(t, tt, "Pear")
	if combo.Value() != "pear" {
		t.Fatalf("Value %q", combo.Value())
	}
}

func TestDropDownButtonOpensItsMenu(t *testing.T) {
	ran := make(chan bool, 1)
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(
		DropDownButton("Tools", MenuItem{Text: "Run", OnClick: func() { ran <- onEvents() }}, MenuItem{Text: "Stop"})))
	tt := formTester(t, f)
	click(t, tt, "Tools")
	if menu := tt.Menu(); len(menu) != 2 || menu[0] != "Run" {
		t.Fatalf("menu %q", menu)
	}
	if err := tt.ChooseMenuItem("Run"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !received(t, ran) {
		t.Fatal("the item's OnClick did not run on the event goroutine")
	}
}

// A ComboBox and a DropDownButton that do not fit go into the overflow menu
// too, under their text.
func TestToolStripOverflowTakesCombosAndDropDowns(t *testing.T) {
	combo := ComboBox(Option{"apple", "Apple"}, Option{"pear", "Pear"}).SetValue("apple")
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(
		ToolButton("Alpha"), ToolButton("Beta"), combo, DropDownButton("Tools", MenuItem{Text: "Run"})))
	tt := ui.NewTester(f.view, 220, 300)
	tt.Frame()
	if err := tt.Click("More"); err != nil {
		t.Fatalf("no overflow button (texts %q): %v", tt.Texts(), err)
	}
	tt.Frame()
	if menu := tt.Menu(); len(menu) != 2 || menu[0] != "Apple" || menu[1] != "Tools" {
		t.Fatalf("overflow menu %q, want the combo's text and the drop-down's", menu)
	}
}

// ui.Toolbar puts what does not fit, from the last, into a menu at its end:
// ToolStrip has that, and no overflow of its own.
func TestToolStripOverflowsTheLastItemsIntoAMenu(t *testing.T) {
	ran := make(chan struct{}, 1)
	var items []Control
	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
		b := ToolButton(name)
		if name == "Epsilon" {
			b.OnClick(func() { ran <- struct{}{} })
		}
		items = append(items, b)
	}
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(items...))
	tt := ui.NewTester(f.view, 800, 600)
	tt.Frame()
	if menu := tt.Menu(); len(menu) != 0 {
		t.Fatalf("a wide window overflows: %q", menu)
	}
	tt.SetSize(200, 300)
	tt.Frame()
	if err := tt.Click("More"); err != nil {
		t.Fatalf("no overflow button in a narrow window (texts %q): %v", tt.Texts(), err)
	}
	tt.Frame()
	menu := tt.Menu()
	if len(menu) == 0 || menu[len(menu)-1] != "Epsilon" {
		t.Fatalf("overflow menu %q: want the last items, ending with Epsilon", menu)
	}
	if err := tt.ChooseMenuItem("Epsilon"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	received(t, ran) // choosing it clicks the button
}
