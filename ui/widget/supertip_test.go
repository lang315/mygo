package widget

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// The tip colors of tipThemes: far from the plain themes' and from each
// other, so that a pixel tells which one drew.
var (
	tipLightFill, tipLightText = ui.Hex("#1a2b3c"), ui.Hex("#f0e0d0")
	tipDarkFill, tipDarkText   = ui.Hex("#d0e0f0"), ui.Hex("#302010")
)

// tipThemes sets tooltip colors in the default themes, as a skin does,
// until the test ends.
func tipThemes(t *testing.T) {
	t.Helper()
	light, dark := ui.LightTheme(), ui.DarkTheme()
	light.Inverse, light.InverseText = tipLightFill, tipLightText
	dark.Inverse, dark.InverseText = tipDarkFill, tipDarkText
	ui.SetDefaultThemes(light, dark)
	t.Cleanup(func() { ui.SetDefaultThemes(nil, nil) })
}

// tipWindow is a Tester of controls with room above them for their tips.
func tipWindow(t *testing.T, controls ...Control) *Tester {
	t.Helper()
	return NewTester(t, VBox(80).Add(append([]Control{Label("Top")}, controls...)...), 400, 400)
}

// hoverTip moves the pointer away long enough for no tip to show at once,
// then rests it on r for d.
func hoverTip(tt *Tester, r ui.Rect, d time.Duration) {
	tt.Move(395, 395)
	tt.Advance(2 * time.Second)
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Advance(d)
	tt.Frame()
}

// colorBox returns the box of the pixels of exactly c in the last frame.
func colorBox(tt *Tester, c ui.Color) (ui.Rect, bool) {
	img := tt.Image()
	b := img.Bounds()
	x0, y0, x1, y1 := b.Max.X, b.Max.Y, -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if p := img.RGBAAt(x, y); p.R == c.R && p.G == c.G && p.B == c.B && p.A == 255 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: float32(x0), Y: float32(y0), W: float32(x1 - x0 + 1), H: float32(y1 - y0 + 1)}, true
}

// textAt returns the box of text s, failing the test when none shows.
func tipTextAt(t *testing.T, tt *Tester, s string) ui.Rect {
	t.Helper()
	r, ok := tt.Tester.Find(s)
	if !ok {
		t.Fatalf("no text %q; texts %q", s, tt.Texts())
	}
	return r
}

func tipNear(a, b, tol float32) bool { return a-b <= tol && b-a <= tol }

// tipGreen is a 20 by 20 bitmap of a color no theme has.
var tipGreen = func() *ui.Bitmap {
	m := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(m, m.Bounds(), &image.Uniform{color.RGBA{0, 255, 0, 255}}, image.Point{}, draw.Src)
	return ui.NewBitmap(m)
}()

// tipGreenBox returns the box of the bitmap's pixels in the last frame.
func tipGreenBox(tt *Tester) (ui.Rect, bool) { return colorBox(tt, ui.RGB(0, 255, 0)) }

// The SuperTip shows after the text tooltip's delay, at its place, and hides
// on a click and on Escape.
func TestSuperTipTiming(t *testing.T) {
	tipThemes(t)
	plain := Button("Plain").SetTooltip("A plain tip")
	rich := Button("Rich").SetSuperTip(SuperTip{Title: "A rich tip"})
	tt := tipWindow(t, plain, rich)
	gaps := map[string]float32{}
	for _, c := range []struct{ button, tip string }{{"Plain", "A plain tip"}, {"Rich", "A rich tip"}} {
		b := tt.Find(ByRole(ui.RoleButton).Label(c.button)).Bounds()
		hoverTip(tt, b, 500*time.Millisecond)
		if tt.HasText(c.tip) {
			t.Fatalf("%s: the tip shows before the pointer rested 0.6 seconds", c.button)
		}
		tt.Advance(150 * time.Millisecond)
		tt.Frame()
		if !tt.HasText(c.tip) {
			t.Fatalf("%s: no tip once the pointer rested 0.6 seconds; texts %q", c.button, tt.Texts())
		}
		box, ok := colorBox(tt, tipLightFill)
		if !ok {
			t.Fatalf("%s: no tip drawn in the theme's Inverse", c.button)
		}
		if !tipNear(box.X+box.W/2, b.X+b.W/2, 1) {
			t.Errorf("%s: the tip is at %v, not centered above %v", c.button, box, b)
		}
		gaps[c.button] = b.Y - (box.Y + box.H)
	}
	if gaps["Plain"] <= 0 || !tipNear(gaps["Rich"], gaps["Plain"], 1) {
		t.Errorf("the tips are %v above their buttons: want the SuperTip where the text tip goes", gaps)
	}
	// A click hides it.
	b := tt.Find(ByRole(ui.RoleButton).Label("Rich")).Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if tt.HasText("A rich tip") {
		t.Error("the SuperTip shows after a click")
	}
	// Escape hides it.
	hoverTip(tt, b, time.Second)
	if !tt.HasText("A rich tip") {
		t.Fatalf("no SuperTip as the pointer comes back; texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.HasText("A rich tip") {
		t.Error("the SuperTip shows after Escape")
	}
}

// superTipShown is a Tester of a button with tip s, its tip showing.
func superTipShown(t *testing.T, s SuperTip) *Tester {
	t.Helper()
	tt := tipWindow(t, Button("Help").SetSuperTip(s))
	hoverTip(tt, tt.Find(ByRole(ui.RoleButton).Label("Help")).Bounds(), time.Second)
	return tt
}

const tipBody = "Puts the text of the clipboard where the caret is, replacing the selected text, and keeps the clipboard as it was for the next paste."

// Bold title; the body wraps at 320 DIPs with the image at its start (its
// end in RTL); a separator, then the footer. Empty parts take no room.
func TestSuperTipLayout(t *testing.T) {
	tipThemes(t)
	full := SuperTip{Title: "Paste", Body: tipBody, Image: tipGreen, Footer: "Press F1 for help"}
	tt := superTipShown(t, full)
	box, ok := colorBox(tt, tipLightFill)
	if !ok {
		t.Fatalf("no tip; texts %q", tt.Texts())
	}
	if box.W > 320 || box.W < 300 {
		t.Errorf("the tip of a long body is %v wide, want up to 320 DIPs", box.W)
	}
	title, body, footer := tipTextAt(t, tt, "Paste"), tipTextAt(t, tt, tipBody), tipTextAt(t, tt, "Press F1 for help")
	if body.H < 2*title.H {
		t.Errorf("the body is %v high, one line %v: it does not wrap", body.H, title.H)
	}
	img, ok := tipGreenBox(tt)
	if !ok {
		t.Fatal("the image is not drawn")
	}
	if !(title.Y+title.H <= body.Y && body.Y+body.H <= footer.Y) {
		t.Errorf("title %v, body %v, footer %v: want them in that order", title, body, footer)
	}
	if img.X+img.W > body.X || !tipNear(img.Y, body.Y, 2) {
		t.Errorf("the image is at %v, not at the start of the body %v", img, body)
	}
	sep := false
	for y := body.Y + body.H; y < footer.Y; y++ {
		p := tt.Image().RGBAAt(int(box.X+box.W/2), int(y))
		if p.R != tipLightFill.R || p.G != tipLightFill.G || p.B != tipLightFill.B {
			sep = true
		}
	}
	if !sep {
		t.Error("no separator between the body and the footer")
	}
	tt.SetDirection(ui.RightToLeft)
	tt.Frame()
	body = tipTextAt(t, tt, tipBody)
	if img, _ = tipGreenBox(tt); img.X < body.X+body.W {
		t.Errorf("RTL: the image is at %v, not at the end of the body %v", img, body)
	}
	tt.SetDirection(ui.LeftToRight)

	// The title is bold: wider than the same text in the body.
	bold := tipTextAt(t, superTipShown(t, SuperTip{Title: "Paste the words"}), "Paste the words")
	regular := tipTextAt(t, superTipShown(t, SuperTip{Body: "Paste the words"}), "Paste the words")
	if bold.W <= regular.W {
		t.Errorf("the title is %v wide, the same text in the body %v: not bold", bold.W, regular.W)
	}
	// One part alone: the tip is its line and the padding, 8 DIPs a side.
	for _, s := range []SuperTip{{Title: "Xy"}, {Body: "Xy"}, {Footer: "Xy"}} {
		tt := superTipShown(t, s)
		box, ok := colorBox(tt, tipLightFill)
		line := tipTextAt(t, tt, "Xy")
		if !ok || !tipNear(box.H, line.H+16, 1) {
			t.Errorf("%+v: the tip is %v high, its line %v: want the line and 16", s, box.H, line.H)
		}
		if _, ok := tipGreenBox(tt); ok {
			t.Errorf("%+v: an image is drawn", s)
		}
	}
}

// With both set, the SuperTip shows; a zero SuperTip brings the text back.
func TestSuperTipReplacesTooltip(t *testing.T) {
	tipThemes(t)
	b := Button("Copy").SetTooltip("Plain words").SetSuperTip(SuperTip{Title: "Rich title"})
	tt := tipWindow(t, b)
	r := tt.Find(ByRole(ui.RoleButton).Label("Copy")).Bounds()
	hoverTip(tt, r, time.Second)
	if !tt.HasText("Rich title") || tt.HasText("Plain words") {
		t.Errorf("both set: texts %q, want the SuperTip alone", tt.Texts())
	}
	if a, _ := tt.AccessByRole(ui.RoleButton, "Copy"); a.Description != "Rich title" {
		t.Errorf("both set: description %q, want the SuperTip's", a.Description)
	}
	b.SetSuperTip(SuperTip{})
	hoverTip(tt, r, time.Second)
	if tt.HasText("Rich title") || !tt.HasText("Plain words") {
		t.Errorf("SuperTip removed: texts %q, want the text tooltip", tt.Texts())
	}
	if a, _ := tt.AccessByRole(ui.RoleButton, "Copy"); a.Description != "Plain words" {
		t.Errorf("SuperTip removed: description %q, want the tooltip", a.Description)
	}
}

// An ErrorProvider's error shows as the tip in place of the SuperTip, as it
// does of a text tooltip, and the SuperTip stays the description.
func TestSuperTipYieldsToError(t *testing.T) {
	tipThemes(t)
	ep := ErrorProvider()
	tb := TextBox().SetSuperTip(SuperTip{Title: "Rich title"})
	tt := tipWindow(t, tb)
	ep.SetError(tb, "Enter a name")
	tt.Frame()
	hoverTip(tt, tt.Find(ByRole(ui.RoleTextField)).Bounds(), time.Second)
	if !tt.HasText("Enter a name") || tt.HasText("Rich title") {
		t.Errorf("with an error: texts %q, want the error's tip alone", tt.Texts())
	}
	if a, _ := tt.AccessByRole(ui.RoleTextField, ""); a.Description != "Enter a name\nRich title" {
		t.Errorf("with an error: description %q, want the error, then the SuperTip's", a.Description)
	}
}

// Assistive technology reads Title, Body and Footer, joined, as the
// control's description.
func TestSuperTipAccessible(t *testing.T) {
	for _, c := range []struct {
		tip  SuperTip
		want string
	}{
		{SuperTip{Title: "Paste", Body: "Puts the clipboard's text at the caret.", Footer: "Press F1 for help."},
			"Paste. Puts the clipboard's text at the caret. Press F1 for help."},
		{SuperTip{Title: "Paste", Footer: "Press F1"}, "Paste. Press F1"},
		{SuperTip{Body: "Only a body", Image: tipGreen}, "Only a body"},
		{SuperTip{Title: "Done!", Body: "Saved"}, "Done! Saved"},
	} {
		tt := NewTester(t, Button("Go").SetSuperTip(c.tip), 300, 200)
		if a, ok := tt.AccessByRole(ui.RoleButton, "Go"); !ok || a.Description != c.want {
			t.Errorf("%+v: description %q, want %q", c.tip, a.Description, c.want)
		}
	}
	// A Fields row's description stays the control's, as it does over a
	// text tooltip.
	tb := TextBox().SetSuperTip(SuperTip{Title: "Rich"})
	tt := NewTester(t, Fields().SettingsRows().Row("Name", tb).Describe("Your name."), 400, 300)
	if d := descriptionOf(t, tt, "Name"); d != "Your name." {
		t.Errorf("in a described row: description %q, want the row's", d)
	}
}

// The colors come from the skin's tooltip colors (the theme's Inverse), in
// light and dark, and from the theme turned over where it sets none.
func TestSuperTipSkin(t *testing.T) {
	tipThemes(t)
	tt := superTipShown(t, SuperTip{Title: "Skinned", Body: "Body text"})
	if _, ok := colorBox(tt, tipLightFill); !ok {
		t.Error("light: the tip is not filled with the skin's Inverse")
	}
	title := tipTextAt(t, tt, "Skinned")
	if !tipHasColor(tt, title, tipLightText) {
		t.Error("light: the title is not drawn in the skin's InverseText")
	}
	tt.SetDark(true)
	tt.Frame()
	if _, ok := colorBox(tt, tipDarkFill); !ok {
		t.Error("dark: the tip is not filled with the skin's Inverse")
	}
	if !tipHasColor(tt, tipTextAt(t, tt, "Skinned"), tipDarkText) {
		t.Error("dark: the title is not drawn in the skin's InverseText")
	}
	ui.SetDefaultThemes(nil, nil)
	tt.SetDark(false)
	tt.Frame()
	if _, ok := colorBox(tt, ui.LightTheme().Text); !ok {
		t.Error("no skin: the tip is not filled with the theme's Text, as a text tooltip")
	}
}

// hasColor reports whether r holds a pixel of exactly c.
func tipHasColor(tt *Tester, r ui.Rect, c ui.Color) bool {
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if p := tt.Image().RGBAAt(x, y); p.R == c.R && p.G == c.G && p.B == c.B {
				return true
			}
		}
	}
	return false
}

// A control with no SuperTip builds, draws and allocates as one whose
// SuperTip was removed, which is as on main: no func installed.
func TestSuperTipOffIsUnchanged(t *testing.T) {
	never := Button("A").SetTooltip("tip")
	removed := Button("A").SetTooltip("tip").SetSuperTip(SuperTip{Title: "x"}).SetSuperTip(SuperTip{})
	mu.Lock()
	installed := never.withSuperTip != nil || never.superTip != nil || removed.withSuperTip != nil || removed.superTip != nil
	mu.Unlock()
	if installed {
		t.Error("a control with no SuperTip carries one")
	}
	a, b := tipWindow(t, never), tipWindow(t, removed)
	for _, at := range []time.Duration{0, time.Second} {
		ra := a.Find(ByRole(ui.RoleButton).Label("A")).Bounds()
		hoverTip(a, ra, at)
		hoverTip(b, b.Find(ByRole(ui.RoleButton).Label("A")).Bounds(), at)
		if !tipSamePixels(a.Image(), b.Image()) {
			t.Errorf("rested %v: the frames differ", at)
		}
		da, _ := a.AccessByRole(ui.RoleButton, "A")
		db, _ := b.AccessByRole(ui.RoleButton, "A")
		if da.Description != "tip" || db.Description != "tip" {
			t.Errorf("descriptions %q and %q, want the tooltip", da.Description, db.Description)
		}
	}
	plain := ui.NewTester(content(VBox(4).Add(TextBox(), Button("B"))), 300, 100)
	off := ui.NewTester(content(VBox(4).Add(TextBox(), Button("B").SetSuperTip(SuperTip{Title: "x"}).SetSuperTip(SuperTip{}))), 300, 100)
	for range 10 {
		plain.Frame()
		off.Frame()
	}
	if want, got := allocrun.Least(30, plain.Frame), allocrun.Least(30, off.Frame); got != want {
		t.Errorf("a frame with a removed SuperTip allocates %v, one with none %v", got, want)
	}
}

func tipSamePixels(a, b *image.RGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

// A frame at rest of a control with a SuperTip not showing allocates
// nothing more than one with a text tooltip.
func TestSuperTipAllocNothingAtRest(t *testing.T) {
	plain := ui.NewTester(content(VBox(4).Add(Button("B").SetTooltip("tip"))), 300, 100)
	with := ui.NewTester(content(VBox(4).Add(Button("B").SetSuperTip(SuperTip{Title: "x", Body: "y", Footer: "z"}))), 300, 100)
	for range 10 {
		plain.Frame()
		with.Frame()
	}
	if want, got := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame); got > want {
		t.Errorf("a frame of a SuperTip at rest allocates %v, a text tooltip's %v", got, want)
	}
}

// A SuperTip set from another goroutine while it shows shows on the next
// frame.
func TestSuperTipFromGoroutine(t *testing.T) {
	tipThemes(t)
	b := Button("Go").SetSuperTip(SuperTip{Title: "First"})
	tt := tipWindow(t, b)
	hoverTip(tt, tt.Find(ByRole(ui.RoleButton).Label("Go")).Bounds(), time.Second)
	if !tt.HasText("First") {
		t.Fatalf("no SuperTip; texts %q", tt.Texts())
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				b.SetSuperTip(SuperTip{Title: "Last", Body: "from the goroutine", Image: tipGreen})
				return
			default:
			}
			if i%2 == 0 {
				b.SetSuperTip(SuperTip{Title: "Even", Footer: "f"})
			} else {
				b.SetSuperTip(SuperTip{Body: "Odd", Image: tipGreen})
			}
		}
	}()
	for range 50 {
		tt.Frame()
	}
	close(stop)
	wg.Wait()
	tt.Frame()
	if !tt.HasText("Last") || !tt.HasText("from the goroutine") {
		t.Errorf("the SuperTip set from another goroutine does not show; texts %q", tt.Texts())
	}
	done := make(chan struct{})
	go func() { b.SetSuperTip(SuperTip{}); close(done) }()
	<-done
	tt.Frame()
	if tt.HasText("Last") {
		t.Errorf("the SuperTip removed from another goroutine still shows; texts %q", tt.Texts())
	}
}
