package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// lines is a card of n labels.
func lines(n int) *BoxWidget {
	card := VBox(4)
	for i := range n {
		card.Add(Label(fmt.Sprintf("line %d", i)))
	}
	return card
}

// fitOf is the height Show gives a dialog of the content, at width: the oracle
// of a refit, which must measure the same way.
func fitOf(width int, content Control) int {
	d := NewDialog(nil, "oracle", "", width, 0, content)
	var opts mygo.WindowOptions
	opts.Width = width
	d.fitWindow(&opts)
	return opts.Height
}

// fitOfForm is fitOf for a Form, which has no dialog title around its content.
func fitOfForm(width int, content Control) int {
	f := NewForm("oracle", width, 0)
	f.SetContent(content)
	var opts mygo.WindowOptions
	opts.Width = width
	f.fitWindow(&opts)
	return opts.Height
}

// drain waits for the event goroutine to run everything posted so far.
func drain() {
	done := make(chan struct{})
	Post(func() { close(done) })
	<-done
}

func boundsHeight(t *testing.T, title string) (int, func() int) {
	fw := waitFake(t, title)
	h := func() int { return fw.Bounds().Height }
	return h(), h
}

// A shown Form whose content was swapped is fitted to the new content by
// SetSize(width, 0), at width.
func TestShownFormSetSizeOfHeightZeroFitsTheNewContent(t *testing.T) {
	f := NewForm("Refit form", 420, 200)
	f.SetContent(Label("splash"))
	f.Show()
	t.Cleanup(f.Close)
	h0, height := boundsHeight(t, "Refit form")
	f.SetContent(lines(12))
	f.SetSize(600, 0)
	want := fitOfForm(600, lines(12))
	wait.Until(t, "the window fitted", func() bool { return height() == want })
	if want <= h0 {
		t.Fatalf("oracle %d is not taller than the splash's %d", want, h0)
	}
}

func TestShownDialogSetSizeOfHeightZeroFitsTheNewContent(t *testing.T) {
	card := VBox(4).Add(Label("a"))
	d := NewDialog(nil, "Refit dialog", "", 500, 0, card)
	d.Show()
	t.Cleanup(d.Close)
	_, height := boundsHeight(t, "Refit dialog")
	for i := range 10 {
		card.Add(Label(fmt.Sprint("more ", i)))
	}
	d.SetSize(500, 0)
	other := VBox(4).Add(Label("a"))
	for i := range 10 {
		other.Add(Label(fmt.Sprint("more ", i)))
	}
	want := fitOf(500, other)
	wait.Until(t, "the window fitted", func() bool { return height() == want })
}

// Content taller than the work area is clamped as Show clamps, and scrolls.
func TestShownFormRefitIsClampedToTheWorkAreaAndScrolls(t *testing.T) {
	f := NewForm("Refit tall", 420, 200)
	f.SetContent(Label("splash"))
	f.Show()
	t.Cleanup(f.Close)
	_, height := boundsHeight(t, "Refit tall")
	f.SetContent(lines(150))
	f.SetSize(500, 0)
	wait.Until(t, "the window clamped", func() bool { h := height(); return h >= 700 && h <= 875 })
	mu.Lock()
	scrolls := f.fitScroll
	mu.Unlock()
	if !scrolls {
		t.Error("the clamped content does not scroll")
	}
}

// Before Show, SetSize(w, 0) is what it was: the window opens at that width and
// the window default height of 600, and nothing fits it afterwards.
func TestSetSizeOfHeightZeroBeforeShowIsUnchanged(t *testing.T) {
	f := NewForm("Refit before", 420, 200)
	f.SetContent(lines(12))
	f.SetSize(600, 0)
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Refit before")
	if o := fw.Opts; o.Width != 600 || o.Height != 600 || o.UseContentSize {
		t.Errorf("opened %dx%d content size %v, want 600x600 not content size", o.Width, o.Height, o.UseContentSize)
	}
	drain()
	if b := fw.Bounds(); b.Width != 600 || b.Height != 600 {
		t.Errorf("bounds %dx%d, want 600x600", b.Width, b.Height)
	}
}

// The fit is one-shot: later changes of the content leave the window alone.
func TestShownFormRefitIsOneShot(t *testing.T) {
	card := lines(3)
	f := NewForm("Refit once", 420, 200)
	f.SetContent(card)
	f.Show()
	t.Cleanup(f.Close)
	_, height := boundsHeight(t, "Refit once")
	f.SetSize(600, 0)
	want := fitOfForm(600, lines(3))
	wait.Until(t, "the window fitted", func() bool { return height() == want })
	for i := range 20 {
		card.Add(Label(fmt.Sprint("more ", i)))
	}
	drain()
	if h := height(); h != want {
		t.Errorf("height %d became %d", want, h)
	}
}

// A headless form's viewport follows, on the next frame.
func TestHeadlessFormSetSizeOfHeightZeroFitsTheNewContent(t *testing.T) {
	f := NewForm("Refit headless", 420, 200)
	f.SetContent(Label("splash"))
	tt := TestForm(t, f)
	f.SetContent(lines(12))
	f.SetSize(600, 0)
	tt.Settle()
	if want := fitOfForm(600, lines(12)); tt.w != 600 || tt.h != want {
		t.Errorf("viewport %dx%d, want 600x%d", tt.w, tt.h, want)
	}
}

func TestHeadlessDialogSetSizeOfHeightZeroFitsTheNewContent(t *testing.T) {
	card := VBox(4).Add(Label("a"))
	d := NewDialog(nil, "Refit headless dialog", "", 500, 300, card)
	dt := TestDialog(t, d)
	for i := range 10 {
		card.Add(Label(fmt.Sprint("more ", i)))
	}
	d.SetSize(500, 0)
	dt.Settle()
	other := VBox(4).Add(Label("a"))
	for i := range 10 {
		other.Add(Label(fmt.Sprint("more ", i)))
	}
	if want := fitOf(500, other); dt.w != 500 || dt.h != want {
		t.Errorf("viewport %dx%d, want 500x%d", dt.w, dt.h, want)
	}
}

// A later explicit SetSize wins over a refit still pending.
func TestShownFormLaterSetSizeBeatsAPendingRefit(t *testing.T) {
	f := NewForm("Refit later", 420, 200)
	f.SetContent(lines(12))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Refit later")
	f.SetSize(600, 0)
	f.SetSize(800, 500)
	drain()
	if b := fw.Bounds(); b.Width != 800 || b.Height != 500 {
		t.Errorf("bounds %dx%d, want 800x500", b.Width, b.Height)
	}
}

// The measure of a live form leaves its pending announcement for the real
// frame: a screen reader must hear it.
func TestMeasureFitLeavesAPendingAnnouncement(t *testing.T) {
	f := NewForm("Refit announce", 420, 200)
	f.SetContent(lines(3))
	mu.Lock()
	f.announce = "Name is required"
	mu.Unlock()
	f.measureFit(500, nil)
	mu.Lock()
	a := f.announce
	mu.Unlock()
	if a != "Name is required" {
		t.Errorf("announcement %q after the measure", a)
	}
}

// A window low on the screen is moved up to keep its bottom in the work area
// (25 to 900), never above its top, in the one resize.
func TestShownFormRefitMovesUpToStayOnTheWorkArea(t *testing.T) {
	f := NewForm("Refit low", 420, 200)
	f.SetContent(Label("splash"))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Refit low")
	f.Native().SetPosition(500, 400)
	drain()
	f.SetContent(lines(40))
	f.SetSize(600, 0)
	want := fitOfForm(600, lines(40))
	wait.Until(t, "the window fitted", func() bool { return fw.Bounds().Height == want })
	if b := fw.Bounds(); b.X != 500 || b.Y < 25 || b.Y+b.Height > 900 {
		t.Errorf("bounds %+v: want x 500 and inside the work area 25 to 900", b)
	}
	// Not low: it stays where it is.
	f.Native().SetPosition(500, 30)
	drain()
	f.SetContent(lines(3))
	f.SetSize(600, 0)
	want = fitOfForm(600, lines(3))
	wait.Until(t, "the window fitted again", func() bool { return fw.Bounds().Height == want })
	if b := fw.Bounds(); b.X != 500 || b.Y != 30 {
		t.Errorf("bounds %+v: moved from 500,30", b)
	}
}

// A dialog that is refitted still fits at its next Run: a height of 0 means fit.
func TestDialogRefitKeepsFittingAtTheNextShow(t *testing.T) {
	card := lines(3)
	d := NewDialog(nil, "Refit again", "", 500, 0, card)
	d.Show()
	first := waitFake(t, "Refit again")
	d.SetSize(500, 0)
	drain()
	d.Close()
	wait.Until(t, "the first window closed", first.IsClosed)
	card.Add(Label("one more"))
	d.Show()
	t.Cleanup(d.Close)
	fw := waitFake(t, "Refit again")
	if fw.Opts.Height == 0 || fw.Opts.Height == 600 || !fw.Opts.UseContentSize {
		t.Errorf("reopened at %d (content size %v): want it fitted", fw.Opts.Height, fw.Opts.UseContentSize)
	}
}

// A maximized or minimized window is left alone.
func TestShownFormRefitLeavesAMaximizedWindowAlone(t *testing.T) {
	f := NewForm("Refit max", 420, 200)
	f.SetContent(Label("splash"))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Refit max")
	before := fw.Bounds()
	f.Native().Maximize()
	f.SetContent(lines(12))
	f.SetSize(600, 0)
	drain()
	if b := fw.Bounds(); b != before {
		t.Errorf("a maximized window went from %+v to %+v", before, b)
	}
}
