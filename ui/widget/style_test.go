package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAlignInAVBox(t *testing.T) {
	tt := tester(t, VBox(0).Add(Label("L").Align(Start), Label("R").Align(End), Label("C").Align(Center)))
	l, r, c := find(t, tt, "L"), find(t, tt, "R"), find(t, tt, "C")
	if l.X > 10 || r.X+r.W < 790 || c.X < 350 || c.X > 450 {
		t.Fatalf("L %v (want left), R %v (want right), C %v (want center)", l, r, c)
	}
}

func TestPaddingAndMarginUseThemeUnits(t *testing.T) {
	tt := tester(t, VBox(0).Add(Label("P").Align(Start).Padding(2), Label("N").Align(Start),
		Label("M").Align(Start).MarginLeft(3)))
	// Find gives the element's rect, padding included: 2 units a side.
	if p, n := find(t, tt, "P"), find(t, tt, "N"); p.H-n.H < 15.5 || p.H-n.H > 16.5 {
		t.Fatalf("padded height %v, plain %v: want 16 more (2 units of 4 a side)", p.H, n.H)
	}
	if m := find(t, tt, "M"); m.X < 11.5 || m.X > 12.5 {
		t.Fatalf("M at x %v, want 12 (3 units of 4)", m.X)
	}
}

func TestTextSizeGrowsText(t *testing.T) {
	tt := tester(t, VBox(0).Add(Label("base"), Label("big").TextSize(XXL)))
	if b, g := find(t, tt, "base"), find(t, tt, "big"); g.H <= b.H {
		t.Fatalf("XXL height %v not above base %v", g.H, b.H)
	}
}

func TestHExpandInAnHBox(t *testing.T) {
	tt := tester(t, HBox(0).Add(VBox(0).Add(Label("A")).HExpand(), Label("B")))
	if b := find(t, tt, "B"); b.X < 700 {
		t.Fatalf("B at %v: the expanded box did not take the room", b)
	}
}

func TestEveryStyleMethodBuilds(t *testing.T) {
	l := Label("s").PaddingX(1).PaddingY(1).PaddingTop(1).PaddingRight(1).PaddingBottom(1).PaddingLeft(1).
		Margin(1).MarginX(1).MarginY(1).MarginTop(1).MarginRight(1).MarginBottom(1).
		VAlign(Center).VExpand().Expand().Bold().Italic().Mono().Muted().
		TextColor(ui.RGB(255, 0, 0)).Background(ui.RGB(0, 0, 255)).Border(1, ui.RGB(0, 0, 0)).Radius(1).SetSize(100, 20)
	tt := tester(t, l)
	if !tt.HasText("s") {
		t.Fatalf("styled label not shown: %q", tt.Texts())
	}
}

// restyleLocked merges the control's styles under the package's lock, to
// read the merge without a frame.
func (b *core) restyleLocked() { mu.Lock(); defer mu.Unlock(); b.restyle() }

// pixel returns the color drawn in the control showing s, clear of its
// text and of corners up to r DIPs round: at its vertical middle, r+2 DIPs
// in from its left edge. The tester draws at scale 1.
func pixel(t *testing.T, tt *ui.Tester, s string, r float32) color.RGBA {
	t.Helper()
	b := find(t, tt, s)
	return tt.Image().RGBAAt(int(b.X+r+2), int(b.Y+b.H/2))
}

// near reports whether c is within 8 of want on each channel.
func near(c color.RGBA, want ui.Color) bool {
	d := func(a, b uint8) bool { return a-b < 9 || b-a < 9 }
	return d(c.R, want.R) && d(c.G, want.G) && d(c.B, want.B)
}

func TestApplyStylesAControl(t *testing.T) {
	l := Label("styled").Apply(NewStyle().Padding(3).Background(Fixed(red)))
	tt := tester(t, l)
	if c := pixel(t, tt, "styled", 0); !near(c, red) {
		t.Fatalf("background %v, want red", c)
	}
}

func TestMutedThenTextColorKeepsTheColor(t *testing.T) {
	// Today's order still works: a later TextColor wins over Muted.
	l := Label("m").Muted().TextColor(red)
	l.base().restyleLocked()
	if l.style.text.fixed != red {
		t.Fatalf("text %v, want red", l.style.text)
	}
}
