package skin

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// A skin made of Calm's tokens with another Background gives a window that
// Background, and is the active skin by its own name.
func TestNewSkinApplies(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Background = ui.Hex("#fffaf0")
	Use(New("Brand", l, d))
	tt.Frame()
	if got.Background != ui.Hex("#fffaf0") {
		t.Errorf("light Background %v, want #fffaf0", got.Background)
	}
	if Active().Name() != "Brand" {
		t.Errorf("Active is %v, want Brand", activeName())
	}
	tt.SetDark(true)
	if got.Background != d.Background {
		t.Errorf("dark Background %v, want Calm's dark %v", got.Background, d.Background)
	}
}

// Tokens returns a copy, and New copies what it is given: changing either
// afterwards changes neither skin.
func TestTokensAndNewCopy(t *testing.T) {
	red, blue := ui.Hex("#ff0000"), ui.Hex("#0000ff")
	want := Calm.Tokens(false).Background
	if want != ui.Hex("#ffffff") || Calm.Tokens(true).Background != ui.Hex("#15171a") {
		t.Fatalf("Calm's Background is %v light and %v dark, want #ffffff and #15171a", want, Calm.Tokens(true).Background)
	}
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Background = red
	if got := Calm.Tokens(false).Background; got != want {
		t.Errorf("after changing a copy, Calm's Background is %v, want %v", got, want)
	}
	d.Background = red
	if got := Calm.Tokens(true).Background; got == red {
		t.Error("after changing a dark copy, Calm's dark Background is red")
	}
	s := New("B", l, d)
	l.Background, d.Background = blue, blue
	if got := s.Tokens(false).Background; got != red {
		t.Errorf("after changing what New took, the light Background is %v, want red", got)
	}
	if got := s.Tokens(true).Background; got != red {
		t.Errorf("after changing what New took, the dark Background is %v, want red", got)
	}
	u := s.Tokens(false)
	u.Background = blue
	if got := s.Tokens(false).Background; got != red {
		t.Errorf("after changing what Tokens returned, the Background is %v, want red", got)
	}
}

// The tokens' Font is the theme's, and Calm's is the system's.
func TestTokensFont(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(Calm)
	tt.Frame()
	if got.Font != "" {
		t.Errorf("under Calm the Font is %q, want the system's", got.Font)
	}
	l := Calm.Tokens(false)
	l.Font = "Inter"
	Use(New("Brand", l, Calm.Tokens(true)))
	tt.Frame()
	if got.Font != "Inter" {
		t.Errorf("the Font is %q, want Inter", got.Font)
	}
	tt.SetDark(true)
	if got.Font != "" {
		t.Errorf("dark, whose Font is empty, the Font is %q", got.Font)
	}
}

// The tokens' sizes are the theme's Metrics.
func TestTokensSizes(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	l := Calm.Tokens(false)
	l.ControlHeight, l.TabHeight, l.CheckSize = 32, 44, 18
	Use(New("Brand", l, Calm.Tokens(true)))
	tt.Frame()
	if m := got.Metrics; m.ControlHeight != 32 || m.TabHeight != 44 || m.CheckSize != 18 {
		t.Errorf("Metrics %+v, want ControlHeight 32, TabHeight 44, CheckSize 18", m)
	}
}

// Calm passes its check with every palette.
func TestSkinCheckPasses(t *testing.T) {
	if err := Calm.Check(); err != nil {
		t.Errorf("Calm: %v", err)
	}
	for _, p := range palettes {
		if err := Calm.WithPalette(p).Check(); err != nil {
			t.Errorf("Calm with %s: %v", p.Name(), err)
		}
	}
}

// Check names the appearance and the failing pair, and does not stop Use.
func TestSkinCheckNamesTheAppearance(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Text = l.Background
	s := New("Bad", l, d)
	err := s.Check()
	if err == nil || !strings.Contains(err.Error(), "light") || !strings.Contains(err.Error(), "Text on Background") {
		t.Errorf("Text = Background gives %v, want an error naming light and Text on Background", err)
	}
	if err != nil && strings.Contains(err.Error(), "dark") {
		t.Errorf("only light fails, the error is %v", err)
	}
	d.Text = d.Background
	if err := New("Bad", Calm.Tokens(false), d).Check(); err == nil || !strings.Contains(err.Error(), "dark") || strings.Contains(err.Error(), "light") {
		t.Errorf("dark Text = Background gives %v, want an error naming dark alone", err)
	}
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(s)
	tt.Frame()
	if got.Text != got.Background || got.Text != l.Background {
		t.Errorf("Use of a skin that fails its check gives Text %v and Background %v, want both %v", got.Text, got.Background, l.Background)
	}
}

// A palette made outside the package colors the accent, Info, the selection,
// the selected row and the focus ring.
func TestNewPaletteApplies(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	base := ui.Hex("#7c3aed")
	lf := LightFamily(base, ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9"))
	df := DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033"))
	p := NewPalette("Brand", lf, df)
	if p.Name() != "Brand" {
		t.Errorf("Name() is %q, want Brand", p.Name())
	}
	s := Calm.WithPalette(p)
	l := s.Tokens(false)
	if l.Accent.Base != base || l.Info != l.Accent {
		t.Errorf("Accent.Base %v, Info %v; want #7c3aed and the accent", l.Accent.Base, l.Info)
	}
	if l.Selection != base.Mix(white, 0.68) || l.RowSelected != base.Mix(white, 0.82) {
		t.Errorf("Selection %v, RowSelected %v; want the base mixed with white at 0.68 and 0.82", l.Selection, l.RowSelected)
	}
	if d := s.Tokens(true); d.Accent != df || d.RowSelected != df.Base.Alpha(0.16) || d.Selection != df.Base.Alpha(0.35) {
		t.Errorf("dark Accent %v, RowSelected %v, Selection %v", d.Accent, d.RowSelected, d.Selection)
	}
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(s)
	tt.Frame()
	if got.Focus != base {
		t.Errorf("Focus %v, want #7c3aed", got.Focus)
	}
}

// LightFamily mixes the base with white; DarkFamily makes it translucent.
func TestFamilies(t *testing.T) {
	b, h, p, tx, on := ui.Hex("#7c3aed"), ui.Hex("#010203"), ui.Hex("#040506"), ui.Hex("#070809"), ui.Hex("#0a0b0c")
	l := LightFamily(b, h, p, tx)
	if want := (Family{Base: b, Hover: h, Pressed: p, Subtle: b.Mix(white, 0.92), Border: b.Mix(white, 0.70), Text: tx, On: white}); l != want {
		t.Errorf("LightFamily is %+v, want %+v", l, want)
	}
	d := DarkFamily(b, h, p, tx, on)
	if want := (Family{Base: b, Hover: h, Pressed: p, Subtle: b.Alpha(0.14), Border: b.Alpha(0.35), Text: tx, On: on}); d != want {
		t.Errorf("DarkFamily is %+v, want %+v", d, want)
	}
}

// The text on the accent is the accent family's On, in both appearances.
func TestAccentTextIsAccentOn(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Accent.On, d.Accent.On = ui.Hex("#123456"), ui.Hex("#654321")
	for _, s := range []*Skin{Calm, New("Brand", l, d)} {
		Use(s)
		for _, dark := range []bool{false, true} {
			tt.SetDark(dark)
			tt.Frame()
			if want := s.Tokens(dark).Accent.On; got.AccentText != want {
				t.Errorf("%s dark=%v: AccentText %v, want %v", s.Name(), dark, got.AccentText, want)
			}
		}
	}
}

// The rows a grid keeps take the row height of a skin made with New, after
// Calm's.
func TestKeptRowsResizeOnNew(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var builds atomic.Int32
	grid := widget.DataGridView(func(r *sizeRow) string { return r.ID },
		widget.Column[sizeRow]{Title: "Name", Text: func(r *sizeRow) string { builds.Add(1); return r.Name }},
	).SetRows([]sizeRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}).SetSize(0, 180)
	note := widget.Label("one")
	tt := widget.NewTester(t, widget.VBox(8).Add(note, grid), 600, 400)
	Use(Calm)
	tt.Frame()
	rows, _ := gridHeights(t, tt)
	wantRows(t, "after Use(Calm)", rows, 30)
	builds.Store(0)
	note.SetText("two")
	tt.Frame()
	if n := builds.Load(); n != 0 {
		t.Fatalf("a frame that changed only a label built %d cells: the grid keeps no rows, and the test sees nothing", n)
	}
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.RowHeight, d.RowHeight = 40, 40
	Use(New("Tall", l, d))
	tt.Frame()
	rows, _ = gridHeights(t, tt)
	wantRows(t, "after Use(New) with RowHeight 40", rows, 40)
	if l, d := skinhook.Get(false), skinhook.Get(true); l == nil || d == nil || l.RowHeight != 40 || d.RowHeight != 40 {
		t.Errorf("the hook's tokens are %p and %p, want the new skin's, RowHeight 40", l, d)
	}
}

// Zero tokens and families make a skin and a palette that work: Check names
// failing pairs and does not panic, and Use gives the zero colors.
func TestZeroTokens(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	s := New("", Tokens{}, Tokens{})
	if s.Name() != "" || s.Palette() != nil {
		t.Errorf("Name %q, Palette %v; want empty and nil", s.Name(), s.Palette())
	}
	err := s.Check()
	if err == nil || !strings.Contains(err.Error(), "light") || !strings.Contains(err.Error(), "dark") || !strings.Contains(err.Error(), "Text on Background") {
		t.Errorf("zero tokens give %v, want an error naming light, dark and Text on Background", err)
	}
	p := NewPalette("", Family{}, Family{})
	if err := s.WithPalette(p).Check(); err == nil {
		t.Error("zero tokens with a zero palette pass the check")
	}
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(s.WithPalette(p))
	tt.Frame()
	if got.Background != (ui.Color{}) || got.Accent != (ui.Color{}) || got.Font != "" {
		t.Errorf("Background %v, Accent %v, Font %q; want the zero colors and the system's font", got.Background, got.Accent, got.Font)
	}
}

// Check holds each appearance to its own state-distance floor: an Info text
// 40 from green fails the light floor, 45, and passes the dark one, 35.
func TestSkinCheckFloorPerAppearance(t *testing.T) {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Info.Text = ui.Hex("#7a2569")
	err := New("Plum", l, d).Check()
	if err == nil || !strings.Contains(err.Error(), "light") || !strings.Contains(err.Error(), "want 45") {
		t.Errorf("light Info.Text #7a2569 gives %v, want an error naming light and want 45", err)
	}
	if err := New("Plum", Calm.Tokens(false), d).Check(); err != nil {
		t.Errorf("Calm's own: %v", err)
	}
	d.Info.Text = ui.Hex("#7a2569").Mix(ui.Hex("#ffffff"), 0.5)
	if err := New("Plum", Calm.Tokens(false), d).Check(); err != nil && strings.Contains(err.Error(), "want 45") {
		t.Errorf("a dark skin is held to the light floor: %v", err)
	}
}
