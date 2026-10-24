package ui

// Radii returns the radii of the element's corners in DIPs, from the top
// left clockwise, as its builder set them so far in this frame: for a
// layer that changes some corners and keeps the others.
func (h Element) Radii() [4]float32 {
	if n := h.node(); n != nil {
		return n.radius
	}
	return [4]float32{}
}

// BorderWidths returns the widths of the element's border in DIPs (top,
// right, bottom, left), as Radii does.
func (h Element) BorderWidths() [4]float32 {
	if n := h.node(); n != nil {
		return n.border
	}
	return [4]float32{}
}

// Paddings returns the element's padding in DIPs (top, right, bottom,
// left), as Radii does.
func (h Element) Paddings() [4]float32 {
	if n := h.node(); n != nil {
		return n.pad
	}
	return [4]float32{}
}

// Margins returns the element's margins in DIPs (top, right, bottom,
// left), as Radii does.
func (h Element) Margins() [4]float32 {
	if n := h.node(); n != nil {
		return n.margin
	}
	return [4]float32{}
}
