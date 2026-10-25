package twclass

import (
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The cheat sheet of docs/ui/widget/tw.md: the tables between its
// <!-- twclass:name --> markers are made here from the parser's own tables,
// so that the page cannot say other than the parser does. Run
//
//	go test ./internal/twclass -run TestCheatSheet -update
//
// to write them. The classes of the page's ```tw blocks must parse, and those
// of its ```tw-bad blocks must be refused for the reason it gives.

var update = flag.Bool("update", false, "write the generated tables of the cheat sheet")

const sheetPath = "../../docs/ui/widget/tw.md"

func keysByValue[V ~float32](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] < m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func code(ss ...string) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = "`" + s + "`"
	}
	return strings.Join(out, " ")
}

func num(f float32) string { return strconv.FormatFloat(float64(f), 'f', -1, 32) }

// sideNames names the sides of a mask: bit 0 the top, 1 the right, 2 the
// bottom, 3 the left.
func sideNames(mask uint8, names [4]string) string {
	if mask == 0b1111 {
		return "all four"
	}
	var out []string
	for i := range 4 {
		if mask&(1<<i) != 0 {
			out = append(out, names[i])
		}
	}
	return strings.Join(out, " + ")
}

var (
	edgeNames   = [4]string{"top", "right", "bottom", "left"}
	cornerNames = [4]string{"top left", "top right", "bottom right", "bottom left"}
)

// styleOf is the widget.Style method of each kind of declaration
// (ui/widget/tw's build).
var styleOf = map[Kind]string{
	Padding: "PaddingTop … PaddingLeft", Margin: "MarginTop … MarginLeft", Radius: "RadiusTopLeft … RadiusBottomLeft",
	BorderWidth: "BorderTop … BorderLeft", BorderStyle: "BorderStyle", Width: "Width", Height: "Height",
	MinWidth: "MinWidth", MaxWidth: "MaxWidth", MinHeight: "MinHeight", MaxHeight: "MaxHeight",
	Background: "Background", TextColor: "TextColor", BorderColor: "BorderColor", TextSize: "TextSize",
	FontSize: "FontSize", FontWeight: "FontWeight", Italic: "Italic", Mono: "Mono", Opacity: "Opacity",
	Cursor: "Cursor", Shadows: "Shadow, InsetShadow, NoShadow", Align: "Align", VAlign: "VAlign",
	HExpand: "HExpand", VExpand: "VExpand", Transition: "Transition",
}

var kindNames = map[Kind]string{
	Padding: "Padding", Margin: "Margin", Radius: "Radius", BorderWidth: "BorderWidth", BorderStyle: "BorderStyle",
	Width: "Width", Height: "Height", MinWidth: "MinWidth", MaxWidth: "MaxWidth", MinHeight: "MinHeight",
	MaxHeight: "MaxHeight", Background: "Background", TextColor: "TextColor", BorderColor: "BorderColor",
	TextSize: "TextSize", FontSize: "FontSize", FontWeight: "FontWeight", Italic: "Italic", Mono: "Mono",
	Opacity: "Opacity", Cursor: "Cursor", Shadows: "Shadows", Align: "Align", VAlign: "VAlign",
	HExpand: "HExpand", VExpand: "VExpand", Transition: "Transition",
}

// blocks makes the tables of the page, by name.
func blocks() map[string]string {
	out := map[string]string{}

	var b strings.Builder
	b.WriteString("| Class | Sets | Value |\n|---|---|---|\n")
	for _, x := range boxes {
		names, what, unit := edgeNames, "", ""
		switch x.kind {
		case Padding:
			what = "padding"
		case Margin:
			what = "margin"
		default:
			what = strings.ToLower(kindNames[x.kind])
			what = map[string]string{"width": "width", "height": "height", "minwidth": "minimum width", "maxwidth": "maximum width", "minheight": "minimum height", "maxheight": "maximum height"}[what]
		}
		if x.sides != 0 {
			what += " of " + sideNames(x.sides, names)
			unit = "`n` × 4 DIPs"
		} else {
			unit = "`n` × 4 DIPs"
		}
		cls := "`" + x.prefix + "-{n}`"
		if x.kind == Margin {
			cls += " `-" + x.prefix + "-{n}`"
			unit += "; negative with the `-` form"
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", cls, what, unit)
	}
	b.WriteString("| `max-w-none` | no maximum width (a `w-` is no longer capped at its container's room) | |\n")
	out["boxes"] = b.String()

	b.Reset()
	b.WriteString("`n` is one of " + ranges(keysByValue(spacing), spacing, 1) + ".\n\n")
	b.WriteString("`px` is 1 DIP; any other `n` is `n` × 4 DIPs (`Theme.Space(n)`).\n")
	out["spacing"] = b.String()

	b.Reset()
	b.WriteString("| Class | Radius |\n|---|---|\n")
	for _, k := range keysByValue(radii) {
		cls := "rounded"
		if k != "" {
			cls += "-" + k
		}
		v := num(radii[k]*4) + " DIPs"
		if radii[k] == 9999 {
			v = "a full pill or circle"
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", cls, v)
	}
	b.WriteString("\nBefore the size, a side or a corner: ")
	var sides []string
	for _, k := range sortedKeys(corners) {
		sides = append(sides, "`"+k+"` ("+sideNames(corners[k], cornerNames)+")")
	}
	b.WriteString(strings.Join(sides, ", ") + ". `rounded-l`, `rounded-tr-2xl` and `rounded-l-none` all parse.\n")
	out["radius"] = b.String()

	b.Reset()
	b.WriteString("| Class | Width |\n|---|---|\n")
	for _, k := range keysByValue(borderWidths) {
		cls := "border"
		if k != "" {
			cls += "-" + k
		}
		fmt.Fprintf(&b, "| `%s` | %s DIP |\n", cls, num(borderWidths[k]))
	}
	b.WriteString("\nOne side or two: ")
	var bs []string
	for _, k := range sortedKeys(borderSides) {
		bs = append(bs, "`"+k+"` ("+sideNames(borderSides[k], edgeNames)+")")
	}
	b.WriteString(strings.Join(bs, ", ") + ", as `border-t`, `border-x-2`, `border-b-0`. `border-solid` and `border-dashed` set the style. A color is `border-{color}` (see Colors).\n")
	out["border"] = b.String()

	b.Reset()
	b.WriteString("| Class | Size |\n|---|---|\n")
	order := []string{"xs", "sm", "base", "lg", "xl", "2xl", "3xl", "4xl", "5xl", "6xl", "7xl", "8xl", "9xl"}
	for _, k := range order {
		z := textSizes[k]
		v := ""
		if z.Kind == TextSize {
			v = "`widget.Size` step " + num(z.Value) + " (0 is the theme's base size)"
		} else {
			v = num(z.Value) + " DIPs"
		}
		fmt.Fprintf(&b, "| `text-%s` | %s |\n", k, v)
	}
	b.WriteString("\n| Class | Weight |\n|---|---|\n")
	for _, k := range keysByValue(weights) {
		fmt.Fprintf(&b, "| `font-%s` | %s |\n", k, num(weights[k]))
	}
	out["text"] = b.String()

	b.Reset()
	b.WriteString("Palette (Tailwind v3.4): `{color}-{shade}` after `bg-`, `text-` or `border-`.\n\n")
	b.WriteString("Hues: " + code(sortedKeys(palette)...) + "\n\n")
	var shs []string
	for _, k := range keysByValue32(shades) {
		shs = append(shs, k)
	}
	b.WriteString("Shades: " + code(shs...) + "\n\n")
	b.WriteString("Also `black`, `white`, `transparent`, and `default`, the theme's color for the property (`bg-default` is the background, `text-default` the text, `border-default` the border).\n\n")
	b.WriteString("Palette colors are the same in the light and the dark theme.\n")
	out["palette"] = b.String()

	b.Reset()
	b.WriteString("| Class suffix | Theme role |\n|---|---|\n")
	for _, k := range sortedKeys(roles) {
		fmt.Fprintf(&b, "| `bg-%s`, `text-%s`, `border-%s` | `%s` |\n", k, k, k, roles[k])
	}
	b.WriteString("\nThe roles are " + code(Roles...) + ". Theme colors follow the theme, light and dark.\n")
	out["roles"] = b.String()

	b.Reset()
	b.WriteString("| Class | Layers (x, y, blur, spread, alpha) |\n|---|---|\n")
	var shadowNames []string
	for k, d := range fixed {
		if len(d) == 1 && d[0].Kind == Shadows {
			shadowNames = append(shadowNames, k)
		}
	}
	sort.Strings(shadowNames)
	for _, k := range shadowNames {
		var layers []string
		for _, s := range fixed[k][0].Shadows {
			in := ""
			if s.Inset {
				in = " inset"
			}
			layers = append(layers, fmt.Sprintf("(%s, %s, %s, %s, %s)%s", num(s.X), num(s.Y), num(s.Blur), num(s.Spread), num(s.Alpha), in))
		}
		if len(layers) == 0 {
			layers = []string{"none: clears the shadows"}
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", k, strings.Join(layers, " + "))
	}
	b.WriteString("\n`opacity-{n}`, `n` one of: ")
	b.WriteString(ranges(keysByValue(opacities), opacities, 100) + ", in percent.\n\n`cursor-{name}`, `name` one of: " + code(sortedKeys(cursors)...) + ".\n")
	out["effects"] = b.String()

	b.Reset()
	b.WriteString("| Class | Sets |\n|---|---|\n")
	var plain []string
	for k, d := range fixed {
		if len(d) == 1 && d[0].Kind != Shadows && k != "max-w-none" {
			plain = append(plain, k)
		}
	}
	sort.Strings(plain)
	for _, k := range plain {
		fmt.Fprintf(&b, "| `%s` | %s |\n", k, describe(fixed[k][0]))
	}
	out["fixed"] = b.String()

	b.Reset()
	b.WriteString("| Class | Same as |\n|---|---|\n")
	for _, k := range sortedKeys(buttons) {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", k, buttons[k])
	}
	out["buttons"] = b.String()

	b.Reset()
	b.WriteString("| Class | Duration |\n|---|---|\n")
	b.WriteString(fmt.Sprintf("| `transition` | %d ms unless a `duration-` class is given |\n", defaultDuration))
	for _, k := range keysByValue(durations) {
		fmt.Fprintf(&b, "| `%s` | %s ms |\n", k, num(durations[k]))
	}
	out["motion"] = b.String()

	b.Reset()
	b.WriteString("| What a class sets | `widget.Style` method |\n|---|---|\n")
	for k := Padding; k <= Transition; k++ {
		fmt.Fprintf(&b, "| %s | `%s` |\n", kindNames[k], styleOf[k])
	}
	out["styles"] = b.String()
	return out
}

// ranges writes keys, in the order of their values (scaled by scale), with
// the runs of four or more at an even step as "a … b (step s)".
func ranges(keys []string, values map[string]float32, scale float64) string {
	v := func(k string) float64 { return math.Round(float64(values[k])*scale*1e4) / 1e4 }
	var out []string
	for i := 0; i < len(keys); {
		j := i + 1
		if j < len(keys) {
			step := v(keys[j]) - v(keys[i])
			for j+1 < len(keys) && math.Abs(v(keys[j+1])-v(keys[j])-step) < 1e-6 {
				j++
			}
			if j-i+1 >= 4 {
				out = append(out, fmt.Sprintf("`%s` … `%s` (step %s)", keys[i], keys[j], strconv.FormatFloat(step, 'f', -1, 64)))
				i = j + 1
				continue
			}
		}
		out = append(out, "`"+keys[i]+"`")
		i++
	}
	return strings.Join(out, ", ")
}

func keysByValue32(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] < m[keys[j]] })
	return keys
}

// describe says what a fixed utility sets.
func describe(d Decl) string {
	switch d.Kind {
	case Italic:
		return map[float32]string{1: "italic text", 0: "not italic (clears an `italic` earlier in the string; an error in a variant)"}[d.Value]
	case Mono:
		return "the monospace font"
	case HExpand:
		return "grows to take the room along the container (layout: an error in a variant)"
	case VExpand:
		return "grows to take the room across it (layout: an error in a variant)"
	case Align, VAlign:
		axis := "horizontal"
		if d.Kind == VAlign {
			axis = "vertical"
		}
		return axis + " alignment: " + [...]string{"", "start", "center", "end", "fill"}[int(d.Value)] + " (layout: an error in a variant)"
	}
	return kindNames[d.Kind]
}

var markerRE = regexp.MustCompile(`(?s)<!-- twclass:(\w+) -->\n(.*?)<!-- /twclass:\w+ -->`)

func TestCheatSheet(t *testing.T) {
	data, err := os.ReadFile(sheetPath)
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	want := blocks()
	seen := map[string]bool{}
	next := markerRE.ReplaceAllStringFunc(page, func(m string) string {
		name := markerRE.FindStringSubmatch(m)[1]
		seen[name] = true
		body, ok := want[name]
		if !ok {
			t.Errorf("the page has a table twclass:%s that the test does not make", name)
			return m
		}
		return "<!-- twclass:" + name + " -->\n" + body + "<!-- /twclass:" + name + " -->"
	})
	for name := range want {
		if !seen[name] {
			t.Errorf("the page lacks the table twclass:%s", name)
		}
	}
	if next != page {
		if *update {
			if err := os.WriteFile(sheetPath, []byte(next), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		t.Errorf("tables of %s are not the parser's: go test ./internal/twclass -run TestCheatSheet -update", sheetPath)
	}
}

// Every kind of declaration is described in the page's table of Style methods.
func TestCheatSheetKnowsEveryKind(t *testing.T) {
	for k := Padding; k <= Transition; k++ {
		if kindNames[k] == "" || styleOf[k] == "" {
			t.Errorf("kind %d is not described: add it to kindNames and styleOf", k)
		}
	}
	if Transition != Kind(len(kindNames)) {
		t.Errorf("%d kinds named, the last kind is %d", len(kindNames), Transition)
	}
}

var fenceRE = regexp.MustCompile("(?s)```(tw|tw-bad)\n(.*?)```")

// The classes the page shows as working parse, and those it shows as
// refused are refused for the reason it gives.
func TestCheatSheetExamples(t *testing.T) {
	data, err := os.ReadFile(sheetPath)
	if err != nil {
		t.Fatal(err)
	}
	good, bad := 0, 0
	for _, m := range fenceRE.FindAllStringSubmatch(string(data), -1) {
		for _, line := range strings.Split(strings.TrimSpace(m[2]), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			switch m[1] {
			case "tw":
				good++
				if _, b := Parse(line); len(b) > 0 {
					t.Errorf("the page shows %q, which parses with %v", line, b)
				}
			case "tw-bad":
				bad++
				class, reason, ok := strings.Cut(line, " | ")
				if !ok {
					t.Errorf("tw-bad line %q: want `class | reason`", line)
					continue
				}
				_, b := Parse(class)
				if len(b) != 1 || b[0].Reason != reason {
					t.Errorf("the page says %q is refused as %q, Parse says %v", class, reason, b)
				}
			}
		}
	}
	if good < 10 || bad < 10 {
		t.Errorf("the page has %d good and %d refused examples: did the blocks go?", good, bad)
	}
}
