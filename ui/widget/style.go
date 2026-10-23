package widget

import "github.com/egoist/mygo/ui"

// Align places a widget across its box: horizontally in a VBox,
// vertically in an HBox. The zero value keeps the box's default.
type Align uint8

const (
	Start Align = iota + 1
	Center
	End
	Fill
)

func (a Align) ui() ui.Align {
	switch a {
	case Center:
		return ui.Center
	case End:
		return ui.End
	case Fill:
		return ui.Stretch
	}
	return ui.Start
}

// Size is a text size, relative to the theme's: gota's text-xs to text-3xl.
type Size uint8

const (
	Base Size = iota
	XS
	SM
	LG
	XL
	XXL
	XXXL
)

var sizeScale = [...]float32{Base: 1, XS: 0.75, SM: 0.875, LG: 1.125, XL: 1.25, XXL: 1.5, XXXL: 1.875}

// layout places a control in its box: across the box, and growing along
// it. It depends on the box's axis, so it is not part of a Style.
type layout struct {
	align, valign    Align
	hexpand, vexpand bool
}

// apply places e, a child of a box laid out along axis. Under mu.
func (l *layout) apply(e ui.Element) {
	across, grow := l.align, l.vexpand
	if axis == horizontal {
		across, grow = l.valign, l.hexpand
	}
	if across != 0 {
		e.AlignSelf(across.ui())
	}
	if grow {
		e.Grow(1)
	}
}

// apply sets on e what p sets: the four-sided properties all four sides
// at once (ui sets them so), those p does not set 0. key is the control's,
// for its text color's animation. Under mu.
func (p *props) apply(c *ui.Context, e ui.Element, key any) {
	t := c.Theme()
	units := func(v [4]float32) (float32, float32, float32, float32) {
		return t.Space(v[0]), t.Space(v[1]), t.Space(v[2]), t.Space(v[3])
	}
	if p.padSet != 0 {
		a, b, cc, d := units(p.pad)
		e.Padding(a, b, cc, d)
	}
	if p.marginSet != 0 {
		a, b, cc, d := units(p.margin)
		e.Margin(a, b, cc, d)
	}
	if p.radiusSet != 0 {
		a, b, cc, d := units(p.radius)
		e.Radius(a, b, cc, d)
	}
	if p.borderSet != 0 {
		e.BorderWidth(p.border[0], p.border[1], p.border[2], p.border[3])
		col := t.Border
		if p.set&pBorderColor != 0 {
			col = p.borderColor.resolve(t)
		}
		e.BorderColor(col)
	} else if p.set&pBorderColor != 0 {
		e.BorderColor(p.borderColor.resolve(t))
	}
	s := p.set
	if s&pBorderStyle != 0 {
		e.BorderStyle(p.borderStyle)
	}
	if s&pWidth != 0 {
		e.Width(p.width)
	}
	if s&pHeight != 0 {
		e.Height(p.height)
	}
	if s&pMinWidth != 0 {
		e.MinWidth(p.minW)
	}
	if s&pMaxWidth != 0 {
		e.MaxWidth(p.maxW)
	}
	if s&pMinHeight != 0 {
		e.MinHeight(p.minH)
	}
	if s&pMaxHeight != 0 {
		e.MaxHeight(p.maxH)
	}
	if s&pOpacity != 0 {
		e.Opacity(p.opacity)
	}
	if s&pCursor != 0 {
		e.Cursor(p.cursor)
	}
	if s&pBlur != 0 && p.blur > 0 {
		e.Blur(p.blur)
	}
	for _, sh := range p.shadows {
		if sh.inset {
			e.InsetShadow(sh.x, sh.y, sh.blur, sh.spread, sh.c.resolve(t))
		} else {
			e.Shadow(sh.x, sh.y, sh.blur, sh.spread, sh.c.resolve(t))
		}
	}
	if s&pBackground != 0 {
		e.Background(p.background.resolve(t))
	}
	if s&pGradient != 0 {
		e.Gradient(p.gradient.from.resolve(t), p.gradient.to.resolve(t), p.gradient.angle)
	}
	if s&pFontSize != 0 {
		size := p.fontSize
		if size == 0 {
			z := p.textSize
			if int(z) >= len(sizeScale) {
				z = Base
			}
			size = t.FontSize * sizeScale[z]
		}
		e.FontSize(size)
	}
	if s&pBold != 0 {
		e.Bold()
	}
	if s&pItalic != 0 {
		e.Italic()
	}
	if s&pMono != 0 {
		e.Font("monospace")
	}
	if s&pTextColor != 0 {
		col := p.text.resolve(t)
		if p.transition > 0 {
			col = e.AnimateColor(key, col, ui.Animation{Duration: p.transition})
		}
		e.TextColor(col)
	}
	if p.transition > 0 {
		e.Transition(ui.ElementTransition{Colors: true, Duration: p.transition})
	}
}

// textKey keys a control's text color animation.
type textKey int

// own runs fn on the control's own style, which applies over its Apply
// styles, as an inline style over classes.
func (c *common[W]) own(fn func(s Style) Style) *W {
	c.change(func() { c.ownStyle = fn(c.ownStyle); c.restyle() })
	return c.self
}

// lay changes how the control is placed in its box.
func (c *common[W]) lay(fn func(l *layout)) *W {
	c.change(func() { fn(&c.place) })
	return c.self
}

// Apply styles the control with styles, in order: a later one wins, and
// the control's own methods (Padding, Bold, …) win over them all. Apply
// replaces the styles given before; Apply() removes them.
func (c *common[W]) Apply(styles ...Style) *W {
	cp := append([]Style(nil), styles...)
	c.change(func() { c.applied = cp; c.restyle() })
	return c.self
}

// Padding sets the room inside the widget's edges on every side, in theme
// units (4 DIPs): gota's p-2 is Padding(2).
func (c *common[W]) Padding(n float32) *W { return c.own(func(s Style) Style { return s.Padding(n) }) }
func (c *common[W]) PaddingX(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingX(n) })
}
func (c *common[W]) PaddingY(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingY(n) })
}
func (c *common[W]) PaddingTop(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingTop(n) })
}
func (c *common[W]) PaddingRight(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingRight(n) })
}
func (c *common[W]) PaddingBottom(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingBottom(n) })
}
func (c *common[W]) PaddingLeft(n float32) *W {
	return c.own(func(s Style) Style { return s.PaddingLeft(n) })
}

// Margin sets the room around the widget on every side, in theme units:
// gota's mt-3 is MarginTop(3).
func (c *common[W]) Margin(n float32) *W { return c.own(func(s Style) Style { return s.Margin(n) }) }
func (c *common[W]) MarginX(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginX(n) })
}
func (c *common[W]) MarginY(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginY(n) })
}
func (c *common[W]) MarginTop(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginTop(n) })
}
func (c *common[W]) MarginRight(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginRight(n) })
}
func (c *common[W]) MarginBottom(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginBottom(n) })
}
func (c *common[W]) MarginLeft(n float32) *W {
	return c.own(func(s Style) Style { return s.MarginLeft(n) })
}

// Align places the widget horizontally in a VBox: gota's halign-start is
// Align(Start).
func (c *common[W]) Align(a Align) *W { return c.lay(func(l *layout) { l.align = a }) }

// VAlign places the widget vertically in an HBox.
func (c *common[W]) VAlign(a Align) *W { return c.lay(func(l *layout) { l.valign = a }) }

// HExpand gives the widget the free width of its HBox.
func (c *common[W]) HExpand() *W { return c.lay(func(l *layout) { l.hexpand = true }) }

// VExpand gives the widget the free height of its VBox.
func (c *common[W]) VExpand() *W { return c.lay(func(l *layout) { l.vexpand = true }) }

// Expand is HExpand and VExpand.
func (c *common[W]) Expand() *W {
	return c.lay(func(l *layout) { l.hexpand, l.vexpand = true, true })
}

// TextSize sets the size of the text: gota's text-xl is TextSize(XL).
func (c *common[W]) TextSize(z Size) *W { return c.own(func(s Style) Style { return s.TextSize(z) }) }
func (c *common[W]) Bold() *W           { return c.own(func(s Style) Style { return s.Bold() }) }
func (c *common[W]) Italic() *W         { return c.own(func(s Style) Style { return s.Italic() }) }

// Mono draws the text in the system's monospace font.
func (c *common[W]) Mono() *W { return c.own(func(s Style) Style { return s.Mono() }) }

// Muted draws the text in the theme's color for secondary text.
func (c *common[W]) Muted() *W {
	return c.own(func(s Style) Style { return s.TextColor(RoleTextMuted) })
}

func (c *common[W]) TextColor(col ui.Color) *W {
	return c.own(func(s Style) Style { return s.TextColor(Fixed(col)) })
}
func (c *common[W]) Background(col ui.Color) *W {
	return c.own(func(s Style) Style { return s.Background(Fixed(col)) })
}

// Border outlines the widget, width DIPs wide.
func (c *common[W]) Border(width float32, col ui.Color) *W {
	return c.own(func(s Style) Style { return s.BorderWidth(width).BorderColor(Fixed(col)) })
}

// Radius rounds the widget's corners by n theme units.
func (c *common[W]) Radius(n float32) *W { return c.own(func(s Style) Style { return s.Radius(n) }) }

// SetSize sets the widget's width and height in DIPs. A 0 leaves that
// dimension as it was: the natural size if it was never set, and there is
// no way back to it once it is.
func (c *common[W]) SetSize(w, h float32) *W {
	return c.own(func(s Style) Style {
		if w > 0 {
			s = s.Width(w)
		}
		if h > 0 {
			s = s.Height(h)
		}
		return s
	})
}
