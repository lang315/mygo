package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// GalleryItem is an item of a ribbon's gallery: its text, under its image
// when it has one.
type GalleryItem struct {
	Text  string
	Image *ui.Bitmap
}

// RibbonGalleryWidget is a gallery in a ribbon's group, as Office's Styles:
// a row of items, arrows that scroll it a row at a time, and a button that
// opens all the items in a grid under it. AddGallery makes one.
type RibbonGalleryWidget struct {
	common[RibbonGalleryWidget]
	items    []GalleryItem
	columns  int
	sel      int // the item selected, -1 for none
	row      int // the row the strip shows
	onSelect func(i int)
	onHover  func(i int)
	// What only the drawing thread touches, under mu while building.
	open      bool // the drop-down shows; ui's Popover writes it
	wasOpen   bool // it showed in the last build
	seenHides int  // the hides the last build saw: a hide closes the drop-down
	hl        int  // the item the drop-down's arrows are on, -1 for none
	moved     bool // the arrows or the pointer moved hl since the drop-down opened
	popHov    int  // the item of the drop-down under the pointer in the last build
	stripHov  int  // the item of the strip under the pointer in this build
	hover     int  // what OnHover was last given
	cur       int  // the item face builds
	c         *ui.Context
	panelEl   ui.Element // the drop-down's panel, whose active descendant hl is
	// The functions handed to ui, made once.
	strip, arrows, face, grid, cells, upFace, downFace, moreFace func()
	panel                                                        func(panel ui.Element)
}

// AddGallery adds a gallery of items to the ribbon's last group (a new one
// with no title before any), columns items wide and one row high. Its
// arrows scroll it a row at a time, and its drop-down button, or its key
// tip, opens all the items in a grid under it, which the arrow keys move in,
// Enter picks from and Escape closes. Nothing is selected at first.
func (r *RibbonWidget) AddGallery(items []GalleryItem, columns int) *RibbonGalleryWidget {
	g := &RibbonGalleryWidget{items: slices.Clone(items), columns: max(columns, 1), sel: -1, hl: -1, popHov: -1, hover: -1}
	g.init(g)
	g.strip, g.arrows, g.face = g.buildStrip, g.buildArrows, g.buildFace
	g.grid, g.cells, g.panel = g.buildGrid, g.buildCells, g.buildPanel
	g.upFace, g.downFace, g.moreFace = g.buildUp, g.buildDown, g.buildMore
	r.change(func() {
		r.galleryTip = galleryTip
		r.add(g, true)
	})
	return g
}

// Selected returns the index of the item selected, -1 for none.
func (g *RibbonGalleryWidget) Selected() int { mu.Lock(); defer mu.Unlock(); return g.sel }

// SetSelected selects item i, none when i is out of range, and shows its
// row, without running OnSelect.
func (g *RibbonGalleryWidget) SetSelected(i int) *RibbonGalleryWidget {
	g.change(func() { g.selectItem(i) })
	return g
}

// SetItems replaces the items. The selection stays where its item still
// is, else it is none.
func (g *RibbonGalleryWidget) SetItems(items []GalleryItem) *RibbonGalleryWidget {
	g.change(func() {
		g.items = slices.Clone(items)
		if g.sel >= len(g.items) {
			g.sel = -1
		}
		if g.hl >= len(g.items) {
			g.hl = -1
		}
		g.row = min(g.row, max(g.rows()-1, 0))
	})
	return g
}

// OnSelect runs fn with the item's index, on the event goroutine, after the
// user picks an item: a click in the strip or the drop-down, or Enter in the
// drop-down. A setter does not run it.
func (g *RibbonGalleryWidget) OnSelect(fn func(i int)) *RibbonGalleryWidget {
	g.change(func() { g.onSelect = fn })
	return g
}

// OnHover runs fn, on the event goroutine, as the pointer or the drop-down's
// arrow keys move over an item, with the item's index, for a live preview,
// and with -1 when they leave the items, or the drop-down closes.
func (g *RibbonGalleryWidget) OnHover(fn func(i int)) *RibbonGalleryWidget {
	g.change(func() { g.onHover = fn })
	return g
}

// selectItem selects item i, none out of range, and shows its row. Under mu.
func (g *RibbonGalleryWidget) selectItem(i int) {
	if i < 0 || i >= len(g.items) {
		g.sel = -1
		return
	}
	g.sel, g.row = i, i/g.columns
}

// rows is how many rows the items fill. Under mu.
func (g *RibbonGalleryWidget) rows() int { return (len(g.items) + g.columns - 1) / g.columns }

// galleryTip opens the drop-down of tp's command when it is an enabled
// gallery, and reports whether it did: a disabled one ends the tips as any
// disabled command does. Under mu (galleryTip).
func galleryTip(tp ribbonTip) bool {
	g, ok := tp.w.(*RibbonGalleryWidget)
	if !ok || tp.el.IsDisabled() {
		return false
	}
	g.open = true
	return true
}

func (g *RibbonGalleryWidget) build(c *ui.Context) ui.Element {
	if g.seenHides != g.hides { // out of sight since the last build: it closes
		g.open, g.seenHides = false, g.hides
	}
	if g.disabled { // the drop-down is an overlay, out of the disabled box
		g.open = false
	}
	g.c = c
	t := c.Theme()
	box := ui.Row(c).Role(ui.RoleGroup).Border(1, t.Border).Radius(t.Radius).AlignItems(ui.Stretch).Clip()
	g.stripHov = -1
	box.Children(g.strip)
	if g.open && !g.wasOpen {
		g.hl, g.moved, g.popHov = g.sel, false, -1
	}
	if g.open {
		ui.PopoverBase(c, box, &g.open, g.panel)
	}
	if !c.Measuring() {
		g.hovered()
		g.wasOpen = g.open
	}
	return box
}

// hovered posts OnHover when the item under the pointer, or the drop-down's
// arrows, changed. Under mu, at the end of a build for the window.
func (g *RibbonGalleryWidget) hovered() {
	want := g.stripHov
	if g.open {
		want = -1
		if g.moved {
			want = g.hl
		}
	}
	if want == g.hover {
		return
	}
	g.hover = want
	if fn := g.onHover; fn != nil {
		Post(func() { fn(want) })
	}
}

// itemWidth is the width of an item, in the strip and the drop-down.
func itemWidth(t *ui.Theme) float32 { return t.Rem(4.5) }

// buildStrip builds the strip: the row's items, then the arrows. Under mu,
// building.
func (g *RibbonGalleryWidget) buildStrip() {
	c, t := g.c, g.c.Theme()
	ui.Row(c).Gap(t.Space(0.5)).Padding(t.Space(0.5)).Width(float32(g.columns)*(itemWidth(t)+t.Space(0.5)) + t.Space(0.5)).
		AlignItems(ui.Stretch).Children(g.cells)
	ui.Column(c).BorderStartWidth(1).BorderColor(t.Border).Children(g.arrows)
}

// buildCells builds the items of the strip's row, or all of them in the
// drop-down. Under mu, building.
func (g *RibbonGalleryWidget) buildCells() {
	inPanel := g.panelEl != (ui.Element{})
	from, to := 0, len(g.items)
	if !inPanel {
		from = g.row * g.columns
		to = min(from+g.columns, to)
	}
	hov := -1
	for i := from; i < to; i++ {
		e := g.item(i, inPanel)
		if e.Hovered() {
			hov = i
		}
		if e.Clicked() {
			g.pick(i)
		}
	}
	if !inPanel {
		g.stripHov = hov
	} else if !g.c.Measuring() && hov != g.popHov {
		g.popHov = hov
		g.moved = hov >= 0 // off the items: OnHover(-1), the arrows going on from hl
		if hov >= 0 {
			g.hl = hov
		}
	}
}

// item builds item i, marked when selected, lit when the pointer or the
// drop-down's arrows (inPanel) are on it. Under mu, building.
func (g *RibbonGalleryWidget) item(i int, inPanel bool) ui.Element {
	c, t := g.c, g.c.Theme()
	e := ui.ButtonBase(c).Unfocusable().KeepFocus().Column().Width(itemWidth(t)).Radius(t.Radius).
		Padding(t.Space(0.5)).Gap(t.Space(0.5)).AlignItems(ui.Center).Justify(ui.Center).
		Label(g.items[i].Text).Checked(i == g.sel)
	lit := e.Hovered() || inPanel && i == g.hl
	switch {
	case i == g.sel:
		bg, fg := highlightOf(t)
		e.Background(bg).TextColor(fg)
	case lit:
		e.Background(t.SurfaceHover)
	}
	if inPanel && i == g.hl {
		e.Border(1, t.Accent)
		g.panelEl.ActiveDescendant(e) // assistive technology reads the item the arrows are on
	}
	g.cur = i
	e.Children(g.face)
	return e
}

// buildFace builds item cur's image, 32 DIPs at the text scale, and text.
// Under mu, building.
func (g *RibbonGalleryWidget) buildFace() {
	c, t := g.c, g.c.Theme()
	it := g.items[g.cur]
	if it.Image != nil {
		scale := c.Preferences().TextScale
		if scale <= 0 {
			scale = 1
		}
		ui.Image(c, it.Image).Size(32*scale, 32*scale).Fit(ui.Contain)
	}
	ui.Text(c, it.Text).FontSize(t.Rem(0.85)).TextAlign(ui.Center).MaxLines(2).AccessHidden(true) // the item's label
}

// buildArrows builds the up and down arrows, which scroll the strip a row,
// and the button that opens the drop-down. Under mu, building.
func (g *RibbonGalleryWidget) buildArrows() {
	c := g.c
	up := g.arrow(c).Label(l10n.Previous()).Disabled(g.row == 0)
	up.Children(g.upFace)
	if up.Clicked() { // a disabled button is not clicked
		g.row--
	}
	down := g.arrow(c).Label(l10n.Next()).Disabled(g.row >= g.rows()-1)
	down.Children(g.downFace)
	if down.Clicked() {
		g.row++
	}
	more := g.arrow(c).Label(l10n.More()).Expanded(g.open)
	more.Children(g.moreFace)
	if more.Clicked() {
		g.open = !g.open // a press on the gallery is not outside the drop-down: it toggles
	}
}

// arrow builds a button of the arrows' column, a third of its height. Under
// mu, building.
func (g *RibbonGalleryWidget) arrow(c *ui.Context) ui.Element {
	t := c.Theme()
	e := ui.ButtonBase(c).Grow(1).Basis(0).Width(t.Rem(1.25)).AlignItems(ui.Center).Justify(ui.Center).Column()
	if e.Hovered() && !e.IsDisabled() {
		e.Background(t.SurfaceHover)
	}
	return e
}

// buildUp, buildDown and buildMore draw the arrows' faces: chevrons, the
// drop-down's under a line. Under mu, building.
func (g *RibbonGalleryWidget) buildUp() {
	t := g.c.Theme()
	ui.Icon(g.c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted).Rotate(180)
}

func (g *RibbonGalleryWidget) buildDown() {
	t := g.c.Theme()
	ui.Icon(g.c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
}

func (g *RibbonGalleryWidget) buildMore() {
	t := g.c.Theme()
	ui.Row(g.c).Height(1).Width(t.Space(2.5)).Background(t.TextMuted)
	ui.Icon(g.c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
}

// buildPanel builds the drop-down: all the items in a grid, columns wide,
// which takes the keyboard while it shows. Under mu, building.
func (g *RibbonGalleryWidget) buildPanel(panel ui.Element) {
	c, t := g.c, g.c.Theme()
	comboPanel(c, panel)
	panel.Padding(t.Space(1)).Focusable().FocusRing(false).AutoFocus()
	right, left := panel.Shortcut(0, ui.KeyRight), panel.Shortcut(0, ui.KeyLeft)
	down, up := panel.Shortcut(0, ui.KeyDown), panel.Shortcut(0, ui.KeyUp)
	enter := panel.Shortcut(0, ui.KeyEnter)
	if c.Direction() == ui.RightToLeft { // the grid's start is on the right
		right, left = left, right
	}
	step := 0
	switch {
	case right:
		step = 1
	case left:
		step = -1
	case down:
		step = g.columns
	case up:
		step = -g.columns
	}
	if step != 0 {
		j := g.hl + step
		if g.hl < 0 {
			j = 0
		}
		if j >= 0 && j < len(g.items) {
			g.hl, g.moved = j, true
		}
	}
	if enter && g.hl >= 0 { // SetItems keeps hl in range
		g.pick(g.hl)
		return
	}
	g.panelEl = panel
	panel.Children(g.grid)
	g.panelEl = ui.Element{}
}

// buildGrid builds the drop-down's grid. Under mu, building.
func (g *RibbonGalleryWidget) buildGrid() {
	ui.Grid(g.c).Columns(g.columns).Gap(g.c.Theme().Space(0.5)).Children(g.cells)
}

// pick selects item i as the user did, closes the drop-down, and posts
// OnSelect. Under mu, building.
func (g *RibbonGalleryWidget) pick(i int) {
	g.open = false
	g.selectItem(i)
	if fn := g.onSelect; fn != nil {
		Post(func() { fn(i) })
	}
}
