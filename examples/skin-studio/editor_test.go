package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
	"github.com/egoist/mygo/ui/widget/skin/skinfile"
)

// sameSkin fails the test unless a and b are equal on name, both token sets,
// looks, density and palette name.
func sameSkin(t *testing.T, got, want *skin.Skin) {
	t.Helper()
	if got.Name() != want.Name() {
		t.Errorf("name = %q, want %q", got.Name(), want.Name())
	}
	for _, dark := range []bool{false, true} {
		g, w := got.Tokens(dark), want.Tokens(dark)
		reported := false
		for _, p := range tokenPaths() {
			gv, _ := leafAt(reflect.ValueOf(&g).Elem(), p)
			wv, _ := leafAt(reflect.ValueOf(&w).Elem(), p)
			if gv.Interface() != wv.Interface() {
				t.Errorf("tokens(dark=%v) %s = %v, want %v", dark, p, leafText(gv), leafText(wv))
				reported = true
			}
		}
		if g != w && !reported {
			t.Errorf("tokens(dark=%v) differ in a field tokenPaths does not list", dark)
		}
	}
	if got.Looks() != want.Looks() {
		t.Errorf("looks = %+v, want %+v", got.Looks(), want.Looks())
	}
	if got.Density() != want.Density() {
		t.Errorf("density = %v, want %v", got.Density(), want.Density())
	}
	gp, wp := "", ""
	if got.Palette() != nil {
		gp = got.Palette().Name()
	}
	if want.Palette() != nil {
		wp = want.Palette().Name()
	}
	if gp != wp {
		t.Errorf("palette = %q, want %q", gp, wp)
	}
}

// A fresh editor builds its base: Calm with Blue, Classic Compact with
// Indigo, HighContrast with no palette.
func TestEditorBuildIsTheBase(t *testing.T) {
	for key, s := range skins {
		sameSkin(t, newEditor(key).build(), s)
	}
}

// An edited token, a look, the palette and the density reach the skin built,
// and Classic's floor stays: a text between 4.5:1 and 7:1 passes it.
func TestEditorBuild(t *testing.T) {
	e := newEditor("classic")
	e.palette = skin.ClassicSlate.Name()
	e.density = skin.Standard
	e.name = "Mine"
	if err := e.set(false, "Accent.Base", "#7c3aed"); err != nil {
		t.Fatal(err)
	}
	if err := e.set(false, "Text", "#5a5a5a"); err != nil { // 6.9:1 on white
		t.Fatal(err)
	}
	if err := e.setLook("Button", "rounded-none border border-button"); err != nil {
		t.Fatal(err)
	}
	s := e.build()
	if s.Name() != "Mine" {
		t.Errorf("Name = %q, want Mine", s.Name())
	}
	if got, want := s.Tokens(false).Accent.Base, ui.Hex("#7c3aed"); got != want {
		t.Errorf("light Accent.Base = %v, want %v", got, want)
	}
	if got := s.Tokens(true).Accent; got != skin.Classic.WithPalette(skin.ClassicSlate).Tokens(true).Accent {
		t.Errorf("dark Accent = %+v, want Slate's: an edit of light leaked into dark", got)
	}
	if s.Palette().Name() != "Slate" {
		t.Errorf("Palette = %q, want Slate", s.Palette().Name())
	}
	if s.Density() != skin.Standard {
		t.Errorf("Density = %v, want Standard", s.Density())
	}
	if got := s.Looks().Button; got != "rounded-none border border-button" {
		t.Errorf("Looks.Button = %q", got)
	}
	if got, want := s.Looks().GridHeader, skin.Classic.Looks().GridHeader; got != want || want == "" {
		t.Errorf("Looks.GridHeader = %q, want Classic's %q", got, want)
	}
	if err := s.Check(); err != nil {
		t.Errorf("Check: %v; Classic's floor moved", err)
	}
}

// build keeps the floor of the base: the same text that Classic passes fails
// HighContrast's 7:1.
func TestEditorBuildKeepsFloor(t *testing.T) {
	e := newEditor("highcontrast")
	if err := e.set(false, "Text", "#5a5a5a"); err != nil {
		t.Fatal(err)
	}
	err := e.build().Check()
	if err == nil || !strings.Contains(err.Error(), "Text") {
		t.Errorf("Check = %v, want an error on a Text pair: HighContrast's floor is 7:1", err)
	}
}

// An accent makes the palette, named by its hex, and wins over a palette.
func TestEditorAccent(t *testing.T) {
	e := newEditor("calm")
	e.palette = "Teal"
	if err := e.setAccent("#7c3aed"); err != nil {
		t.Fatal(err)
	}
	if got := e.build().Palette().Name(); got != "#7c3aed" {
		t.Errorf("Palette = %q, want #7c3aed", got)
	}
	if err := e.setAccent("#zzz"); err == nil || !strings.Contains(err.Error(), "#zzz") {
		t.Errorf("setAccent(#zzz) = %v, want an error naming it", err)
	}
	if err := e.setAccent("#7c3aed80"); err == nil || !strings.Contains(err.Error(), "#7c3aed80") {
		t.Errorf("setAccent(#7c3aed80) = %v, want an error: an accent is opaque", err)
	}
	if got := e.accent; got != "#7c3aed" {
		t.Errorf("accent = %q after a refused one, want #7c3aed", got)
	}
	if err := e.setAccent(""); err != nil {
		t.Fatal(err)
	}
	if got := e.build().Palette().Name(); got != "Teal" {
		t.Errorf("Palette = %q after the accent is cleared, want Teal", got)
	}
}

// A refused value gives an error naming the path and leaves the model as it
// was; the bounds themselves are accepted.
func TestEditorSetErrors(t *testing.T) {
	cases := []struct {
		name, path, value string
	}{
		{"short hex", "Text", "#zzz"},
		{"not hex digits", "Text", "#gggggg"},
		{"no hash", "Text", "7c3aed"},
		{"7 digits", "Text", "#7c3aed1"},
		{"empty color", "Text", ""},
		{"unknown path", "Nope", "#ffffff"},
		{"a struct, not a leaf", "Accent", "#ffffff"},
		{"unknown field of a struct", "Accent.Nope", "#ffffff"},
		{"past a leaf", "Text.R", "1"},
		{"chart out of range", "Chart.8", "#ffffff"},
		{"bad number", "RowHeight", "tall"},
		{"empty number", "RowHeight", ""},
		{"size under 0", "RowHeight", "-1"},
		{"size over 400", "RowHeight", "401"},
		{"not a number", "RowHeight", "NaN"},
		{"weight under 100", "Label.Weight", "99"},
		{"weight over 900", "Label.Weight", "901"},
		{"weight not whole", "Label.Weight", "400.5"},
		{"bad bool", "ScrollbarStyle.Square", "maybe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEditor("calm")
			before := e.build()
			err := e.set(false, c.path, c.value)
			if err == nil {
				t.Fatalf("set(%q, %q) = nil, want an error", c.path, c.value)
			}
			if !strings.Contains(err.Error(), c.path) {
				t.Errorf("error %q does not name the path %q", err, c.path)
			}
			sameSkin(t, e.build(), before)
			sameSkin(t, e.build(), skin.Calm)
		})
	}
	for _, c := range []struct{ path, value string }{
		{"RowHeight", "0"}, {"RowHeight", "400"}, {"Label.Weight", "100"}, {"Label.Weight", "900"},
		{"Text", "#AABBCC"}, {"Text", "#aabbcc80"},
	} {
		e := newEditor("calm")
		if err := e.set(true, c.path, c.value); err != nil {
			t.Errorf("set(%q, %q) = %v, want it accepted", c.path, c.value, err)
		}
	}
}

// A look needs classes tw knows; "" takes the look off; an unknown kind is
// refused.
func TestEditorSetLook(t *testing.T) {
	e := newEditor("calm")
	if err := e.setLook("Button", "bg-nonsense"); err == nil || !strings.Contains(err.Error(), "bg-nonsense") {
		t.Errorf("setLook(bg-nonsense) = %v, want tw's error", err)
	}
	if e.build().Looks().Button != "" {
		t.Error("a refused look was applied")
	}
	if err := e.setLook("Nope", "border"); err == nil || !strings.Contains(err.Error(), "Nope") {
		t.Errorf("setLook(Nope) = %v, want an error naming the kind", err)
	}
	if err := e.setLook("Button", "border"); err != nil {
		t.Fatal(err)
	}
	if err := e.setLook("Button", ""); err != nil {
		t.Fatal(err)
	}
	if got := e.build().Looks().Button; got != "" {
		t.Errorf("Looks.Button = %q after clearing, want none", got)
	}
}

// Review focus 2: Save, then Open gives a skin equal on name, both token
// sets, looks, density and palette or accent; a file saved against Classic
// opens with base Classic.
func TestEditorSaveOpen(t *testing.T) {
	calm := newEditor("calm")
	calm.name = "Brand"
	calm.density = skin.Compact
	if err := calm.setAccent("#7c3aed"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		dark        bool
		path, value string
	}{{false, "Background", "#fffaf0"}, {true, "Danger.Base", "#ff5555"}, {false, "RowHeight", "28.5"}, {true, "Font", "Inter"}, {false, "Chart.3", "#11223344"}} {
		if err := calm.set(c.dark, c.path, c.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := calm.setLook("TabHeader", "px-4 selected:bg-surface"); err != nil {
		t.Fatal(err)
	}

	classic := newEditor("classic")
	classic.palette = skin.ClassicSlate.Name()
	classic.density = skin.Standard
	if err := classic.set(true, "Accent.Base", "#123456"); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name string
		e    *editor
		base string
	}{{"calm", calm, "calm"}, {"classic", classic, "classic"}} {
		t.Run(c.name, func(t *testing.T) {
			data := c.e.save()
			if !strings.Contains(string(data), `"base": "`+skins[c.base].Name()+`"`) {
				t.Errorf("the file is not against %s:\n%s", c.base, data)
			}
			got := newEditor("highcontrast") // anything but the base
			if err := got.open(data); err != nil {
				t.Fatalf("open: %v\n%s", err, data)
			}
			if got.base != c.base {
				t.Errorf("base = %q, want %q", got.base, c.base)
			}
			sameSkin(t, got.build(), c.e.build())
			if !reflect.DeepEqual(got.save(), data) {
				t.Errorf("saved again:\n%s\nwant\n%s", got.save(), data)
			}
		})
	}
}

// Saved against its base, a skin nothing changed has no token, palette or
// density in its file.
func TestEditorSaveWritesTheDifference(t *testing.T) {
	want := "{\n  \"version\": 1,\n  \"name\": \"Classic\",\n  \"base\": \"Classic\"\n}\n"
	if got := string(newEditor("classic").save()); got != want {
		t.Errorf("save =\n%s\nwant\n%s", got, want)
	}
	e := newEditor("calm")
	if err := e.set(false, "Background", "#fffaf0"); err != nil {
		t.Fatal(err)
	}
	want = "{\n  \"version\": 1,\n  \"name\": \"Calm\",\n  \"base\": \"Calm\",\n  \"light\": {\n    \"Background\": \"#fffaf0\"\n  }\n}\n"
	if got := string(e.save()); got != want {
		t.Errorf("save =\n%s\nwant\n%s", got, want)
	}
}

// A file made by skinfile.Format against no base, or that is not a skin, is
// read or refused whole.
func TestEditorOpen(t *testing.T) {
	e := newEditor("classic")
	if err := e.set(false, "Text", "#5a5a5a"); err != nil {
		t.Fatal(err)
	}
	before := e.build()
	for _, c := range []struct{ name, data string }{
		{"empty", ""},
		{"not json", "{"},
		{"unknown base", `{"version": 1, "name": "X", "base": "Nope"}`},
		{"unknown key", `{"version": 1, "name": "X", "base": "Calm", "nope": 1}`},
		{"bad color", `{"version": 1, "name": "X", "base": "Calm", "light": {"Text": "red"}}`},
		{"bad class", `{"version": 1, "name": "X", "base": "Calm", "looks": {"Button": "bg-nonsense"}}`},
	} {
		if err := e.open([]byte(c.data)); err == nil {
			t.Errorf("open(%s) = nil, want an error", c.name)
		}
		sameSkin(t, e.build(), before)
	}

	everything := skin.Calm.WithPalette(skin.Teal).WithDensity(skin.Compact)
	got := newEditor("classic")
	if err := got.open(skinfile.Format(everything, nil)); err != nil {
		t.Fatal(err)
	}
	sameSkin(t, got.build(), everything)
	if got.base != "calm" {
		t.Errorf("base = %q, want calm", got.base)
	}
}

// reset puts a base back as it ships: its palette, density, looks and name,
// no token changed.
func TestEditorReset(t *testing.T) {
	e := newEditor("calm")
	e.palette, e.density, e.name = "Teal", skin.Compact, "X"
	if err := e.setAccent("#7c3aed"); err != nil {
		t.Fatal(err)
	}
	if err := e.set(false, "Text", "#000000"); err != nil {
		t.Fatal(err)
	}
	if err := e.setLook("Button", "border"); err != nil {
		t.Fatal(err)
	}
	for key, s := range skins {
		e.reset(key)
		sameSkin(t, e.build(), s)
		if e.base != key {
			t.Errorf("base = %q, want %q", e.base, key)
		}
	}
}

// "none" is the tool strip's no skin: there is nothing to edit, so the
// editor takes Calm.
func TestEditorResetNone(t *testing.T) {
	e := newEditor("none")
	if e.base != "calm" {
		t.Errorf("base = %q, want calm", e.base)
	}
	sameSkin(t, e.build(), skin.Calm)
	e.reset("highcontrast")
	e.reset("none")
	sameSkin(t, e.build(), skin.Calm)
}

// A file may hold what the panel's boxes do not take (a size of 450, a
// weight of 50, a fractional one): open keeps it, and a skin built is the
// skin Parse gave.
func TestEditorOpenKeepsAFilesValues(t *testing.T) {
	data := `{"version": 1, "name": "Wide", "base": "HighContrast",
	  "light": {"RowHeight": 450, "Caption": {"Weight": 50}, "Label": {"Weight": 450}, "FontSize": 12.25},
	  "dark": {"Icon": 0}}`
	want, err := skinfile.Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	e := newEditor("calm")
	if err := e.open([]byte(data)); err != nil {
		t.Fatal(err)
	}
	sameSkin(t, e.build(), want)
	if e.base != "highcontrast" {
		t.Errorf("base = %q, want highcontrast", e.base)
	}
	for _, c := range []struct{ path, want string }{{"RowHeight", "450"}, {"Caption.Weight", "50"}, {"FontSize", "12.25"}} {
		if got := tokenText(e.build().Tokens(false), c.path); got != c.want {
			t.Errorf("%s = %q, want %q", c.path, got, c.want)
		}
	}
}

// Every base opens as itself.
func TestEditorOpenEveryBase(t *testing.T) {
	for key, s := range skins {
		e := newEditor("none")
		if err := e.open(skinfile.Format(s, s)); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if e.base != key {
			t.Errorf("base = %q, want %q", e.base, key)
		}
		sameSkin(t, e.build(), s)
	}
}

// assign refuses a field of a kind the editor has no text for, so a new kind
// in Tokens fails TestEditorAllFields instead of being dropped.
func TestEditorAssignUnknownKind(t *testing.T) {
	var s struct{ M map[int]int }
	if err := assign(reflect.ValueOf(&s).Elem().Field(0), "M", "x", true); err == nil || !strings.Contains(err.Error(), "M") {
		t.Errorf("assign(map) = %v, want an error naming M", err)
	}
}

// leaves lists the paths of the leaves of a Tokens by reflection, as the
// test reads them, not as the editor does.
func leaves(v reflect.Value, prefix string) []string {
	var out []string
	switch {
	case v.Type() == reflect.TypeFor[ui.Color]():
		return []string{prefix}
	case v.Kind() == reflect.Struct:
		for i := range v.NumField() {
			name := v.Type().Field(i).Name
			if prefix != "" {
				name = prefix + "." + name
			}
			out = append(out, leaves(v.Field(i), name)...)
		}
	case v.Kind() == reflect.Array:
		for i := range v.Len() {
			out = append(out, leaves(v.Index(i), prefix+"."+string(rune('0'+i)))...)
		}
	default:
		return []string{prefix}
	}
	return out
}

// Every leaf of skin.Tokens, in every built-in skin and both appearances,
// takes its own value back through set: no field is missing, and a new field
// of another kind fails here.
func TestEditorAllFields(t *testing.T) {
	paths := leaves(reflect.ValueOf(skin.Calm.Tokens(false)), "")
	if len(paths) != 107 {
		t.Errorf("Tokens has %d leaves, the editor was written for 107: check the new ones", len(paths))
	}
	if got := tokenPaths(); !reflect.DeepEqual(got, paths) {
		t.Errorf("tokenPaths = %v, want the leaves %v", got, paths)
	}
	for key := range skins {
		for _, dark := range []bool{false, true} {
			e := newEditor(key)
			tk := e.build().Tokens(dark)
			for _, p := range paths {
				v := tokenText(tk, p)
				if v == "" && !strings.HasSuffix(p, "Font") {
					t.Fatalf("%s dark=%v: tokenText(%q) is empty", key, dark, p)
				}
				if err := e.set(dark, p, v); err != nil {
					t.Errorf("%s dark=%v: set(%q, %q): %v", key, dark, p, v, err)
				}
			}
			sameSkin(t, e.build(), skins[key])
		}
	}
}
