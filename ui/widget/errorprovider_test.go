package widget

import (
	"image/color"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
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
	before, blueBefore := redPixels(tt, box), bluePixels(tt, box)
	if blueBefore == 0 {
		t.Fatal("the TextBox icon does not draw at all")
	}
	ep.SetError(tb, "bad")
	tt.Settle()
	if redPixels(tt, box) <= before {
		t.Error("the error mark did not draw over a TextBox with an icon")
	}
	if got := bluePixels(tt, box); got < blueBefore/2 {
		t.Errorf("the TextBox icon went from %d to %d pixels once the error mark drew: they must both draw", blueBefore, got)
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

func bluePixels(tt *Tester, r ui.Rect) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c.B > 200 && c.R < 60 && c.G < 60 {
				n++
			}
		}
	}
	return n
}

// The error reaches assistive technology as an invalid value, and the
// control's own tooltip stays.
func TestErrorProviderMarksTheValueInvalidAndKeepsTheTooltip(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox().SetTooltip("Your work email")
	f := NewForm("Err invalid", 400, 200)
	f.SetContent(VBox(4).Add(Label("Email"), tb))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Err invalid")
	ep.SetError(tb, "Enter an email address")
	var node platform.AccessNode
	for i := 0; i < 20 && node.Description == ""; i++ {
		frame(fw)
		for _, n := range accessNodes(fw) {
			if n.Role == platform.RoleTextField {
				node = n
			}
		}
		if node.States&platform.AccessInvalid != 0 {
			break
		}
	}
	if node.States&platform.AccessInvalid == 0 {
		t.Errorf("the control is not invalid (states %b)", node.States)
	}
	if !strings.Contains(node.Description, "Enter an email address") || !strings.Contains(node.Description, "Your work email") {
		t.Errorf("description %q: want the error and the tooltip", node.Description)
	}
}

// A failed validation is announced once, with the first failing control's
// message.
func TestValidateChildrenAnnouncesTheFirstError(t *testing.T) {
	ep := ErrorProvider()
	a, b := TextBox(), TextBox()
	fail := func(w *TextBoxWidget, msg string) func() bool {
		return func() bool { ep.SetError(w, msg); return false }
	}
	a.OnValidating(fail(a, "first problem"))
	b.OnValidating(fail(b, "second problem"))
	f := NewForm("v", 400, 200)
	f.SetContent(VBox(4).Add(a, b))
	tt := TestForm(t, f)
	tt.Settle()
	if f.ValidateChildren() {
		t.Fatal("validation passed")
	}
	tt.Frame()
	if got := tt.Announcements(); len(got) != 1 || got[0] != "first problem" {
		t.Errorf("announced %q, want the first message once", got)
	}
}

// Every control the form holds is validated, including the pages of a tab
// control that are not showing, but not a hidden control; a disabled one is.
func TestValidateChildrenReachesInactiveTabsAndSkipsHiddenOnly(t *testing.T) {
	var ran []string
	mk := func(name string, ok bool) *TextBoxWidget {
		tb := TextBox()
		tb.OnValidating(func() bool { ran = append(ran, name); return ok })
		return tb
	}
	hidden, disabled := mk("hidden", false), mk("disabled", true)
	hidden.SetVisible(false)
	disabled.SetDisabled(true)
	tabs := TabControl().AddPage("one", VBox(4).Add(mk("page0", true))).AddPage("two", GroupBox("g").Add(mk("page1", false)))
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(tabs, hidden, disabled, Grid(4, 4).Attach(mk("grid", true), 0, 0, 1, 1), Fields().Row("Name", mk("field", true))))
	tt := TestForm(t, f)
	tt.Settle()
	if f.ValidateChildren() {
		t.Error("a failing control on a tab that is not showing was not noticed")
	}
	want := []string{"page0", "page1", "disabled", "grid", "field"}
	if !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v (layout order, hidden skipped, disabled validated)", ran, want)
	}
}

func TestErrorProviderLetsGoOfControlsTheAppDropped(t *testing.T) {
	ep := ErrorProvider()
	func() {
		tb := TextBox()
		ep.SetError(tb, "gone soon")
	}()
	for range 5 {
		runtime.GC()
	}
	if ep.HasErrors() {
		t.Error("the provider keeps a control nothing else holds")
	}
}

// The mark does not cover the text of a TextBox: the end padding grows.
func TestErrorMarkDoesNotCoverTheTextOfATextBox(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox().SetText(strings.Repeat("W", 80))
	tt := NewTester(t, VBox(8).Add(tb), 300, 100)
	tt.Find(ByRole(ui.RoleTextField)).Focus().Key(0, ui.KeyEnd) // the end of the text is at the end of the box
	tt.Settle()
	box := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	darkBefore := darkPixelsInEnd(tt, box)
	ep.SetError(tb, "bad")
	tt.Settle()
	if darkBefore == 0 {
		t.Fatal("no text at the end of the box before the error")
	}
	if darkAfter := darkPixelsInEnd(tt, box); darkAfter > darkBefore/2 {
		t.Errorf("text pixels in the room before the mark: %d before the error, %d after: the text runs under the mark", darkBefore, darkAfter)
	}
}

func darkPixelsInEnd(tt *Tester, r ui.Rect) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X + r.W - 27); x < int(r.X+r.W-21); x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c.R < 90 && c.G < 90 && c.B < 90 {
				n++
			}
		}
	}
	return n
}
