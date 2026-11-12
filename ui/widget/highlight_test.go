package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The option the keys or the pointer are on: the accent and AccentText
// without row colors, as before them; RowSelected with Text on it when
// RowSelectedText is zero; both when set.
func TestHighlightOf(t *testing.T) {
	th := ui.LightTheme()
	if bg, fg := highlightOf(th); bg != th.Accent || fg != th.AccentText {
		t.Errorf("plain: %v on %v, want AccentText %v on Accent %v", fg, bg, th.AccentText, th.Accent)
	}
	th.RowSelected = ui.Hex("#e8f0fd")
	if bg, fg := highlightOf(th); bg != th.RowSelected || fg != th.Text {
		t.Errorf("RowSelected alone: %v on %v, want Text %v on %v", fg, bg, th.Text, th.RowSelected)
	}
	th.RowSelectedText = ui.Hex("#102030")
	if bg, fg := highlightOf(th); bg != th.RowSelected || fg != th.RowSelectedText {
		t.Errorf("both: %v on %v, want %v on %v", fg, bg, th.RowSelectedText, th.RowSelected)
	}
}

// Without a skin, the option under the pointer of a drop-down and of an
// editable combo fills with the accent, as before the row colors.
func TestComboHighlightKeepsThePlainAccent(t *testing.T) {
	accent := ui.LightTheme().Accent
	for _, editable := range []bool{false, true} {
		c := ComboBox(Option{"a", "Apple"}, Option{"k", "Kale"})
		if editable {
			c.Editable()
		}
		tt := NewTester(t, c, 300, 200)
		if editable {
			focus(tt.Tester, 1)
			tt.Tester.Key(0, ui.KeyDown)
		} else {
			tt.Find(ByRole(ui.RolePopUpButton)).Click()
		}
		tt.Frame()
		r := tt.Find(ByLabel("Kale").Nth(0)).Bounds() // the option, before its text
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		p := tt.Image().RGBAAt(int(r.X+r.W-6), int(r.Y+r.H/2))
		if got := (ui.Color{R: p.R, G: p.G, B: p.B, A: p.A}); got != accent {
			t.Errorf("editable %v: the option under the pointer fills %v, want the accent %v", editable, got, accent)
		}
	}
}
