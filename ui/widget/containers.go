package widget

import (
	"cmp"
	"slices"

	"github.com/egoist/mygo/ui"
)

// GroupBoxWidget draws a border with a title around one widget.
type GroupBoxWidget struct {
	common[GroupBoxWidget]
	title  string
	titleW Control
	child  Control
}

// GroupBox makes a frame titled title.
func GroupBox(title string) *GroupBoxWidget {
	f := &GroupBoxWidget{title: title}
	f.init(f)
	return f
}

// Add sets the widget inside the frame; the frame's title widget is
// ignored.
func (f *GroupBoxWidget) Add(w Control) *GroupBoxWidget {
	f.change(func() {
		if w != f.titleW {
			if f.child != nil {
				orphan(f.child)
			}
			f.child = w
			adopt(&f.core, w)
		}
	})
	return f
}

// SetTitleWidget shows w as the title instead of the text; the frame's
// child is ignored.
func (f *GroupBoxWidget) SetTitleWidget(w Control) *GroupBoxWidget {
	f.change(func() {
		if w != f.child {
			f.titleW = w
		}
	})
	return f
}

func (f *GroupBoxWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(2)).Padding(t.Space(3)).Border(1, t.Border).Radius(t.Radius).Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		switch {
		case f.titleW != nil:
			render(c, f.titleW)
		case f.title != "":
			ui.Text(c, f.title).Bold()
		}
		if f.child != nil {
			e := render(c, f.child)
			if fillsAcross(f.child) {
				e.Grow(1) // what scrolls takes the room the frame has
			}
		}
	})
}

// GridWidget places widgets in cells of rows and columns.
type GridWidget struct {
	common[GridWidget]
	colSp, rowSp float32
	cells        []gridCell
}

type gridCell struct {
	w                       Control
	col, row, width, height int
}

// holeKey keys the empty cells a grid fills its holes with.
type holeKey struct{ col, row int }

// Grid makes a grid whose columns are colSpacing DIPs apart and rows
// rowSpacing. Cells must not overlap: a widget whose cell another's span
// covers is not shown. A cell's Align and VAlign follow the box around
// the grid.
func Grid(colSpacing, rowSpacing int) *GridWidget {
	g := &GridWidget{colSp: float32(colSpacing), rowSp: float32(rowSpacing)}
	g.init(g)
	return g
}

// Attach puts w in the cell at col, row, spanning width columns and height
// rows. A widget attached again moves; one attached to a taken cell
// replaces the widget there.
func (g *GridWidget) Attach(w Control, col, row, width, height int) *GridWidget {
	g.change(func() {
		g.cells = slices.DeleteFunc(g.cells, func(have gridCell) bool {
			return have.w == w || have.col == col && have.row == row
		})
		g.cells = append(g.cells, gridCell{w, col, row, max(width, 1), max(height, 1)})
	})
	return g
}

// build places the cells in row order: ui's grid flows children into the
// next free cell, so the holes before each cell get empty boxes.
func (g *GridWidget) build(c *ui.Context) ui.Element {
	cols, rows := 0, 0
	for _, cl := range g.cells {
		cols, rows = max(cols, cl.col+cl.width), max(rows, cl.row+cl.height)
	}
	at := make(map[[2]int]*gridCell, len(g.cells))
	taken := make([]bool, cols*rows)
	for i := range g.cells {
		cl := &g.cells[i]
		at[[2]int{cl.col, cl.row}] = cl
	}
	e := ui.Grid(c).GapX(g.colSp).GapY(g.rowSp)
	if cols > 0 {
		e.Columns(cols)
	}
	return e.Children(func() {
		for r := range rows {
			for col := range cols {
				if taken[r*cols+col] {
					continue
				}
				cl, ok := at[[2]int{col, r}]
				if !ok {
					ui.Box(c.Key(holeKey{col, r}))
					continue
				}
				for dr := range cl.height {
					for dc := range cl.width {
						if r+dr < rows && col+dc < cols {
							taken[(r+dr)*cols+col+dc] = true
						}
					}
				}
				if cl.w.base().hidden { // keep its cells, or ui flows the next ones in
					keeping.hold(cl.w.base()) // and the widget, to redraw when it shows
					ui.Box(c.Key(holeKey{col, r})).ColumnSpan(cl.width).RowSpan(cl.height)
					continue
				}
				render(c, cl.w).ColumnSpan(cl.width).RowSpan(cl.height)
			}
		}
	})
}

// ScrollPanelWidget scrolls a widget larger than its room.
type ScrollPanelWidget struct {
	common[ScrollPanelWidget]
	child Control
}

func ScrollPanel(w Control) *ScrollPanelWidget {
	s := &ScrollPanelWidget{child: w}
	s.init(s)
	return s
}

func (s *ScrollPanelWidget) build(c *ui.Context) ui.Element {
	return ui.ScrollBoth(c).Grow(1).Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		render(c, s.child)
	})
}

// SeparatorWidget is a line between widgets.
type SeparatorWidget struct{ common[SeparatorWidget] }

func Separator() *SeparatorWidget {
	s := &SeparatorWidget{}
	s.init(s)
	return s
}

func (s *SeparatorWidget) build(c *ui.Context) ui.Element { return ui.Divider(c) }

// TabControlWidget shows one of its pages, chosen by tabs.
type TabControlWidget struct {
	common[TabControlWidget]
	labels   []string
	pages    []Control
	page     mirror[int]
	onSwitch func()
}

func TabControl() *TabControlWidget {
	n := &TabControlWidget{}
	n.init(n)
	return n
}

// AddPage adds a page showing w under a tab labeled label.
func (n *TabControlWidget) AddPage(label string, w Control) *TabControlWidget {
	n.change(func() { n.labels, n.pages = append(n.labels, label), append(n.pages, w) })
	return n
}

// Page returns the index of the page shown.
func (n *TabControlWidget) Page() int { mu.Lock(); defer mu.Unlock(); return n.page.value }

func (n *TabControlWidget) SetPage(i int) *TabControlWidget {
	n.change(func() { n.page.set(i) })
	return n
}

// OnSwitch runs fn after the page shown changes.
func (n *TabControlWidget) OnSwitch(fn func()) *TabControlWidget {
	n.change(func() { n.onSwitch = fn })
	return n
}

func (n *TabControlWidget) build(c *ui.Context) ui.Element {
	return ui.Column(c).Gap(c.Theme().Space(2)).Children(func() {
		p, _ := n.page.bind()
		ui.Tabs(c, p, n.labels...).OnChange(onChange(&n.page, n.onSwitch))
		// Tabs clamps *p while it builds, without OnChange: see it here.
		if changed := *p != n.page.value; changed {
			n.page.value = *p
			if n.onSwitch != nil {
				Post(n.onSwitch)
			}
		}
		if i := *p; i >= 0 && i < len(n.pages) {
			outer := axis
			axis = vertical
			render(c, n.pages[i])
			axis = outer
		}
	})
}

// SidebarWidget shows one of its pages, chosen in a list at its side.
type SidebarWidget struct {
	common[SidebarWidget]
	names, titles []string
	pages         []Control
	selected      mirror[string]
	width         mirror[float32]
}

func Sidebar() *SidebarWidget {
	s := &SidebarWidget{}
	s.width.set(200)
	s.init(s)
	return s
}

// Add adds a page named name, listed as title, showing w. A name already
// there is ignored.
func (s *SidebarWidget) Add(name, title string, w Control) *SidebarWidget {
	s.change(func() {
		if slices.Contains(s.names, name) {
			return
		}
		s.names, s.titles, s.pages = append(s.names, name), append(s.titles, title), append(s.pages, w)
	})
	return s
}

// Remove removes the page named name.
func (s *SidebarWidget) Remove(name string) *SidebarWidget {
	s.change(func() {
		for i, n := range s.names {
			if n == name {
				s.names = append(s.names[:i], s.names[i+1:]...)
				s.titles = append(s.titles[:i], s.titles[i+1:]...)
				s.pages = append(s.pages[:i], s.pages[i+1:]...)
				return
			}
		}
	})
	return s
}

// Selected returns the name of the page shown.
func (s *SidebarWidget) Selected() string { mu.Lock(); defer mu.Unlock(); return s.selected.value }

func (s *SidebarWidget) Select(name string) *SidebarWidget {
	s.change(func() { s.selected.set(name) })
	return s
}

// SetMenuWidth sets the width of the list, in DIPs.
func (s *SidebarWidget) SetMenuWidth(w float32) *SidebarWidget {
	s.change(func() { s.width.set(w) })
	return s
}

func (s *SidebarWidget) build(c *ui.Context) ui.Element {
	sel, _ := s.selected.bind()
	if !slices.Contains(s.names, *sel) && len(s.names) > 0 {
		*sel = s.names[0]
		s.selected.value = *sel
	}
	width, _ := s.width.bind()
	return ui.Split(c, width, func() {
		ui.Sidebar(c, sel, func() {
			for i, name := range s.names {
				ui.SidebarItem(c, name, nil, s.titles[i])
			}
		}).OnChange(onChange(&s.selected, nil)).Fill()
	}, func() {
		for i, name := range s.names {
			if name == *sel {
				outer := axis
				axis = vertical
				render(c, s.pages[i])
				axis = outer
			}
		}
	}).Fill()
}

// CardWidget is a titled box whose body can be hidden.
type CardWidget struct {
	common[CardWidget]
	title, body Control
	hideBody    bool
}

func Card() *CardWidget {
	k := &CardWidget{}
	k.init(k)
	return k
}

// SetTitle shows w as the card's title; the card's body is ignored.
func (k *CardWidget) SetTitle(w Control) *CardWidget {
	k.change(func() {
		if w != k.body {
			k.title = w
		}
	})
	return k
}

// SetBody shows w as the card's body; the card's title is ignored.
func (k *CardWidget) SetBody(w Control) *CardWidget {
	k.change(func() {
		if w != k.title {
			k.body = w
		}
	})
	return k
}

// ShowBody shows the body, or hides it leaving the title.
func (k *CardWidget) ShowBody(on bool) *CardWidget { k.change(func() { k.hideBody = !on }); return k }

func (k *CardWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(t.Space(2)).Padding(t.Space(3)).Border(1, t.Border).Radius(t.Radius).
		Background(t.Surface).Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		if k.title != nil {
			render(c, k.title)
		}
		if k.body != nil && !k.hideBody {
			render(c, k.body)
		}
	})
}

// FlowPanelWidget lays out widgets in a row that wraps onto new lines.
type FlowPanelWidget struct {
	common[FlowPanelWidget]
	spacing float32
	items   []Control
}

func FlowPanel(spacing int) *FlowPanelWidget {
	f := &FlowPanelWidget{spacing: float32(spacing)}
	f.init(f)
	return f
}

func (f *FlowPanelWidget) Add(ws ...Control) *FlowPanelWidget {
	f.change(func() {
		for _, w := range ws {
			if !slices.Contains(f.items, w) {
				f.items = append(f.items, w)
				adopt(&f.core, w)
			}
		}
	})
	return f
}

func (f *FlowPanelWidget) Clear() *FlowPanelWidget {
	f.change(func() {
		for _, w := range f.items {
			orphan(w)
		}
		f.items = nil
	})
	return f
}

func (f *FlowPanelWidget) build(c *ui.Context) ui.Element {
	return ui.Row(c).Wrap().Gap(f.spacing).Children(func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, w := range f.items {
			render(c, w)
		}
	})
}

func (n *TabControlWidget) fillsAcross() bool { return true }

func (f *GroupBoxWidget) fillsAcross() bool    { return f.child != nil && fillsAcross(f.child) }
func (s *ScrollPanelWidget) fillsAcross() bool { return true }

func (f *GroupBoxWidget) children() []Control    { return []Control{f.titleW, f.child} }
func (s *ScrollPanelWidget) children() []Control { return []Control{s.child} }
func (n *TabControlWidget) children() []Control  { return n.pages }
func (s *SidebarWidget) children() []Control     { return s.pages }
func (k *CardWidget) children() []Control        { return []Control{k.title, k.body} }
func (f *FlowPanelWidget) children() []Control   { return f.items }

// children of a grid, row by row.
func (g *GridWidget) children() []Control {
	cells := slices.Clone(g.cells)
	slices.SortStableFunc(cells, func(a, b gridCell) int { return cmp.Or(cmp.Compare(a.row, b.row), cmp.Compare(a.col, b.col)) })
	cs := make([]Control, len(cells))
	for i, c := range cells {
		cs[i] = c.w
	}
	return cs
}
