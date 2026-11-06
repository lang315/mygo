package widget

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

// fitted is a tester of d, as high as its content at its width, as its window is.
func fitted(t *testing.T, d *Dialog) *DialogTester {
	dt := TestDialog(t, d)
	d.SetSize(420, 0)
	dt.Frame()
	return dt
}

// The caption shows the title; the dialog does not repeat it.
func TestMessageTitleShowsOnce(t *testing.T) {
	d, _ := messageDialog(nil, Info, 0, "Heading", "Body", nil, nil, "OK")
	dt := TestDialog(t, d)
	n := 0
	for _, s := range dt.Texts() {
		if s == "Heading" {
			n++
		}
	}
	if n != 0 {
		t.Fatalf("the title shows %d times in the view: %q", n, dt.Texts())
	}
	// A plain NewDialog keeps its bold title.
	p := NewDialog(nil, "Plain", "", 300, 200, Label("x"))
	pt := TestDialog(t, p)
	if !strings.Contains(strings.Join(pt.Texts(), "|"), "Plain") {
		t.Fatalf("NewDialog lost its title: %q", pt.Texts())
	}
}

// A message dialog has the close button only, as a WinForms MessageBox.
func TestMessageHasCloseButtonOnly(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Frame?", "") })
	fw := waitFake(t, "Frame?")
	o := fw.Opts
	if o.Minimizable || o.Maximizable || o.Resizable || !o.Closable {
		t.Fatalf("minimizable %v maximizable %v resizable %v closable %v", o.Minimizable, o.Maximizable, o.Resizable, o.Closable)
	}
	press(t, fw, "No")
	received(t, got)
	// Other dialogs keep theirs.
	d := NewDialog(nil, "Other", "", 300, 200, Label("x"))
	d.Show()
	t.Cleanup(d.Close)
	if o := waitFake(t, "Other").Opts; !o.Minimizable || !o.Maximizable {
		t.Fatalf("NewDialog lost minimize or maximize: %v %v", o.Minimizable, o.Maximizable)
	}
}

func TestConfirmYesIsTrue(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Yes?", "") })
	press(t, waitFake(t, "Yes?"), "Yes")
	if !received(t, got) {
		t.Fatal("Yes returned false")
	}
}

func TestConfirmNoIsFalse(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "No?", "") })
	press(t, waitFake(t, "No?"), "No")
	if received(t, got) {
		t.Fatal("No returned true")
	}
}

// No is the default: Enter answers it.
func TestConfirmEnterIsNo(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Enter no?", "") })
	fw := waitFake(t, "Enter no?")
	frame(fw)
	key(fw, platform.KeyEnter)
	if received(t, got) {
		t.Fatal("Enter returned true")
	}
}

// Yes comes first, then No.
func TestConfirmButtonsYesThenNo(t *testing.T) {
	d, _ := messageDialog(nil, Question, 0, "Order", "", nil, nil, "Yes", "No")
	dt := fitted(t, d)
	yes, no := find(t, dt.Tester.Tester, "Yes"), find(t, dt.Tester.Tester, "No")
	if yes.X >= no.X {
		t.Fatalf("Yes at %v is not left of No at %v", yes, no)
	}
}

// The dialog is as high as its text, so more lines make it taller.
func TestMessageHeightFitsTheText(t *testing.T) {
	height := func(title, text string) int {
		got := runAsync(func() bool { Alert(nil, Info, title, text); return true })
		fw := waitFake(t, title)
		h := fw.Opts.Height
		press(t, fw, "OK")
		received(t, got)
		return h
	}
	short := height("Short", "Saved.")
	long := height("Long", strings.Repeat("The file is in use by another program and cannot be replaced. ", 6))
	if long <= short {
		t.Fatalf("long text %d high, short %d", long, short)
	}
}

// One line sits beside the icon's middle; several leave the icon on top.
func TestMessageIconAlignsWithText(t *testing.T) {
	place := func(text string) (iconBox image.Rectangle, textBox ui.Rect) {
		d, _ := messageDialog(nil, Info, 0, "Align", text, nil, nil, "OK")
		dt := fitted(t, d)
		img := dt.Image()
		iconBox = image.Rectangle{Min: img.Bounds().Max}
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < 60; x++ { // the icon is at the left
				if c := img.RGBAAt(x, y); c.B > 150 && c.R < 60 && c.G > 90 && c.G < 150 { // the icon's blue
					iconBox.Min.X, iconBox.Min.Y = min(iconBox.Min.X, x), min(iconBox.Min.Y, y)
					iconBox.Max.X, iconBox.Max.Y = max(iconBox.Max.X, x+1), max(iconBox.Max.Y, y+1)
				}
			}
		}
		if iconBox.Empty() {
			t.Fatal("no icon drawn")
		}
		return iconBox, find(t, dt.Tester.Tester, text)
	}
	ic, tx := place("Saved.")
	if d := float32(ic.Min.Y+ic.Max.Y)/2 - (tx.Y + tx.H/2); d < -4 || d > 4 {
		t.Errorf("one line: icon %v, text %v: middles differ by %v", ic, tx, d)
	}
	long := strings.Repeat("The file is in use by another program and cannot be replaced. ", 4)
	ic, tx = place(long)
	if tx.H < 2*float32(ic.Dy()) || float32(ic.Min.Y) > tx.Y+4 {
		t.Errorf("many lines: icon %v, text %v: the icon is not at the top", ic, tx)
	}
}

// hue samples: each Kind draws its colors.
func TestMessageIconColors(t *testing.T) {
	type want struct {
		kind       Kind
		body, mark color.RGBA
	}
	for _, w := range []want{
		{Error, color.RGBA{0xd3, 0x2f, 0x2f, 255}, color.RGBA{255, 255, 255, 255}},
		{Warning, color.RGBA{0xfb, 0xc0, 0x2d, 255}, color.RGBA{0, 0, 0, 255}},
		{Info, color.RGBA{0x19, 0x76, 0xd2, 255}, color.RGBA{255, 255, 255, 255}},
		{Question, color.RGBA{0x19, 0x76, 0xd2, 255}, color.RGBA{255, 255, 255, 255}},
	} {
		tt := tester(t, kindIcon(w.kind))
		img := tt.Image()
		b := iconBounds(img)
		// Sample the body near the left edge of the shape and the mark at the middle.
		bodyPt := image.Pt(b.Min.X+b.Dx()/2, b.Max.Y-b.Dy()/10)
		if w.kind != Warning {
			bodyPt = image.Pt(b.Min.X+b.Dx()/10*2, b.Min.Y+b.Dy()/2)
		}
		if !closeTo(img.RGBAAt(bodyPt.X, bodyPt.Y), w.body) {
			t.Errorf("kind %d: body at %v is %v, want about %v", w.kind, bodyPt, img.RGBAAt(bodyPt.X, bodyPt.Y), w.body)
		}
		// The mark is at the icon's centre: without it, the centre is the body's.
		c := image.Pt((b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2)
		found := false
		for y := c.Y - 3; y <= c.Y+3; y++ {
			for x := c.X - 3; x <= c.X+3; x++ {
				found = found || closeTo(img.RGBAAt(x, y), w.mark)
			}
		}
		if !found {
			t.Errorf("kind %d: no pixel near %v at the centre %v", w.kind, w.mark, c)
		}
	}
}

// iconBounds is the box of the pixels that are not the page's background.
func iconBounds(img *image.RGBA) image.Rectangle {
	bg := img.RGBAAt(img.Bounds().Max.X-1, img.Bounds().Max.Y-1)
	r := image.Rectangle{Min: img.Bounds().Max}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y) != bg {
				r.Min.X, r.Min.Y = min(r.Min.X, x), min(r.Min.Y, y)
				r.Max.X, r.Max.Y = max(r.Max.X, x+1), max(r.Max.Y, y+1)
			}
		}
	}
	return r
}

func closeTo(a, b color.RGBA) bool {
	d := func(x, y uint8) float64 { return math.Abs(float64(x) - float64(y)) }
	return d(a.R, b.R) < 40 && d(a.G, b.G) < 40 && d(a.B, b.B) < 40
}

func TestCaptionShowsTitle(t *testing.T) {
	for _, c := range []struct {
		goos      string
		hasParent bool
		want      bool
	}{
		{"windows", false, true}, {"windows", true, true},
		{"linux", false, true}, {"linux", true, true},
		{"darwin", false, true}, {"darwin", true, false}, // a sheet has no caption
		{"js", false, false}, {"js", true, false}, // a browser sets only the page's title
	} {
		if got := captionShowsTitle(c.goos, c.hasParent); got != c.want {
			t.Errorf("%s, parent %v: %v, want %v", c.goos, c.hasParent, got, c.want)
		}
	}
}

// With a parent, the dialog shows its title itself where the caption does not.
func TestMessageWithParentShowsTitleWhereNoCaptionDoes(t *testing.T) {
	parent := NewForm("Parent", 300, 200)
	d, _ := messageDialog(parent, Warning, 0, "Delete?", "", nil, nil, "Yes", "No")
	dt := fitted(t, d)
	shown := false
	for _, s := range dt.Texts() {
		shown = shown || s == "Delete?"
	}
	if want := !captionShowsTitle(runtime.GOOS, true); shown != want {
		t.Fatalf("title shown %v, want %v on %s: %q", shown, want, runtime.GOOS, dt.Texts())
	}
}

// InputSelect keeps room for its open list, which is drawn inside the window.
func TestInputSelectListFitsTheWindow(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputSelect(nil, Question, "Pick", "Which", Option{"a", "opta"}, Option{"b", "optb"}, Option{"c", "optc"})
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Pick")
	frame(fw)
	press(t, fw, "") // the combo
	found := false
	for _, n := range accessNodes(fw) {
		if n.Label == "optc" {
			found = true
			if n.Bounds.Y+n.Bounds.H > float64(fw.Opts.Height) {
				t.Errorf("optc ends at %v, below the window's %d", n.Bounds.Y+n.Bounds.H, fw.Opts.Height)
			}
		}
	}
	if !found {
		t.Fatalf("no optc in %q", labels(fw))
	}
	key(fw, platform.KeyEscape)
	key(fw, platform.KeyEscape)
	received(t, got)
}

// A long label makes InputSelect's dialog taller, with the combo and OK
// still inside the window.
func TestInputSelectLongLabelKeepsComboAndOKInWindow(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputSelect(nil, Question, "Long", strings.Repeat("Choose the one that suits this machine best. ", 14), Option{"a", "opta"})
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Long")
	if fw.Opts.Height <= 240 {
		t.Errorf("height %d: a long label did not make it taller than 240", fw.Opts.Height)
	}
	frame(fw)
	seen := 0
	for _, n := range accessNodes(fw) {
		if n.Label == "OK" || n.Label == "" { // the combo has no label
			seen++
		}
		if (n.Label == "OK" || n.Label == "Cancel" || n.Label == "") && n.Bounds.Y+n.Bounds.H > float64(fw.Opts.Height) {
			t.Errorf("%q (%v) ends at %v, below the window's %d", n.Label, n.Role, n.Bounds.Y+n.Bounds.H, fw.Opts.Height)
		}
	}
	if seen != 2 {
		t.Errorf("found %d of the combo and OK in %q", seen, labels(fw))
	}
	press(t, fw, "Cancel")
	received(t, got)
}
