package skin

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/i18n"
	_ "github.com/egoist/mygo/locales/vi"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The picker's boxes, in their order.
const (
	skinBox = iota
	paletteBox
	densityBox
)

// picker shows Picker(skins...) after Use(start), and turns the skin off
// when the test ends.
func picker(t *testing.T, start *Skin, skins ...*Skin) *widget.Tester {
	t.Helper()
	Use(start)
	t.Cleanup(func() { Use(nil) })
	return widget.NewTester(t, Picker(skins...), 600, 600)
}

func box(tt *widget.Tester, i int) *widget.Node {
	return tt.Find(widget.ByRole(ui.RolePopUpButton).Nth(i))
}

// choose opens box i and clicks its option text, as the user does.
func choose(t *testing.T, tt *widget.Tester, i int, text string) {
	t.Helper()
	box(tt, i).Click()
	tt.Find(widget.ByRole(ui.RoleListItem).Label(text)).Click()
}

// options opens box i and returns its options' texts, then closes it.
func options(t *testing.T, tt *widget.Tester, i int) []string {
	t.Helper()
	box(tt, i).Click()
	var out []string
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleListItem)) {
		out = append(out, n.Label())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	return out
}

// shows reports whether the window shows the text s.
func shows(tt *widget.Tester, s string) bool { return tt.HasText(s) }

func activeID(t *testing.T) string {
	t.Helper()
	a := Active()
	if a == nil {
		t.Fatal("no skin is active")
	}
	return a.ID()
}

func TestPickerUses(t *testing.T) {
	tt := picker(t, Calm)
	if got := options(t, tt, skinBox); !slices.Equal(got, []string{"Calm", "Classic", "HighContrast"}) {
		t.Errorf("Picker() lists %q, want Builtin", got)
	}
	choose(t, tt, skinBox, "Classic")
	if a := Active(); a == nil || a.Name() != "Classic" {
		t.Fatalf("after choosing Classic, Active is %v", a)
	}
	// The density chosen stays: Calm's Standard, not Classic's own Compact.
	if got := activeID(t); got != "Classic/Indigo/standard" {
		t.Errorf("after choosing Classic, Active is %s, want Classic/Indigo/standard", got)
	}
	choose(t, tt, densityBox, "Compact")
	if a := Active(); a.Name() != "Classic" || a.Density() != Compact {
		t.Errorf("after choosing Compact, Active is %s, want Classic compact", a.ID())
	}
	choose(t, tt, paletteBox, "Slate")
	if got := activeID(t); got != "Classic/Slate/compact" {
		t.Errorf("after choosing Slate, Active is %s, want Classic/Slate/compact", got)
	}
}

func TestPickerPaletteFollowsSkin(t *testing.T) {
	tt := picker(t, Calm.WithPalette(Indigo))
	choose(t, tt, skinBox, "Classic")
	// Classic's own Indigo, not Calm's: its light accent is #3949ab.
	if a := Active(); a.ID() != "Classic/Indigo/standard" || a.Tokens(false).Accent.Base != ui.Hex("#3949ab") {
		t.Errorf("Calm/Indigo then Classic gives %s, accent %v, want Classic/Indigo, #3949ab", a.ID(), a.Tokens(false).Accent.Base)
	}
	if got := options(t, tt, paletteBox); !slices.Equal(got, []string{"Indigo", "Slate"}) {
		t.Errorf("Classic's palettes %q, want Indigo, Slate", got)
	}
	choose(t, tt, skinBox, "HighContrast")
	if !box(tt, paletteBox).Disabled() {
		t.Error("the palette box is enabled for HighContrast, which has no palettes")
	}
	if got := activeID(t); got != "HighContrast//standard" {
		t.Errorf("HighContrast gives %s", got)
	}
	choose(t, tt, skinBox, "Calm")
	if box(tt, paletteBox).Disabled() {
		t.Error("the palette box stays disabled for Calm")
	}
	if got := activeID(t); got != "Calm/Blue/standard" {
		t.Errorf("HighContrast then Calm gives %s, want Calm's own Blue", got)
	}
	// Teal, which Classic lacks, gives Classic its own palette.
	choose(t, tt, paletteBox, "Teal")
	choose(t, tt, skinBox, "Classic")
	if got := activeID(t); got != "Classic/Indigo/standard" {
		t.Errorf("Calm/Teal then Classic gives %s, want Classic/Indigo/standard", got)
	}
}

// Choosing the density keeps the skin and the palette, an accent too.
func TestPickerDensityKeepsPalette(t *testing.T) {
	tt := picker(t, Calm.WithPalette(Teal))
	choose(t, tt, densityBox, "Compact")
	if got := activeID(t); got != "Calm/Teal/compact" {
		t.Errorf("Calm/Teal then Compact gives %s", got)
	}
	tt = picker(t, Calm.WithAccent(ui.Hex("#7c3aed")))
	choose(t, tt, densityBox, "Compact")
	if got := activeID(t); got != "Calm/#7c3aed/compact" {
		t.Errorf("Calm's accent then Compact gives %s", got)
	}
}

func TestPickerStartsAtActive(t *testing.T) {
	tt := picker(t, Classic.WithPalette(ClassicSlate))
	for _, s := range []string{"Classic", "Slate", "Compact"} {
		if !shows(tt, s) {
			t.Errorf("the picker does not show %q: %q", s, tt.Texts())
		}
	}
	if shows(tt, "Calm") || shows(tt, "Indigo") {
		t.Errorf("the picker shows another skin or palette: %q", tt.Texts())
	}
}

// With no skin on, or one not offered, the picker shows the first skin and
// turns nothing on until the user chooses.
func TestPickerStartsAtFirst(t *testing.T) {
	tt := picker(t, nil)
	tt.Settle()
	if a := Active(); a != nil {
		t.Errorf("the picker turned on %s by itself", a.ID())
	}
	if !shows(tt, "Calm") || !shows(tt, "Blue") || !shows(tt, "Standard") {
		t.Errorf("the picker does not show Calm, Blue, Standard: %q", tt.Texts())
	}
	tt = picker(t, HighContrast, Classic, Calm)
	tt.Settle()
	if a := Active(); a != HighContrast {
		t.Error("the picker called Use by itself: Active is no longer the HighContrast the app set")
	}
	if !shows(tt, "Classic") || !shows(tt, "Indigo") || !shows(tt, "Compact") {
		t.Errorf("the picker does not show Classic, Indigo, Compact: %q", tt.Texts())
	}
}

func TestPickerLabels(t *testing.T) {
	tt := picker(t, Calm)
	for _, s := range []string{"Skin", "Palette", "Density"} {
		if !tt.Has(widget.ByLabel(s)) {
			t.Errorf("no label %q: %q", s, tt.Texts())
		}
	}
}

func TestPickerLabelsVietnamese(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "vi" })
	t.Cleanup(func() { i18n.SetLocaleFunc(nil) })
	tt := picker(t, Calm)
	for _, s := range []string{"Giao diện", "Bảng màu", "Mật độ", "Tiêu chuẩn"} {
		if !shows(tt, s) {
			t.Errorf("no %q under vi: %q", s, tt.Texts())
		}
	}
	if got := options(t, tt, densityBox); !slices.Equal(got, []string{"Tiêu chuẩn", "Thu gọn"}) {
		t.Errorf("the densities under vi: %q", got)
	}
}

func TestPickerOwnSkins(t *testing.T) {
	mine := New("Mine", Calm.Tokens(false), Calm.Tokens(true))
	tt := picker(t, nil, mine, Calm)
	if got := options(t, tt, skinBox); !slices.Equal(got, []string{"Mine", "Calm"}) {
		t.Errorf("the skins %q, want Mine, Calm", got)
	}
	if !box(tt, paletteBox).Disabled() {
		t.Error("the palette box is enabled for Mine, which has no palettes")
	}
	choose(t, tt, skinBox, "Calm")
	choose(t, tt, skinBox, "Mine")
	if got := activeID(t); got != "Mine//standard" {
		t.Errorf("choosing Mine gives %s", got)
	}
	if Active().Tokens(false) != mine.Tokens(false) {
		t.Error("choosing Mine does not give the app's tokens")
	}
}

// The picker lists a lazy skin by name and builds it only when the user
// chooses it.
func TestPickerBuildsLazySkinWhenChosen(t *testing.T) {
	built := 0
	lazy := &Skin{name: "Lazy", lazy: &lazySkin{build: func() *Skin {
		built++
		return New("Lazy", Calm.Tokens(false), Calm.Tokens(true))
	}}}
	tt := picker(t, nil, Calm, lazy)
	if got := options(t, tt, skinBox); !slices.Equal(got, []string{"Calm", "Lazy"}) {
		t.Errorf("the skins %q", got)
	}
	if built != 0 {
		t.Fatalf("the picker built the lazy skin %d times before it was chosen", built)
	}
	choose(t, tt, skinBox, "Lazy")
	if built != 1 || Active().Name() != "Lazy" {
		t.Errorf("after choosing it: built %d times, Active %s", built, Active().ID())
	}
}
