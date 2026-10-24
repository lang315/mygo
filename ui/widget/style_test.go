package widget

import (
	"image/color"
	"testing"
	"time"

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
	d := func(a, b uint8) bool { x := int(a) - int(b); return x > -9 && x < 9 }
	return d(c.R, want.R) && d(c.G, want.G) && d(c.B, want.B)
}

// page is the color of the page at the bottom left of the window: well
// outside the few controls these tests show at its top, which a box
// stretches across its width.
func page(tt *ui.Tester) color.RGBA {
	img := tt.Image()
	r := img.Bounds()
	return img.RGBAAt(r.Min.X+2, r.Max.Y-3)
}

// pixelLeft returns the color drawn left of the text s, in the padding of
// the control around it: tt.Find gives a button's label text, and the
// padding beside it shows the button's fill. It fails if that is the page's
// color, as a padding under 4 DIPs would give.
func pixelLeft(t *testing.T, tt *ui.Tester, s string) color.RGBA {
	t.Helper()
	b := find(t, tt, s)
	c := tt.Image().RGBAAt(int(b.X-4), int(b.Y+b.H/2))
	if c == page(tt) {
		t.Fatalf("the point left of %q shows the page %v, not the control", s, c)
	}
	return c
}

// mix is c over bg at the opacity f.
func mix(c ui.Color, bg color.RGBA, f float32) ui.Color {
	m := func(a, b uint8) uint8 { return uint8(float32(a)*f + float32(b)*(1-f) + 0.5) }
	return ui.RGB(m(c.R, bg.R), m(c.G, bg.G), m(c.B, bg.B))
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

var green = ui.RGB(0, 160, 0)

// card is padded so pixel samples its background.
func card() Style { return NewStyle().Padding(3).Radius(1) }

func centerOf(t *testing.T, tt *ui.Tester, s string) (float32, float32) {
	t.Helper()
	b := find(t, tt, s)
	return b.X + b.W/2, b.Y + b.H/2
}

func TestStyleHoverShowsAndGoes(t *testing.T) {
	l := Label("hov").Apply(card().Background(Fixed(red)).Hover(NewStyle().Background(Fixed(blue))))
	tt := tester(t, VBox(0).Add(l, Label("elsewhere")))
	tt.Move(centerOf(t, tt, "hov"))
	if c := pixel(t, tt, "hov", 4); !near(c, blue) {
		t.Fatalf("hovered %v, want blue", c)
	}
	tt.Move(centerOf(t, tt, "elsewhere"))
	if c := pixel(t, tt, "hov", 4); !near(c, red) {
		t.Fatalf("after leaving %v, want red", c)
	}
}

func TestStylePressedWinsOverHover(t *testing.T) {
	b := Button("press").Apply(card().Background(Fixed(red)).
		Hover(NewStyle().Background(Fixed(blue))).Pressed(NewStyle().Background(Fixed(green))))
	tt := tester(t, b)
	x, y := centerOf(t, tt, "press")
	tt.Move(x, y)
	tt.Press(x, y)
	if c := pixelLeft(t, tt, "press"); !near(c, green) {
		t.Fatalf("pressed %v, want green", c)
	}
	tt.Release(x, y)
	if c := pixelLeft(t, tt, "press"); !near(c, blue) {
		t.Fatalf("released, still hovered: %v, want blue", c)
	}
}

func TestStyleDisabledWinsOverAll(t *testing.T) {
	l := Label("dis").Apply(card().Background(Fixed(red)).
		Hover(NewStyle().Background(Fixed(blue))).Disabled(NewStyle().Background(Fixed(green))))
	tt := tester(t, l)
	tt.Move(centerOf(t, tt, "dis"))
	l.SetDisabled(true)
	tt.Frame()
	// ui draws a disabled element at 50% opacity, over the page.
	want := mix(green, page(tt), 0.5)
	if c := pixel(t, tt, "dis", 4); !near(c, want) {
		t.Fatalf("disabled and hovered %v, want green at half opacity %v", c, want)
	}
}

func TestStyleDarkAndDarkHover(t *testing.T) {
	l := Label("dk").Apply(card().Background(Fixed(red)).Dark(NewStyle().Background(Fixed(blue))).
		Hover(NewStyle().Background(Fixed(green)).Dark(NewStyle().Background(Fixed(ui.RGB(200, 200, 0))))))
	tt := tester(t, VBox(0).Add(l, Label("away")))
	tt.SetDark(true)
	if c := pixel(t, tt, "dk", 4); !near(c, blue) {
		t.Fatalf("dark %v, want blue", c)
	}
	tt.Move(centerOf(t, tt, "dk"))
	if c := pixel(t, tt, "dk", 4); !near(c, ui.RGB(200, 200, 0)) {
		t.Fatalf("dark hover %v, want yellow", c)
	}
}

func TestStyleRoleFollowsTheTheme(t *testing.T) {
	l := Label("role").Apply(card().Background(RoleAccent))
	tt := tester(t, l)
	light := pixel(t, tt, "role", 4)
	tt.SetDark(true)
	dark := pixel(t, tt, "role", 4)
	if !near(light, ui.LightTheme().Accent) {
		t.Fatalf("light %v, want the light accent %v", light, ui.LightTheme().Accent)
	}
	if near(dark, ui.LightTheme().Accent) && ui.LightTheme().Accent != ui.DarkTheme().Accent {
		t.Fatalf("dark %v still the light accent", dark)
	}
}

func TestOwnStyleWinsAndPerSideKeepsTheRest(t *testing.T) {
	l := Label("own").Apply(card().Background(Fixed(red)), NewStyle().Background(Fixed(blue))).PaddingTop(0)
	tt := tester(t, VBox(0).Add(l, Label("plain")))
	if c := pixel(t, tt, "own", 4); !near(c, blue) {
		t.Fatalf("later Apply style should win: %v", c)
	}
	l.Background(green)
	tt.Frame()
	if c := pixel(t, tt, "own", 4); !near(c, green) {
		t.Fatalf("own style should win over Apply: %v", c)
	}
	o, p := find(t, tt, "own"), find(t, tt, "plain")
	if d := o.H - p.H; d < 11.5 || d > 12.5 { // 3 units below only: 12 DIPs
		t.Fatalf("own height %v vs plain %v: PaddingTop(0) should keep the bottom's 3 units", o.H, p.H)
	}
}

func TestUnsetPropertiesKeepTheButtonsHover(t *testing.T) {
	b := Button("native").Apply(NewStyle().Bold())
	tt := tester(t, VBox(0).Add(b, Label("away")))
	before := pixelLeft(t, tt, "native")
	tt.Move(centerOf(t, tt, "native"))
	after := pixelLeft(t, tt, "native")
	if before == after {
		t.Fatalf("ui.Button's own hover color is gone: %v both", before)
	}
}

func TestStyleTransitionFades(t *testing.T) {
	const d = 200 * time.Millisecond
	l := Label("fade").Apply(card().Background(Fixed(ui.RGB(0, 0, 0))).
		Hover(NewStyle().Background(Fixed(ui.RGB(240, 240, 240)))).Transition(d))
	tt := tester(t, VBox(0).Add(l, Label("away")))
	tt.Move(centerOf(t, tt, "fade"))
	tt.Advance(d / 2)
	mid := pixel(t, tt, "fade", 4)
	if mid.R < 20 || mid.R > 220 {
		t.Fatalf("halfway %v, want between black and white", mid)
	}
	tt.Advance(d)
	if end := pixel(t, tt, "fade", 4); !near(end, ui.RGB(240, 240, 240)) {
		t.Fatalf("after the transition %v, want the hover color", end)
	}
}

// The documented case: the base style sets a TextColor too, so the text has
// a color to fade from. (With a variant's alone it jumps: the style cannot
// know the color the control's text rests in.)
func TestStyleTextColorFadesWhenTheBaseSetsOneToo(t *testing.T) {
	const d = 200 * time.Millisecond
	l := Label("TTTTTTTT").Apply(NewStyle().TextColor(Fixed(ui.RGB(0, 0, 0))).
		Hover(NewStyle().TextColor(Fixed(ui.RGB(230, 0, 0)))).Transition(d)).TextSize(XXXL)
	tt := tester(t, VBox(0).Add(l, Label("away")))
	// The thresholds are relative to what this test measures at rest and at
	// the end: how much of a glyph pixel is text depends on the platform's
	// text coverage, so it samples one pixel the text covers fully (the
	// darkest at rest) throughout, and the ease-out curve is most of the way
	// at d/2, so the middle is sampled at d/4.
	b, img := find(t, tt, "TTTTTTTT"), tt.Image()
	px, py, dark := 0, 0, 3*256
	for y := int(b.Y); y < int(b.Y+b.H); y++ {
		for x := int(b.X); x < int(b.X+b.W); x++ {
			if c := img.RGBAAt(x, y); int(c.R)+int(c.G)+int(c.B) < dark {
				px, py, dark = x, y, int(c.R)+int(c.G)+int(c.B)
			}
		}
	}
	red := func() int { return int(tt.Image().RGBAAt(px, py).R) }
	rest := red()
	tt.Move(centerOf(t, tt, "TTTTTTTT"))
	tt.Advance(d / 4)
	mid := red()
	tt.Advance(d)
	end := red()
	span := end - rest
	if span <= 100 || mid <= rest+span/10 || mid >= end-span/10 {
		t.Fatalf("text red at rest %d, a quarter into the fade %d, at the end %d: want the end more than 100 above rest and the middle at least a tenth of that from both", rest, mid, end)
	}
}

// Apply while frames build: the merge is made under the package's lock
// (go test -race sees if it is not), and the frame after the last call
// shows its style.
func TestApplyFromAnotherGoroutineWhileFramesBuild(t *testing.T) {
	a := card().Background(Fixed(red))
	b := card().Background(Fixed(blue))
	l := Label("race")
	tt := tester(t, l)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 200 {
			if i%2 == 0 {
				l.Apply(a)
			} else {
				l.Apply(b)
			}
		}
	}()
	for running := true; running; {
		tt.Frame()
		select {
		case <-done:
			running = false
		default:
		}
	}
	tt.Frame()
	if c := pixel(t, tt, "race", 4); !near(c, blue) { // the 200th call, i = 199, applies b
		t.Fatalf("last Apply not shown: %v", c)
	}
}

// The docs say so: a style that sets a background gives a static one, unless
// it has a Hover variant too.
func TestStyleStaticBackgroundTurnsTheButtonsHoverOff(t *testing.T) {
	b := Button("stat").Apply(card().Background(Fixed(red)))
	tt := tester(t, VBox(0).Add(b, Label("away")))
	before := pixelLeft(t, tt, "stat")
	tt.Move(centerOf(t, tt, "stat"))
	if after := pixelLeft(t, tt, "stat"); after != before {
		t.Fatalf("hover changed a static background: %v then %v", before, after)
	}
}

func TestStyleBadValuesDoNotPanic(t *testing.T) {
	l := Label("bad").Apply(NewStyle().Opacity(2).Padding(-3).Radius(1, 2, 3).Blur(-1))
	tt := tester(t, l)
	if !tt.HasText("bad") {
		t.Fatalf("not shown: %q", tt.Texts())
	}
}

// box is a Custom control: a column padded and outlined as its builder
// sets it, with the text "in" at its content's top left.
func box(build func(e ui.Element) ui.Element) *CustomWidget {
	return Custom(func(c *ui.Context) ui.Element {
		return build(ui.Column(c).Children(func() { ui.Text(c, "in") }))
	})
}

func TestOneSideKeepsTheOthers(t *testing.T) {
	styled := box(func(e ui.Element) ui.Element { return e.Padding(8) }).Apply(NewStyle().PaddingTop(1))
	want := box(func(e ui.Element) ui.Element { return e.Padding(4, 8, 8, 8) })
	got, ref := find(t, tester(t, VBox(0).Add(styled)), "in"), find(t, tester(t, VBox(0).Add(want)), "in")
	if got != ref {
		t.Fatalf("PaddingTop(1) put the text at %v, want %v: the other sides must keep 8", got, ref)
	}
}

func TestOneSideKeepsLogicalPadding(t *testing.T) {
	styled := box(func(e ui.Element) ui.Element { return e.PaddingStart(12) }).Apply(NewStyle().PaddingTop(1))
	want := box(func(e ui.Element) ui.Element { return e.PaddingStart(12).PaddingY(4) })
	if got, ref := find(t, tester(t, VBox(0).Add(styled)), "in"), find(t, tester(t, VBox(0).Add(want)), "in"); got.X != ref.X {
		t.Fatalf("PaddingTop(1) moved the text to x %v from %v", got.X, ref.X)
	}
}

func TestOneCornerKeepsTheOthers(t *testing.T) {
	red := ui.RGB(230, 0, 0)
	w := box(func(e ui.Element) ui.Element { return e.Width(60).Height(60).Background(red).Radius(16) }).
		Apply(NewStyle().RadiusTopLeft(0))
	tt := tester(t, VBox(0).Add(w))
	in := find(t, tt, "in") // the column's top left: no padding
	tl := tt.Image().RGBAAt(int(in.X)+1, int(in.Y)+1)
	tr := tt.Image().RGBAAt(int(in.X)+58, int(in.Y)+1)
	if !near(tl, red) {
		t.Fatalf("top-left corner %v, want square and red", tl)
	}
	if near(tr, red) {
		t.Fatalf("top-right corner %v is red: it must keep its 16 DIP radius", tr)
	}
}

func TestOneBorderSideKeepsTheOthers(t *testing.T) {
	black := ui.RGB(0, 0, 0)
	w := box(func(e ui.Element) ui.Element {
		return e.Width(60).Height(60).Background(ui.RGB(255, 255, 255)).BorderWidth(3).BorderColor(black)
	}).Apply(NewStyle().BorderLeft(0).BorderColor(Fixed(black)))
	tt := tester(t, VBox(0).Add(w))
	in := find(t, tt, "in")
	// The box is at the window's left edge, x 0: its border, where it has
	// one, covers x 0 to 2, and the text sits inside it.
	left := tt.Image().RGBAAt(1, int(in.Y)+20)
	right := tt.Image().RGBAAt(int(in.X)+58, int(in.Y)+20)
	if near(left, black) {
		t.Fatalf("left edge %v is still black after BorderLeft(0)", left)
	}
	if !near(right, black) {
		t.Fatalf("right edge %v lost its 3 DIP border", right)
	}
}

func TestStyleAlignMatchesTheMethod(t *testing.T) {
	styled := find(t, tester(t, VBox(0).Add(Label("x").Apply(NewStyle().Align(End)))), "x")
	method := find(t, tester(t, VBox(0).Add(Label("x").Align(End))), "x")
	plain := find(t, tester(t, VBox(0).Add(Label("x"))), "x")
	if styled != method {
		t.Fatalf("Style.Align(End) put x at %v, the method at %v", styled, method)
	}
	if styled == plain {
		t.Fatal("Align(End) did not move the label")
	}
}

func TestOwnAlignWinsOverApplied(t *testing.T) {
	got := find(t, tester(t, VBox(0).Add(Label("x").Apply(NewStyle().Align(End)).Align(Start))), "x")
	want := find(t, tester(t, VBox(0).Add(Label("x").Align(Start))), "x")
	if got != want {
		t.Fatalf("x at %v, want %v: the widget's own Align wins", got, want)
	}
}

func TestLayoutInAVariantIsIgnored(t *testing.T) {
	tt := tester(t, VBox(0).Add(Label("x").Apply(NewStyle().Hover(NewStyle().Align(End)))))
	before := find(t, tt, "x")
	tt.Move(before.X+1, before.Y+before.H/2)
	tt.Frame()
	if after := find(t, tt, "x"); after != before {
		t.Fatalf("hover moved x from %v to %v", before, after)
	}
}

func TestStyleHExpandMatchesTheMethod(t *testing.T) {
	styled := find(t, tester(t, HBox(0).Add(Button("a").Apply(NewStyle().HExpand()), Button("b"))), "b")
	method := find(t, tester(t, HBox(0).Add(Button("a").HExpand(), Button("b"))), "b")
	if styled != method {
		t.Fatalf("b at %v with Style.HExpand, %v with the method", styled, method)
	}
}

func TestFontWeightBoldsText(t *testing.T) {
	heavy := find(t, tester(t, VBox(0).Add(Label("Wide text here").Align(Start).Apply(NewStyle().FontWeight(700)))), "Wide text here")
	plain := find(t, tester(t, VBox(0).Add(Label("Wide text here").Align(Start))), "Wide text here")
	if heavy.W <= plain.W {
		t.Fatalf("weight 700 is %v wide, plain %v: want wider", heavy.W, plain.W)
	}
}
