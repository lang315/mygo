package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Apply styles the number's box: its background shows behind the text.
func TestNumericUpDownApplyStylesTheNumberBox(t *testing.T) {
	red := ui.RGB(200, 0, 0)
	n := NumericUpDown(0, 10, 1).SetValue(5).Apply(NewStyle().Background(Fixed(red)))
	tt := tester(t, n)
	var text ui.Rect
	for _, nd := range tt.Nodes() {
		if nd.Role == ui.RoleTextField {
			text = nd.Bounds
		}
	}
	if c := tt.Image().RGBAAt(int(text.X+text.W-6), int(text.Y+text.H/2)); !near(c, red) {
		t.Fatalf("the number's box is %v, want %v", c, red)
	}
}

// The arrows are joined at the end of the box, the up one over the down
// one; SideButtons gives the look of before, − and + beside the box.
func TestNumericUpDownArrowsAndSideButtons(t *testing.T) {
	n := NumericUpDown(0, 10, 1).SetValue(5)
	tt := tester(t, n)
	up, _ := tt.Find("Increase")
	down, _ := tt.Find("Decrease")
	if up.X != down.X || up.Y >= down.Y {
		t.Fatalf("Increase %v is not over Decrease %v", up, down)
	}
	n.SideButtons()
	tt.Frame()
	up, _ = tt.Find("Increase")
	down, _ = tt.Find("Decrease")
	if up.Y != down.Y || up.X <= down.X {
		t.Fatalf("SideButtons: Increase %v is not after Decrease %v", up, down)
	}
	if _, buttons := spinNames(tt); !slices.Equal(buttons, []string{"Decrease", "Increase"}) {
		t.Fatalf("SideButtons: buttons %q", buttons)
	}
	if !tt.HasText("+") {
		t.Fatalf("SideButtons: no + in %q", tt.Texts())
	}
}

// The keys and the arrows step the number and run OnChange.
func TestNumericUpDownKeysAndArrowsStep(t *testing.T) {
	changed := make(chan float64, 8)
	var n *NumericUpDownWidget
	NumericUpDown(0, 10, 1).AssignTo(&n).SetValue(5).OnChange(func() { changed <- n.Value() })
	tt := tester(t, n)
	focus(tt, 1)
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if v := received(t, changed); v != 6 {
		t.Fatalf("Up: %v", v)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	received(t, changed)
	if v := received(t, changed); v != 4 {
		t.Fatalf("Down, Down from 6: %v", v)
	}
	if n.Value() != 4 {
		t.Fatalf("Down, Down from 6: %v", n.Value())
	}
	click(t, tt, "Increase")
	if v := received(t, changed); v != 5 {
		t.Fatalf("the up arrow: %v", v)
	}
}

// Padding from Apply, as tw's p-2, pads the text: the arrows stay joined
// at the end of the box, as high as it.
func TestNumericUpDownPaddingLeavesTheArrowsAtTheEdge(t *testing.T) {
	n := NumericUpDown(0, 10, 1).Apply(NewStyle().Padding(2)).SetAccessibleName("Count")
	tt := tester(t, HBox(0).Add(n))
	box, _ := tt.Find("Count")
	up, _ := tt.Find("Increase")
	down, _ := tt.Find("Decrease")
	if end := box.X + box.W; up.X+up.W < end-1.5 {
		t.Fatalf("up arrow %v ends before the box %v", up, box)
	}
	if up.Y > box.Y+1.5 || down.Y+down.H < box.Y+box.H-1.5 {
		t.Fatalf("arrows %v, %v not as high as the box %v", up, down, box)
	}
	var text ui.Rect
	for _, nd := range tt.Nodes() {
		if nd.Role == ui.RoleTextField {
			text = nd.Bounds
		}
	}
	if text.X < box.X+7.5 {
		t.Fatalf("text %v not padded 8 in the box %v", text, box)
	}
}

// A negative step steps by its absolute value.
func TestNumericUpDownNegativeStep(t *testing.T) {
	n := NumericUpDown(0, 10, -1).SetValue(5)
	tt := tester(t, n)
	click(t, tt, "Increase")
	if n.Value() != 6 {
		t.Fatalf("Increase with step -1: %v, want 6", n.Value())
	}
}
