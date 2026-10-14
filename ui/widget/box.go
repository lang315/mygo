package widget

import "github.com/egoist/mygo/ui"

// BoxWidget lays out widgets in a column (VBox) or a row (HBox).
type BoxWidget struct {
	common[BoxWidget]
	dir     direction
	spacing float32
	items   []boxItem
	buttons bool // a ButtonBox: items at the end
}

type boxItem struct {
	w                 Widget
	expand, fill, end bool
}

// wrapKey keys the room of an item packed to expand without filling it.
type wrapKey int

// VBox makes a column of widgets spacing DIPs apart.
func VBox(spacing int) *BoxWidget { return newBox(vertical, spacing) }

// HBox makes a row of widgets spacing DIPs apart.
func HBox(spacing int) *BoxWidget { return newBox(horizontal, spacing) }

// ButtonBox makes a row whose widgets sit at its end, as a dialog's
// buttons.
func ButtonBox() *BoxWidget {
	b := newBox(horizontal, 8)
	b.buttons = true
	return b
}

func newBox(d direction, spacing int) *BoxWidget {
	b := &BoxWidget{dir: d, spacing: float32(spacing)}
	b.init(b)
	return b
}

// Add adds widgets after the others at the start, at their natural size.
// A widget already in the box stays where it is.
func (b *BoxWidget) Add(ws ...Widget) *BoxWidget {
	change(func() {
		for _, w := range ws {
			b.add(boxItem{w: w})
		}
	})
	return b
}

// Pack adds w after the others at the start. With expand it takes a share
// of the free room; with fill it grows to fill that share, else it sits in
// the middle of it.
func (b *BoxWidget) Pack(w Widget, expand, fill bool) *BoxWidget {
	change(func() { b.add(boxItem{w: w, expand: expand, fill: fill}) })
	return b
}

// PackEnd adds w at the end, before the widgets packed at the end earlier.
func (b *BoxWidget) PackEnd(w Widget, expand, fill bool) *BoxWidget {
	change(func() { b.add(boxItem{w: w, expand: expand, fill: fill, end: true}) })
	return b
}

func (b *BoxWidget) add(it boxItem) {
	for _, have := range b.items {
		if have.w == it.w {
			return
		}
	}
	b.items = append(b.items, it)
}

// Remove takes w out of the box.
func (b *BoxWidget) Remove(w Widget) *BoxWidget {
	change(func() {
		for i, it := range b.items {
			if it.w == w {
				b.items = append(b.items[:i], b.items[i+1:]...)
				return
			}
		}
	})
	return b
}

// Clear takes every widget out of the box.
func (b *BoxWidget) Clear() *BoxWidget { change(func() { b.items = nil }); return b }

func (b *BoxWidget) build(c *ui.Context) ui.Element {
	var e ui.Element
	if b.dir == vertical {
		e = ui.Column(c)
	} else {
		e = ui.Row(c)
	}
	e.Gap(b.spacing)
	return e.Children(func() {
		outer := axis
		axis = b.dir
		defer func() { axis = outer }()
		if b.buttons {
			ui.Spacer(c)
			ui.Grid(c).Columns(max(len(b.items), 1)).GapX(b.spacing).Children(func() {
				for _, it := range b.items {
					render(c, it.w)
				}
			})
			return
		}
		grows := false
		for _, it := range b.items {
			if !it.end && it.expand {
				grows = true
			}
		}
		for _, it := range b.items {
			if !it.end {
				b.item(c, it)
			}
		}
		end := false
		for _, it := range b.items {
			end = end || it.end
		}
		if end && !grows {
			ui.Spacer(c)
		}
		for i := len(b.items) - 1; i >= 0; i-- {
			if b.items[i].end {
				b.item(c, b.items[i])
			}
		}
	})
}

func (b *BoxWidget) item(c *ui.Context, it boxItem) {
	switch {
	case !it.expand || it.w.base().hidden: // a hidden widget takes no room
		render(c, it.w)
	case it.fill:
		render(c, it.w).Grow(1)
	default:
		ui.Box(c.Key(wrapKey(it.w.base().id))).Grow(1).AlignItems(ui.Center).Justify(ui.Center).
			Children(func() { render(c, it.w) })
	}
}
