package widget

import (
	"slices"
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

// lineAt is the point frac of the way from the circle of step a to that of
// step b, on the line joining them.
func lineAt(t *testing.T, tt *Tester, a, b string, frac float32, vertical, rtl bool) ui.Rect {
	t.Helper()
	pa, pb := dotFill(t, tt, a, vertical, rtl), dotFill(t, tt, b, vertical, rtl)
	// dotFill is 4 DIPs in from the circle's start edge: back to its center.
	pa.X, pb.X = pa.X+stepDot/2-4, pb.X+stepDot/2-4
	return ui.Rect{X: pa.X + (pb.X-pa.X)*frac, Y: pa.Y + (pb.Y-pa.Y)*frac}
}

// checkLines checks the lines of a bar of fourSteps at step 2: accent from
// One to Three, the border's color from Three to Four, near each end.
func checkLines(t *testing.T, tt *Tester, th *ui.Theme, vertical, rtl bool) {
	t.Helper()
	for _, seg := range [][2]string{{"One", "Two"}, {"Two", "Three"}, {"Three", "Four"}} {
		for _, frac := range []float32{0.35, 0.65} { // past the circles' edges
			at := lineAt(t, tt, seg[0], seg[1], frac, vertical, rtl)
			got := tt.Image().RGBAAt(int(at.X), int(at.Y))
			want := th.Accent
			if seg[1] == "Four" {
				want = th.Border
			}
			if !near(got, want) {
				t.Errorf("vertical %v, RTL %v: the line from %s to %s, %v of the way, is %v; want %v", vertical, rtl, seg[0], seg[1], frac, got, want)
			}
		}
	}
	// No line leads out of the first circle or the last: before One (above
	// it), after Four (below it), 6 DIPs out.
	for _, end := range []struct {
		step string
		out  float32 // -1 toward the start, 1 toward the end
	}{{"One", -1}, {"Four", 1}} {
		at := dotFill(t, tt, end.step, vertical, rtl)
		at.X += stepDot/2 - 4 // the circle's center
		off := end.out * (stepDot/2 + 6)
		if vertical {
			at.Y += off
		} else if rtl {
			at.X -= off
		} else {
			at.X += off
		}
		got := tt.Image().RGBAAt(int(at.X), int(at.Y))
		if near(got, th.Accent) || near(got, th.Border) {
			t.Errorf("vertical %v, RTL %v: a line leads out of %s: %v at %v", vertical, rtl, end.step, got, at)
		}
	}
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
	// The current circle has its number alone, no check: as much ink as a
	// bold 3 on the accent, drawn apart.
	r = stepText(t, tt, "Three")
	got := countNear(tt.Image(), ui.Rect{X: r.X + r.W/2 - stepDot/2, Y: r.Y - stepTitleGap - stepDot, W: stepDot, H: stepDot}, false, th.AccentText, 8)
	ref := NewTester(t, Custom(func(c *ui.Context) ui.Element {
		t := c.Theme()
		return ui.Row(c).Width(stepDot).Height(stepDot).Radius(stepDot / 2).Background(t.Accent).TextColor(t.AccentText).
			AlignItems(ui.Center).Justify(ui.Center).Children(func() { ui.Text(c, "3").SingleLine().FontWeight(600) })
	}), 100, 100)
	ref.Settle()
	want := countNear(ref.Image(), ui.Rect{W: stepDot, H: stepDot}, false, th.AccentText, 8)
	if want == 0 || got > want+4 || got < want-4 {
		t.Errorf("the current circle has %d pixels of ink, a bold 3 alone in the accent's text %d: a check, or another color?", got, want)
	}
	checkLines(t, tt, th, false, false)
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
	checkLines(t, tt, th, true, false)
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
	checkLines(t, tt, th, false, true)
	bar.SetVertical(true)
	tt.Settle()
	if at := dotFill(t, tt, "One", true, true); !isAccent(tt, at, th) {
		t.Errorf("RTL vertical: the circle after One is %v, want the accent", tt.Image().RGBAAt(int(at.X), int(at.Y)))
	}
	checkLines(t, tt, th, true, true)
}

// Case 3b: the setters from another goroutine while frames run, the steps
// growing and shrinking, show on a following frame without a race.
func TestStepProgressBarFromGoroutine(t *testing.T) {
	bar := StepProgressBar(fourSteps...)
	tt, _ := stepTester(t, bar)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			steps := fourSteps[:1+i%4]
			if i%3 == 0 {
				steps = append(slices.Clone(fourSteps), "Five", "Six")
			}
			bar.SetSteps(steps...).SetStep(i % 6).SetVertical(i%2 == 0)
			_ = bar.Step()
		}
		bar.SetSteps("a", "b", "c").SetStep(1).SetVertical(true)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	a, c := stepText(t, tt, "a"), stepText(t, tt, "c")
	if bar.Step() != 1 || c.Y <= a.Y {
		t.Fatalf("after the goroutine: step %d, a at %v, c at %v; want 1, c below a", bar.Step(), a, c)
	}
	if _, ok := tt.Tester.Find("Six"); ok {
		t.Errorf("an old step shows: %q", tt.Texts())
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
