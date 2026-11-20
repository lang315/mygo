package skinfile

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

// same reports where got differs from want on every getter a file keeps.
func same(t *testing.T, got, want *skin.Skin) {
	t.Helper()
	if got.Name() != want.Name() {
		t.Errorf("Name = %q, want %q", got.Name(), want.Name())
	}
	if got.Tokens(false) != want.Tokens(false) {
		t.Errorf("light tokens\n got %+v\nwant %+v", got.Tokens(false), want.Tokens(false))
	}
	if got.Tokens(true) != want.Tokens(true) {
		t.Errorf("dark tokens\n got %+v\nwant %+v", got.Tokens(true), want.Tokens(true))
	}
	if got.Looks() != want.Looks() {
		t.Errorf("Looks = %+v, want %+v", got.Looks(), want.Looks())
	}
	if got.Density() != want.Density() {
		t.Errorf("Density = %v, want %v", got.Density(), want.Density())
	}
	if paletteName(got) != paletteName(want) {
		t.Errorf("Palette = %q, want %q", paletteName(got), paletteName(want))
	}
}

// paletteName is s's palette's name, "" for none.
func paletteName(s *skin.Skin) string {
	if p := s.Palette(); p != nil {
		return p.Name()
	}
	return ""
}

// brand is a skin that changes a token of each kind, with an alpha.
func brand() *skin.Skin {
	s := skin.Calm.WithPalette(skin.Teal).WithDensity(skin.Compact)
	l, d := s.Tokens(false), s.Tokens(true)
	l.Background, l.Danger.Base, l.RowHeight, l.Title.Weight, l.Font = ui.Hex("#fffaf0"), ui.Hex("#c62828"), 31.5, 700, "Inter"
	d.Background = ui.Hex("#12121280")
	d.ScrollbarStyle.Square, d.CardShadow.Blur = true, 0.1
	d.Chart[3] = ui.Hex("#00ff0001")
	return s.Derive("Brand", l, d).WithLooks(skin.Looks{Button: "rounded-none border border-button"})
}

// roundTrips are the skins every file must keep, written against Calm.
func roundTrips() map[string]*skin.Skin {
	return map[string]*skin.Skin{
		"Calm":           skin.Calm,
		"Teal":           skin.Calm.WithPalette(skin.Teal),
		"accent compact": skin.Calm.WithAccent(ui.Hex("#7c3aed")).WithDensity(skin.Compact),
		"looks": skin.Calm.WithLooks(skin.Looks{
			Button:     "rounded-sm border border-button",
			TabHeader:  "rounded-t border border-b-0 bg-chrome selected:bg-surface",
			GridHeader: "bg-chrome border-r border-strong",
		}),
		// Case 11: the looks of the status components come back.
		"status looks": skin.Calm.WithLooks(skin.Looks{
			Badge:      "rounded-sm border border-status",
			InfoBar:    "border-2 border-status",
			EmptyState: "bg-chrome",
		}),
		"brand": brand(),
	}
}

// TestRoundTrip: Parse(Format(s, Calm)) is s, and so is Parse(Format(s,
// nil)).
func TestRoundTrip(t *testing.T) {
	for name, s := range roundTrips() {
		for _, base := range []*skin.Skin{skin.Calm, nil} {
			t.Run(name, func(t *testing.T) {
				data := Format(s, base)
				got, err := Parse(data)
				if err != nil {
					t.Fatalf("%v\n%s", err, data)
				}
				same(t, got, s)
			})
		}
	}
}

// TestRoundTripOwnBase: the other skins MyGo ships, each against itself
// and against nil.
func TestRoundTripOwnBase(t *testing.T) {
	for _, c := range []struct {
		name    string
		s, base *skin.Skin
	}{
		{"Classic", skin.Classic, skin.Classic},
		{"Classic Slate", skin.Classic.WithPalette(skin.ClassicSlate), skin.Classic},
		{"Classic Standard", skin.Classic.WithDensity(skin.Standard), skin.Classic},
		{"HighContrast", skin.HighContrast, skin.HighContrast},
		{"HighContrast accent", skin.HighContrast.WithAccent(ui.Hex("#7c3aed")), skin.HighContrast},
	} {
		for _, base := range []*skin.Skin{c.base, nil} {
			t.Run(c.name, func(t *testing.T) {
				data := Format(c.s, base)
				if base == nil && !strings.Contains(string(data), "\"base\": \"Calm\",\n") {
					t.Errorf("a nil base wrote no base Calm:\n%s", data)
				}
				got, err := Parse(data)
				if err != nil {
					t.Fatalf("%v\n%s", err, data)
				}
				if base == nil {
					// Against Calm only what the file holds is kept: tokens, looks, density.
					if got.Tokens(false) != c.s.Tokens(false) || got.Tokens(true) != c.s.Tokens(true) ||
						got.Looks() != c.s.Looks() || got.Density() != c.s.Density() {
						t.Errorf("lost through a nil base:\n%s", data)
					}
					return
				}
				same(t, got, c.s)
			})
		}
	}
	if got := string(Format(skin.Classic, skin.Classic)); got != "{\n  \"version\": 1,\n  \"name\": \"Classic\",\n  \"base\": \"Classic\"\n}\n" {
		t.Errorf("Classic against itself:\n%s", got)
	}
}

// TestParseFloorKept: a file based on HighContrast keeps its 7:1 floor.
func TestParseFloorKept(t *testing.T) {
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"HighContrast","light":{"Text":"#6b6b6b"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Check(); err == nil || !strings.Contains(err.Error(), "want 7:1") {
		t.Errorf("Check = %v, want text under 7:1 reported", err)
	}
}

// TestRoundTripNoPalette:a skin of New has no palette, which the tokens
// carry.
func TestRoundTripNoPalette(t *testing.T) {
	l, d := skin.Calm.Tokens(false), skin.Calm.Tokens(true)
	l.Accent.Base = ui.Hex("#abcdef")
	s := skin.New("Mine", l, d)
	for _, base := range []*skin.Skin{skin.Calm, nil} {
		data := Format(s, base)
		if strings.Contains(string(data), `"palette"`) || strings.Contains(string(data), `"accent"`) {
			t.Errorf("a palette written for a skin without one:\n%s", data)
		}
		got, err := Parse(data)
		if err != nil {
			t.Fatalf("%v\n%s", err, data)
		}
		if got.Tokens(false) != l || got.Tokens(true) != d || got.Name() != "Mine" {
			t.Errorf("tokens or name lost:\n%s", data)
		}
	}
}

// TestRoundTripForeignPalette: a palette not of the base's list is carried
// by the tokens, not named.
func TestRoundTripForeignPalette(t *testing.T) {
	p := skin.NewPalette("Brand", skin.LightFamily(ui.Hex("#7c3aed"), ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9")),
		skin.DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033")))
	odd := skin.NewPalette("#nothex", skin.LightFamily(ui.Hex("#7c3aed"), ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9")),
		skin.DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033")))
	for _, s := range []*skin.Skin{skin.Calm.WithPalette(p), skin.Calm.WithPalette(odd)} {
		data := Format(s, skin.Calm)
		got, err := Parse(data)
		if err != nil {
			t.Fatalf("%v\n%s", err, data)
		}
		if got.Tokens(false) != s.Tokens(false) || got.Tokens(true) != s.Tokens(true) {
			t.Errorf("tokens lost:\n%s", data)
		}
	}
}

// TestFormatMinimal: a skin against itself is its name and base alone.
func TestFormatMinimal(t *testing.T) {
	want := "{\n  \"version\": 1,\n  \"name\": \"Calm\",\n  \"base\": \"Calm\"\n}\n"
	if got := string(Format(skin.Calm, skin.Calm)); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// TestFormatDiff: only what differs from the base, after its palette and
// density, in Tokens' order.
func TestFormatDiff(t *testing.T) {
	want := `{
  "version": 1,
  "name": "Brand",
  "base": "Calm",
  "palette": "Teal",
  "density": "compact",
  "light": {
    "Background": "#fffaf0",
    "Danger": {
      "Base": "#c62828"
    },
    "RowHeight": 31.5,
    "Title": {
      "Weight": 700
    },
    "Font": "Inter"
  },
  "dark": {
    "Background": "#12121280",
    "Chart": ["#60a5fa", "#4ade80", "#fb923c", "#00ff0001", "#22d3ee", "#f472b6", "#facc15", "#9ca3af"],
    "CardShadow": {
      "Blur": 0.1
    },
    "ScrollbarStyle": {
      "Square": true
    }
  },
  "looks": {
    "Button": "rounded-none border border-button"
  }
}
`
	if got := string(Format(brand(), skin.Calm)); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// TestFormatLooksDiff: against a base with looks, a kind the skin keeps is
// not written, and one it drops is written "".
func TestFormatLooksDiff(t *testing.T) {
	l, d := skin.Calm.Tokens(false), skin.Calm.Tokens(true)
	base := skin.Calm.WithLooks(skin.Looks{GridHeader: "bg-chrome", Button: "rounded"}).Derive("Looky", l, d)
	bases["Looky"] = base
	t.Cleanup(func() { delete(bases, "Looky") })
	s := base.WithLooks(skin.Looks{Button: "rounded"})
	want := "{\n  \"version\": 1,\n  \"name\": \"Looky\",\n  \"base\": \"Looky\",\n  \"looks\": {\n    \"GridHeader\": \"\"\n  }\n}\n"
	data := Format(s, base)
	if string(data) != want {
		t.Errorf("got\n%s\nwant\n%s", data, want)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	same(t, got, s)
}

// TestFormatBaseNoPalette: against a base of New, with no palette and no
// palette list, the skin's palette is carried by the tokens.
func TestFormatBaseNoPalette(t *testing.T) {
	base := skin.New("Plain", skin.Calm.Tokens(false), skin.Calm.Tokens(true))
	got := string(Format(skin.Calm.WithPalette(skin.Teal), base))
	if strings.Contains(got, `"palette"`) || !strings.Contains(got, "\"Base\": \"#0f766e\"") {
		t.Errorf("want Teal's accent in the tokens and no palette:\n%s", got)
	}
}

// TestFormatAccent: a palette WithAccent made is written as the accent.
func TestFormatAccent(t *testing.T) {
	got := string(Format(skin.Calm.WithAccent(ui.Hex("#7c3aed")), skin.Calm))
	want := "{\n  \"version\": 1,\n  \"name\": \"Calm\",\n  \"base\": \"Calm\",\n  \"accent\": \"#7c3aed\"\n}\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// TestFormatNilBase: with no base, every token in light and dark, and the
// base Calm.
func TestFormatNilBase(t *testing.T) {
	data := string(Format(skin.Calm, nil))
	if !strings.Contains(data, "\"base\": \"Calm\",\n") || !strings.Contains(data, "\"palette\": \"Blue\",\n") {
		t.Errorf("no base Calm or palette Blue:\n%s", data)
	}
	light, dark, ok := strings.Cut(data, "\"dark\": {")
	if !ok {
		t.Fatalf("no dark:\n%s", data)
	}
	ty := reflect.TypeFor[skin.Tokens]()
	for i := range ty.NumField() {
		key := "\n    \"" + ty.Field(i).Name + "\": "
		if !strings.Contains(light, key) {
			t.Errorf("light has no %s", ty.Field(i).Name)
		}
		if !strings.Contains(dark, key) {
			t.Errorf("dark has no %s", ty.Field(i).Name)
		}
	}
	got, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	same(t, got, skin.Calm)
}
