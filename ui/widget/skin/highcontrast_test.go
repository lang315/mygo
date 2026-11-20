package skin

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/tw"
)

// HighContrast passes its own floors: text at 7:1.
func TestHighContrastCheck(t *testing.T) {
	if err := HighContrast.Check(); err != nil {
		t.Errorf("HighContrast.Check(): %v", err)
	}
	if f := HighContrast.textFloor(); f != 7 {
		t.Errorf("HighContrast's text floor is %v, want 7", f)
	}
}

// HighContrast's values are the plan's.
func TestHighContrastValues(t *testing.T) {
	for _, c := range []struct {
		dark                   bool
		background, text, base string
	}{
		{true, "#000000", "#ffffff", "#ffff00"},
		{false, "#ffffff", "#000000", "#00009f"},
	} {
		tk := HighContrast.Tokens(c.dark)
		if tk.Background != ui.Hex(c.background) || tk.Text != ui.Hex(c.text) || tk.Accent.Base != ui.Hex(c.base) {
			t.Errorf("dark=%v: Background %v, Text %v, Accent.Base %v; want %s, %s, %s",
				c.dark, hexOf(tk.Background), hexOf(tk.Text), hexOf(tk.Accent.Base), c.background, c.text, c.base)
		}
		if tk.TextMuted != tk.Text {
			t.Errorf("dark=%v: TextMuted %v, want Text %v", c.dark, tk.TextMuted, tk.Text)
		}
		if tk.FocusRingWidth != 3 {
			t.Errorf("dark=%v: FocusRingWidth %v, want 3", c.dark, tk.FocusRingWidth)
		}
		if tk.CardShadow != (Shadow{}) || tk.PopupShadow != (Shadow{}) {
			t.Errorf("dark=%v: CardShadow %+v, PopupShadow %+v; want none", c.dark, tk.CardShadow, tk.PopupShadow)
		}
		if tk.ControlHeight != 36 || tk.RowHeight != 30 {
			t.Errorf("dark=%v: ControlHeight %v, RowHeight %v; want Calm's Standard 36, 30", c.dark, tk.ControlHeight, tk.RowHeight)
		}
	}
	if p := HighContrast.Palettes(); p != nil {
		t.Errorf("HighContrast.Palettes() is %v, want nil", p)
	}
	if HighContrast.Name() != "HighContrast" {
		t.Errorf("HighContrast.Name() is %q", HighContrast.Name())
	}
}

// The selected row is inverted: its text is the Background, black on the
// dark cyan row and white on the light purple one, at 7:1 or more.
func TestHighContrastSelectedRowInverted(t *testing.T) {
	for _, c := range []struct {
		dark      bool
		row, text string
	}{
		{true, "#1aebff", "#000000"},
		{false, "#37006e", "#ffffff"},
	} {
		tk := HighContrast.Tokens(c.dark)
		th := themeOf(&tk, c.dark)
		if th.RowSelected != ui.Hex(c.row) || th.RowSelectedText != ui.Hex(c.text) {
			t.Errorf("dark=%v: the selected row is %v on %v, want %s on %s", c.dark, hexOf(th.RowSelectedText), hexOf(th.RowSelected), c.text, c.row)
		}
		if k := contrastOf(th.RowSelectedText, th.RowSelected); k < 7 {
			t.Errorf("dark=%v: the selected row's text is %.2f:1, want 7", c.dark, k)
		}
		// Text selection is a tint the skin's own text reads on at 7:1.
		if k := ratio(&tk, tk.Text, tk.Selection); k < 7 {
			t.Errorf("dark=%v: Text on Selection %v is %.2f:1, want 7", c.dark, tk.Selection, k)
		}
	}
}

// A selected row's text is Text where Text reads better on it, as Calm's,
// and Background where Background does; check measures the same pick.
func TestSelectedRowTextPick(t *testing.T) {
	tk := Calm.light
	if th := themeOf(&tk, false); th.RowSelectedText != tk.Text {
		t.Errorf("Calm: RowSelectedText %v, want Text %v", th.RowSelectedText, tk.Text)
	}
	tk.RowSelected = ui.Hex("#202020") // white reads on it, Text does not
	if th := themeOf(&tk, false); th.RowSelectedText != tk.Background {
		t.Errorf("a dark row: RowSelectedText %v, want Background %v", th.RowSelectedText, tk.Background)
	}
	if err := check(&tk, false); err != nil {
		t.Errorf("a dark row with white text on it: %v", err)
	}
}

// A Button, a TextBox and a GroupBox each draw a 2 px border in the border
// color, light and dark: the bottom edge's two rows are it, the row above
// is not.
func TestHighContrastBorders(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tk := HighContrast.Tokens(dark)
			w := widget.VBox(12).Add(widget.Button("OK"), widget.TextBox().SetAccessibleName("box"),
				widget.GroupBox("Group").Add(widget.Label("inside")).SetSize(0, 80))
			tt := showSkin(t, w, 300, 300, HighContrast, dark, 1)
			img := tt.Image()
			// The group box's frame is no node: it is the 80 DIPs under the
			// TextBox and the gap of 12.
			box := tt.Find(widget.ByLabel("box")).Bounds()
			group := ui.Rect{X: box.X, Y: box.Y + box.H + 12, W: box.W, H: 80}
			for _, c := range []struct {
				name   string
				r      ui.Rect
				border ui.Color
			}{
				{"Button", tt.Find(widget.ByLabel("OK")).Bounds(), tk.ButtonBorder},
				{"TextBox", tt.Find(widget.ByLabel("box")).Bounds(), tk.ControlBorder},
				{"GroupBox", group, tk.BorderStrong},
			} {
				x, bottom := int(c.r.X+c.r.W/2), int(c.r.Y+c.r.H)-1
				for i, want := range []bool{true, true, false} {
					got := colorAt(img, x, bottom-i)
					if (dist(got, c.border) <= 3) != want {
						t.Errorf("%s: row %d above its bottom edge is %v; the border %v there: %v", c.name, i, got, c.border, want)
					}
				}
			}
		})
	}
}

// WithAccent on HighContrast derives for its floor: S3's named colors each
// make a skin that passes Check at 7:1.
func TestHighContrastWithAccent(t *testing.T) {
	for _, h := range []string{"#ffd400", "#e0f2ff", "#0a0a0a", "#d32f2f", "#2e7d32", "#f59e0b", "#7c3aed", "#1c71d8"} {
		s := HighContrast.WithAccent(ui.Hex(h))
		if err := s.Check(); err != nil {
			t.Errorf("HighContrast.WithAccent(%s): %v", h, err)
		}
		if f := s.textFloor(); f != 7 {
			t.Errorf("HighContrast.WithAccent(%s): the floor is %v, want 7", h, f)
		}
	}
}

// WithPalette on HighContrast is not refused: the skin takes the palette,
// and Check reports the pairs under 7:1, naming the floor.
func TestHighContrastWithPalette(t *testing.T) {
	s := HighContrast.WithPalette(Blue)
	if s.Palette() != Blue || s.Tokens(false).Accent != Blue.light.accent {
		t.Fatalf("HighContrast.WithPalette(Blue) has the palette %v", s.Palette())
	}
	err := s.Check()
	if err == nil || !strings.Contains(err.Error(), "want 7:1") {
		t.Errorf("HighContrast.WithPalette(Blue).Check() is %v, want the pairs under 7:1", err)
	}
	t.Logf("HighContrast.WithPalette(Blue): %v", err)
}

// Every class of HighContrast's looks is one tw knows.
func TestHighContrastLooksParse(t *testing.T) {
	l := HighContrast.Looks()
	for name, cl := range map[string]string{"Button": l.Button, "ToolButton": l.ToolButton, "Field": l.Field,
		"ComboBox": l.ComboBox, "GroupBox": l.GroupBox, "Card": l.Card} {
		if cl == "" {
			t.Errorf("%s has no look", name)
		}
		if _, err := tw.Parse(cl); err != nil {
			t.Errorf("%s %q: %v", name, cl, err)
		}
	}
}
