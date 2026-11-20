// Package skinfile reads and writes skins as JSON files: a file names the
// skin it starts from, its palette or accent color, its density, the tokens
// it changes in light and dark, and its looks.
//
//	{
//	  "version": 1,
//	  "name": "Brand",
//	  "base": "Calm",
//	  "accent": "#7c3aed",
//	  "density": "compact",
//	  "light": { "Background": "#fffaf0", "Font": "Inter", "Danger": { "Base": "#c62828" } },
//	  "dark": { "Background": "#121212" },
//	  "looks": { "Button": "rounded-none border border-button" }
//	}
//
// An app embeds the file and turns the skin on at startup; mygo dev
// relaunches the app on each edit of an embedded file, so the skin shows as
// it is saved:
//
//	//go:embed brand.json
//	var brandJSON []byte
//
//	func must(s *skin.Skin, err error) *skin.Skin {
//		if err != nil {
//			log.Fatal(err)
//		}
//		return s
//	}
//
//	skin.Use(must(skinfile.Parse(brandJSON)))
//
// An app may also read a file at run time and pass its bytes to Parse.
//
// The keys of light and dark are the names of skin.Tokens' fields; a field
// that is a struct (a Family, a Type, a Shadow, the ScrollbarStyle) is an
// object of its own fields, each optional. Colors are "#rrggbb" or
// "#rrggbbaa", sizes numbers of DIPs, and Chart an array of all 8 colors.
// Looks are tw classes, a kind of look replacing the base's whole.
//
// An app that imports package skin but not this one links no JSON reader.
package skinfile

import (
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/jsonlite"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
	"github.com/egoist/mygo/ui/widget/tw"
)

// bases are the skins a file can start from, by name.
var bases = map[string]*skin.Skin{"Calm": skin.Calm}

// member is a key of the file's object: its value's text (a string's
// content, an object's JSON) and offset.
type member struct {
	text []byte
	off  int
	set  bool
}

// Parse returns the skin of the file data. It is applied in this order: the
// base skin, then its palette or the accent, then the density, then the
// tokens of light and dark, then the looks. An unknown key, a value of the
// wrong kind or an unknown class is an error naming its path and line, as
//
//	light.Background: want a color #rrggbb or #rrggbbaa, got "red" (line 6)
//
// Parse does not check the skin's contrast: Skin.Check does.
func Parse(data []byte) (*skin.Skin, error) {
	d := &decoder{file: data, r: jsonlite.NewReader(data)}
	var version bool
	var accentColor ui.Color
	var name, base, palette, accent, density, light, dark, looks member
	strs := map[string]*member{"name": &name, "base": &base, "palette": &palette, "density": &density}
	objs := map[string]*member{"light": &light, "dark": &dark, "looks": &looks}
	err := d.object("", func(key, path string, off int) error {
		d.r.Peek()
		voff := d.r.Offset()
		switch m, o := strs[key], objs[key]; {
		case key == "version":
			v, off, err := d.read(path)
			if err != nil {
				return err
			}
			if string(v) != "1" {
				return d.fail(path, off, "unsupported version "+cut(string(v))+", want 1")
			}
			version = true
		case key == "accent":
			c, err := d.color(path)
			accentColor, accent = c, member{off: voff, set: true}
			return err
		case m != nil:
			s, err := d.str(path)
			*m = member{[]byte(s), voff, true}
			return err
		case o != nil:
			v, off, err := d.read(path)
			*o = member{v, off, true}
			return err
		default:
			return d.fail(path, off, "unknown key")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	end := d.r.Offset() - 1 // the object's }
	if err := d.r.End(); err != nil {
		return nil, d.fail("", d.r.Offset(), "trailing data after the object")
	}
	switch {
	case !version:
		return nil, d.fail("version", end, "missing")
	case !name.set:
		return nil, d.fail("name", end, "missing")
	case !base.set:
		return nil, d.fail("base", end, "missing")
	}

	s := bases[string(base.text)]
	if s == nil {
		return nil, d.fail("base", base.off, "unknown skin "+quote(string(base.text)))
	}
	if palette.set && accent.set {
		key, off := "accent", accent.off
		if palette.off > off {
			key, off = "palette", palette.off
		}
		return nil, d.fail(key, off, "palette and accent together; give one")
	}
	if palette.set {
		p := findPalette(s, string(palette.text))
		if p == nil {
			var names []string
			for _, p := range s.Palettes() {
				names = append(names, p.Name())
			}
			return nil, d.fail("palette", palette.off, "no palette "+quote(string(palette.text))+" in "+s.Name()+"; it has "+strings.Join(names, ", "))
		}
		s = s.WithPalette(p)
	}
	if accent.set {
		s = s.WithAccent(accentColor)
	}
	if density.set {
		switch string(density.text) {
		case "standard":
			s = s.WithDensity(skin.Standard)
		case "compact":
			s = s.WithDensity(skin.Compact)
		default:
			return nil, d.fail("density", density.off, `want "standard" or "compact", got `+quote(string(density.text)))
		}
	}

	lt, dt := s.Tokens(false), s.Tokens(true)
	if light.set {
		if err := d.at(light).value(reflect.ValueOf(&lt).Elem(), "light"); err != nil {
			return nil, err
		}
	}
	if dark.set {
		if err := d.at(dark).value(reflect.ValueOf(&dt).Elem(), "dark"); err != nil {
			return nil, err
		}
	}
	s = s.Derive(string(name.text), lt, dt)

	if looks.set {
		l := s.Looks()
		if err := d.at(looks).looks(&l); err != nil {
			return nil, err
		}
		s = s.WithLooks(l)
	}
	return s, nil
}

// at returns a decoder of the object m.
func (d *decoder) at(m member) *decoder {
	return &decoder{file: d.file, base: m.off, r: jsonlite.NewReader(m.text)}
}

// looks reads the object of looks into l: each kind's classes, which tw
// must know.
func (d *decoder) looks(l *skin.Looks) error {
	v := reflect.ValueOf(l).Elem()
	return d.object("looks", func(key, path string, off int) error {
		f, ok := v.Type().FieldByName(key)
		if !ok || !f.IsExported() || len(f.Index) != 1 {
			return d.fail(path, off, "unknown key")
		}
		d.r.Peek()
		voff := d.r.Offset()
		s, err := d.str(path)
		if err != nil {
			return err
		}
		if _, err := tw.Parse(s); err != nil {
			return d.fail(path, voff, strings.TrimPrefix(err.Error(), "tw: "))
		}
		v.Field(f.Index[0]).SetString(s)
		return nil
	})
}

// findPalette returns s's palette of the name, or nil.
func findPalette(s *skin.Skin, name string) *skin.Palette {
	for _, p := range s.Palettes() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// quote quotes s as JSON, cut short.
func quote(s string) string { return cut(string(jsonlite.AppendQuote(nil, s))) }

// cut cuts s to 40 bytes, for an error.
func cut(s string) string {
	if len(s) <= 40 {
		return s
	}
	n := 40
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
