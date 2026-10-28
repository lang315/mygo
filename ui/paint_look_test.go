package ui

import "testing"

// Painting a node leaves it as it was built: its styleFn changes what the
// frame paints, not the node, which a later paint of the same node reads
// again (repaintFrame now, a retained row later).
func TestPaintLeavesTheNodeAsBuilt(t *testing.T) {
	var box *node
	var built nodeLook
	tt := coreNewTester(func(c *context) {
		box = coreBox(c).Width(40).Height(20)
		box.styleFn = func(e *node) {
			e.bg, e.borderC, e.ts.color = RGB(255, 0, 0), RGB(0, 255, 0), RGB(0, 0, 255)
			e.Opacity(0.5)
		}
		built = box.look()
	}, 100, 100)
	tt.Frame()
	if got := box.look(); got != built {
		t.Fatalf("after paint the node holds %+v, built %+v", got, built)
	}
	// What the styleFn set was painted: the frame shows its red at half
	// opacity, which on the white window is (255, 127, 127).
	if c := tt.Image().RGBAAt(10, 10); c.R < 250 || c.G < 110 || c.G > 145 || c.B < 110 || c.B > 145 {
		t.Fatalf("the box painted %v, not the styleFn's red at half opacity", c)
	}
}

// painted is the look a node paints with in the frame it was just in: its
// built look with its styleFn applied to a copy, as Painter.element does, now
// that painting leaves the node itself as built. It does not apply a color
// transition.
func painted(n *node) nodeLook {
	c := *n
	if c.styleFn != nil {
		c.styleFn(&c)
	}
	return c.look()
}
