package ui

import (
	"testing"
	"time"
)

// A copy of an element going with its exit transition starts from the look
// the element last showed, a hovered row's hover color, and not from the look
// it was built with (painting leaves the node as built).
func TestExitCopyStartsFromTheLookThePaintShowed(t *testing.T) {
	th := LightTheme()
	shown := true
	var row *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			if shown {
				row = coreBox(c).Key("row").Size(80, 30).Transition(ElementTransition{Position: true, Exit: &Motion{Opacity: 0}})
				row.builtWith(th)
				row.styleFn = hoverSurface
			}
		})
	}, 200, 100)
	r := row.Bounds()
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	shown = false
	*now = now.Add(16 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for _, r := range tt.rt.trans {
		if r.ghost != nil {
			ghost = r.ghost
		}
	}
	if ghost == nil {
		t.Fatal("no copy going: the test shows nothing")
	}
	if ghost.bg != th.SurfaceHover {
		t.Errorf("the copy starts with the background %v, want the hover color %v the row showed", ghost.bg, th.SurfaceHover)
	}
}

// A copy going while its element's colors were moving (Colors) starts from the
// colors shown, between the built and the hover color.
func TestExitCopyStartsFromTheColorsMidTransition(t *testing.T) {
	th := LightTheme()
	shown := true
	var row *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			if shown {
				row = coreBox(c).Key("row").Size(80, 30).Transition(ElementTransition{Colors: true, Duration: time.Second, Ease: Linear, Exit: &Motion{Opacity: 0}})
				row.builtWith(th)
				row.styleFn = hoverSurface
			}
		})
	}, 200, 100)
	r := row.Bounds()
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	shown = false
	*now = now.Add(16 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for _, r := range tt.rt.trans {
		if r.ghost != nil {
			ghost = r.ghost
		}
	}
	if ghost == nil {
		t.Fatal("no copy going: the test shows nothing")
	}
	if ghost.bg.A == 0 || ghost.bg == th.SurfaceHover {
		t.Errorf("the copy starts with the background %v, want a color between transparent and %v", ghost.bg, th.SurfaceHover)
	}
}

// A node with a color transition and no styleFn is painted as built too:
// the transition writes its colors as it paints.
func TestPaintLeavesATransitioningNodeAsBuilt(t *testing.T) {
	col := RGB(255, 0, 0)
	var box *node
	var built nodeLook
	tt, now := clockTester(func(c *context) {
		box = coreBox(c).Key("box").Size(40, 20).Background(col).Transition(ElementTransition{Colors: true, Duration: time.Second, Ease: Linear})
		built = box.look()
	}, 100, 100)
	col = RGB(0, 0, 255) // a change the next frame builds, which the transition moves to
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if got := box.look(); got != built {
		t.Errorf("after paint the node holds %+v, built %+v", got, built)
	}
}
