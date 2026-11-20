package skin

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDeriveKeeps: Derive takes the name and the tokens it is given, and
// keeps the skin's density, looks, palette, palette list and floor.
func TestDeriveKeeps(t *testing.T) {
	l := Looks{Button: "rounded-none border border-button", GridHeader: "bg-chrome"}
	base := Calm.WithPalette(Teal).WithDensity(Compact).WithLooks(l)
	lt, dt := Calm.Tokens(false), Calm.Tokens(true)
	lt.Background, dt.Background = ui.Hex("#fffaf0"), ui.Hex("#121212")
	lt.RowHeight, dt.Font = 31, "Inter"
	// Text between the 4.5:1 of Calm's floor and 7:1: a skin whose floor is
	// 4.5 passes it, one of 7 would not.
	lt.Text = lt.TextMuted
	if r := contrastOf(lt.Text, lt.Background); r < 4.5 || r >= 7 {
		t.Fatalf("the test's text is %.2f:1 on its background, want between 4.5 and 7", r)
	}

	s := base.Derive("X", lt, dt)
	if s.Name() != "X" {
		t.Errorf("Name = %q, want X", s.Name())
	}
	if s.Tokens(false) != lt {
		t.Error("Tokens(false) are not the tokens given")
	}
	if s.Tokens(true) != dt {
		t.Error("Tokens(true) are not the tokens given")
	}
	if s.Density() != Compact {
		t.Errorf("Density = %v, want Compact", s.Density())
	}
	if s.Looks() != l {
		t.Errorf("Looks = %+v, want %+v", s.Looks(), l)
	}
	if s.parsed == nil || s.parsed[0] == nil {
		t.Error("the looks' Styles are gone")
	}
	if s.Palette() != Teal {
		t.Errorf("Palette = %p, want Teal", s.Palette())
	}
	got, want := s.Palettes(), Calm.Palettes()
	if len(got) != len(want) || len(want) != 5 {
		t.Fatalf("Palettes = %d, want Calm's 5", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Palettes[%d] = %s, want %s", i, got[i].Name(), want[i].Name())
		}
	}
	if err := s.Check(); err != nil {
		t.Errorf("Check of the derived skin: %v; the floor moved", err)
	}
}

// TestDeriveLazy: a skin made at its first use derives from the skin it
// makes, under the new name and tokens.
func TestDeriveLazy(t *testing.T) {
	lt, dt := Classic.Tokens(false), Classic.Tokens(true)
	lt.Background, dt.Background = ui.Hex("#fffaf0"), ui.Hex("#121212")
	s := Classic.Derive("X", lt, dt)
	if s.Name() != "X" {
		t.Errorf("Name = %q, want X", s.Name())
	}
	if s.Tokens(false) != lt || s.Tokens(true) != dt {
		t.Error("tokens are not the tokens given")
	}
	if s.Density() != Compact {
		t.Errorf("Density = %v, want Classic's Compact", s.Density())
	}
	if s.Looks() != classicLooks || s.Looks().TabHeader == "" {
		t.Errorf("Looks = %+v, want Classic's", s.Looks())
	}
	if got := s.Palettes(); len(got) != 2 || got[0] != ClassicIndigo || got[1] != ClassicSlate {
		t.Errorf("Palettes = %v, want ClassicIndigo, ClassicSlate", got)
	}
	if Classic.Name() != "Classic" || Classic.Tokens(false).Background == lt.Background {
		t.Error("Derive changed Classic")
	}
}

// TestDeriveKeepsFloor: a skin derived from HighContrast keeps its 7:1
// floor: text at 5:1, which Calm's floor passes, is reported.
func TestDeriveKeepsFloor(t *testing.T) {
	lt, dt := HighContrast.Tokens(false), HighContrast.Tokens(true)
	lt.Text = ui.Hex("#6b6b6b")
	if r := contrastOf(lt.Text, lt.Background); r < 4.5 || r >= 7 {
		t.Fatalf("the test's text is %.2f:1 on its background, want between 4.5 and 7", r)
	}
	err := HighContrast.Derive("X", lt, dt).Check()
	if err == nil || !strings.Contains(err.Error(), "light: Text on Background") || !strings.Contains(err.Error(), "want 7:1") {
		t.Errorf("Check = %v, want light Text on Background under 7:1", err)
	}
}

// TestDeriveReceiverUnchanged:Derive leaves its skin as it was.
func TestDeriveReceiverUnchanged(t *testing.T) {
	l0, d0, name := Calm.Tokens(false), Calm.Tokens(true), Calm.Name()
	lt, dt := l0, d0
	lt.Background, dt.Background = ui.Hex("#fffaf0"), ui.Hex("#121212")
	Calm.Derive("X", lt, dt)
	if Calm.Tokens(false) != l0 || Calm.Tokens(true) != d0 {
		t.Error("Derive changed Calm's tokens")
	}
	if Calm.Name() != name {
		t.Errorf("Calm's name is now %q", Calm.Name())
	}
}
