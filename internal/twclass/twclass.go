// Package twclass parses Tailwind-style class strings into declarations:
// for ui/widget/tw, which makes widget Styles of them, and internal/uivet,
// which checks literal strings. It imports neither ui nor widget, so the
// CLI does not link them.
package twclass

import (
	"slices"
	"strings"
)

// Variant is the state a declaration applies in.
type Variant uint8

const (
	Base Variant = iota
	Hover
	Active
	Focus
	Disabled
)

// Key is where a declaration applies: a state, in the dark theme or not.
type Key struct {
	Variant Variant
	Dark    bool
}

// Kind is what a declaration sets.
type Kind uint8

const (
	Padding     Kind = iota + 1 // Sides, Value in theme units
	Margin                      // Sides, Value in theme units, maybe negative
	Radius                      // Sides as corners, Value in theme units
	BorderWidth                 // Sides, Value in DIPs
	BorderStyle                 // Value 0 solid, 1 dashed
	Width                       // Value in DIPs, as the five below
	Height
	MinWidth
	MaxWidth
	MinHeight
	MaxHeight
	Background // Color
	TextColor  // Color
	BorderColor
	TextSize   // Value: widget.Size's order, 0 base, 1 xs, 2 sm, 3 lg, 4 xl, 5 2xl, 6 3xl
	FontSize   // Value in DIPs
	FontWeight // Value 100 to 900
	Italic     // Value 1 italic, 0 not
	Mono
	Opacity // Value 0 to 1
	Cursor  // Name
	Shadows // Shadows; none clears them
	Align   // Value: widget.Align's order, 1 start, 2 center, 3 end, 4 fill
	VAlign
	HExpand
	VExpand
	Transition // Value in milliseconds

	kindEnd // after the last kind: loops over the kinds stop before it
)

// Color is a palette color, RGBA as 0xRRGGBBAA, or a theme role by name
// (the names in Roles).
type Color struct {
	RGBA uint32
	Role string
}

// Shadow is one layer of a box shadow, in DIPs, black at Alpha.
type Shadow struct {
	X, Y, Blur, Spread, Alpha float32
	Inset                     bool
}

// Decl is one property a class sets. Sides holds, for the four-sided
// kinds, bit 0 for the top (Radius: the top left corner), 1 the right
// (top right), 2 the bottom (bottom right), 3 the left (bottom left).
type Decl struct {
	Kind    Kind
	Sides   uint8
	Value   float32
	Color   Color
	Name    string
	Shadows []Shadow
}

// Rule is a declaration and where it applies.
type Rule struct {
	Key Key
	Decl
}

// Bad is a class Parse could not use, and why.
type Bad struct{ Class, Reason string }

// The reasons of Bad.
const (
	Unknown             = "unknown class"
	LayoutInVariant     = "layout in a variant"
	TransitionInVariant = "transition in a variant"
	NotItalicInVariant  = "not-italic in a variant"
	Stack               = "unsupported variant stack"
)

// defaultDuration is transition's, in milliseconds, as Tailwind's.
const defaultDuration = 150

// Parse returns the declarations of the classes in order, so a later one
// on the same property wins, and the classes it could not use. A
// transition comes last, as one declaration. It never panics.
func Parse(classes string) (rules []Rule, bad []Bad) {
	var duration float32
	transition := false
	for _, class := range expand(strings.Fields(classes)) {
		key, utility, reason := variants(class)
		if reason != "" {
			bad = append(bad, Bad{class, reason})
			continue
		}
		if ms, ok := durations[utility]; ok || utility == "transition" {
			if key != (Key{}) {
				bad = append(bad, Bad{class, TransitionInVariant})
				continue
			}
			transition = true
			if ok {
				duration = ms
			}
			continue
		}
		decls, ok := parseUtility(utility)
		if !ok {
			bad = append(bad, Bad{class, Unknown})
			continue
		}
		if key != (Key{}) && layout(decls) {
			bad = append(bad, Bad{class, LayoutInVariant})
			continue
		}
		if key != (Key{}) && notItalic(decls) { // a Style cannot clear italic
			bad = append(bad, Bad{class, NotItalicInVariant})
			continue
		}
		for _, d := range decls {
			d.Shadows = slices.Clone(d.Shadows) // the table's own must not leak
			rules = append(rules, Rule{key, d})
		}
	}
	if transition {
		if duration == 0 {
			duration = defaultDuration
		}
		rules = append(rules, Rule{Key{}, Decl{Kind: Transition, Value: duration}})
	}
	return rules, bad
}

// expand replaces gota's button classes by what they stand for.
func expand(fields []string) []string {
	var out []string
	for _, f := range fields {
		if x, ok := buttons[f]; ok {
			out = append(out, strings.Fields(x)...)
		} else {
			out = append(out, f)
		}
	}
	return out
}

var states = map[string]Variant{"hover": Hover, "active": Active, "focus": Focus, "disabled": Disabled}

// variants splits class's prefixes off: dark with at most one state.
func variants(class string) (key Key, utility, reason string) {
	parts := strings.Split(class, ":")
	utility = parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		if p == "dark" {
			if key.Dark {
				return key, "", Stack
			}
			key.Dark = true
			continue
		}
		v, ok := states[p]
		if !ok {
			return key, "", Unknown
		}
		if key.Variant != Base {
			return key, "", Stack
		}
		key.Variant = v
	}
	if utility == "" {
		return key, "", Unknown
	}
	return key, utility, ""
}

func notItalic(decls []Decl) bool {
	for _, d := range decls {
		if d.Kind == Italic && d.Value == 0 {
			return true
		}
	}
	return false
}

func layout(decls []Decl) bool {
	for _, d := range decls {
		switch d.Kind {
		case Align, VAlign, HExpand, VExpand:
			return true
		}
	}
	return false
}
