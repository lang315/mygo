package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// edges is what a button draws on its outline row (the middle of its top
// edge, where no corner reaches) and in its padding, at scale 1.
type edges struct{ border, fill color.RGBA }

// edgesOf reads the button named label of tt, whose bounds include its border.
func edgesOf(t *testing.T, tt *Tester, label string) edges {
	t.Helper()
	b := tt.Find(ByLabel(label)).Bounds()
	img := tt.Image()
	return edges{
		border: img.RGBAAt(int(b.X+b.W/2), int(b.Y)),
		fill:   img.RGBAAt(int(b.X)+6, int(b.Y+b.H/2)),
	}
}

// edgesInStates shows w, sets the theme, and reads its edges at rest,
// hovered and pressed.
func edgesInStates(t *testing.T, w Control, label string, dark bool) (rest, hover, pressed edges) {
	t.Helper()
	tt := NewTester(t, w, 400, 200)
	tt.SetDark(dark)
	tt.Frame()
	return statesOf(t, tt, label)
}

// statesOf reads the edges of the button label of tt at rest, hovered and
// pressed.
func statesOf(t *testing.T, tt *Tester, label string) (rest, hover, pressed edges) {
	t.Helper()
	rest = edgesOf(t, tt, label)
	b := tt.Find(ByLabel(label)).Bounds()
	x, y := b.X+b.W/2, b.Y+b.H/2
	tt.Move(x, y)
	hover = edgesOf(t, tt, label)
	tt.Press(x, y)
	pressed = edgesOf(t, tt, label)
	return rest, hover, pressed
}

// Case 4 (no skin) and 2: a Suggested button draws its border in its fill, at
// rest, hovered and pressed.
func TestSuggestedBorderIsItsFillNoSkin(t *testing.T) {
	for _, dark := range []bool{false, true} {
		th := ui.LightTheme()
		if dark {
			th = ui.DarkTheme()
		}
		rest, hover, pressed := edgesInStates(t, Button("Save").Suggested(), "Save", dark)
		plain, _, _ := edgesInStates(t, Button("Save"), "Save", dark)
		for _, c := range []struct {
			state string
			e     edges
			want  ui.Color
		}{{"rest", rest, th.Accent}, {"hover", hover, th.AccentHover}, {"pressed", pressed, th.AccentPressed}} {
			if c.e.border != c.e.fill {
				t.Errorf("dark %v %s: border %v, fill %v: want the same", dark, c.state, c.e.border, c.e.fill)
			}
			if !near(c.e.border, c.want) {
				t.Errorf("dark %v %s: border %v, want %v", dark, c.state, c.e.border, c.want)
			}
		}
		if rest.border == plain.border {
			t.Errorf("dark %v: Suggested's border %v is the plain button's", dark, rest.border)
		}
	}
}

// Case 3 and 4: a Destructive button draws its border in its fill in each state.
func TestDestructiveBorderIsItsFillNoSkin(t *testing.T) {
	for _, dark := range []bool{false, true} {
		rest, hover, pressed := edgesInStates(t, Button("Delete").Destructive(), "Delete", dark)
		plain, _, _ := edgesInStates(t, Button("Delete"), "Delete", dark)
		for state, e := range map[string]edges{"rest": rest, "hover": hover, "pressed": pressed} {
			if e.border != e.fill {
				t.Errorf("dark %v %s: border %v, fill %v: want the same", dark, state, e.border, e.fill)
			}
		}
		if rest.border == plain.border {
			t.Errorf("dark %v: Destructive's border %v is the plain button's", dark, rest.border)
		}
	}
}

// Case 5: the border keeps its width, so a Suggested or Destructive button is
// as large as a plain one.
func TestFilledBorderKeepsTheSizeNoSkin(t *testing.T) {
	size := func(b *ButtonWidget) (w, h float32) {
		tt := NewTester(t, b, 400, 200)
		r := tt.Find(ByLabel("Go")).Bounds()
		return r.W, r.H
	}
	pw, ph := size(Button("Go"))
	for name, b := range map[string]*ButtonWidget{"suggested": Button("Go").Suggested(), "destructive": Button("Go").Destructive()} {
		if w, h := size(b); w != pw || h != ph {
			t.Errorf("%s %v x %v, plain %v x %v", name, w, h, pw, ph)
		}
	}
}

// Case 6: the look's border is its own style, as its background is: a later
// own call (Border) wins, an applied border color loses.
func TestFilledBorderIsTheOwnStyle(t *testing.T) {
	accent := ui.LightTheme().Accent
	for name, c := range map[string]struct {
		b    *ButtonWidget
		want ui.Color
	}{
		"Border after Suggested":     {Button("Go").Suggested().Border(1, red), red},
		"applied BorderColor":        {Button("Go").Suggested().Apply(NewStyle().BorderColor(Fixed(red))), accent},
		"applied BorderColor, other": {Button("Go").Apply(NewStyle().BorderColor(Fixed(red))).Suggested(), accent},
	} {
		tt := NewTester(t, c.b, 400, 200)
		tt.SetDark(false)
		tt.Frame()
		if e := edgesOf(t, tt, "Go"); !near(e.border, c.want) {
			t.Errorf("%s: border %v, want %v", name, e.border, c.want)
		}
	}
}

// Case 8 (no skin): a tool button's Suggested and Destructive draw no border
// apart from their fill.
func TestToolButtonFilledBorderNoSkin(t *testing.T) {
	for name, b := range map[string]*ToolButtonWidget{"suggested": ToolButton("Go").Suggested(), "destructive": ToolButton("Go").Destructive()} {
		rest, hover, pressed := edgesInStates(t, ToolStrip(b), "Go", false)
		for state, e := range map[string]edges{"rest": rest, "hover": hover, "pressed": pressed} {
			if e.border != e.fill {
				t.Errorf("%s %s: border %v, fill %v: want the same", name, state, e.border, e.fill)
			}
		}
	}
}
