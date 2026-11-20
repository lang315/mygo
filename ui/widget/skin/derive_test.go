package skin

import (
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

// TestDeriveReceiverUnchanged: Derive leaves its skin as it was.
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
