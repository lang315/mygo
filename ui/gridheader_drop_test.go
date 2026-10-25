package ui

import (
	"image"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// Dragging items over a header shows where they would go as a line across the
// header, at its top on its upper half and at its bottom on the lower, not as
// the vertical line beside an item.
func TestGridViewHeaderDropLineIsAcrossTheHeader(t *testing.T) {
	sel := -1
	s := &GridState{Selected: &sel, Key: func(i int) any { return i }, Reorder: func([]int, int) {}}
	s.Header = func(i int) bool { return i == 0 || i == 5 }
	tt := coreNewTester(func(c *context) {
		coreGridView(c, s, 10, 100, 60, func(i int) {
			if s.Header(i) {
				coreTextf(c, "H%d", i)
			} else {
				coreTextf(c, "Item %d", i)
			}
		}).Grow(1)
	}, 448, 400)
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	hn := accessNode(t, tt.h.access, platform.RoleListItem, "H5")
	from, _ := tt.Find("Item 2")
	accent := tt.rt.theme.Accent
	// accentRun counts the accent pixels along y, from x0 to x1.
	accentRun := func(img *image.RGBA, y, x0, x1 int) int {
		n := 0
		for x := x0; x < x1; x++ {
			if p := img.RGBAAt(x, y); p.R == accent.R && p.G == accent.G && p.B == accent.B {
				n++
			}
		}
		return n
	}
	for _, tc := range []struct {
		name    string
		y       float32
		lineAt  int // the row of pixels the line should be on
		otherAt int // and one it should not
	}{
		{"upper half", float32(hn.Bounds.Y) + 2, int(hn.Bounds.Y), int(hn.Bounds.Y + hn.Bounds.H)},
		{"lower half", float32(hn.Bounds.Y+hn.Bounds.H) - 2, int(hn.Bounds.Y + hn.Bounds.H), int(hn.Bounds.Y)},
	} {
		tt.Press(from.X, from.Y)
		tt.Move(from.X+20, from.Y)
		tt.Move(224, tc.y)
		img := tt.Image()
		got, other := accentRun(img, tc.lineAt, 20, 430), accentRun(img, tc.otherAt, 20, 430)
		tt.Release(224, tc.y)
		if got < 300 || other > 0 {
			t.Errorf("dragging over the %s of H5: %d accent pixels along the line at y %d (want a line across), %d at y %d (want none)", tc.name, got, tc.lineAt, other, tc.otherAt)
		}
	}
}
