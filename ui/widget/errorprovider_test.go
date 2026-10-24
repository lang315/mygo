package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// redPixels counts the pixels of a clearly red hue inside r of the frame.
func redPixels(tt *Tester, r ui.Rect) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c.R > 180 && c.G < 90 && c.B < 90 {
				n++
			}
		}
	}
	return n
}

func TestErrorProviderMarksAControlAndClearsIt(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox()
	tt := NewTester(t, VBox(8).Add(Label("Name"), tb), 400, 200)
	box := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	before := redPixels(tt, box)
	ep.SetError(tb, "Name is required")
	tt.Settle()
	if after := redPixels(tt, box); after <= before {
		t.Errorf("no red mark on the control: %d red pixels before, %d after", before, after)
	}
	if got := ep.Error(tb); got != "Name is required" || !ep.HasErrors() {
		t.Errorf("Error = %q, HasErrors %v", got, ep.HasErrors())
	}
	ep.SetError(tb, "")
	tt.Settle()
	if redPixels(tt, box) != before || ep.HasErrors() || ep.Error(tb) != "" {
		t.Errorf("the mark stayed after SetError(\"\")")
	}
}

func TestErrorProviderClearTakesEveryMarkOff(t *testing.T) {
	ep := ErrorProvider()
	a, b := TextBox(), TextBox()
	NewTester(t, VBox(8).Add(a, b), 400, 200)
	ep.SetError(a, "one").SetError(b, "two")
	if !ep.HasErrors() {
		t.Fatal("no errors")
	}
	ep.Clear()
	if ep.HasErrors() || ep.Error(a) != "" || ep.Error(b) != "" {
		t.Error("Clear left errors")
	}
}

// The mark keeps the TextBox's own icon, which draws in the same place.
func TestErrorMarkAndATextBoxIconBothDraw(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox().Icon(solidBitmap(color.RGBA{0, 0, 255, 255}))
	tt := NewTester(t, VBox(8).Add(tb), 400, 100)
	box := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	before := redPixels(tt, box)
	ep.SetError(tb, "bad")
	tt.Settle()
	if redPixels(tt, box) <= before {
		t.Error("the error mark did not draw over a TextBox with an icon")
	}
}

func TestValidateChildrenRunsEveryHandlerAndSaysIfAnyCancelled(t *testing.T) {
	ep := ErrorProvider()
	name, age := TextBox(), TextBox()
	var ran []string
	name.OnValidating(func() bool {
		ran = append(ran, "name")
		if name.Text() == "" {
			ep.SetError(name, "required")
			return false
		}
		ep.SetError(name, "")
		return true
	})
	age.OnValidating(func() bool { ran = append(ran, "age"); return true })
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(name, age))
	tt := TestForm(t, f)
	tt.Settle()
	if f.ValidateChildren() {
		t.Error("ValidateChildren said true with an empty name")
	}
	if len(ran) != 2 || ran[0] != "name" || ran[1] != "age" || ep.Error(name) != "required" {
		t.Errorf("ran %v, error %q: every handler runs, in tree order", ran, ep.Error(name))
	}
	name.SetText("Ann")
	if !f.ValidateChildren() || ep.HasErrors() {
		t.Error("ValidateChildren with a good name")
	}
}

// Nothing validates by itself: not a focus change, not a typed character.
func TestNothingValidatesUntilValidateChildrenIsCalled(t *testing.T) {
	name, other := TextBox(), TextBox()
	called := 0
	name.OnValidating(func() bool { called++; return true })
	tt := NewTester(t, VBox(4).Add(name, other), 400, 200)
	tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focus().Type("abc")
	tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	tt.Settle()
	if called != 0 {
		t.Errorf("OnValidating ran %d times without ValidateChildren", called)
	}
}

// A hidden control is not validated, and one removed from the form is not.
func TestValidateChildrenSkipsHiddenControls(t *testing.T) {
	hidden := TextBox()
	called := 0
	hidden.OnValidating(func() bool { called++; return false })
	hidden.SetVisible(false)
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(Label("x"), hidden))
	tt := TestForm(t, f)
	tt.Settle()
	if !f.ValidateChildren() || called != 0 {
		t.Errorf("a hidden control was validated (%d calls)", called)
	}
}

// The message reaches assistive technology as the control's description.
func TestErrorProviderMessageIsTheAccessibleDescription(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox()
	f := NewForm("Err a11y", 400, 200)
	f.SetContent(VBox(4).Add(Label("Name"), tb))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Err a11y")
	ep.SetError(tb, "Name is required")
	got := ""
	for i := 0; i < 20 && got == ""; i++ {
		frame(fw)
		for _, n := range accessNodes(fw) {
			if n.Description == "Name is required" {
				got = n.Description
			}
		}
	}
	if got == "" {
		t.Error("no node described by the error")
	}
}
