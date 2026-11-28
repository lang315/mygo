package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// stepTester shows w, with a control before it that keeps the frame's theme
// in the returned Theme.
func stepTester(t *testing.T, w Control) (*Tester, *ui.Theme) {
	t.Helper()
	th := new(ui.Theme)
	keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
	tt := NewTester(t, VBox(0).Add(keep, w), 600, 400)
	tt.Settle()
	return tt, th
}

// stepText is the box of the text s, which must show.
func stepText(t *testing.T, tt *Tester, s string) ui.Rect {
	t.Helper()
	r, ok := tt.Tester.Find(s)
	if !ok {
		t.Fatalf("no text %q in %q", s, tt.Texts())
	}
	return r
}

// dotFill is the color inside the circle of the step titled title, a few
// DIPs in from its start edge, away from its number or check: the circle is
// above the title, or before it when vertical, at its start (end in RTL).
func dotFill(t *testing.T, tt *Tester, title string, vertical, rtl bool) ui.Rect {
	t.Helper()
	r := stepText(t, tt, title)
	var cx, cy float32
	if vertical {
		cy = r.Y + r.H/2
		cx = r.X - stepTitleGap - stepDot/2
		if rtl {
			cx = r.X + r.W + stepTitleGap + stepDot/2
		}
	} else {
		cx, cy = r.X+r.W/2, r.Y-stepTitleGap-stepDot/2
	}
	return ui.Rect{X: cx - stepDot/2 + 4, Y: cy}
}

func isAccent(tt *Tester, at ui.Rect, th *ui.Theme) bool {
	return near(tt.Image().RGBAAt(int(at.X), int(at.Y)), th.Accent)
}

var fourSteps = []string{"One", "Two", "Three", "Four"}

// Case 1: with 4 steps and step 2, steps 0–1 are done (a check, no number),
// 2 is current (accented, its number), 3 is ahead (its number, not
// accented). Left to right.
func TestStepProgressBarStates(t *testing.T) {
	bar := StepProgressBar(fourSteps...).SetStep(2)
	tt, th := stepTester(t, bar)
	if got := bar.Step(); got != 2 {
		t.Fatalf("Step() = %d, want 2", got)
	}
	for _, n := range []string{"1", "2"} {
		if _, ok := tt.Tester.Find(n); ok {
			t.Errorf("done step %s shows its number, want a check", n)
		}
	}
	for _, n := range []string{"3", "4"} {
		stepText(t, tt, n)
	}
	for _, s := range fourSteps[:3] {
		if at := dotFill(t, tt, s, false, false); !isAccent(tt, at, th) {
			t.Errorf("the circle of %q is %v, want the accent %v", s, tt.Image().RGBAAt(int(at.X), int(at.Y)), th.Accent)
		}
	}
	if at := dotFill(t, tt, "Four", false, false); isAccent(tt, at, th) {
		t.Errorf("the circle of the step ahead is the accent")
	}
	// The check is drawn in the done circle: the accent's text color at its
	// middle row, somewhere.
	r := stepText(t, tt, "One")
	cx, cy := r.X+r.W/2, r.Y-stepTitleGap-stepDot/2
	inked := false
	for x := cx - stepDot/2 + 2; x < cx+stepDot/2-2; x++ {
		for y := cy - 4; y <= cy+4; y++ {
			if near(tt.Image().RGBAAt(int(x), int(y)), th.AccentText) {
				inked = true
			}
		}
	}
	if !inked {
		t.Errorf("the done circle of One draws no check in %v", th.AccentText)
	}
	// Left to right on one line.
	prev := stepText(t, tt, "One")
	for _, s := range fourSteps[1:] {
		b := stepText(t, tt, s)
		if b.X <= prev.X || b.Y != prev.Y {
			t.Errorf("%q at %v after %v: want to its right, on its line", s, b, prev)
		}
		prev = b
	}
}

// Case 2: SetVertical(true) stacks the steps, each circle before its title.
func TestStepProgressBarVertical(t *testing.T) {
	bar := StepProgressBar(fourSteps...).SetStep(2).SetVertical(true)
	tt, th := stepTester(t, bar)
	prev := stepText(t, tt, "One")
	for _, s := range fourSteps[1:] {
		b := stepText(t, tt, s)
		if b.Y <= prev.Y || b.X != prev.X {
			t.Errorf("%q at %v after %v: want below it, at its start", s, b, prev)
		}
		prev = b
	}
	if at := dotFill(t, tt, "One", true, false); !isAccent(tt, at, th) {
		t.Errorf("vertical: the circle before One is %v, want the accent", tt.Image().RGBAAt(int(at.X), int(at.Y)))
	}
	if at := dotFill(t, tt, "Four", true, false); isAccent(tt, at, th) {
		t.Errorf("vertical: the circle before Four is the accent")
	}
}

// Case 3: right to left, the first step is at the right, and a vertical
// bar's circles are after (right of) their titles.
func TestStepProgressBarRightToLeft(t *testing.T) {
	bar := StepProgressBar(fourSteps...).SetStep(2)
	tt, th := stepTester(t, bar)
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	if one, four := stepText(t, tt, "One"), stepText(t, tt, "Four"); one.X <= four.X {
		t.Errorf("RTL: One at %v, Four at %v: want One at the right", one, four)
	}
	if at := dotFill(t, tt, "Three", false, true); !isAccent(tt, at, th) {
		t.Errorf("RTL: the current circle is not the accent")
	}
	bar.SetVertical(true)
	tt.Settle()
	if at := dotFill(t, tt, "One", true, true); !isAccent(tt, at, th) {
		t.Errorf("RTL vertical: the circle after One is %v, want the accent", tt.Image().RGBAAt(int(at.X), int(at.Y)))
	}
}

// Case 4: the bounds. SetStep clamps to the steps; SetSteps that drops the
// current step moves it to the last; no steps draw nothing.
func TestStepProgressBarBounds(t *testing.T) {
	bar := StepProgressBar(fourSteps...)
	if got := bar.SetStep(9).Step(); got != 3 {
		t.Errorf("SetStep(9) of 4 steps: Step() = %d, want 3", got)
	}
	if got := bar.SetStep(-1).Step(); got != 0 {
		t.Errorf("SetStep(-1): Step() = %d, want 0", got)
	}
	bar.SetStep(3).SetSteps("a", "b")
	if got := bar.Step(); got != 1 {
		t.Errorf("SetSteps of 2 at step 3: Step() = %d, want 1", got)
	}
	tt, _ := stepTester(t, bar)
	stepText(t, tt, "a")
	if _, ok := tt.Tester.Find("One"); ok {
		t.Errorf("SetSteps kept the old titles: %q", tt.Texts())
	}
	empty := StepProgressBar()
	tt, _ = stepTester(t, empty)
	if got := empty.Step(); got != 0 {
		t.Errorf("no steps: Step() = %d, want 0", got)
	}
	if got := empty.SetStep(2).Step(); got != 0 {
		t.Errorf("no steps, SetStep(2): Step() = %d, want 0", got)
	}
	tt.Settle()
}
