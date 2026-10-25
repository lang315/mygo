//go:build !mygo_noinspector

package widget

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

func inspectedButton(t *testing.T) (*Tester, *ButtonWidget) {
	t.Helper()
	b := Button("Save").
		Apply(NewStyle().Padding(2).Radius(1).Background(RoleAccent).
			Hover(NewStyle().Background(Fixed(ui.RGB(255, 0, 0)))).
			Dark(NewStyle().TextColor(RoleText)).
			Pressed(NewStyle().Opacity(0.5))).
		Bold()
	return NewTester(t, b, 1000, 600), b
}

func TestInspectorShowsWhatAControlIsStyledWith(t *testing.T) {
	tt, _ := inspectedButton(t)
	if err := tt.Inspect("Save"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Widget")
	for _, s := range []string{
		"*widget.ButtonWidget", "disabled: false",
		"Apply, style 1 of 1", "padding: 2 units (8 DIPs)", "radius: 1 units (4 DIPs)", "background: role accent",
		"hover:background: #ff0000", "dark:color: role text", "pressed:opacity: 0.5",
		"Own methods (Padding, Bold, …)", "font-weight: 700",
	} {
		if !tt.HasText(s) {
			t.Errorf("the Widget tab lacks %q: %q", s, tt.Texts())
		}
	}
}

func TestInspectorForcesTheStatesOfAWidget(t *testing.T) {
	tt, _ := inspectedButton(t)
	if err := tt.Inspect("Save"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Widget")
	centre := func() color.RGBA {
		r := tt.Find(ByLabel("Save")).Bounds()
		return tt.Image().RGBAAt(int(r.X)+3, int(r.Y+r.H/2)) // beside the text
	}
	red := color.RGBA{255, 0, 0, 255}
	if got := centre(); got == red {
		t.Fatalf("the button is red before it is hovered: %v", got)
	}
	if err := tt.Click("Force hover"); err != nil {
		t.Fatal(err)
	}
	if got := centre(); got != red {
		t.Errorf("forced hover: the button is %v, want the Style's hover %v", got, red)
	}
	if err := tt.Click("Force hover"); err != nil {
		t.Fatal(err)
	}
	if got := centre(); got == red {
		t.Errorf("hover unforced: the button is still %v", got)
	}
}

func TestNotesAreLeftOnlyWhileTheInspectorIsOpen(t *testing.T) {
	tt, _ := inspectedButton(t)
	before := notesLeft.Load()
	for range 3 {
		tt.Frame()
	}
	if got := notesLeft.Load() - before; got != 0 {
		t.Fatalf("%d notes left with the inspector closed", got)
	}
	if err := tt.Inspect("Save"); err != nil {
		t.Fatal(err)
	}
	if notesLeft.Load() == before {
		t.Error("no note left with the inspector open")
	}
}
