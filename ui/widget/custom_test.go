package widget

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCustomBuildsItsElements(t *testing.T) {
	tt := tester(t, Custom(func(c *ui.Context) ui.Element { return ui.Text(c, "drawn") }))
	if !tt.HasText("drawn") {
		t.Fatalf("no drawn in %q", tt.Texts())
	}
}

func TestCustomNilOrZeroShowsNothing(t *testing.T) {
	box := VBox(4).Add(
		Custom(nil),
		Custom(func(c *ui.Context) ui.Element { return ui.Element{} }),
		Label("after"),
	)
	tt := tester(t, box)
	if !tt.HasText("after") {
		t.Fatalf("the label after empty customs is missing: %q", tt.Texts())
	}
}

func TestCustomHasCommonMethods(t *testing.T) {
	cw := Custom(func(c *ui.Context) ui.Element { return ui.Text(c, "x") })
	tt := tester(t, cw)
	cw.SetVisible(false)
	tt.Frame()
	if tt.HasText("x") {
		t.Fatalf("hidden custom shown: %q", tt.Texts())
	}
}

// gauge is a user control: a type of the app that embeds *CustomWidget,
// as WinForms' UserControl derives from Control.
type gauge struct {
	*CustomWidget
	value int
}

func newGauge() *gauge {
	g := &gauge{}
	g.CustomWidget = Custom(func(c *ui.Context) ui.Element {
		return ui.Text(c, "value "+strconv.Itoa(g.value))
	})
	return g
}

func (g *gauge) SetValue(v int) *gauge { Change(func() { g.value = v }); return g }

func TestUserControlSetterFromAnotherGoroutine(t *testing.T) {
	g := newGauge()
	tt := tester(t, g)
	done := make(chan struct{})
	go func() { g.SetValue(7); close(done) }()
	<-done
	tt.Frame()
	if !tt.HasText("value 7") {
		t.Fatalf("texts %q, want value 7", tt.Texts())
	}
}

func TestUserControlInTwoBoxesShowsInBoth(t *testing.T) {
	g := newGauge().SetValue(3)
	tt := tester(t, VBox(0).Add(VBox(0).Add(g), VBox(0).Add(g)))
	if n := count(tt, "value 3"); n != 2 {
		t.Fatalf("value 3 shown %d times, want 2: %q", n, tt.Texts())
	}
}
