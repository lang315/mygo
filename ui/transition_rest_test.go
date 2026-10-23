package ui

import (
	"image/color"
	"testing"
	"time"
)

// colorRows is a clipped column of 200 rows with a Colors Transition each,
// scrolled by a translation of dy, painted in the color bg.
func colorRows(dy *float32, bg *Color) func(c *context) {
	return func(c *context) {
		coreBox(c).Fill().Clip().Children(func() {
			coreColumn(c).Translate(0, *dy).Children(func() {
				for i := range 200 {
					coreBox(c).Key(i).Height(28).Background(*bg).Transition(ElementTransition{Colors: true, Ease: Linear})
				}
			})
		})
	}
}

// Rows with a Colors Transition are painted only where they show, as rows
// without one are (TestPaintCullsScrolledRows).
func TestPaintCullsColorTransitions(t *testing.T) {
	var dy float32
	bg := RGB(0, 0, 255)
	tt := coreNewTester(colorRows(&dy, &bg), 400, 300)
	for _, d := range []float32{0, -1000, -4000} {
		dy = d
		before := tt.rt.culled
		tt.Frame()
		// 11 rows show at most; the others are skipped whole.
		if n := tt.rt.culled - before; n < 188 || n > 190 {
			t.Errorf("at %v, %d rows skipped", d, n)
		}
	}
}

// A row that changed color while it was off screen shows its color once
// scrolled into view, and a row in view still fades.
func TestTransitionColorsOfRowsScrolledIntoView(t *testing.T) {
	var dy float32
	bg := RGB(0, 0, 255)
	view := colorRows(&dy, &bg)
	now := time.Unix(1_000_000, 0)
	tt := coreNewTester(view, 400, 300)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	pixel := func(y int) color.RGBA { return tt.Image().RGBAAt(10, y) }
	bg = RGB(255, 0, 0)
	tt.Frame()
	now = now.Add(100 * time.Millisecond) // halfway of the default 200 ms
	tt.Frame()
	if p := pixel(10); p.R < 110 || p.R > 145 || p.B < 110 || p.B > 145 {
		t.Errorf("a row in view, halfway: %v", p)
	}
	now = now.Add(time.Second)
	tt.Frame()
	dy = -2800 // row 100 on top
	tt.Frame()
	if p := pixel(10); p.R != 255 || p.B != 0 {
		t.Errorf("a row scrolled into view: %v", p)
	}
}

// What a Colors-only Transition does not move, it lays out at once; and a
// Transition that moves things, given to the same element later, starts
// from where the element was.
func TestColorsTransitionMovesNothingThenPositionStartsFromHere(t *testing.T) {
	x := float32(0)
	tr := ElementTransition{Colors: true}
	tt, now := clockTester(func(c *context) {
		coreBox(c).Size(100, 100).Children(func() {
			coreText(c, "Moving").Absolute().Left(x).Transition(tr)
		})
	}, 300, 200)
	x = 50
	tt.Frame()
	if r := at(t, tt, "Moving"); r.X != 50 {
		t.Fatalf("with Colors alone, it shows at %v, want 50", r.X)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	tr = ElementTransition{Position: true, Ease: Linear, Duration: 100 * time.Millisecond}
	x = 150
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if r := at(t, tt, "Moving"); r.X < 90 || r.X > 110 {
		t.Errorf("halfway from 50 to 150, it shows at %v", r.X)
	}
}

// A Colors-only child of a box that a Transition resizes shows where the
// layout put it, as one without a Transition does (here, at the end of
// the box's final width), not where the resized box would put it.
func TestColorsOnlyChildStaysWhereTheLayoutPutIt(t *testing.T) {
	w := float32(100)
	tt, now := clockTester(func(c *context) {
		keep("parent", coreBox(c).Key("parent")).Width(w).Height(50).Transition(ElementTransition{Size: true, Ease: Linear}).Children(func() {
			coreText(c, "Right").Key("child").AlignSelf(End).Transition(ElementTransition{Colors: true})
		})
	}, 400, 100)
	w = 300
	tt.Frame()
	*now = now.Add(100 * time.Millisecond) // halfway of the default 200 ms
	tt.Frame()
	p := keyed(tt, "parent")
	if p.w < 190 || p.w > 210 {
		t.Fatalf("halfway, the parent is %v wide, want about 200", p.w)
	}
	r := at(t, tt, "Right")
	if edge := r.X + r.W; edge < p.x+299 || edge > p.x+301 {
		t.Errorf("the child's right edge is at %v, want %v, the end of the final width", edge, p.x+300)
	}
}

// An element culled in the middle of a fade stops being Transitioning once
// the fade is over: nothing paints it again to end it.
func TestColorsFadeEndsForAnElementCulledMidFade(t *testing.T) {
	var dy float32
	bg := RGB(0, 0, 255)
	now := time.Unix(1_000_000, 0)
	tt := coreNewTester(colorRows(&dy, &bg), 400, 300)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	bg = RGB(255, 0, 0)
	tt.Frame()
	now = now.Add(50 * time.Millisecond)
	tt.Frame()
	dy = -4000 // the rows that were fading are out of view
	tt.Frame()
	now = now.Add(5 * time.Second)
	tt.Frame()
	moving := 0
	for _, r := range tt.rt.trans {
		if r.colorMoving {
			moving++
		}
	}
	if moving != 0 {
		t.Errorf("%d elements still fade, long after", moving)
	}
}
