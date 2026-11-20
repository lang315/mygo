package skin

import (
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// sameSkin fails the test unless got is want: the same name, tokens in both
// appearances, palette name, density and looks.
func sameSkin(t *testing.T, id string, got, want *Skin) {
	t.Helper()
	if got == nil {
		t.Errorf("FromID(%q) is nil, want %s", id, want.ID())
		return
	}
	palette := func(s *Skin) string {
		if p := s.Palette(); p != nil {
			return p.Name()
		}
		return "<nil>"
	}
	switch {
	case got.Name() != want.Name():
		t.Errorf("FromID(%q): name %q, want %q", id, got.Name(), want.Name())
	case palette(got) != palette(want):
		t.Errorf("FromID(%q): palette %s, want %s", id, palette(got), palette(want))
	case got.Density() != want.Density():
		t.Errorf("FromID(%q): density %v, want %v", id, got.Density(), want.Density())
	case !reflect.DeepEqual(got.Tokens(false), want.Tokens(false)):
		t.Errorf("FromID(%q): the light tokens differ", id)
	case !reflect.DeepEqual(got.Tokens(true), want.Tokens(true)):
		t.Errorf("FromID(%q): the dark tokens differ", id)
	case got.Looks() != want.Looks():
		t.Errorf("FromID(%q): looks %+v, want %+v", id, got.Looks(), want.Looks())
	}
}

func TestBuiltin(t *testing.T) {
	b := Builtin()
	var names []string
	for _, s := range b {
		names = append(names, s.Name())
		if nameOf(s) != s.Name() {
			t.Errorf("the name without building is %q, built %q", nameOf(s), s.Name())
		}
	}
	if want := []string{"Calm", "Classic", "HighContrast"}; !slices.Equal(names, want) {
		t.Fatalf("Builtin names %q, want %q", names, want)
	}
	// The lazy skins' literals hold their names, so listing them builds none.
	if Classic.name != "Classic" || HighContrast.name != "HighContrast" {
		t.Errorf("the lazy literals are named %q and %q", Classic.name, HighContrast.name)
	}
	if b[0] != Calm || b[1] != Classic || b[2] != HighContrast {
		t.Errorf("Builtin is not the package's skins")
	}
	b[0] = nil
	if again := Builtin(); again[0] != Calm {
		t.Errorf("a change to Builtin's slice shows in the next call: %v", again[0])
	}
}

func TestID(t *testing.T) {
	for _, c := range []struct {
		s    *Skin
		want string
	}{
		{Calm, "Calm/Blue/standard"},
		{Classic, "Classic/Indigo/compact"},
		{HighContrast, "HighContrast//standard"},
		{Calm.WithAccent(ui.Hex("#7c3aed")), "Calm/#7c3aed/standard"},
		{Calm.WithPalette(Teal).WithDensity(Compact), "Calm/Teal/compact"},
		{Calm.WithPalette(Cham), "Calm/Chàm/standard"},
		{Classic.WithDensity(Standard), "Classic/Indigo/standard"},
		// A light color's base darkens; the ID keeps the color asked for.
		{Calm.WithAccent(ui.Hex("#ffd700")), "Calm/#ffd700/standard"},
	} {
		if got := c.s.ID(); got != c.want {
			t.Errorf("ID() = %q, want %q", got, c.want)
		}
	}
}

// Every built-in skin, with each of its palettes and its own, at each
// density, and with an accent, comes back from its ID.
func TestFromIDRoundTrip(t *testing.T) {
	var all []*Skin
	for _, b := range Builtin() {
		for _, d := range []Density{Standard, Compact} {
			all = append(all, b.WithDensity(d))
			for _, p := range b.Palettes() {
				all = append(all, b.WithPalette(p).WithDensity(d))
			}
		}
	}
	all = append(all,
		Calm.WithAccent(ui.Hex("#7c3aed")),
		Calm.WithAccent(ui.Hex("#ffd700")).WithDensity(Compact),
		Classic.WithAccent(ui.Hex("#0f766e")),
		HighContrast.WithAccent(ui.Hex("#7c3aed")))
	if len(all) != 2*(1+5)+2*(1+2)+2*1+4 {
		t.Fatalf("%d skins to round-trip, want 24", len(all))
	}
	for _, x := range all {
		id := x.ID()
		got := FromID(id)
		sameSkin(t, id, got, x)
		if got != nil && got.ID() != id {
			t.Errorf("FromID(%q).ID() = %q", id, got.ID())
		}
	}
}

// An ID of the skin's own palette left empty means that palette.
func TestFromIDEmptyPalette(t *testing.T) {
	sameSkin(t, "Calm//standard", FromID("Calm//standard"), Calm)
	sameSkin(t, "Classic//compact", FromID("Classic//compact"), Classic)
}

// A bad ID gives nil, never a panic.
func TestFromIDBad(t *testing.T) {
	for _, id := range []string{
		"", "Calm", "Calm/Blue", "Calm/Blue/standard/x", "Nope/Blue/standard",
		"Calm/Pink/standard", "Calm/#zzz/standard", "Calm/Blue/wide",
		"/Blue/standard", "Calm/Blue/", "Calm/Blue/Compact", "calm/Blue/standard",
		"Calm/#/standard", "Calm/#7c3aed0/standard", "Calm/#7c3/standard", "Calm/#-7c3ae/standard", "Calm/#zzzzzz/standard",
		"Calm/Slate/standard", // Classic's palette, not Calm's
		"HighContrast/Blue/standard",
		"Calm/blue/standard",
	} {
		if got := FromID(id); got != nil {
			t.Errorf("FromID(%q) = %s, want nil", id, got.ID())
		}
	}
}

// An app's skin named as one of MyGo's wins over it, the first of extra
// over the next.
func TestFromIDExtraWins(t *testing.T) {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Background = ui.Hex("#fffaf0")
	mine := New("Calm", l, d)
	l.Background = ui.Hex("#000001")
	second := New("Calm", l, d)
	got := FromID("Calm//standard", nil, mine, second)
	if got == nil {
		t.Fatal("FromID of the app's Calm is nil")
	}
	if bg := got.Tokens(false).Background; bg != ui.Hex("#fffaf0") {
		t.Errorf("FromID gives the background %v, want the app's #fffaf0", bg)
	}
	// The app's Calm has no palette named Blue, and MyGo's Calm is not tried.
	if got := FromID("Calm/Blue/standard", mine); got != nil {
		t.Errorf("FromID(Calm/Blue/standard, mine) = %s, want nil", got.ID())
	}
	// Without extra, MyGo's.
	if bg := FromID("Calm//standard").Tokens(false).Background; bg == ui.Hex("#fffaf0") {
		t.Errorf("FromID without extra gives the app's Calm")
	}
}

// A skin made with New and given a palette comes back from its ID when the
// app passes it, its own palette found by its name.
func TestFromIDOwnPalette(t *testing.T) {
	brand := NewPalette("Brand",
		LightFamily(ui.Hex("#7c3aed"), ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9")),
		DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033")))
	s := New("Mine", Calm.Tokens(false), Calm.Tokens(true)).WithPalette(brand).WithDensity(Compact)
	if id := s.ID(); id != "Mine/Brand/compact" {
		t.Fatalf("ID() = %q, want Mine/Brand/compact", id)
	}
	sameSkin(t, s.ID(), FromID(s.ID(), s), s)
	if got := FromID(s.ID()); got != nil {
		t.Errorf("FromID without the app's skin = %s, want nil", got.ID())
	}
}

// FromID finds a skin by name without building the lazy skins before it.
func TestFromIDLeavesLazySkinsUnbuilt(t *testing.T) {
	built := 0
	lazy := &Skin{name: "Lazy", lazy: &lazySkin{build: func() *Skin {
		built++
		return New("Lazy", Calm.Tokens(false), Calm.Tokens(true))
	}}}
	if got := FromID("Calm/Blue/standard", lazy); got == nil || got.Name() != "Calm" {
		t.Fatalf("FromID(Calm/Blue/standard, lazy) = %v", got)
	}
	if built != 0 {
		t.Errorf("FromID of another skin built the lazy one %d times", built)
	}
	if got := FromID("Lazy//standard", lazy); got == nil || built != 1 {
		t.Errorf("FromID(Lazy//standard) = %v, built %d times, want once", got, built)
	}
}
