package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// showFit shows a dialog of the size asked, its content a title and a card of
// n labels, and returns its fake window.
func showFit(t *testing.T, title string, width, height, n int) *Dialog {
	t.Helper()
	card := VBox(4)
	for i := range n {
		card.Add(Label(fmt.Sprintf("line %d", i)))
	}
	d := NewDialog(nil, title, "", width, height, card)
	d.Show()
	t.Cleanup(d.Close)
	return d
}

// A dialog of height 0 opens as high as its content at its width, with that
// as the size of its page area, not the 600 of a window.
func TestDialogOfHeightZeroOpensAtTheHeightOfItsContent(t *testing.T) {
	showFit(t, "Fit one", 500, 0, 3)
	fw := waitFake(t, "Fit one")
	if h := fw.Opts.Height; h < 60 || h > 250 {
		t.Errorf("height %d: want the content's (a title and three labels), not 600", h)
	}
	if !fw.Opts.UseContentSize || fw.Opts.Width != 500 {
		t.Errorf("content size %v, width %d: the measured height is of the page area", fw.Opts.UseContentSize, fw.Opts.Width)
	}
}

// Width 0 is the window default, and the height is measured at it.
func TestDialogOfWidthZeroIsMeasuredAtTheDefaultWidth(t *testing.T) {
	showFit(t, "Fit two", 0, 0, 2)
	fw := waitFake(t, "Fit two")
	if fw.Opts.Width != 800 || fw.Opts.Height >= 600 {
		t.Errorf("%dx%d", fw.Opts.Width, fw.Opts.Height)
	}
}

// More content than the screen has opens as high as the work area allows, and
// the content scrolls.
func TestDialogTallerThanTheWorkAreaIsClampedAndScrolls(t *testing.T) {
	d := showFit(t, "Fit tall", 500, 0, 120)
	fw := waitFake(t, "Fit tall")
	if h := fw.Opts.Height; h > 875 || h < 700 {
		t.Errorf("height %d: want the work area's 875 less a margin", h)
	}
	mu.Lock()
	scrolls := d.fitScroll
	mu.Unlock()
	if !scrolls {
		t.Error("the clamped content does not scroll")
	}
	hasScroll := false
	for _, n := range accessNodes(fw) {
		hasScroll = hasScroll || n.Role == platform.RoleScroll
	}
	if !hasScroll {
		t.Error("the clamped content is not in a scroll container")
	}
}

// A dialog that fits does not scroll, and one sized by the app is not fitted.
func TestDialogFitScrollsOnlyWhenClampedAndASizeFromTheAppWins(t *testing.T) {
	d := showFit(t, "Fit short", 500, 0, 2)
	mu.Lock()
	scrolls := d.fitScroll
	mu.Unlock()
	if scrolls {
		t.Error("a dialog that fits scrolls")
	}
	showFit(t, "Fit fixed", 500, 321, 3)
	if h := waitFake(t, "Fit fixed").Opts.Height; h != 321 {
		t.Errorf("a given height became %d", h)
	}
	d2 := NewDialog(nil, "Fit set", "", 500, 0, Label("x"))
	d2.SetSize(500, 333)
	d2.Show()
	t.Cleanup(d2.Close)
	if h := waitFake(t, "Fit set").Opts.Height; h != 333 {
		t.Errorf("SetSize after NewDialog: height %d, want it to win", h)
	}
}

// Later changes of the content do not resize an open dialog.
func TestDialogFitIsMeasuredAtShowOnly(t *testing.T) {
	card := VBox(4).Add(Label("a"))
	d := NewDialog(nil, "Fit once", "", 500, 0, card)
	d.Show()
	t.Cleanup(d.Close)
	fw := waitFake(t, "Fit once")
	before := fw.Opts.Height
	for i := range 20 {
		card.Add(Label(fmt.Sprint("more ", i)))
	}
	if fw.Opts.Height != before {
		t.Errorf("height %d became %d", before, fw.Opts.Height)
	}
}
