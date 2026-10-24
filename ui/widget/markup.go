package widget

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// ParseMarkup returns the spans of text in a subset of Pango markup, for
// Label.Rich and the like, where gota used markup: span with foreground
// (fgcolor, color), background (bgcolor) and weight, and b, i and u.
//
//	widget.Label("").Rich(widget.ParseMarkup(`<span foreground='white' background='green' weight='800'>Done</span> saved`)...)
//
// Colors are #rgb, #rrggbb or a name of X11's, as Pango reads them
// (white, black, red, green, blue, yellow, gray, orange, purple, cyan,
// magenta, pink, brown); weight is 100 to 900 or thin, light, normal, bold,
// ultrabold or heavy. The entities &amp; &lt; &gt; &apos; &quot; and
// &#N; &#xN; are text. Nothing else is: markup it does not know, or that is
// malformed, never fails. An unknown tag, or an unknown or bad attribute,
// is ignored and its words stay; a closing tag with no opening one is
// ignored; a tag left open styles the rest; a "<" that starts no tag, and
// an unknown entity, stay as they are.
func ParseMarkup(s string) []ui.Span {
	var (
		spans []ui.Span
		text  strings.Builder
		style ui.Span
		stack []markupFrame
	)
	flush := func() {
		if text.Len() > 0 {
			sp := style
			sp.Text = text.String()
			spans = append(spans, sp)
			text.Reset()
		}
	}
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == '<':
			j := strings.IndexByte(s[i:], '>')
			if j < 0 || !startsTag(s[i+1:]) {
				text.WriteByte(c)
				i++
				continue
			}
			tag := s[i+1 : i+j]
			i += j + 1
			flush()
			if name, ok := strings.CutPrefix(tag, "/"); ok {
				name = strings.ToLower(strings.TrimSpace(name))
				for k := len(stack) - 1; k >= 0; k-- {
					if stack[k].name == name {
						style = stack[k].before
						stack = stack[:k]
						break
					}
				}
				continue
			}
			selfClosing := strings.HasSuffix(tag, "/")
			name, attrs := splitTag(strings.TrimSuffix(tag, "/"))
			frame := markupFrame{name: name, before: style}
			switch name {
			case "b":
				style.Weight = 700
			case "i":
				style.Italic = true
			case "u":
				style.Underline = true
			case "span":
				applySpanAttrs(&style, attrs)
			}
			if !selfClosing {
				stack = append(stack, frame)
			} else {
				style = frame.before
			}
		case c == '&':
			if r, n := entity(s[i:]); n > 0 {
				text.WriteString(r)
				i += n
			} else {
				text.WriteByte(c)
				i++
			}
		default:
			text.WriteByte(c)
			i++
		}
	}
	flush()
	return spans
}

type markupFrame struct {
	name   string
	before ui.Span // the style outside the tag
}

// startsTag reports whether s, what follows a "<", begins a tag.
func startsTag(s string) bool {
	s = strings.TrimPrefix(s, "/")
	return s != "" && (s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z')
}

// splitTag returns a tag's lower-case name and its attributes.
func splitTag(tag string) (name string, attrs map[string]string) {
	tag = strings.TrimSpace(tag)
	end := strings.IndexAny(tag, " \t\r\n")
	if end < 0 {
		return strings.ToLower(tag), nil
	}
	name, rest := strings.ToLower(tag[:end]), tag[end:]
	attrs = map[string]string{}
	for {
		rest = strings.TrimLeft(rest, " \t\r\n")
		eq := strings.IndexByte(rest, '=')
		if eq <= 0 {
			return name, attrs
		}
		key := strings.ToLower(strings.TrimSpace(rest[:eq]))
		rest = strings.TrimLeft(rest[eq+1:], " \t\r\n")
		if rest == "" {
			return name, attrs
		}
		var val string
		if q := rest[0]; q == '\'' || q == '"' {
			k := strings.IndexByte(rest[1:], q)
			if k < 0 {
				return name, attrs
			}
			val, rest = rest[1:1+k], rest[k+2:]
		} else {
			k := strings.IndexAny(rest, " \t\r\n")
			if k < 0 {
				k = len(rest)
			}
			val, rest = rest[:k], rest[k:]
		}
		attrs[key] = val
	}
}

func applySpanAttrs(s *ui.Span, attrs map[string]string) {
	for k, v := range attrs {
		switch k {
		case "foreground", "fgcolor", "color":
			if c, ok := markupColor(v); ok {
				s.Color = c
			}
		case "background", "bgcolor":
			if c, ok := markupColor(v); ok {
				s.Background = c
			}
		case "weight", "font_weight":
			if w, ok := markupWeight(v); ok {
				s.Weight = w
			}
		}
	}
}

var markupWeights = map[string]int{"thin": 100, "ultralight": 200, "light": 300, "normal": 400, "medium": 500, "semibold": 600, "bold": 700, "ultrabold": 800, "heavy": 900}

func markupWeight(v string) (int, bool) {
	if w, ok := markupWeights[strings.ToLower(v)]; ok {
		return w, true
	}
	n, err := strconv.Atoi(v)
	return min(max(n, 100), 900), err == nil && n > 0
}

// markupColors are X11's colors, which Pango's names are.
var markupColors = map[string]ui.Color{
	"black": ui.RGB(0, 0, 0), "white": ui.RGB(255, 255, 255), "red": ui.RGB(255, 0, 0), "green": ui.RGB(0, 255, 0),
	"blue": ui.RGB(0, 0, 255), "yellow": ui.RGB(255, 255, 0), "gray": ui.RGB(190, 190, 190), "grey": ui.RGB(190, 190, 190),
	"orange": ui.RGB(255, 165, 0), "purple": ui.RGB(160, 32, 240), "cyan": ui.RGB(0, 255, 255),
	"magenta": ui.RGB(255, 0, 255), "pink": ui.RGB(255, 192, 203), "brown": ui.RGB(165, 42, 42),
}

func markupColor(v string) (ui.Color, bool) {
	if h, ok := strings.CutPrefix(v, "#"); ok {
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		n, err := strconv.ParseUint(h, 16, 32)
		if len(h) != 6 || err != nil {
			return ui.Color{}, false
		}
		return ui.RGB(uint8(n>>16), uint8(n>>8), uint8(n)), true
	}
	c, ok := markupColors[strings.ToLower(v)]
	return c, ok
}

// entity returns the text of the entity s starts with and its length, or
// 0 when it is none that is known.
func entity(s string) (string, int) {
	end := strings.IndexByte(s, ';')
	if end < 2 || end > 10 {
		return "", 0
	}
	switch name := s[1:end]; name {
	case "amp":
		return "&", end + 1
	case "lt":
		return "<", end + 1
	case "gt":
		return ">", end + 1
	case "apos":
		return "'", end + 1
	case "quot":
		return `"`, end + 1
	default:
		if name[0] == '#' {
			base, digits := 10, name[1:]
			if len(digits) > 0 && (digits[0] == 'x' || digits[0] == 'X') {
				base, digits = 16, digits[1:]
			}
			if n, err := strconv.ParseUint(digits, base, 32); err == nil && n > 0 && n <= 0x10FFFF && (n < 0xD800 || n > 0xDFFF) {
				return string(rune(n)), end + 1
			}
		}
	}
	return "", 0
}
