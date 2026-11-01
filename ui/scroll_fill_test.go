package ui

import "testing"

func bottomOf(tt *Tester, label string) float32 {
	for _, n := range tt.Nodes() {
		if n.Label == label && n.Role == RoleButton {
			return n.Bounds.Y + n.Bounds.H
		}
	}
	return -1
}

// A scroll container that fills its viewport lays its content out at least as
// high as the viewport, so that what grows (Grow) fills the room of a short
// content and the content scrolls when it is long; without FillViewport the
// content takes its own height.
func TestScrollFillViewport(t *testing.T) {
	build := func(fill bool, lines int) func(c *Context) {
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				s := Scroll(c).Grow(1)
				if fill {
					s.FillViewport()
				}
				s.Children(func() {
					Column(c).Fill().Children(func() {
						for i := range lines {
							Textf(c, "line %d", i)
						}
						Spacer(c) // takes the free room
						Button(c, "Apply")
					})
				})
			})
		}
	}
	short := NewTester(build(true, 2), 300, 300)
	short.Frame()
	if b := bottomOf(short, "Apply"); b < 280 {
		t.Errorf("fill: Apply ends at %v in a 300 high viewport, want the bottom", b)
	}
	plain := NewTester(build(false, 2), 300, 300)
	plain.Frame()
	if b := bottomOf(plain, "Apply"); b >= 200 {
		t.Errorf("no fill: Apply ends at %v, want it right after the content", b)
	}
	long := NewTester(build(true, 60), 300, 300)
	long.Frame()
	if b := bottomOf(long, "Apply"); b < 300 {
		t.Errorf("long content: Apply ends at %v, want it below the fold", b)
	}
	long.Scroll(100, 100, 0, 5000)
	long.Frame()
	if b := bottomOf(long, "Apply"); b < 0 || b > 300 {
		t.Errorf("after scrolling Apply ends at %v, want it in view", b)
	}
}
