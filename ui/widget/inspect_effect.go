package widget

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// The Style in effect for the Widget tab: each property the control's Styles
// set, with the value that wins in the state of the frame and where it came
// from, and the values it won over. It replays what restyle and effective
// do, in their order, over the layers Apply was given and the control's own
// methods, so that it shows the layer and the variant, which the merged Style
// does not keep.

// frameState is the state a frame styled a control in.
type frameState struct{ dark, hover, focused, pressed, disabled bool }

// on lists the variants that hold, in the order effective applies them.
func stateOn(st frameState) [nVariants]bool {
	return [nVariants]bool{vDark: st.dark, vHover: st.hover, vFocused: st.focused, vPressed: st.pressed, vDisabled: st.disabled}
}

func stateText(st frameState) string {
	var on []string
	for k, b := range stateOn(st) {
		if b {
			on = append(on, variantNames[k])
		}
	}
	if len(on) == 0 {
		return "no state"
	}
	return strings.Join(on, ", ")
}

// styleLayer is a Style of a control and what names it in the tab.
type styleLayer struct {
	name string
	s    Style
}

// layersOf lists the Styles that make b's, in the order restyle merges them.
// Under mu.
func layersOf(b *core) []styleLayer {
	var out []styleLayer
	for i, s := range b.applied {
		name := fmt.Sprintf("Apply, style %d", i+1)
		if s.src != nil {
			name += " (" + clip(s.src.classes, 40) + ")"
		}
		out = append(out, styleLayer{name, s})
	}
	if !b.ownStyle.empty() {
		out = append(out, styleLayer{"Own methods", b.ownStyle})
	}
	return out
}

// clip shortens s to n runes, with an ellipsis.
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// isLayout reports whether name is a property that places a control, which
// applies from the base only. A function, not a map, so that nothing is made at
// start-up.
func isLayout(name string) bool {
	switch name {
	case "align", "valign", "hexpand", "vexpand":
		return true
	}
	return false
}

// candidate is a value a layer set for a property.
type candidate struct {
	from  string // "Apply, style 1", "hover: Own methods"
	value string
}

// candidates lists, by property, the values the layers set that count in the
// state st, from the first to apply to the last, which wins: the base of each
// layer in order, then, for each variant that holds, its Style in each layer
// and, in the dark theme, the dark: inside it. A layout property set in a
// variant does not count: only the base's place the control.
func candidates(th *ui.Theme, layers []styleLayer, st frameState) (names []string, by map[string][]candidate) {
	by = map[string][]candidate{}
	add := func(from string, p *props, variant bool) {
		group := map[string][]string{}
		var order []string
		for _, f := range p.describeSides(th, "", false) {
			if variant && isLayout(f.Name) {
				continue
			}
			if _, ok := group[f.Name]; !ok {
				order = append(order, f.Name)
			}
			group[f.Name] = append(group[f.Name], f.Value)
		}
		for _, n := range order {
			if _, ok := by[n]; !ok {
				names = append(names, n)
			}
			by[n] = append(by[n], candidate{from, strings.Join(group[n], "; ")})
		}
	}
	for _, l := range layers {
		add(l.name, &l.s.props, false)
	}
	on := stateOn(st)
	for k := range nVariants {
		if !on[k] {
			continue
		}
		for _, l := range layers {
			if v := l.s.v[k]; v != nil {
				add(variantNames[k]+": "+l.name, &v.props, true)
			}
		}
		if k == vDark {
			continue
		}
		for _, l := range layers {
			if v := l.s.v[k]; v != nil && st.dark && v.v[vDark] != nil {
				add(variantNames[k]+":dark: "+l.name, &v.v[vDark].props, true)
			}
		}
	}
	return names, by
}

// inEffect describes the Style in effect: a heading naming the state, then
// each property with the value that wins and where it came from, and below it
// the values it won over, struck through, the last first.
func inEffect(th *ui.Theme, layers []styleLayer, st frameState) []ui.InspectorField {
	if len(layers) == 0 {
		return nil
	}
	names, by := candidates(th, layers, st)
	names = joinSides(names, by)
	out := []ui.InspectorField{{Name: "In effect (" + stateText(st) + ")"}}
	for _, n := range names {
		cs := by[n]
		win := cs[len(cs)-1]
		out = append(out, ui.InspectorField{Name: n, Value: win.value + "   " + win.from})
		for i := len(cs) - 2; i >= 0; i-- {
			out = append(out, ui.InspectorField{Name: n, Value: cs[i].value + "   " + cs[i].from, Struck: true})
		}
	}
	return out
}

// joinSides shows the four sides of padding, margin, radius or border width
// as one line when every layer that set one set all four to the same value,
// as Padding(2) does; sides that differ stay apart. It changes by, adding the
// property's name and taking the sides out, and returns the names.
func joinSides(names []string, by map[string][]candidate) []string {
	for _, g := range []struct {
		base  string
		sides [4]string
	}{
		{"padding", [4]string{"padding-top", "padding-right", "padding-bottom", "padding-left"}},
		{"margin", [4]string{"margin-top", "margin-right", "margin-bottom", "margin-left"}},
		{"radius", [4]string{"radius-top-left", "radius-top-right", "radius-bottom-right", "radius-bottom-left"}},
		{"border-width", [4]string{"border-width-top", "border-width-right", "border-width-bottom", "border-width-left"}},
	} {
		first, ok := by[g.sides[0]]
		for _, sd := range g.sides[1:] {
			cs, has := by[sd]
			if !has || !equalCandidates(first, cs) {
				ok = false
			}
		}
		if !ok {
			continue
		}
		by[g.base] = first
		var kept []string
		for _, n := range names {
			switch n {
			case g.sides[0]:
				kept = append(kept, g.base)
			case g.sides[1], g.sides[2], g.sides[3]:
			default:
				kept = append(kept, n)
			}
		}
		names = kept
		for _, sd := range g.sides {
			delete(by, sd)
		}
	}
	return names
}

func equalCandidates(a, b []candidate) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
