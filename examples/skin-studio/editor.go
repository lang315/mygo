package main

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
	"github.com/egoist/mygo/ui/widget/skin/skinfile"
	"github.com/egoist/mygo/ui/widget/tw"
)

// editor is the skin being edited: a base skin of skins, its palette or an
// accent, a density, a name, the tokens changed in each appearance (by path,
// as "Accent.Base", with the value as text) and the looks. build makes the
// skin of it, as skinfile.Parse does of a file. The editor is not safe for
// concurrent use: the studio drives it from its main thread.
type editor struct {
	base    string // a key of skins
	palette string // a palette of the base, "" for its own
	accent  string // "#rrggbb", which wins over palette; "" for none
	density skin.Density
	name    string
	changed [2]map[string]string // light, dark: path to value
	looks   skin.Looks
}

// newEditor returns an editor of the base as it ships.
func newEditor(base string) *editor {
	e := &editor{}
	e.reset(base)
	return e
}

// reset puts the editor back to the base as it ships: its palette, density,
// looks and name, with no token changed. A base that is no skin ("none")
// has nothing to edit: the editor takes Calm.
func (e *editor) reset(base string) {
	s := skins[base]
	if s == nil {
		base, s = "calm", skin.Calm
	}
	e.base, e.palette, e.accent, e.density, e.name = base, "", "", s.Density(), s.Name()
	if p := s.Palette(); p != nil {
		e.palette = p.Name()
	}
	e.changed = [2]map[string]string{{}, {}}
	e.looks = s.Looks()
}

// derived returns the skin before any token is changed: the base with the
// palette or the accent and the density.
func (e *editor) derived() *skin.Skin {
	s := skins[e.base]
	if c, ok := parseColor(e.accent); ok {
		s = s.WithAccent(c)
	} else {
		for _, p := range s.Palettes() {
			if p.Name() == e.palette {
				s = s.WithPalette(p)
			}
		}
	}
	return s.WithDensity(e.density)
}

// build returns the skin being edited: derived, then Derive with the name and
// the changed tokens, which keeps the base's floor, then the looks.
func (e *editor) build() *skin.Skin {
	d := e.derived()
	lt, dt := d.Tokens(false), d.Tokens(true)
	for i, t := range []*skin.Tokens{&lt, &dt} {
		for path, text := range e.changed[i] {
			// set and open keep each value as the leaf reads it, and a
			// file's values may lie outside the range the user types in.
			v, _ := leafAt(reflect.ValueOf(t).Elem(), path)
			_ = assign(v, path, text, false)
		}
	}
	return d.Derive(e.name, lt, dt).WithLooks(e.looks)
}

// tokenText returns the token at path in t as text: "#rrggbb", a number, a
// name or true; "" for a path that is no leaf.
func tokenText(t skin.Tokens, path string) string {
	v, err := leafAt(reflect.ValueOf(&t).Elem(), path)
	if err != nil {
		return ""
	}
	return leafText(v)
}

// set changes the token at path in the appearance, the value as get gives it.
// A value it refuses (the error names the path) leaves the editor as it was.
func (e *editor) set(dark bool, path, value string) error {
	var t skin.Tokens
	v, err := leafAt(reflect.ValueOf(&t).Elem(), path)
	if err != nil {
		return err
	}
	if err := assign(v, path, value, true); err != nil {
		return err
	}
	e.changed[b2i(dark)][path] = leafText(v)
	return nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// setLook sets the look of a kind of control, one of the names of
// skin.Looks' fields, to Tailwind classes tw knows; "" takes it off.
func (e *editor) setLook(kind, classes string) error {
	f := reflect.ValueOf(&e.looks).Elem().FieldByName(kind)
	if !f.IsValid() {
		return fmt.Errorf("%s: not a kind of look", kind)
	}
	if _, err := tw.Parse(classes); err != nil {
		return fmt.Errorf("%s: %s", kind, strings.TrimPrefix(err.Error(), "tw: "))
	}
	f.SetString(classes)
	return nil
}

// setAccent makes the skin's palette from one color, "#rrggbb"; "" goes back
// to the palette.
func (e *editor) setAccent(hex string) error {
	if hex == "" {
		e.accent = ""
		return nil
	}
	c, ok := parseColor(hex)
	if !ok || c.A != 255 {
		return fmt.Errorf("accent: want a color #rrggbb, got %q", hex)
	}
	e.accent = hexText(c)
	return nil
}

// save returns the skin file of the edit: what it changes from its base.
func (e *editor) save() []byte { return skinfile.Format(e.build(), skins[e.base]) }

// open reads a skin file into the editor, against the base it names; a file
// it refuses leaves the editor as it was.
func (e *editor) open(data []byte) error {
	s, err := skinfile.Parse(data)
	if err != nil {
		return err
	}
	// Parse read the file, so it is an object with a "base" of one of the
	// skins of skins.
	var base string
	r := jsonlite.NewReader(data)
	_ = r.Object(func(key string) error {
		if key != "base" {
			return r.Skip()
		}
		s, err := r.String()
		base = strings.ToLower(s)
		return err
	})
	n := &editor{base: base, density: s.Density(), name: s.Name(), looks: s.Looks()}
	if p := s.Palette(); p != nil {
		if c, ok := parseColor(p.Name()); ok {
			n.accent = hexText(c)
		} else {
			n.palette = p.Name()
		}
	}
	d := n.derived()
	for i := range 2 {
		n.changed[i] = map[string]string{}
		got, was := s.Tokens(i == 1), d.Tokens(i == 1)
		for _, path := range tokenPaths() {
			g, _ := leafAt(reflect.ValueOf(&got).Elem(), path)
			w, _ := leafAt(reflect.ValueOf(&was).Elem(), path)
			if g.Interface() != w.Interface() {
				n.changed[i][path] = leafText(g)
			}
		}
	}
	*e = *n
	return nil
}

// tokenPaths returns the path of every leaf of skin.Tokens, in the order of
// its fields: a color, a number, a name or a flag, as "Accent.Base" or
// "Chart.3".
func tokenPaths() []string {
	var out []string
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		switch {
		case t == colorType:
			out = append(out, prefix)
		case t.Kind() == reflect.Struct:
			for i := range t.NumField() {
				name := t.Field(i).Name
				if prefix != "" {
					name = prefix + "." + name
				}
				walk(t.Field(i).Type, name)
			}
		case t.Kind() == reflect.Array:
			for i := range t.Len() {
				walk(t.Elem(), prefix+"."+strconv.Itoa(i))
			}
		default:
			out = append(out, prefix)
		}
	}
	walk(reflect.TypeFor[skin.Tokens](), "")
	return out
}

var colorType = reflect.TypeFor[ui.Color]()

// leafAt returns the leaf of the struct v at path, an error naming the path
// for one that is not a leaf of it.
func leafAt(v reflect.Value, path string) (reflect.Value, error) {
	for _, part := range strings.Split(path, ".") {
		switch {
		case v.Type() == colorType:
			return v, fmt.Errorf("%s: no field %q: a color is one value", path, part)
		case v.Kind() == reflect.Struct:
			v = v.FieldByName(part)
		case v.Kind() == reflect.Array:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= v.Len() {
				return v, fmt.Errorf("%s: no element %q", path, part)
			}
			v = v.Index(i)
		default:
			return v, fmt.Errorf("%s: no field %q: a %s is one value", path, part, v.Kind())
		}
		if !v.IsValid() {
			return v, fmt.Errorf("%s: no field %q", path, part)
		}
	}
	if v.Type() != colorType && (v.Kind() == reflect.Struct || v.Kind() == reflect.Array) {
		return v, fmt.Errorf("%s: a group of fields, not one: name one of them", path)
	}
	return v, nil
}

// assign sets the settable leaf v, whose path is path, to the text value: a
// color is #rrggbb or #rrggbbaa, a size a number, a weight a whole number,
// a flag true or false. A user's value (typed) must also be a size from 0 to
// 400 and a weight from 100 to 900, the ranges of the edit panel's number
// boxes; a file's may lie outside them.
func assign(v reflect.Value, path, value string, typed bool) error {
	switch {
	case v.Type() == colorType:
		c, ok := parseColor(value)
		if !ok {
			return fmt.Errorf("%s: want a color #rrggbb or #rrggbbaa, got %q", path, value)
		}
		v.Set(reflect.ValueOf(c))
	case v.Kind() == reflect.Float32:
		f, err := strconv.ParseFloat(value, 32)
		if err != nil || math.IsNaN(f) || typed && (f < 0 || f > 400) {
			return fmt.Errorf("%s: want a size from 0 to 400, got %q", path, value)
		}
		v.SetFloat(f)
	case v.Kind() == reflect.Int:
		n, err := strconv.Atoi(value)
		if err != nil || typed && (n < 100 || n > 900) {
			return fmt.Errorf("%s: want a weight from 100 to 900, got %q", path, value)
		}
		v.SetInt(int64(n))
	case v.Kind() == reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil || value != "true" && value != "false" {
			return fmt.Errorf("%s: want true or false, got %q", path, value)
		}
		v.SetBool(b)
	case v.Kind() == reflect.String:
		v.SetString(value)
	default:
		return fmt.Errorf("%s: a field of kind %s", path, v.Kind())
	}
	return nil
}

// leafText returns the leaf v as text, as setValue reads it.
func leafText(v reflect.Value) string {
	switch {
	case v.Type() == colorType:
		return hexText(v.Interface().(ui.Color))
	case v.Kind() == reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case v.Kind() == reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	case v.Kind() == reflect.Bool:
		return strconv.FormatBool(v.Bool())
	}
	return v.String()
}

// hexText returns c as "#rrggbb", or "#rrggbbaa" if it is not opaque.
func hexText(c ui.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// parseColor reads "#rrggbb" or "#rrggbbaa".
func parseColor(s string) (ui.Color, bool) {
	if len(s) != 7 && len(s) != 9 || s[0] != '#' {
		return ui.Color{}, false
	}
	var b [4]byte
	b[3] = 255
	for i := range (len(s) - 1) / 2 {
		n, err := strconv.ParseUint(s[1+2*i:3+2*i], 16, 8)
		if err != nil {
			return ui.Color{}, false
		}
		b[i] = byte(n)
	}
	return ui.Color{R: b[0], G: b[1], B: b[2], A: b[3]}, true
}
