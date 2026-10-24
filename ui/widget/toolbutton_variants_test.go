package widget

import (
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// look is the colors of a button left of its text at rest, hovered and
// pressed.
type look struct{ rest, hover, pressed color.RGBA }

// buttonLook shows w, a Button or a ToolButton in a ToolStrip, in the light
// or the dark theme and returns its look.
func buttonLook(t *testing.T, w Control, text string, dark bool) look {
	t.Helper()
	tt := tester(t, w)
	tt.SetDark(dark)
	tt.Frame()
	l := look{rest: pixelLeft(t, tt, text)}
	x, y := centerOf(t, tt, text)
	tt.Move(x, y)
	l.hover = pixelLeft(t, tt, text)
	tt.Press(x, y)
	l.pressed = pixelLeft(t, tt, text)
	return l
}

func inStrip(b *ToolButtonWidget) Control { return ToolStrip(b) }

// A ToolButton's Suggested and Destructive draw as the Button's, in both
// themes: the accent or the danger color, and the hover and pressed looks.
func TestToolButtonVariantsLookAsTheButtons(t *testing.T) {
	for _, dark := range []bool{false, true} {
		th := ui.LightTheme()
		if dark {
			th = ui.DarkTheme()
		}
		t.Run(map[bool]string{false: "light", true: "dark"}[dark], func(t *testing.T) {
			got := buttonLook(t, inStrip(ToolButton("Save").Suggested()), "Save", dark)
			want := buttonLook(t, Button("Save").Suggested(), "Save", dark)
			if got != want {
				t.Errorf("suggested: ToolButton %v, Button %v", got, want)
			}
			if !near(got.rest, th.Accent) || !near(got.hover, th.AccentHover) || !near(got.pressed, th.AccentPressed) {
				t.Errorf("suggested %v: want the accent %v, hover %v, pressed %v", got, th.Accent, th.AccentHover, th.AccentPressed)
			}

			got = buttonLook(t, inStrip(ToolButton("Delete").Destructive()), "Delete", dark)
			want = buttonLook(t, Button("Delete").Destructive(), "Delete", dark)
			if got != want {
				t.Errorf("destructive: ToolButton %v, Button %v", got, want)
			}
			if !near(got.rest, th.Danger) {
				t.Errorf("destructive %v: want the danger color %v", got.rest, th.Danger)
			}
			if got.hover == got.rest || got.pressed == got.hover {
				t.Errorf("destructive %v: hover and pressed must differ from rest and each other", got)
			}
		})
	}
}

// A disabled ToolButton looks as a disabled Button of the variant does, and
// not as an enabled one.
func TestToolButtonVariantsDisabledLookAsTheButtons(t *testing.T) {
	for name, c := range map[string]struct {
		tool func() *ToolButtonWidget
		btn  func() *ButtonWidget
	}{
		"suggested": {
			func() *ToolButtonWidget { return ToolButton("Go").Suggested() },
			func() *ButtonWidget { return Button("Go").Suggested() },
		},
		"destructive": {
			func() *ToolButtonWidget { return ToolButton("Go").Destructive() },
			func() *ButtonWidget { return Button("Go").Destructive() },
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, dark := range []bool{false, true} {
				enabled := buttonLook(t, inStrip(c.tool()), "Go", dark).rest
				got := buttonLook(t, inStrip(c.tool().SetDisabled(true)), "Go", dark).rest
				want := buttonLook(t, c.btn().SetDisabled(true), "Go", dark).rest
				if got != want {
					t.Errorf("dark %v: disabled ToolButton %v, disabled Button %v", dark, got, want)
				}
				if got == enabled {
					t.Errorf("dark %v: disabled ToolButton looks enabled (%v)", dark, got)
				}
			}
		})
	}
}

// The variants are the button's own style: they win over applied ones, as
// the Button's do.
func TestToolButtonSuggestedIsTheWidgetsOwnStyle(t *testing.T) {
	th := ui.LightTheme()
	b := ToolButton("Save").Apply(NewStyle().Background(Fixed(red))).Suggested()
	if c := buttonLook(t, inStrip(b), "Save", false).rest; !near(c, th.Accent) {
		t.Fatalf("applied red over Suggested: %v, want the accent", c)
	}
}

// An overflowed Destructive button is a destructive item of the overflow
// menu. A Suggested one is a normal item: menus have no suggested role.
func TestToolStripOverflowShowsADestructiveButtonAsADestructiveItem(t *testing.T) {
	f := NewForm("Shell", 800, 600).ToolStrip(ToolStrip(
		ToolButton("Alpha"), ToolButton("Beta"),
		ToolButton("Delete all").Destructive(), ToolButton("Start").Suggested(),
		ToolButton("Keep").Destructive().Suggested()))
	tt := ui.NewTester(f.view, 110, 300)
	tt.Frame()
	if err := tt.Click("More"); err != nil {
		t.Fatalf("no overflow button (texts %q): %v", tt.Texts(), err)
	}
	tt.Frame()
	menu := tt.Menu()
	if !slices.Contains(menu, "Delete all") || !slices.Contains(menu, "Start") {
		t.Fatalf("overflow menu %q: want Delete all and Start in it", menu)
	}
	if !tt.MenuItemDestructive("Delete all") {
		t.Error("Delete all is a normal item in the overflow menu")
	}
	if tt.MenuItemDestructive("Start") {
		t.Error("Start, which is Suggested, is a destructive item")
	}
	if !slices.Contains(menu, "Keep") || tt.MenuItemDestructive("Keep") {
		t.Errorf("Keep, made Suggested after Destructive: in the menu %v, destructive %v", slices.Contains(menu, "Keep"), tt.MenuItemDestructive("Keep"))
	}
}
