package tw

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// edges reads a button's top edge (its middle), its left edge (its middle)
// and its padding, at scale 1, in the state tt is in.
func edgesOf(t *testing.T, tt *widget.Tester) (top, left, fill color.RGBA) {
	t.Helper()
	r := tt.Find(widget.ByLabel("Go")).Bounds()
	img := tt.Image()
	return img.RGBAAt(int(r.X+r.W/2), int(r.Y)), img.RGBAAt(int(r.X), int(r.Y+r.H/2)), img.RGBAAt(int(r.X)+6, int(r.Y+r.H/2))
}

func edges(t *testing.T, b *widget.ButtonWidget) (top, left, fill color.RGBA) {
	t.Helper()
	tt := widget.NewTester(t, b, 400, 200)
	tt.SetDark(false)
	tt.Frame()
	return edgesOf(t, tt)
}

// is reports whether c is within 8 of want on each channel.
func is(c color.RGBA, want ui.Color) bool {
	d := func(a, b uint8) bool { return max(int(a)-int(b), int(b)-int(a)) <= 8 }
	return d(c.R, want.R) && d(c.G, want.G) && d(c.B, want.B)
}

// Case 6: an applied tw border class loses to Suggested's own border, as an
// applied background does.
func TestSuggestedOwnBorderBeatsTwBorderClasses(t *testing.T) {
	top, _, fill := edges(t, widget.Button("Go").Suggested().Apply(S("border-red-500 border-l-0 rounded-l-none")))
	if top != fill {
		t.Errorf("top edge %v, want the fill %v: the own border wins", top, fill)
	}
}

// tw's suggested-action and destructive-action draw the border in the fill
// too.
func TestTwActionsDrawTheBorderInTheFill(t *testing.T) {
	for _, class := range []string{"suggested-action", "destructive-action"} {
		top, left, fill := edges(t, widget.Button("Go").Apply(S(class)))
		if top != fill {
			t.Errorf("%q: top edge %v, want the fill %v", class, top, fill)
		}
		if left != fill {
			t.Errorf("%q: left edge %v, want the fill %v", class, left, fill)
		}
	}
}

// border-l-0 leaves no left border, whether suggested-action is in the
// classes or not. The border is red, which the fill is not, so a left border
// would show.
func TestBorderL0LeavesNoLeftBorder(t *testing.T) {
	red := ui.Hex("#ef4444") // border-red-500
	for _, class := range []string{"border-red-500 border-l-0 rounded-l-none", "suggested-action border-red-500 border-l-0 rounded-l-none"} {
		top, left, fill := edges(t, widget.Button("Go").Apply(S(class)))
		if !is(top, red) {
			t.Errorf("%q: top edge %v, want the red %v", class, top, red)
		}
		if is(left, red) || left != fill {
			t.Errorf("%q: left edge %v, want the fill %v with no left border", class, left, fill)
		}
	}
}

// suggested-action's hover and active borders are classes of their own, so a
// border color the app adds ("border-red-500") holds at rest only: hovered
// and pressed, the border follows the fill again. This pins that.
func TestTwSuggestedActionBorderColorHoldsAtRestOnly(t *testing.T) {
	red := ui.Hex("#ef4444")
	th := ui.LightTheme()
	tt := widget.NewTester(t, widget.Button("Go").Apply(S("suggested-action border-red-500")), 400, 200)
	tt.SetDark(false)
	tt.Frame()
	if top, _, _ := edgesOf(t, tt); !is(top, red) {
		t.Errorf("rest: top edge %v, want red", top)
	}
	r := tt.Find(widget.ByLabel("Go")).Bounds()
	x, y := r.X+r.W/2, r.Y+r.H/2
	tt.Move(x, y)
	if top, _, fill := edgesOf(t, tt); top != fill || !is(top, th.AccentHover) {
		t.Errorf("hover: top edge %v, fill %v, want both the accent hover %v", top, fill, th.AccentHover)
	}
	tt.Press(x, y)
	if top, _, fill := edgesOf(t, tt); top != fill || !is(top, th.AccentPressed) {
		t.Errorf("pressed: top edge %v, fill %v, want both the accent pressed %v", top, fill, th.AccentPressed)
	}

	// The documented way to keep it: the same color in the states.
	tt = widget.NewTester(t, widget.Button("Go").Apply(S("suggested-action border-red-500 hover:border-red-500 active:border-red-500")), 400, 200)
	tt.SetDark(false)
	tt.Frame()
	tt.Move(x, y)
	if top, _, _ := edgesOf(t, tt); !is(top, red) {
		t.Errorf("hover with hover:border-red-500: top edge %v, want red", top)
	}
	tt.Press(x, y)
	if top, _, _ := edgesOf(t, tt); !is(top, red) {
		t.Errorf("pressed with active:border-red-500: top edge %v, want red", top)
	}
}
