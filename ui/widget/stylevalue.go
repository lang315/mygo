package widget

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Color is a fixed color, or a role of the theme read as each frame
// builds, so that it follows the theme, dark included. The zero Color is
// unset: a Style given one changes nothing.
type Color struct {
	fixed ui.Color
	role  role
	ok    bool
}

type role uint8

const (
	roleNone role = iota
	roleBackground
	roleSurface
	roleSurfaceHover
	roleSurfacePressed
	roleBorder
	roleText
	roleTextMuted
	roleAccent
	roleAccentHover
	roleAccentPressed
	roleAccentText
	roleDanger
	roleWarning
	roleSuccess
	roleSelection
	roleFocus
	roleInverse
	roleInverseText
)

// Fixed is the color c, whatever the theme.
func Fixed(c ui.Color) Color { return Color{fixed: c, ok: true} }

// Hex is Fixed(ui.Hex(s)).
func Hex(s string) Color { return Fixed(ui.Hex(s)) }

// The colors of the theme (ui.Theme), by role.
var (
	RoleBackground     = Color{role: roleBackground, ok: true}
	RoleSurface        = Color{role: roleSurface, ok: true}
	RoleSurfaceHover   = Color{role: roleSurfaceHover, ok: true}
	RoleSurfacePressed = Color{role: roleSurfacePressed, ok: true}
	RoleBorder         = Color{role: roleBorder, ok: true}
	RoleText           = Color{role: roleText, ok: true}
	RoleTextMuted      = Color{role: roleTextMuted, ok: true}
	RoleAccent         = Color{role: roleAccent, ok: true}
	RoleAccentHover    = Color{role: roleAccentHover, ok: true}
	RoleAccentPressed  = Color{role: roleAccentPressed, ok: true}
	RoleAccentText     = Color{role: roleAccentText, ok: true}
	RoleDanger         = Color{role: roleDanger, ok: true}
	RoleWarning        = Color{role: roleWarning, ok: true}
	RoleSuccess        = Color{role: roleSuccess, ok: true}
	RoleSelection      = Color{role: roleSelection, ok: true}
	RoleFocus          = Color{role: roleFocus, ok: true}
	RoleInverse        = Color{role: roleInverse, ok: true}
	RoleInverseText    = Color{role: roleInverseText, ok: true}
)

// resolve returns the color for the theme t.
func (c Color) resolve(t *ui.Theme) ui.Color {
	switch c.role {
	case roleBackground:
		return t.Background
	case roleSurface:
		return t.Surface
	case roleSurfaceHover:
		return t.SurfaceHover
	case roleSurfacePressed:
		return t.SurfacePressed
	case roleBorder:
		return t.Border
	case roleText:
		return t.Text
	case roleTextMuted:
		return t.TextMuted
	case roleAccent:
		return t.Accent
	case roleAccentHover:
		return t.AccentHover
	case roleAccentPressed:
		return t.AccentPressed
	case roleAccentText:
		return t.AccentText
	case roleDanger:
		return t.Danger
	case roleWarning:
		return t.Warning
	case roleSuccess:
		return t.Success
	case roleSelection:
		return t.Selection
	case roleFocus:
		return t.Focus
	case roleInverse: // left zero, tooltips use Text (docs/ui/styling.md)
		if t.Inverse == (ui.Color{}) {
			return t.Text
		}
		return t.Inverse
	case roleInverseText:
		if t.InverseText == (ui.Color{}) {
			return t.Background
		}
		return t.InverseText
	}
	return c.fixed
}

// prop is a property a Style sets, beside the sides of its padding,
// margin, radius and border width.
type prop uint32

const (
	pBorderStyle prop = 1 << iota
	pWidth
	pHeight
	pMinWidth
	pMaxWidth
	pMinHeight
	pMaxHeight
	pOpacity
	pCursor
	pBlur
	pShadows
	pGradient
	pBackground
	pBorderColor
	pTextColor
	pFontSize
	pItalic
	pMono
	pTransition
	pAlign
	pVAlign
	pHExpand
	pVExpand
	pFontWeight

	pEnd // after the last property: loops over the properties stop before it
)

// pLayout are the properties that place a control in its box. They count
// in the base style only, and a control with no other is not restyled.
const pLayout = pAlign | pVAlign | pHExpand | pVExpand

type shadow struct {
	x, y, blur, spread float32
	c                  Color
	inset              bool
}

type gradient struct {
	from, to Color
	angle    float32
}

// props are what a Style sets: set says which properties, and the 4-bit
// masks which sides (top, right, bottom, left) or corners (top-left,
// top-right, bottom-right, bottom-left) of the four-sided ones.
type props struct {
	set                                     prop
	padSet, marginSet, radiusSet, borderSet uint8
	pad, margin, radius, border             [4]float32
	borderStyle                             ui.BorderStyle
	width, height, minW, maxW, minH, maxH   float32
	opacity, blur                           float32
	cursor                                  ui.Cursor
	shadows                                 []shadow // never written once a Style holds it
	gradient                                gradient
	background, borderColor, text           Color
	textSize                                Size    // when fontSize is 0
	fontSize                                float32 // DIPs
	transition                              time.Duration
	align, valign                           Align
	weight                                  int
	// src says where the Style came from (Source), for the inspector.
	src *string
}

type variant uint8

// The variants, in the order they apply: a later one wins, as in
// Tailwind v3.
const (
	vDark variant = iota
	vHover
	vFocused
	vPressed
	vDisabled
	nVariants
)

// Style is how a control looks: box, color and text properties, and the
// variants that apply on top while a state holds. The zero Style sets
// nothing. A Style never changes once made: every method returns a new
// one, so one Style can style many controls, from any goroutine.
type Style struct {
	props
	v [nVariants]*Style
}

// NewStyle returns a Style that sets nothing, to chain setters onto.
func NewStyle() Style { return Style{} }

// sides sets *dst from v as CSS reads 1, 2 or 4 values (top, right,
// bottom, left; or corners from the top left); other counts change
// nothing. nonNeg clamps negative values to 0.
func sides(dst *[4]float32, mask *uint8, v []float32, nonNeg bool) {
	var four [4]float32
	switch len(v) {
	case 1:
		four = [4]float32{v[0], v[0], v[0], v[0]}
	case 2:
		four = [4]float32{v[0], v[1], v[0], v[1]}
	case 4:
		four = [4]float32{v[0], v[1], v[2], v[3]}
	default:
		return
	}
	for i := range four {
		side(dst, mask, i, four[i], nonNeg)
	}
}

func side(dst *[4]float32, mask *uint8, i int, x float32, nonNeg bool) {
	if nonNeg && x < 0 {
		x = 0
	}
	dst[i] = x
	*mask |= 1 << i
}

// Padding sets the room inside the control's edges, in theme units (4
// DIPs by default): 1, 2 or 4 values, as CSS.
func (s Style) Padding(v ...float32) Style { sides(&s.pad, &s.padSet, v, true); return s }
func (s Style) PaddingX(n float32) Style {
	side(&s.pad, &s.padSet, 1, n, true)
	side(&s.pad, &s.padSet, 3, n, true)
	return s
}
func (s Style) PaddingY(n float32) Style {
	side(&s.pad, &s.padSet, 0, n, true)
	side(&s.pad, &s.padSet, 2, n, true)
	return s
}
func (s Style) PaddingTop(n float32) Style    { side(&s.pad, &s.padSet, 0, n, true); return s }
func (s Style) PaddingRight(n float32) Style  { side(&s.pad, &s.padSet, 1, n, true); return s }
func (s Style) PaddingBottom(n float32) Style { side(&s.pad, &s.padSet, 2, n, true); return s }
func (s Style) PaddingLeft(n float32) Style   { side(&s.pad, &s.padSet, 3, n, true); return s }

// Margin sets the room around the control, in theme units; it may be
// negative, as in CSS.
func (s Style) Margin(v ...float32) Style { sides(&s.margin, &s.marginSet, v, false); return s }
func (s Style) MarginX(n float32) Style {
	side(&s.margin, &s.marginSet, 1, n, false)
	side(&s.margin, &s.marginSet, 3, n, false)
	return s
}
func (s Style) MarginY(n float32) Style {
	side(&s.margin, &s.marginSet, 0, n, false)
	side(&s.margin, &s.marginSet, 2, n, false)
	return s
}
func (s Style) MarginTop(n float32) Style    { side(&s.margin, &s.marginSet, 0, n, false); return s }
func (s Style) MarginRight(n float32) Style  { side(&s.margin, &s.marginSet, 1, n, false); return s }
func (s Style) MarginBottom(n float32) Style { side(&s.margin, &s.marginSet, 2, n, false); return s }
func (s Style) MarginLeft(n float32) Style   { side(&s.margin, &s.marginSet, 3, n, false); return s }

// Radius rounds the corners, in theme units: 1, 2 or 4 values from the
// top left, as CSS's border-radius.
func (s Style) Radius(r ...float32) Style { sides(&s.radius, &s.radiusSet, r, true); return s }

// BorderWidth sets the border's width in DIPs: 1, 2 or 4 values, as CSS.
// Its color is BorderColor's, or the theme's border color.
func (s Style) BorderWidth(v ...float32) Style { sides(&s.border, &s.borderSet, v, true); return s }

// RadiusTopLeft and the other corners round one corner, in theme units;
// the corners not set keep the control's.
func (s Style) RadiusTopLeft(r float32) Style  { side(&s.radius, &s.radiusSet, 0, r, true); return s }
func (s Style) RadiusTopRight(r float32) Style { side(&s.radius, &s.radiusSet, 1, r, true); return s }
func (s Style) RadiusBottomRight(r float32) Style {
	side(&s.radius, &s.radiusSet, 2, r, true)
	return s
}
func (s Style) RadiusBottomLeft(r float32) Style { side(&s.radius, &s.radiusSet, 3, r, true); return s }

// BorderTop and the other sides set one side's border width in DIPs; the
// sides not set keep the control's.
func (s Style) BorderTop(w float32) Style    { side(&s.border, &s.borderSet, 0, w, true); return s }
func (s Style) BorderRight(w float32) Style  { side(&s.border, &s.borderSet, 1, w, true); return s }
func (s Style) BorderBottom(w float32) Style { side(&s.border, &s.borderSet, 2, w, true); return s }
func (s Style) BorderLeft(w float32) Style   { side(&s.border, &s.borderSet, 3, w, true); return s }

func (s Style) BorderStyle(b ui.BorderStyle) Style {
	s.borderStyle, s.set = b, s.set|pBorderStyle
	return s
}

// Width, Height and their bounds are in DIPs.
func (s Style) Width(w float32) Style     { s.width, s.set = max(w, 0), s.set|pWidth; return s }
func (s Style) Height(h float32) Style    { s.height, s.set = max(h, 0), s.set|pHeight; return s }
func (s Style) MinWidth(w float32) Style  { s.minW, s.set = max(w, 0), s.set|pMinWidth; return s }
func (s Style) MaxWidth(w float32) Style  { s.maxW, s.set = max(w, 0), s.set|pMaxWidth; return s }
func (s Style) MinHeight(h float32) Style { s.minH, s.set = max(h, 0), s.set|pMinHeight; return s }
func (s Style) MaxHeight(h float32) Style { s.maxH, s.set = max(h, 0), s.set|pMaxHeight; return s }

// Opacity is from 0, transparent, to 1.
func (s Style) Opacity(o float32) Style {
	s.opacity, s.set = min(max(o, 0), 1), s.set|pOpacity
	return s
}
func (s Style) Cursor(c ui.Cursor) Style { s.cursor, s.set = c, s.set|pCursor; return s }

// Blur blurs the control and its children, sigma DIPs.
func (s Style) Blur(sigma float32) Style { s.blur, s.set = max(sigma, 0), s.set|pBlur; return s }

// Shadow adds a box shadow, as CSS's box-shadow: several stack.
func (s Style) Shadow(x, y, blur, spread float32, c Color) Style {
	return s.addShadow(shadow{x, y, max(blur, 0), spread, c, false})
}

// InsetShadow adds an inner shadow, as CSS's box-shadow: inset.
func (s Style) InsetShadow(x, y, blur, spread float32, c Color) Style {
	return s.addShadow(shadow{x, y, max(blur, 0), spread, c, true})
}

func (s Style) addShadow(sh shadow) Style {
	if !sh.c.ok {
		return s
	}
	// A new array: never append into one another Style holds.
	n := make([]shadow, len(s.shadows), len(s.shadows)+1)
	copy(n, s.shadows)
	s.shadows, s.set = append(n, sh), s.set|pShadows
	return s
}

// NoShadow removes the shadows, those of the styles under this one too.
func (s Style) NoShadow() Style { s.shadows, s.set = nil, s.set|pShadows; return s }

// Gradient fills the background with a linear gradient, angle degrees
// clockwise from upwards (90 goes left to right).
func (s Style) Gradient(from, to Color, angle float32) Style {
	if !from.ok || !to.ok {
		return s
	}
	s.gradient, s.set = gradient{from, to, angle}, s.set|pGradient
	return s
}

func (s Style) Background(c Color) Style {
	if c.ok {
		s.background, s.set = c, s.set|pBackground
	}
	return s
}
func (s Style) BorderColor(c Color) Style {
	if c.ok {
		s.borderColor, s.set = c, s.set|pBorderColor
	}
	return s
}
func (s Style) TextColor(c Color) Style {
	if c.ok {
		s.text, s.set = c, s.set|pTextColor
	}
	return s
}

// TextSize sizes the text relative to the theme's, as the control
// methods' TextSize; FontSize in DIPs.
func (s Style) TextSize(z Size) Style {
	s.textSize, s.fontSize, s.set = z, 0, s.set|pFontSize
	return s
}
func (s Style) FontSize(dips float32) Style {
	if dips > 0 {
		s.fontSize, s.set = dips, s.set|pFontSize
	}
	return s
}

// Bold is FontWeight(700), as in ui: the later of the two wins.
func (s Style) Bold() Style   { return s.FontWeight(700) }
func (s Style) Italic() Style { s.set |= pItalic; return s }
func (s Style) Mono() Style   { s.set |= pMono; return s }

// FontWeight sets the weight of the text, 100 (thin) to 900 (black).
func (s Style) FontWeight(w int) Style {
	s.weight, s.set = min(max(w, 100), 900), s.set|pFontWeight
	return s
}

// Align, VAlign, HExpand and VExpand place the control in its box, as the
// control's methods of the same names. They count in the base style only:
// a variant's are ignored.
func (s Style) Align(a Align) Style  { s.align, s.set = a, s.set|pAlign; return s }
func (s Style) VAlign(a Align) Style { s.valign, s.set = a, s.set|pVAlign; return s }
func (s Style) HExpand() Style       { s.set |= pHExpand; return s }
func (s Style) VExpand() Style       { s.set |= pVExpand; return s }

// layout returns how p places a control in its box.
func (p *props) layout() layout {
	l := layout{hexpand: p.set&pHExpand != 0, vexpand: p.set&pVExpand != 0}
	if p.set&pAlign != 0 {
		l.align = p.align
	}
	if p.set&pVAlign != 0 {
		l.valign = p.valign
	}
	return l
}

// Transition fades the background, border and text colors over d as they
// change, as a hover does; 0 turns it off. The text color fades only if
// the style sets a TextColor of its own, besides the one a variant sets:
// a Style cannot know the color a control's text rests in (a Button's is
// the theme's AccentText), so without it the text jumps.
func (s Style) Transition(d time.Duration) Style {
	s.transition, s.set = max(d, 0), s.set|pTransition
	return s
}

// The variants: v applies on top while the state holds. A state's
// variant may have a Dark variant of its own, for the dark theme (as
// Tailwind's dark:hover:); deeper variants are ignored.
func (s Style) Hover(v Style) Style    { return s.with(vHover, v) }
func (s Style) Focused(v Style) Style  { return s.with(vFocused, v) }
func (s Style) Pressed(v Style) Style  { return s.with(vPressed, v) }
func (s Style) Disabled(v Style) Style { return s.with(vDisabled, v) }
func (s Style) Dark(v Style) Style     { return s.with(vDark, v) }

func (s Style) with(k variant, v Style) Style { s.v[k] = &v; return s }

// Source notes where s came from, as the Tailwind classes that made it (tw.S
// does), for the inspector's Widget tab, which shows it beside the style. It
// changes nothing else. A Style made by Merge has no source: it is made of
// several. An empty from takes the source off, as to compare two Styles that
// differ in nothing else.
func (s Style) Source(from string) Style {
	s.src = nil
	if from != "" {
		s.src = &from
	}
	return s
}

// Merge returns s with what o sets on top, side by side and variant by
// variant: later styles win, as the order of Apply's arguments.
func (s Style) Merge(o Style) Style {
	s.src = nil // made of two
	s.props.over(&o.props)
	for k := range nVariants {
		switch {
		case o.v[k] == nil:
		case s.v[k] == nil:
			s.v[k] = o.v[k]
		default:
			m := s.v[k].Merge(*o.v[k])
			s.v[k] = &m
		}
	}
	return s
}

// empty reports whether s sets nothing.
func (s Style) empty() bool {
	if s.set != 0 || s.padSet|s.marginSet|s.radiusSet|s.borderSet != 0 {
		return false
	}
	for _, v := range s.v {
		if v != nil {
			return false
		}
	}
	return true
}

// over sets on p what o sets.
func (p *props) over(o *props) {
	for i := range 4 {
		bit := uint8(1) << i
		if o.padSet&bit != 0 {
			p.pad[i] = o.pad[i]
		}
		if o.marginSet&bit != 0 {
			p.margin[i] = o.margin[i]
		}
		if o.radiusSet&bit != 0 {
			p.radius[i] = o.radius[i]
		}
		if o.borderSet&bit != 0 {
			p.border[i] = o.border[i]
		}
	}
	p.padSet |= o.padSet
	p.marginSet |= o.marginSet
	p.radiusSet |= o.radiusSet
	p.borderSet |= o.borderSet
	s := o.set
	if s&pBorderStyle != 0 {
		p.borderStyle = o.borderStyle
	}
	if s&pWidth != 0 {
		p.width = o.width
	}
	if s&pHeight != 0 {
		p.height = o.height
	}
	if s&pMinWidth != 0 {
		p.minW = o.minW
	}
	if s&pMaxWidth != 0 {
		p.maxW = o.maxW
	}
	if s&pMinHeight != 0 {
		p.minH = o.minH
	}
	if s&pMaxHeight != 0 {
		p.maxH = o.maxH
	}
	if s&pOpacity != 0 {
		p.opacity = o.opacity
	}
	if s&pCursor != 0 {
		p.cursor = o.cursor
	}
	if s&pBlur != 0 {
		p.blur = o.blur
	}
	if s&pShadows != 0 {
		p.shadows = o.shadows
	}
	if s&pGradient != 0 {
		p.gradient = o.gradient
	}
	if s&pBackground != 0 {
		p.background = o.background
	}
	if s&pBorderColor != 0 {
		p.borderColor = o.borderColor
	}
	if s&pTextColor != 0 {
		p.text = o.text
	}
	if s&pFontSize != 0 {
		p.textSize, p.fontSize = o.textSize, o.fontSize
	}
	if s&pTransition != 0 {
		p.transition = o.transition
	}
	if s&pAlign != 0 {
		p.align = o.align
	}
	if s&pVAlign != 0 {
		p.valign = o.valign
	}
	if s&pFontWeight != 0 {
		p.weight = o.weight
	}
	p.set |= s
}

// effective returns the properties of s for a frame: its own with the
// variants whose states hold on top, in Tailwind's order. It allocates
// nothing.
func (s *Style) effective(dark, hover, focused, pressed, disabled bool) props {
	p := s.props
	on := [nVariants]bool{vDark: dark, vHover: hover, vFocused: focused, vPressed: pressed, vDisabled: disabled}
	for k := range nVariants {
		v := s.v[k]
		if !on[k] || v == nil {
			continue
		}
		p.over(&v.props)
		if dark && k != vDark && v.v[vDark] != nil {
			p.over(&v.v[vDark].props)
		}
	}
	return p
}
