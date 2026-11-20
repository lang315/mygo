package skin

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// check measures t at the floor of Calm, 4.5, as the tests before floors did.
func check(t *skinhook.Tokens, dark bool) error { return checkFloor(t, dark, 4.5) }

// withFloor is s with the text floor f, as a built-in skin sets its own.
func withFloor(s *Skin, f float64) *Skin {
	c := *s
	c.floor = f
	return &c
}

// A skin's own floor is what Check asks of text: Calm's tokens with a text
// of 4.83:1 on the light Background pass at 4.5 and fail at 7, naming 7, and
// the dark Text, far over 7:1, is not reported.
func TestCheckUsesSkinFloor(t *testing.T) {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Text = ui.Hex("#6b7280") // 4.83:1 on white
	if err := New("X", l, d).Check(); err != nil && strings.Contains(err.Error(), "Text on Background:") {
		t.Errorf("at the floor 4.5, Check reports Text on Background: %v", err)
	}
	err := withFloor(New("X", l, d), 7).Check()
	if err == nil {
		t.Fatal("at the floor 7, Check passes a text of 4.83:1")
	}
	msg := err.Error()
	if n := strings.Count(msg, "Text on Background:"); n != 1 {
		t.Errorf("at the floor 7, Check names Text on Background %d times, want once (light only): %v", n, msg)
	}
	if !strings.Contains(msg, "Text on Background: 4.83:1, want 7:1") {
		t.Errorf("at the floor 7, Check does not report Text on Background at 4.83:1 against 7:1: %v", msg)
	}
}

// Calm keeps the floor 4.5 and passes it.
func TestCalmFloorUnchanged(t *testing.T) {
	if err := Calm.Check(); err != nil {
		t.Errorf("Calm.Check(): %v", err)
	}
	if f := Calm.textFloor(); f != 4.5 {
		t.Errorf("Calm's text floor is %v, want 4.5", f)
	}
}

// WithAccent derives for the floor of the skin it is called on: white on the
// light base reads at 7:1 for a skin of floor 7, though #1c71d8 gives 4.8:1,
// and it is no black: its text on the selected row reads too. The skin's
// muted text is its text, as HighContrast's, so the row can reach 7:1.
func TestWithAccentMeetsFloor(t *testing.T) {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.TextMuted, d.TextMuted = l.Text, d.Text
	base := withFloor(New("X", l, d), 7).WithAccent(ui.Hex("#1c71d8")).Tokens(false).Accent.Base
	if k := contrastOf(white, base); k < 7 {
		t.Errorf("WithAccent(#1c71d8) at the floor 7 gives the base %v, white on it %.2f:1, want 7", hexOf(base), k)
	}
	if lum, _ := lightChroma(base); lum < 0.2 {
		t.Errorf("WithAccent(#1c71d8) at the floor 7 gives the base %v, Oklab lightness %.2f: the search ran to black", hexOf(base), lum)
	}
	if base := Calm.WithAccent(ui.Hex("#1c71d8")).Tokens(false).Accent.Base; base != ui.Hex("#1c71d8") {
		t.Errorf("WithAccent(#1c71d8) at Calm's floor gives %v, want #1c71d8 kept", hexOf(base))
	}
}

// WithPalette, WithDensity, WithAccent and WithLooks keep the skin's floor.
func TestFloorSurvivesWith(t *testing.T) {
	s := withFloor(Calm, 7)
	for name, c := range map[string]*Skin{
		"WithPalette": s.WithPalette(Teal),
		"WithDensity": s.WithDensity(Compact),
		"WithAccent":  s.WithAccent(ui.Hex("#7c3aed")),
		"WithLooks":   s.WithLooks(Looks{Button: "border"}),
	} {
		if f := c.textFloor(); f != 7 {
			t.Errorf("%s: the floor is %v, want 7", name, f)
		}
	}
}
