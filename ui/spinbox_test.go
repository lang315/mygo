package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// A SpinBox edits, steps and names as a NumberInput does.
func TestSpinBoxStepsTypesAndNames(t *testing.T) {
	v := 5.0
	submitted := 0
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).AlignItems(Start).Children(func() {
			if SpinBox(c, &v, 0, 10, 0.5).Label("Count").Submitted() {
				submitted++
			}
		})
	}, 400, 100)
	tt.Frame()
	fields, buttons := numberNodes(tt)
	if !slices.Equal(fields, []string{"Count"}) || !slices.Equal(buttons, []string{"Count Increase", "Count Decrease"}) {
		t.Fatalf("fields %q, buttons %q", fields, buttons)
	}
	if err := tt.Click("Increase"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Decrease")
	tt.Click("Decrease")
	if v != 4.5 {
		t.Errorf("one step up and two down from 5: %v", v)
	}
	r, _ := tt.Find("Count")
	tt.ClickAt(r.X+10, r.Y+r.H/2)
	tt.Key(Cmd, KeyA)
	tt.Type("7.5")
	if v != 7.5 {
		t.Errorf("typed 7.5: %v", v)
	}
	tt.Key(0, KeyUp)
	if v != 8 {
		t.Errorf("Up from 7.5: %v", v)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	if v != 7 {
		t.Errorf("Down twice from 8: %v", v)
	}
	// A composition, as an input method's, goes in as typing does.
	tt.Key(Cmd, KeyA)
	tt.Compose("3", 1)
	tt.Type("3")
	if v != 3 {
		t.Errorf("composed 3: %v", v)
	}
	tt.Key(0, KeyEnter)
	if submitted != 1 {
		t.Errorf("Enter: submitted %d times, want 1", submitted)
	}
	for range 30 {
		tt.Key(0, KeyUp)
	}
	tt.Frame()
	for _, n := range tt.Nodes() {
		if n.Label == "Count Increase" && !n.Disabled {
			t.Errorf("Increase enabled at the top, %v", v)
		}
	}
}

// Its arrows are joined at the end of its box, the up one over the down
// one, and a click in the box past the text focuses the text.
func TestSpinBoxArrowsAreJoinedAtTheEndOfTheBox(t *testing.T) {
	v := 5.0
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).AlignItems(Start).Children(func() {
			SpinBox(c, &v, 0, 10, 1).Label("Count")
		})
	}, 400, 100)
	tt.Frame()
	box, _ := tt.Find("Count")
	up, _ := tt.Find("Increase")
	down, _ := tt.Find("Decrease")
	var text Rect
	for _, n := range tt.Nodes() {
		if n.Role == RoleTextField {
			text = n.Bounds
		}
	}
	if up.X != down.X || up.Y+up.H > down.Y+0.5 {
		t.Fatalf("Increase %v is not over Decrease %v", up, down)
	}
	if up.X < text.X+text.W-0.5 || up.X+up.W < box.X+box.W-2 {
		t.Fatalf("arrows at %v are not past the text %v at the end of the box %v", up, text, box)
	}
	if h := down.Y + down.H - up.Y; h < box.H-2.5 {
		t.Fatalf("arrows %v high in a box %v", h, box)
	}
}

// focusedLabel is the label of the element with the keyboard focus.
func focusedLabel(tt *Tester) string {
	for _, n := range tt.Nodes() {
		if n.Focused {
			return n.Label
		}
	}
	return ""
}

// Its arrows are no stops of Tab, and a press on one leaves the keyboard
// in the text, as WinForms' NumericUpDown does.
func TestSpinBoxArrowsKeepTheKeyboardInTheText(t *testing.T) {
	v := 5.0
	tt := NewTester(func(c *Context) {
		Row(c).Gap(8).Padding(10).Children(func() {
			Button(c, "Before")
			SpinBox(c, &v, 0, 10, 1).Label("Count")
			Button(c, "After")
		})
	}, 600, 100)
	tt.Frame()
	var order []string
	for range 3 {
		tt.Key(0, KeyTab)
		tt.Frame()
		order = append(order, focusedLabel(tt))
	}
	if !slices.Equal(order, []string{"Before", "Count", "After"}) {
		t.Fatalf("Tab went %q", order)
	}
	if err := tt.Click("Increase"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if v != 6 || focusedLabel(tt) != "Count" {
		t.Fatalf("after a click on the up arrow: %v, focus on %q", v, focusedLabel(tt))
	}
	tt.Key(0, KeyUp)
	if v != 7 {
		t.Fatalf("Up after a click on the up arrow: %v", v)
	}
}

// Held, an arrow keeps stepping, faster after a while.
func TestSpinBoxHeldArrowRepeats(t *testing.T) {
	v := 0.0
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).AlignItems(Start).Children(func() {
			SpinBox(c, &v, 0, 100, 1).Label("Count")
		})
	}, 400, 100)
	tt.Frame()
	up, _ := tt.Find("Increase")
	x, y := up.X+up.W/2, up.Y+up.H/2
	tt.Press(x, y)
	if v != 1 {
		t.Fatalf("a press stepped to %v, want 1", v)
	}
	tt.Advance(700 * time.Millisecond)
	tt.Release(x, y)
	tt.Frame()
	// 1, then one each 80 ms past 400 ms: 300 ms held past it is 3 more.
	if v != 4 {
		t.Fatalf("held 0.7 s: %v, want 4", v)
	}
	tt.Advance(time.Second)
	if v != 4 {
		t.Fatalf("released, it went on to %v", v)
	}
}

// Right to left, the arrows are at the box's start, its left, flush with
// its border, and the line between them and the text is on their right.
func TestSpinBoxRightToLeft(t *testing.T) {
	v := 5.0
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).AlignItems(Start).Children(func() {
			SpinBox(c, &v, 0, 10, 1).Label("Count")
		})
	}, 400, 100)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	box, _ := tt.Find("Count")
	up, _ := tt.Find("Increase")
	var text Rect
	for _, n := range tt.Nodes() {
		if n.Role == RoleTextField {
			text = n.Bounds
		}
	}
	if up.X > box.X+1.5 || text.X < up.X+up.W {
		t.Fatalf("arrows %v not at the start of the box %v, before the text %v", up, box, text)
	}
	img := tt.Image()
	y := int(up.Y + up.H)
	border := img.RGBAAt(int(box.X+box.W-0.5), y)
	if d := img.RGBAAt(int(up.X+up.W), y); d != border {
		t.Fatalf("no line right of the arrows: %v, the border %v", d, border)
	}
}

// NumberInput keeps its look for a negative step: − then +, which steps
// by step; SpinBox steps by its absolute value.
func TestNegativeStep(t *testing.T) {
	a, b := 5.0, 5.0
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).Gap(10).AlignItems(Start).Children(func() {
			NumberInput(c, &a, 0, 10, -1)
			SpinBox(c, &b, 0, 10, -1).Label("Spin")
		})
	}, 400, 200)
	tt.Frame()
	plus, ok := tt.Find("+")
	if !ok {
		t.Fatalf("no + in %q", tt.Texts())
	}
	tt.ClickAt(plus.X+plus.W/2, plus.Y+plus.H/2)
	if a != 4 {
		t.Fatalf("+ with step -1 took 5 to %v, want 4 as before", a)
	}
	for _, n := range tt.Nodes() {
		if n.Label == "Spin Increase" {
			tt.ClickAt(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
		}
	}
	if b != 6 {
		t.Fatalf("Increase with step -1 took 5 to %v, want 6", b)
	}
}

// Assistive technology presses an arrow: one step.
func TestSpinBoxArrowPressedByAssistiveTechnologyStepsOnce(t *testing.T) {
	v := 5.0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).AlignItems(Start).Children(func() {
			coreSpinBox(c, &v, 0, 10, 1).Label("Count")
		})
	}, 400, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	up := accessNode(t, tt.h.access, platform.RoleButton, "Count Increase")
	if up.Actions&platform.ActionPress == 0 {
		t.Fatalf("the up arrow takes no press: %+v", up)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: up.ID, Action: platform.AccessPress})
	tt.Frame()
	tt.Frame()
	if v != 6 {
		t.Fatalf("pressed by assistive technology: %v, want 6", v)
	}
}
