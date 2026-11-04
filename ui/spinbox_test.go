package ui

import (
	"slices"
	"testing"
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
