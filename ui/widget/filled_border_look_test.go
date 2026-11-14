package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A Border given after Suggested or Destructive is red in every state, not
// only at rest: the look's hover and pressed colors do not lay over it.
func TestOwnBorderWinsInEveryState(t *testing.T) {
	for name, w := range map[string]func() Control{
		"suggested":          func() Control { return Button("Go").Suggested().Border(1, red) },
		"destructive":        func() Control { return Button("Go").Destructive().Border(1, red) },
		"tool suggested":     func() Control { return ToolStrip(ToolButton("Go").Suggested().Border(1, red)) },
		"tool destructive":   func() Control { return ToolStrip(ToolButton("Go").Destructive().Border(1, red)) },
		"suggested, twice":   func() Control { return Button("Go").Suggested().Destructive().Suggested().Border(1, red) },
		"destructive, twice": func() Control { return Button("Go").Destructive().Suggested().Destructive().Border(1, red) },
	} {
		for _, dark := range []bool{false, true} {
			rest, hover, pressed := edgesInStates(t, w(), "Go", dark)
			for state, e := range map[string]edges{"rest": rest, "hover": hover, "pressed": pressed} {
				if !near(e.border, red) {
					t.Errorf("%s, dark %v, %s: border %v, want red", name, dark, state, e.border)
				}
			}
		}
	}
}

// A Border given before the look is replaced by it, in every state.
func TestLookReplacesAnEarlierBorder(t *testing.T) {
	rest, hover, pressed := edgesInStates(t, Button("Go").Border(1, red).Suggested(), "Go", false)
	for state, e := range map[string]edges{"rest": rest, "hover": hover, "pressed": pressed} {
		if e.border != e.fill {
			t.Errorf("%s: border %v, fill %v: want the same", state, e.border, e.fill)
		}
	}
}

// Suggested, Destructive and Suggested again leave nothing of the looks
// between: the button is as one that was only Suggested, in every state.
func TestLooksLeaveNoStaleVariants(t *testing.T) {
	type states struct{ rest, hover, pressed edges }
	read := func(b *ButtonWidget) states {
		r, h, p := edgesInStates(t, b, "Go", false)
		return states{r, h, p}
	}
	if got, want := read(Button("Go").Suggested().Destructive().Suggested()), read(Button("Go").Suggested()); got != want {
		t.Errorf("Suggested, Destructive, Suggested: %v, only Suggested: %v", got, want)
	}
	if got, want := read(Button("Go").Destructive().Suggested().Destructive()), read(Button("Go").Destructive()); got != want {
		t.Errorf("Destructive, Suggested, Destructive: %v, only Destructive: %v", got, want)
	}
}

// A hover look the app gave its own (an opacity) stays through the looks that
// follow it: Destructive does not set opacity in its hover, so Suggested
// takes nothing of it away.
func TestSuggestedAfterDestructiveKeepsTheOwnHover(t *testing.T) {
	th := ui.LightTheme()
	b := Button("Go").Destructive()
	b.own(func(s Style) Style { return s.Hover(NewStyle().Opacity(0.5)) })
	b.Suggested()
	tt := lightTester(t, b)
	pg := page(tt)
	tt.Move(centerOf(t, tt, "Go"))
	if c := pixelLeft(t, tt, "Go"); !near(c, mix(th.AccentHover, pg, 0.5)) {
		t.Fatalf("hovered %v, want the accent hover at 0.5 over the page", c)
	}
}

// Under High Contrast the look keeps the plain button's strong outline, not
// a border in its fill.
func TestFilledBorderKeepsTheOutlineUnderHighContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		read := func(b *ButtonWidget) (rest, hover, pressed edges) {
			tt := NewTester(t, b, 400, 200)
			tt.SetDark(dark)
			tt.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
			tt.Frame()
			return statesOf(t, tt, "Go")
		}
		plain, _, _ := read(Button("Go"))
		for name, b := range map[string]*ButtonWidget{"suggested": Button("Go").Suggested(), "destructive": Button("Go").Destructive()} {
			rest, hover, pressed := read(b)
			for state, e := range map[string]edges{"rest": rest, "hover": hover, "pressed": pressed} {
				if e.border != plain.border {
					t.Errorf("%s, dark %v, %s: border %v, want the plain outline %v", name, dark, state, e.border, plain.border)
				}
				if e.border == e.fill {
					t.Errorf("%s, dark %v, %s: border %v is the fill", name, dark, state, e.border)
				}
			}
		}
	}
}
