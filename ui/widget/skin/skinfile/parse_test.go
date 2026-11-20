package skinfile

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

// TestParseMinimal: version, name and base alone make the base under the
// file's name.
func TestParseMinimal(t *testing.T) {
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"Calm"}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "X" {
		t.Errorf("Name = %q, want X", s.Name())
	}
	if s.Tokens(false) != skin.Calm.Tokens(false) || s.Tokens(true) != skin.Calm.Tokens(true) {
		t.Error("tokens differ from Calm's")
	}
	if s.Palette().Name() != "Blue" || s.Density() != skin.Standard || s.Looks() != (skin.Looks{}) {
		t.Errorf("palette %s, density %v, looks %+v: want Calm's", s.Palette().Name(), s.Density(), s.Looks())
	}
	if got := len(s.Palettes()); got != 5 {
		t.Errorf("%d palettes, want Calm's 5", got)
	}
}

// TestParseOrder: the palette, then the density, then the tokens, whatever
// order the keys come in.
func TestParseOrder(t *testing.T) {
	s, err := Parse([]byte(`{
  "light": { "Accent": { "Base": "#123456" } },
  "density": "compact",
  "palette": "Teal",
  "base": "Calm",
  "name": "X",
  "version": 1
}`))
	if err != nil {
		t.Fatal(err)
	}
	teal := skin.Calm.WithPalette(skin.Teal)
	l, d := s.Tokens(false), s.Tokens(true)
	if l.Accent.Base != ui.Hex("#123456") {
		t.Errorf("light Accent.Base = %v, want #123456", l.Accent.Base)
	}
	want := teal.Tokens(false)
	want.Accent.Base = ui.Hex("#123456")
	if l != want {
		t.Error("light tokens: want Teal's but Accent.Base")
	}
	if l.Accent.Hover != ui.Hex("#0c665f") {
		t.Errorf("light Accent.Hover = %v, want Teal's #0c665f", l.Accent.Hover)
	}
	if d != teal.Tokens(true) {
		t.Error("dark tokens: want Teal's")
	}
	if s.Palette().Name() != "Teal" || s.Density() != skin.Compact || s.Name() != "X" {
		t.Errorf("palette %s, density %v, name %s: want Teal, Compact, X", s.Palette().Name(), s.Density(), s.Name())
	}
}

// TestParseAccent: an accent makes the palette WithAccent makes.
func TestParseAccent(t *testing.T) {
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"Calm","accent":"#7c3aed"}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.Palette().Name() != "#7c3aed" {
		t.Errorf("Palette = %q, want #7c3aed", s.Palette().Name())
	}
	want := skin.Calm.WithAccent(ui.Hex("#7c3aed"))
	if s.Tokens(false) != want.Tokens(false) || s.Tokens(true) != want.Tokens(true) {
		t.Error("tokens differ from Calm.WithAccent(#7c3aed)'s")
	}
}

// TestParseLooks: a kind of looks replaces the base's; the others stay.
func TestParseLooks(t *testing.T) {
	bases["Looky"] = skin.Calm.WithLooks(skin.Looks{GridHeader: "bg-chrome", Button: "rounded"})
	t.Cleanup(func() { delete(bases, "Looky") })
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"Looky","looks":{"Button":"rounded-none border border-button"}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := skin.Looks{GridHeader: "bg-chrome", Button: "rounded-none border border-button"}
	if s.Looks() != want {
		t.Errorf("Looks = %+v, want %+v", s.Looks(), want)
	}
	// "" takes the base's look away.
	s, err = Parse([]byte(`{"version":1,"name":"X","base":"Looky","looks":{"GridHeader":""}}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := (skin.Looks{Button: "rounded"}); s.Looks() != want {
		t.Errorf("Looks = %+v, want %+v", s.Looks(), want)
	}
}

// TestParseAlpha: #rrggbbaa keeps its alpha.
func TestParseAlpha(t *testing.T) {
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"Calm","light":{"Background":"#11223344"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Tokens(false).Background; got != (ui.Color{R: 0x11, G: 0x22, B: 0x33, A: 0x44}) {
		t.Errorf("Background = %+v, want #11223344", got)
	}
}

// TestParseValues: every kind of value a Tokens field has.
func TestParseValues(t *testing.T) {
	s, err := Parse([]byte(`{"version":1,"name":"X","base":"Calm","dark":{
  "RowHeight": 31.5, "Font": "Inter", "Title": {"Weight": 700},
  "Danger": {"Base": "#c62828"}, "CardShadow": {"Blur": 12},
  "ScrollbarStyle": {"Square": true, "Thumb": "#ff000080"},
  "Chart": ["#000001","#000002","#000003","#000004","#000005","#000006","#000007","#000008"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := skin.Calm.Tokens(true)
	want.RowHeight, want.Font, want.Title.Weight = 31.5, "Inter", 700
	want.Danger.Base, want.CardShadow.Blur = ui.Hex("#c62828"), 12
	want.ScrollbarStyle.Square, want.ScrollbarStyle.Thumb = true, ui.Hex("#ff000080")
	for i := range want.Chart {
		want.Chart[i] = ui.RGB(0, 0, uint8(i+1))
	}
	if got := s.Tokens(true); got != want {
		t.Errorf("dark tokens = %+v,\nwant %+v", got, want)
	}
	if s.Tokens(false) != skin.Calm.Tokens(false) {
		t.Error("light tokens changed")
	}
}

// TestParseErrors: each error names its path and its line.
func TestParseErrors(t *testing.T) {
	head := "{\n  \"version\": 1,\n  \"name\": \"X\",\n  \"base\": \"Calm\",\n"
	inLight := func(member string) string { return head + "  \"light\": {\n    " + member + "\n  }\n}" }
	for _, c := range []struct{ name, in, want string }{
		{"unknown key", inLight(`"Bacground": "#ffffff"`), "light.Bacground: unknown key (line 6)"},
		{"bad color", inLight(`"Background": "red"`), `light.Background: want a color #rrggbb or #rrggbbaa, got "red" (line 6)`},
		{"short color", inLight(`"Background": "#fff"`), `light.Background: want a color #rrggbb or #rrggbbaa, got "#fff" (line 6)`},
		{"color not hex", inLight(`"Background": "#ffffzz"`), `light.Background: want a color #rrggbb or #rrggbbaa, got "#ffffzz" (line 6)`},
		{"color without #", inLight(`"Background": "ffffff"`), `light.Background: want a color #rrggbb or #rrggbbaa, got "ffffff" (line 6)`},
		{"nested path", inLight(`"Danger": { "Base": 3 }`), `light.Danger.Base: want a color #rrggbb or #rrggbbaa, got 3 (line 6)`},
		{"wrong type", inLight(`"RowHeight": "tall"`), `light.RowHeight: want a number, got "tall" (line 6)`},
		{"float32 overflow", inLight(`"RowHeight": 1e300`), `light.RowHeight: want a number a float32 holds, got 1e300 (line 6)`},
		{"int not integer", inLight(`"Title": { "Weight": 1.5 }`), `light.Title.Weight: want an integer, got 1.5 (line 6)`},
		{"bool wrong type", inLight(`"ScrollbarStyle": { "Flush": 1 }`), `light.ScrollbarStyle.Flush: want true or false, got 1 (line 6)`},
		{"string wrong type", inLight(`"Font": null`), `light.Font: want a string, got null (line 6)`},
		{"struct wrong type", inLight(`"Title": 14`), `light.Title: want an object, got 14 (line 6)`},
		{"chart of 7", inLight(`"Chart": ["#000000","#000000","#000000","#000000","#000000","#000000","#000000"]`), "light.Chart: want 8 colors, got 7 (line 6)"},
		{"chart of 9", inLight(`"Chart": ["#000000","#000000","#000000","#000000","#000000","#000000","#000000","#000000","#000000"]`), "light.Chart: want 8 colors, got 9 (line 6)"},
		{"chart element", inLight(`"Chart": ["#000000","#000000","red"]`), `light.Chart[2]: want a color #rrggbb or #rrggbbaa, got "red" (line 6)`},
		{"chart not array", inLight(`"Chart": "#000000"`), `light.Chart: want an array of 8, got "#000000" (line 6)`},
		{"unexported field", inLight(`"wide": 1`), "light.wide: unknown key (line 6)"},
		{"duplicate key", inLight("\"Background\": \"#ffffff\",\n    \"Background\": \"#000000\""), "light.Background: duplicate key (line 7)"},
		{"duplicate top key", head + "  \"name\": \"Y\"\n}", "name: duplicate key (line 5)"},
		{"light not object", head + "  \"light\": 3\n}", "light: want an object, got 3 (line 5)"},
		{"unknown top key", head + "  \"colour\": 1\n}", "colour: unknown key (line 5)"},
		{"unknown base", "{\n  \"version\": 1,\n  \"name\": \"X\",\n  \"base\": \"Foo\"\n}", `base: unknown skin "Foo" (line 4)`},
		{"palette and accent", head + "  \"palette\": \"Teal\",\n  \"accent\": \"#7c3aed\"\n}", "accent: palette and accent together; give one (line 6)"},
		{"accent and palette", head + "  \"accent\": \"#7c3aed\",\n  \"palette\": \"Teal\"\n}", "palette: palette and accent together; give one (line 6)"},
		{"bad accent", head + "  \"accent\": \"violet\"\n}", `accent: want a color #rrggbb or #rrggbbaa, got "violet" (line 5)`},
		{"unknown palette", head + "  \"palette\": \"Pink\"\n}", `palette: no palette "Pink" in Calm; it has Blue, Chàm, Indigo, Ink, Teal (line 5)`},
		{"bad density", head + "  \"density\": \"loose\"\n}", `density: want "standard" or "compact", got "loose" (line 5)`},
		{"version 2", "{\n  \"version\": 2,\n  \"name\": \"X\",\n  \"base\": \"Calm\"\n}", "version: unsupported version 2, want 1 (line 2)"},
		{"version 2 with a new key", "{\n  \"version\": 2,\n  \"tint\": 1\n}", "version: unsupported version 2, want 1 (line 2)"},
		{"version string", "{\n  \"version\": \"1\"\n}", `version: unsupported version "1", want 1 (line 2)`},
		{"missing version", "{\n  \"name\": \"X\",\n  \"base\": \"Calm\"\n}", "version: missing (line 4)"},
		{"missing name", "{\n  \"version\": 1,\n  \"base\": \"Calm\"\n}", "name: missing (line 4)"},
		{"missing base", "{\n  \"version\": 1,\n  \"name\": \"X\"\n}", "base: missing (line 4)"},
		{"name wrong type", "{\n  \"version\": 1,\n  \"name\": 7\n}", "name: want a string, got 7 (line 3)"},
		{"unknown class", head + "  \"looks\": {\n    \"Button\": \"bg-nonsense\"\n  }\n}", `looks.Button: unknown class "bg-nonsense" (line 6)`},
		{"unknown look", head + "  \"looks\": {\n    \"Buton\": \"rounded\"\n  }\n}", "looks.Buton: unknown key (line 6)"},
		{"look wrong type", head + "  \"looks\": {\n    \"Button\": 1\n  }\n}", "looks.Button: want a string, got 1 (line 6)"},
		{"trailing data", head + "  \"density\": \"compact\"\n}\n{}", "trailing data after the object (line 7)"},
		{"malformed JSON on line 3", "{\n  \"version\": 1,\n  \"name\": \"X\",,\n  \"base\": \"Calm\"\n}", "invalid JSON (line 3)"},
		{"malformed in light", inLight(`"Background": "#ffffff",`), "light: invalid JSON (line 7)"},
		{"not an object", "[]", "want an object, got [] (line 1)"},
		{"empty", "", "invalid JSON (line 1)"},
		{"long value capped", inLight(`"Background": "` + strings.Repeat("a", 100) + `"`), `light.Background: want a color #rrggbb or #rrggbbaa, got "` + strings.Repeat("a", 39) + `… (line 6)`},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, err := Parse([]byte(c.in))
			if err == nil {
				t.Fatalf("Parse returned %v and no error", s.Name())
			}
			if s != nil {
				t.Error("Parse returned a skin with its error")
			}
			if err.Error() != c.want {
				t.Errorf("error\n got %s\nwant %s", err, c.want)
			}
		})
	}
}

// TestParseDeep: input deep or big returns an error, without overflowing
// the stack or hanging.
func TestParseDeep(t *testing.T) {
	head := `{"version":1,"name":"X","base":"Calm","light":`
	for name, in := range map[string]string{
		"top":           strings.Repeat("[", 10000),
		"top deeper":    strings.Repeat("[", 100000),
		"in light":      head + strings.Repeat("[", 10000),
		"objects":       head + strings.Repeat(`{"a":`, 20000),
		"in a value":    head + `{"Title":` + strings.Repeat("[", 10000) + "}}",
		"10 MB color":   head + `{"Background":"` + strings.Repeat("a", 10<<20) + `"}}`,
		"10 MB unended": head + `{"Font":"` + strings.Repeat("a", 10<<20),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(in))
			if err == nil {
				t.Fatal("no error")
			}
			if len(err.Error()) > 200 {
				t.Errorf("error of %d bytes", len(err.Error()))
			}
		})
	}
}

// TestDecoderKinds: every field of Tokens is of a kind the decoder reads.
func TestDecoderKinds(t *testing.T) {
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		if ty == colorType {
			return
		}
		switch ty.Kind() {
		case reflect.Struct:
			for i := range ty.NumField() {
				f := ty.Field(i)
				if !f.IsExported() {
					t.Errorf("%s.%s: unexported field outside ui.Color", path, f.Name)
					continue
				}
				walk(f.Type, path+"."+f.Name)
			}
		case reflect.Array:
			walk(ty.Elem(), path+"[]")
		case reflect.Float32, reflect.Int, reflect.String, reflect.Bool:
		default:
			t.Errorf("%s: kind %v the decoder does not read", path, ty.Kind())
		}
	}
	walk(reflect.TypeFor[skin.Tokens](), "Tokens")
}
