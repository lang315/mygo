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

// style is how a widget looks. Lengths are in theme units (Space).
type style struct {
	pad, margin               [4]float32 // top, right, bottom, left
	align, valign             Align
	hexpand, vexpand          bool
	size                      Size
	bold, italic, mono, muted bool
	color, background         *ui.Color
	border                    float32
	borderColor               *ui.Color
	radius                    float32
	width, height             float32 // DIPs; 0 is the natural size
}

// apply styles e, a child of a box laid out along axis. Under mu.
func (s *style) apply(c *ui.Context, e ui.Element) {
	t := c.Theme()
	sp := func(v [4]float32) []float32 {
		return []float32{t.Space(v[0]), t.Space(v[1]), t.Space(v[2]), t.Space(v[3])}
	}
	if s.pad != [4]float32{} {
		e.Padding(sp(s.pad)...)
	}
	if s.margin != [4]float32{} {
		e.Margin(sp(s.margin)...)
	}
	across, grow := s.align, s.vexpand
	if axis == horizontal {
		across, grow = s.valign, s.hexpand
	}
	if across != 0 {
		e.AlignSelf(across.ui())
	}
	if grow {
		e.Grow(1)
	}
	if s.size != Base {
		e.FontSize(t.FontSize * sizeScale[s.size])
	}
	if s.bold {
		e.Bold()
	}
	if s.italic {
		e.Italic()
	}
	if s.mono {
		e.Font("monospace")
	}
	if s.muted {
		e.TextColor(t.TextMuted)
	}
	if s.color != nil {
		e.TextColor(*s.color)
	}
	if s.background != nil {
		e.Background(*s.background)
	}
	if s.border > 0 {
		col := t.Border
		if s.borderColor != nil {
			col = *s.borderColor
		}
		e.Border(s.border, col)
	}
	if s.radius > 0 {
		e.Radius(t.Space(s.radius))
	}
	if s.width > 0 {
		e.Width(s.width)
	}
	if s.height > 0 {
		e.Height(s.height)
	}
}

func (c *common[W]) style(fn func(s *style)) *W { c.change(func() { fn(&c.st) }); return c.self }

// Padding sets the room inside the widget's edges on every side, in theme
// units (4 DIPs): gota's p-2 is Padding(2).
func (c *common[W]) Padding(n float32) *W {
	return c.style(func(s *style) { s.pad = [4]float32{n, n, n, n} })
}
func (c *common[W]) PaddingX(n float32) *W {
	return c.style(func(s *style) { s.pad[1], s.pad[3] = n, n })
}
func (c *common[W]) PaddingY(n float32) *W {
	return c.style(func(s *style) { s.pad[0], s.pad[2] = n, n })
}
func (c *common[W]) PaddingTop(n float32) *W    { return c.style(func(s *style) { s.pad[0] = n }) }
func (c *common[W]) PaddingRight(n float32) *W  { return c.style(func(s *style) { s.pad[1] = n }) }
func (c *common[W]) PaddingBottom(n float32) *W { return c.style(func(s *style) { s.pad[2] = n }) }
func (c *common[W]) PaddingLeft(n float32) *W   { return c.style(func(s *style) { s.pad[3] = n }) }

// Margin sets the room around the widget on every side, in theme units:
// gota's mt-3 is MarginTop(3).
func (c *common[W]) Margin(n float32) *W {
	return c.style(func(s *style) { s.margin = [4]float32{n, n, n, n} })
}
func (c *common[W]) MarginX(n float32) *W {
	return c.style(func(s *style) { s.margin[1], s.margin[3] = n, n })
}
func (c *common[W]) MarginY(n float32) *W {
	return c.style(func(s *style) { s.margin[0], s.margin[2] = n, n })
}
func (c *common[W]) MarginTop(n float32) *W    { return c.style(func(s *style) { s.margin[0] = n }) }
func (c *common[W]) MarginRight(n float32) *W  { return c.style(func(s *style) { s.margin[1] = n }) }
func (c *common[W]) MarginBottom(n float32) *W { return c.style(func(s *style) { s.margin[2] = n }) }
func (c *common[W]) MarginLeft(n float32) *W   { return c.style(func(s *style) { s.margin[3] = n }) }

// Align places the widget horizontally in a VBox: gota's halign-start is
// Align(Start).
func (c *common[W]) Align(a Align) *W { return c.style(func(s *style) { s.align = a }) }

// VAlign places the widget vertically in an HBox.
func (c *common[W]) VAlign(a Align) *W { return c.style(func(s *style) { s.valign = a }) }

// HExpand gives the widget the free width of its HBox.
func (c *common[W]) HExpand() *W { return c.style(func(s *style) { s.hexpand = true }) }

// VExpand gives the widget the free height of its VBox.
func (c *common[W]) VExpand() *W { return c.style(func(s *style) { s.vexpand = true }) }

// Expand is HExpand and VExpand.
func (c *common[W]) Expand() *W {
	return c.style(func(s *style) { s.hexpand, s.vexpand = true, true })
}

// TextSize sets the size of the text: gota's text-xl is TextSize(XL).
func (c *common[W]) TextSize(z Size) *W { return c.style(func(s *style) { s.size = z }) }
func (c *common[W]) Bold() *W           { return c.style(func(s *style) { s.bold = true }) }
func (c *common[W]) Italic() *W         { return c.style(func(s *style) { s.italic = true }) }

// Mono draws the text in the system's monospace font.
func (c *common[W]) Mono() *W { return c.style(func(s *style) { s.mono = true }) }

// Muted draws the text in the theme's color for secondary text.
func (c *common[W]) Muted() *W { return c.style(func(s *style) { s.muted = true }) }

func (c *common[W]) TextColor(col ui.Color) *W {
	return c.style(func(s *style) { s.color = &col })
}
func (c *common[W]) Background(col ui.Color) *W {
	return c.style(func(s *style) { s.background = &col })
}

// Border outlines the widget, width DIPs wide.
func (c *common[W]) Border(width float32, col ui.Color) *W {
	return c.style(func(s *style) { s.border, s.borderColor = width, &col })
}

// Radius rounds the widget's corners by n theme units.
func (c *common[W]) Radius(n float32) *W { return c.style(func(s *style) { s.radius = n }) }

// SetSize sets the widget's width and height in DIPs; 0 keeps the natural
// size.
func (c *common[W]) SetSize(w, h float32) *W {
	return c.style(func(s *style) { s.width, s.height = w, h })
}
