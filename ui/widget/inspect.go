package widget

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// The note a control leaves for the inspector (ui.Context.Inspecting): its
// type, whether it is disabled, and the Styles that style it, layer by layer
// (Apply's, then its own methods'), each property set, by variant. The fields
// are made when the note is shown, for the one control chosen, not in each
// frame.

// notesLeft counts the notes left, for the tests that they are left only
// while the inspector is open.
var notesLeft atomic.Int64

// leaveNote leaves w's note on its element e. Call it only if c.Inspecting().
func (b *core) leaveNote(c *ui.Context, w Control, e ui.Element) {
	notesLeft.Add(1)
	th := c.Theme()
	e.InspectNote(fmt.Sprintf("%T", w), func() []ui.InspectorField { return b.inspect(w, th, e.IsDisabled()) })
}

// inspect returns the fields of w's note. It runs when the note is shown,
// with no view building. inert is whether the element is disabled, by itself
// or by an ancestor.
func (b *core) inspect(w Control, th *ui.Theme, inert bool) []ui.InspectorField {
	mu.Lock()
	defer mu.Unlock()
	disabled := strconv.FormatBool(b.disabled)
	if inert && !b.disabled {
		disabled += " (inherited: true)"
	}
	fs := []ui.InspectorField{
		{Name: "Control"},
		{Name: "type", Value: fmt.Sprintf("%T", w)},
		{Name: "disabled", Value: disabled},
	}
	layers := 0
	for i, s := range b.applied {
		fs = append(fs, ui.InspectorField{Name: fmt.Sprintf("Apply, style %d of %d", i+1, len(b.applied))})
		fs = append(fs, s.describe(th)...)
		layers++
	}
	if !b.ownStyle.empty() {
		fs = append(fs, ui.InspectorField{Name: "Own methods (Padding, Bold, …)"})
		fs = append(fs, b.ownStyle.describe(th)...)
		layers++
	}
	if layers == 0 {
		fs = append(fs, ui.InspectorField{Name: "No Style"})
	}
	return fs
}

var variantNames = [nVariants]string{vDark: "dark", vHover: "hover", vFocused: "focus", vPressed: "pressed", vDisabled: "disabled"}

// describe lists what s sets, then its variants, in the order they apply:
// each as it is set, not as it resolves.
func (s Style) describe(th *ui.Theme) []ui.InspectorField {
	out := s.props.describe(th, "")
	for k := range nVariants {
		v := s.v[k]
		if v == nil {
			continue
		}
		out = append(out, v.props.describe(th, variantNames[k]+":")...)
		if k != vDark && v.v[vDark] != nil {
			out = append(out, v.v[vDark].props.describe(th, variantNames[k]+":dark:")...)
		}
	}
	return out
}

var (
	edgeNames   = [4]string{"top", "right", "bottom", "left"}
	cornerNames = [4]string{"top-left", "top-right", "bottom-right", "bottom-left"}
)

// describe lists the properties p sets: those of set and the sides of its
// four-sided ones, whose masks say which, each with its unit.
func (p *props) describe(th *ui.Theme, prefix string) (out []ui.InspectorField) {
	add := func(name, value string) {
		out = append(out, ui.InspectorField{Name: prefix + name, Value: value})
	}
	dips := func(v float32) string { return num(v) + " DIPs" }
	units := func(v float32) string { return num(v) + " units (" + dips(th.Space(v)) + ")" }
	// The layout properties apply only from the base: a variant's are ignored.
	inVariant := ""
	if prefix != "" {
		inVariant = " (no effect in a variant)"
	}
	four := func(name string, mask uint8, v [4]float32, names [4]string, format func(float32) string) {
		if mask == 0b1111 && v[0] == v[1] && v[1] == v[2] && v[2] == v[3] {
			add(name, format(v[0]))
			return
		}
		for i := range 4 {
			if mask&(1<<i) != 0 {
				add(name+"-"+names[i], format(v[i]))
			}
		}
	}
	four("padding", p.padSet, p.pad, edgeNames, units)
	four("margin", p.marginSet, p.margin, edgeNames, units)
	four("radius", p.radiusSet, p.radius, cornerNames, units)
	four("border-width", p.borderSet, p.border, edgeNames, dips)
	for bit := prop(1); bit < pEnd; bit <<= 1 {
		if p.set&bit == 0 {
			continue
		}
		switch bit {
		case pBorderStyle:
			add("border-style", map[ui.BorderStyle]string{ui.BorderSolid: "solid", ui.BorderDashed: "dashed"}[p.borderStyle])
		case pWidth:
			add("width", dips(p.width))
		case pHeight:
			add("height", dips(p.height))
		case pMinWidth:
			add("min-width", dips(p.minW))
		case pMaxWidth:
			add("max-width", dips(p.maxW))
		case pMinHeight:
			add("min-height", dips(p.minH))
		case pMaxHeight:
			add("max-height", dips(p.maxH))
		case pOpacity:
			add("opacity", num(p.opacity))
		case pCursor:
			add("cursor", cursorName(p.cursor))
		case pBlur:
			add("blur", dips(p.blur))
		case pShadows:
			if len(p.shadows) == 0 {
				add("shadow", "none")
			}
			for _, sh := range p.shadows {
				inset := ""
				if sh.inset {
					inset = " inset"
				}
				add("shadow", fmt.Sprintf("x %s, y %s, blur %s, spread %s, %s%s", dips(sh.x), dips(sh.y), dips(sh.blur), dips(sh.spread), sh.c.inspectText(), inset))
			}
		case pGradient:
			add("gradient", fmt.Sprintf("%s to %s at %s°", p.gradient.from.inspectText(), p.gradient.to.inspectText(), num(p.gradient.angle)))
		case pBackground:
			add("background", p.background.inspectText())
		case pBorderColor:
			add("border-color", p.borderColor.inspectText())
		case pTextColor:
			add("color", p.text.inspectText())
		case pFontSize:
			if p.fontSize > 0 {
				add("font-size", dips(p.fontSize))
			} else {
				z := p.textSize
				if int(z) >= len(sizeNames) {
					z = Base // as apply does
				}
				add("font-size", sizeNames[z])
			}
		case pItalic:
			add("font-style", "italic")
		case pMono:
			add("font-family", "monospace")
		case pTransition:
			add("transition", p.transition.String())
		case pAlign:
			add("align", alignName(p.align)+inVariant)
		case pVAlign:
			add("valign", alignName(p.valign)+inVariant)
		case pHExpand:
			add("hexpand", "true"+inVariant)
		case pVExpand:
			add("vexpand", "true"+inVariant)
		case pFontWeight:
			add("font-weight", strconv.Itoa(p.weight))
		}
	}
	return out
}

var cursorNames = map[ui.Cursor]string{
	ui.CursorDefault: "default", ui.CursorPointer: "pointer", ui.CursorText: "text", ui.CursorMove: "move",
	ui.CursorResizeEW: "ew-resize", ui.CursorResizeNS: "ns-resize", ui.CursorResizeNWSE: "nwse-resize",
	ui.CursorResizeNESW: "nesw-resize", ui.CursorNotAllowed: "not-allowed", ui.CursorCrosshair: "crosshair",
	ui.CursorGrab: "grab", ui.CursorGrabbing: "grabbing", ui.CursorResizeN: "n-resize", ui.CursorResizeE: "e-resize",
	ui.CursorResizeS: "s-resize", ui.CursorResizeW: "w-resize", ui.CursorResizeColumn: "col-resize",
	ui.CursorResizeRow: "row-resize", ui.CursorVerticalText: "vertical-text", ui.CursorCopy: "copy",
	ui.CursorAlias: "alias", ui.CursorContextMenu: "context-menu", ui.CursorNone: "none",
}

func cursorName(c ui.Cursor) string {
	if n, ok := cursorNames[c]; ok {
		return n
	}
	return strconv.Itoa(int(c))
}

var sizeNames = [...]string{Base: "base", XS: "xs", SM: "sm", LG: "lg", XL: "xl", XXL: "2xl", XXXL: "3xl"}

func alignName(a Align) string {
	switch a {
	case Start:
		return "start"
	case Center:
		return "center"
	case End:
		return "end"
	case Fill:
		return "fill"
	}
	return "default"
}

// num writes v without trailing zeros.
func num(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }

var colorRoleNames = [...]string{
	roleBackground: "background", roleSurface: "surface", roleSurfaceHover: "surface-hover", roleSurfacePressed: "surface-pressed",
	roleBorder: "border", roleText: "text", roleTextMuted: "text-muted", roleAccent: "accent", roleAccentHover: "accent-hover",
	roleAccentPressed: "accent-pressed", roleAccentText: "accent-text", roleDanger: "danger", roleWarning: "warning",
	roleSuccess: "success", roleSelection: "selection", roleFocus: "focus", roleInverse: "inverse", roleInverseText: "inverse-text",
}

// inspectText names a theme role (as the theme's field), or writes a fixed
// color as #rrggbb, with its alpha when it is not 255.
func (c Color) inspectText() string {
	switch {
	case !c.ok:
		return "unset"
	case c.role != roleNone && int(c.role) < len(colorRoleNames):
		return "role " + colorRoleNames[c.role]
	}
	f := c.fixed
	s := fmt.Sprintf("#%02x%02x%02x", f.R, f.G, f.B)
	if f.A != 255 {
		s += fmt.Sprintf("%02x", f.A)
	}
	return strings.ToLower(s)
}
