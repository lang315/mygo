package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestButtonsClickedInAZoomedWindow checks that a click at a button's box,
// in the window's DIPs, reaches that button when the content is zoomed.
func TestButtonsClickedInAZoomedWindow(t *testing.T) {
	var clicks []string
	form := VBox(10).Add(
		Button("One").OnClick(func() { clicks = append(clicks, "One") }),
		Button("Two").OnClick(func() { clicks = append(clicks, "Two") }),
	)
	tt := NewTester(t, form, 400, 300)
	one := tt.Find(ByLabel("One")).Bounds()
	tt.SetZoom(2)
	// The button fills the window's width, in the zoomed layout as in the
	// unzoomed one; its height is the content's, twice.
	want := ui.Rect{X: one.X, Y: 2 * one.Y, W: one.W, H: 2 * one.H}
	if got := tt.Find(ByLabel("One")).Bounds(); got != want {
		t.Errorf("One at %v zoomed 2 times, %v unzoomed: want %v", got, one, want)
	}
	tt.Find(ByLabel("Two")).Click()
	tt.Find(ByLabel("One")).Click()
	if want := []string{"Two", "One"}; !slices.Equal(clicks, want) {
		t.Errorf("clicked %q, want %q", clicks, want)
	}
}
