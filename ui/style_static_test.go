package ui

import (
	"testing"
	"unsafe"
)

// A node's size is kept: its fields go in the padding.
func TestNodeSizeIsKept(t *testing.T) {
	if n := unsafe.Sizeof(node{}); n != 1752 {
		t.Errorf("unsafe.Sizeof(node{}) = %d, want 1752", n)
	}
}

// formOfInputs is n buttons of each kind and n text inputs in a column,
// all in view, with the nodes of the last frame kept in nodes.
func formOfInputs(n int, nodes *[]*node, th func() *Theme) *Tester {
	values := make([]string, n)
	return coreNewTester(func(c *context) {
		if th != nil {
			c.SetTheme(th())
		}
		*nodes = (*nodes)[:0]
		coreColumn(c).Gap(2).Children(func() {
			for i := range n {
				*nodes = append(*nodes, coreButton(c, "plain"))
				*nodes = append(*nodes, corePrimaryButton(c, "primary"))
				*nodes = append(*nodes, coreTextInput(c, &values[i]))
			}
		})
	}, 400, n*120+200)
}

// TestButtonsAndTextInputsMakeNothingPerFrame: the look of a button and of a
// text input is a function that reads the node, not a closure made for each
// one in each frame. The slope is what each adds to a frame.
func TestButtonsAndTextInputsMakeNothingPerFrame(t *testing.T) {
	frames := func(n int) float64 {
		var nodes []*node
		tt := formOfInputs(n, &nodes, nil)
		for range 3 {
			tt.Frame()
		}
		return testing.AllocsPerRun(20, tt.Frame)
	}
	slope := (frames(60) - frames(20)) / 40 // per (button, primary button, text input)
	t.Logf("allocations per button, primary button and text input: %.2f", slope)
	if slope > 0.5 {
		t.Errorf("a button, a primary button and a text input make %.2f allocations in a frame, want none", slope)
	}
}

// What a button and a text box paint follows the theme, light or dark, and the
// pointer, of the frame: the static functions read them as they paint.
func TestButtonsAndTextInputsFollowTheThemeAndThePointer(t *testing.T) {
	light, dark := LightTheme(), DarkTheme()
	cur := light
	var nodes []*node
	tt := formOfInputs(1, &nodes, func() *Theme { return cur })
	tt.Frame()
	plain, primary, input := func() *node { return nodes[0] }, func() *node { return nodes[1] }, func() *node { return nodes[2] }
	for _, c := range []struct {
		name string
		th   *Theme
	}{{"light", light}, {"dark", dark}, {"light again", light}} {
		cur = c.th
		tt.Frame()
		tt.Frame()
		if plain().bg != c.th.Surface || primary().bg != c.th.Accent {
			t.Errorf("%s: backgrounds %v and %v, want %v and %v", c.name, plain().bg, primary().bg, c.th.Surface, c.th.Accent)
		}
		for _, hover := range []struct {
			n    func() *node
			want Color
		}{{plain, c.th.SurfaceHover}, {primary, c.th.AccentHover}} {
			r := hover.n().Bounds()
			tt.Move(r.X+r.W/2, r.Y+r.H/2)
			tt.Frame()
			if got := hover.n().bg; got != hover.want {
				t.Errorf("%s: hovered background %v, want %v", c.name, got, hover.want)
			}
			tt.Press(r.X+r.W/2, r.Y+r.H/2)
			tt.Frame()
			want := c.th.SurfacePressed
			if hover.n() == primary() {
				want = c.th.AccentPressed
			}
			if got := hover.n().bg; got != want {
				t.Errorf("%s: pressed background %v, want %v", c.name, got, want)
			}
			tt.Release(r.X+r.W/2, r.Y+r.H/2)
			tt.Move(395, 315)
			tt.Frame()
		}
		r := input().Bounds()
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		if got, want := input().borderC, c.th.Border.Mix(c.th.Text, 0.25); got != want {
			t.Errorf("%s: hovered input border %v, want %v", c.name, got, want)
		}
		tt.Move(395, 315)
		tt.Frame()
	}
}
