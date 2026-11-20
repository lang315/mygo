package skinfile

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

// Format returns the file of the skin s, written as what s changes from
// base: its palette or accent, its density if not base's, the tokens that differ from
// base's after that palette and density, and the kinds of looks that
// differ. A nil base writes every token and every look s has, against
// Calm. Keys come in the order of Tokens' fields, two spaces a level.
//
// A palette that is neither one of base's Palettes nor an accent's is not
// named: the tokens carry it.
func Format(s, base *skin.Skin) []byte {
	all := base == nil
	if all {
		base = skin.Calm
	}
	ref := base
	var o object
	o.key("version")
	o.b = append(o.b, '1')
	o.key("name")
	o.b = jsonlite.AppendQuote(o.b, s.Name())
	o.key("base")
	o.b = jsonlite.AppendQuote(o.b, base.Name())
	if p := s.Palette(); p != nil && (all || base.Palette() == nil || p.Name() != base.Palette().Name()) {
		if c, ok := parseColor(p.Name()); ok {
			o.key("accent")
			o.b = jsonlite.AppendQuote(o.b, p.Name())
			ref = ref.WithAccent(c)
		} else if q := findPalette(base, p.Name()); q != nil {
			o.key("palette")
			o.b = jsonlite.AppendQuote(o.b, p.Name())
			ref = ref.WithPalette(q)
		}
	}
	if d := s.Density(); d != base.Density() {
		o.key("density")
		o.b = append(o.b, map[skin.Density]string{skin.Standard: `"standard"`, skin.Compact: `"compact"`}[d]...)
	}
	for i, key := range [...]string{"light", "dark"} {
		dark := i == 1
		if v := value(reflect.ValueOf(s.Tokens(dark)), reflect.ValueOf(ref.Tokens(dark)), all, 2); v != nil {
			o.key(key)
			o.b = append(o.b, v...)
		}
	}
	l := object{indent: 2}
	got, was := reflect.ValueOf(s.Looks()), reflect.ValueOf(ref.Looks())
	for i := range got.NumField() {
		g := got.Field(i).String()
		if all && g != "" || !all && g != was.Field(i).String() {
			l.key(got.Type().Field(i).Name)
			l.b = jsonlite.AppendQuote(l.b, g)
		}
	}
	if v := l.end(); v != nil {
		o.key("looks")
		o.b = append(o.b, v...)
	}
	return append(o.end(), '\n')
}

// object writes an object's members, indent spaces in.
type object struct {
	b      []byte
	indent int
}

// key starts the member key.
func (o *object) key(k string) {
	if o.b == nil {
		o.b = append(o.b, '{')
	} else {
		o.b = append(o.b, ',')
	}
	o.b = append(o.b, '\n')
	o.b = append(o.b, strings.Repeat(" ", o.indent+2)...)
	o.b = jsonlite.AppendQuote(o.b, k)
	o.b = append(o.b, ": "...)
}

// end closes the object, nil for one with no members.
func (o *object) end() []byte {
	if o.b == nil {
		return nil
	}
	o.b = append(o.b, '\n')
	o.b = append(o.b, strings.Repeat(" ", o.indent)...)
	return append(o.b, '}')
}

// value writes the field v, a Tokens or a part of one, at indent: nil when
// it equals was, unless all.
func value(v, was reflect.Value, all bool, indent int) []byte {
	if v.Type() != colorType && v.Kind() == reflect.Struct {
		o := object{indent: indent}
		for i := range v.NumField() {
			if b := value(v.Field(i), was.Field(i), all, indent+2); b != nil {
				o.key(v.Type().Field(i).Name)
				o.b = append(o.b, b...)
			}
		}
		return o.end()
	}
	if !all && v.Equal(was) {
		return nil
	}
	switch {
	case v.Type() == colorType:
		return appendColor(nil, v.Interface().(ui.Color))
	case v.Kind() == reflect.Array:
		b := []byte{'['}
		for i := range v.Len() {
			if i > 0 {
				b = append(b, ", "...)
			}
			b = appendColor(b, v.Index(i).Interface().(ui.Color))
		}
		return append(b, ']')
	case v.Kind() == reflect.Float32:
		return strconv.AppendFloat(nil, v.Float(), 'g', -1, 32)
	case v.Kind() == reflect.Int:
		return strconv.AppendInt(nil, v.Int(), 10)
	case v.Kind() == reflect.Bool:
		return strconv.AppendBool(nil, v.Bool())
	case v.Kind() == reflect.String:
		return jsonlite.AppendQuote(nil, v.String())
	}
	panic("skinfile: a field of kind " + v.Kind().String()) // TestDecoderKinds keeps Tokens to the kinds above
}

// appendColor appends c as "#rrggbb", or "#rrggbbaa" if it is not opaque.
func appendColor(b []byte, c ui.Color) []byte {
	if c.A == 255 {
		return fmt.Appendf(b, `"#%02x%02x%02x"`, c.R, c.G, c.B)
	}
	return fmt.Appendf(b, `"#%02x%02x%02x%02x"`, c.R, c.G, c.B, c.A)
}
