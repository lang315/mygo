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

// Before Show, SetSize(w, 0) is what it was: the window opens at that size,
// and nothing fits it afterwards.
func TestSetSizeOfHeightZeroBeforeShowIsUnchanged(t *testing.T) {
	f := NewForm("Refit before", 420, 200)
	f.SetContent(lines(12))
	f.SetSize(600, 0)
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "Refit before")
	if fw.Opts.Height != 0 && fw.Opts.Height != 600 || fw.Opts.Width != 600 || fw.Opts.UseContentSize {
		t.Errorf("opened %dx%d content size %v", fw.Opts.Width, fw.Opts.Height, fw.Opts.UseContentSize)
	}
	drain()
	if h := fw.Bounds().Height; h != fw.Opts.Height && h != 600 {
		t.Errorf("height became %d", h)
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
