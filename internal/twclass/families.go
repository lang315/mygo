package twclass

import "strings"

// box is a family of spacing or size utilities: prefix-<scale>.
type box struct {
	prefix string
	kind   Kind
	sides  uint8
}

var boxes = []box{
	{"p", Padding, 0b1111}, {"px", Padding, 0b1010}, {"py", Padding, 0b0101},
	{"pt", Padding, 0b0001}, {"pr", Padding, 0b0010}, {"pb", Padding, 0b0100}, {"pl", Padding, 0b1000},
	{"m", Margin, 0b1111}, {"mx", Margin, 0b1010}, {"my", Margin, 0b0101},
	{"mt", Margin, 0b0001}, {"mr", Margin, 0b0010}, {"mb", Margin, 0b0100}, {"ml", Margin, 0b1000},
	{"w", Width, 0}, {"h", Height, 0}, {"min-w", MinWidth, 0}, {"max-w", MaxWidth, 0},
	{"min-h", MinHeight, 0}, {"max-h", MaxHeight, 0},
}

// parseUtility returns what utility, a class without its variants, sets.
func parseUtility(u string) ([]Decl, bool) {
	if d, ok := fixed[u]; ok {
		return d, true
	}
	neg := strings.HasPrefix(u, "-")
	if neg {
		u = u[1:]
	}
	for _, b := range boxes {
		v, ok := strings.CutPrefix(u, b.prefix+"-")
		if !ok {
			continue
		}
		n, ok := spacing[v]
		if !ok || neg && b.kind != Margin {
			return nil, false
		}
		if neg {
			n = -n
		}
		if b.sides == 0 { // a size: Tailwind's unit is 4 DIPs
			return []Decl{{Kind: b.kind, Value: n * 4}}, true
		}
		return []Decl{{Kind: b.kind, Sides: b.sides, Value: n}}, true
	}
	if neg {
		return nil, false
	}
	switch {
	case u == "rounded" || strings.HasPrefix(u, "rounded-"):
		return rounded(strings.TrimPrefix(strings.TrimPrefix(u, "rounded"), "-"))
	case u == "border" || strings.HasPrefix(u, "border-"):
		return border(strings.TrimPrefix(strings.TrimPrefix(u, "border"), "-"))
	case strings.HasPrefix(u, "text-"):
		v := u[len("text-"):]
		if z, ok := textSizes[v]; ok {
			return []Decl{z}, true
		}
		if c, ok := color(v, "text"); ok {
			return []Decl{{Kind: TextColor, Color: c}}, true
		}
	case strings.HasPrefix(u, "bg-"):
		if c, ok := color(u[len("bg-"):], "bg"); ok {
			return []Decl{{Kind: Background, Color: c}}, true
		}
	case strings.HasPrefix(u, "font-"):
		if w, ok := weights[u[len("font-"):]]; ok {
			return []Decl{{Kind: FontWeight, Value: w}}, true
		}
	case strings.HasPrefix(u, "opacity-"):
		if o, ok := opacities[u[len("opacity-"):]]; ok {
			return []Decl{{Kind: Opacity, Value: o}}, true
		}
	case strings.HasPrefix(u, "cursor-"):
		if v := u[len("cursor-"):]; cursors[v] {
			return []Decl{{Kind: Cursor, Name: v}}, true
		}
	}
	return nil, false
}

var corners = map[string]uint8{"t": 0b0011, "r": 0b0110, "b": 0b1100, "l": 0b1001, "tl": 0b0001, "tr": 0b0010, "br": 0b0100, "bl": 0b1000}

// rounded parses what follows rounded: [side-]size, both optional.
func rounded(rest string) ([]Decl, bool) {
	sides, size := uint8(0b1111), rest
	if c, r, found := strings.Cut(rest, "-"); found {
		s, ok := corners[c]
		if !ok {
			return nil, false
		}
		sides, size = s, r
	} else if s, ok := corners[rest]; ok {
		sides, size = s, ""
	}
	v, ok := radii[size]
	if !ok {
		return nil, false
	}
	return []Decl{{Kind: Radius, Sides: sides, Value: v}}, true
}

var borderSides = map[string]uint8{"x": 0b1010, "y": 0b0101, "t": 0b0001, "r": 0b0010, "b": 0b0100, "l": 0b1000}

// border parses what follows border: a style, [side-]width, or a color.
func border(rest string) ([]Decl, bool) {
	switch rest {
	case "solid":
		return []Decl{{Kind: BorderStyle, Value: 0}}, true
	case "dashed":
		return []Decl{{Kind: BorderStyle, Value: 1}}, true
	}
	sides, width := uint8(0b1111), rest
	if s, w, found := strings.Cut(rest, "-"); found {
		if b, ok := borderSides[s]; ok {
			sides, width = b, w
		}
	} else if b, ok := borderSides[rest]; ok {
		sides, width = b, ""
	}
	if w, ok := borderWidths[width]; ok {
		return []Decl{{Kind: BorderWidth, Sides: sides, Value: w}}, true
	}
	if sides == 0b1111 {
		if c, ok := color(rest, "border"); ok {
			return []Decl{{Kind: BorderColor, Color: c}}, true
		}
	}
	return nil, false
}

// color parses a color utility's value after prefix (bg, text or border):
// a palette color, black, white, transparent, or a theme token.
func color(v, prefix string) (Color, bool) {
	switch v {
	case "black":
		return Color{RGBA: 0x000000ff}, true
	case "white":
		return Color{RGBA: 0xffffffff}, true
	case "transparent":
		return Color{}, true
	case "default":
		return Color{Role: defaults[prefix]}, true
	}
	if r, ok := roles[v]; ok {
		return Color{Role: r}, true
	}
	hue, shade, ok := strings.Cut(v, "-")
	if !ok {
		return Color{}, false
	}
	row, ok := palette[hue]
	i, ok2 := shades[shade]
	if !ok || !ok2 {
		return Color{}, false
	}
	return Color{RGBA: row[i]<<8 | 0xff}, true
}
