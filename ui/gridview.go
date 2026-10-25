package ui

import (
	"math"
	"runtime"
	"strings"
)

// GridState is the state of a GridView: how its items are told apart and
// chosen, as a List's rows are, and the place of its rows.
type GridState struct {
	Handle

	// Key returns an identity for item i, as ListState.Key does for a
	// row: the choice and the items' state follow it.
	Key func(item int) any
	// Selected, when set, lets a click choose an item, as the arrows do
	// while the grid has the keyboard focus: *Selected is the item chosen,
	// -1 for none. The grid's Changed reports a new choice, and its
	// Submitted a double click or Enter.
	Selected *int
	// Selection, when set, lets the user choose several items, held by
	// their keys, as a List's Selection: Cmd-click (Ctrl-click on Linux
	// and Windows) adds an item or takes it out, Shift-click and Shift
	// with the arrows, Home, End, Page Up and Page Down (not on macOS,
	// where those scroll) choose those from the item last chosen, in place
	// of the others, and besides them with Cmd held for a click or Cmd+Shift
	// for the keys; Cmd+A chooses all. Selected, set as well, is the item
	// last chosen.
	Selection Selector
	// Label returns the text of item i: typing its first letters while
	// the grid has the focus chooses it, and assistive technology reads it
	// as the item's name.
	Label func(item int) string
	// Reorder, when set, lets the user drag items to another place in the
	// grid, as ListState.Reorder does rows: the items chosen, when the item
	// dragged is one, else that item, in order, and the item they go
	// before, the number of items for the end.
	Reorder func(items []int, to int)
	// Checked, when set, shows a checkbox on each item, holding the keys of
	// the items checked apart from those chosen, as ListState.Checked does for
	// a table's rows. A checkbox shows on the item checked, on the item the
	// pointer is over, and on all of them while any is checked. A click on it
	// checks or unchecks its item and leaves the choice; Space, while the grid
	// has the focus, checks the items chosen, or unchecks them when all are
	// checked. CheckedChanged reports a change by the user.
	Checked Selector
	// ItemMenu, when set, is the context menu of the items: it fills the menu
	// for items, the items chosen in order. A right click on an item not
	// chosen chooses it alone first; in a grid that does not choose, items is
	// the item clicked. The menu key and Shift+F10 open it on the items
	// chosen, and nothing opens when none is.
	ItemMenu func(m *Menu, items []int)
	// Copy, when set, puts the text it returns for items, the items chosen in
	// order, on the clipboard on Cmd+C (Ctrl+C on Linux and Windows) or the
	// Edit menu's Copy while the grid has the keyboard focus. Nothing is
	// copied when no item is chosen, and "" leaves the clipboard as it was.
	Copy func(items []int) string
	// AccentChoice shows the items chosen in the accent color with its text,
	// focused or not, as a Table's rows are, instead of a tint (a photo
	// library's) or, unfocused, a grey.
	AccentChoice bool
	// PlainChecks draws the check boxes of Checked without the panel that
	// keeps them readable over a picture: for items that have none.
	PlainChecks bool

	rows ListState
	// cells are the items of the cells the last frame built, by their
	// elements' IDs, and built those of this frame.
	cells, built map[uint64]int
	// cols is how many columns the items took in the last frame, and pivot
	// the item Shift chooses from.
	cols, pivot int
	pivotSet    bool // SetPivot was called before the first frame
	started     bool
	rt          *engine

	// span is how far from the pivot the last Shift-extension went, and
	// except the keys of the items chosen before it reached them: going back
	// takes out the others between the pivot and pivot+span.
	// endKey is the key of the item at pivot+span, and endGen the
	// Selection's generation, as the extension left them: if either has
	// changed, the app moved the items or changed the choice, and the
	// extension is forgotten.
	span   int
	except map[any]struct{}
	endKey any
	endGen uint64
}

// Pivot returns the item Shift extends the choice from, -1 for none: the item
// last chosen without Shift. Call it after the grid is built.
func (s *GridState) Pivot() int { return s.pivot }

// SetPivot makes item the one Shift extends the choice from, without changing
// the choice: for an app that shows the same items in another view and keeps
// the pivot by key. Call it after the grid is built once, with an item of the
// frame about to be built.
func (s *GridState) SetPivot(item int) {
	if item != s.pivot {
		s.span, s.except = 0, exceptReset(s.except)
	}
	s.pivot, s.pivotSet = item, true
}

// ScrollTo scrolls the grid to show item at align, as ListState.ScrollTo
// does a row.
func (s *GridState) ScrollTo(item int, align Align) {
	s.rows.ScrollTo(item/max(s.cols, 1), align)
}

// ScrollIntoView scrolls the grid as little as shows item.
func (s *GridState) ScrollIntoView(item int) {
	s.rows.ScrollIntoView(item / max(s.cols, 1))
}

// gridFit is how many columns a grid's items took, which its layout checks
// against the width its rows have: items at least minW wide, gap apart.
type gridFit struct {
	cols      int
	minW, gap float32
}

// key returns the key of item i.
func (s *GridState) key(i int) any {
	if s.Key != nil {
		return s.Key(i)
	}
	return i
}

// GridView creates a scroll container of n items in a grid, as Photos'
// library and Finder's icon view, built with item(i): as many columns as
// fit items at least minWidth wide, which share the width left, and rows
// height high, built only as they show, so grids of millions of items are
// as fast as small ones. Gap spaces the items.
//
// s keeps the place and says how items are chosen, or nil: with
// s.Selected, a click chooses an item, as the arrow keys do in two
// dimensions while the grid has the keyboard focus, with Home, End and the
// first letters of an item's Label.
//
//	app.photos.Selected = &app.photo
//	ui.GridView(c, &app.photos, len(photos), 140, 120, func(i int) {
//		ui.Image(c, thumbs[i]).Fit(ui.Contain).Grow(1)
//	}).Grow(1)
func coreGridView(c *context, s *GridState, n int, minWidth, height float32, item func(i int)) *node {
	t := c.theme
	e := coreScroll(c)
	e.widget, e.role = "GridView", RoleList
	if s == nil {
		s = coreLocal(e, "grid", func() GridState { return GridState{} })
	}
	wrapElement(e).Bind(&s.Handle)
	n = max(n, 0)
	gap := t.Space(2)
	// The columns: those that fit the width the rows had in the last
	// frame, or the window's, first. The layout builds the next frame
	// again where they changed.
	width := s.rows.width
	if width <= 0 {
		width = c.w
	}
	cols := max(1, int(math.Floor(float64((width+gap)/(minWidth+gap)))))
	if s.cols != cols && s.cols > 0 && s.started {
		// Keep the rows in view on the same items.
		first, _ := s.rows.Visible()
		s.rows.ScrollTo(first*s.cols/cols, Start)
	}
	if !s.started && !s.pivotSet {
		s.pivot = -1 // no item to extend from
	}
	s.cols, s.started = cols, true
	rows := (n + cols - 1) / cols
	if s.Selected != nil {
		e.Focusable()
		e.choosesItems = true
		if s.Label != nil {
			e.flags |= flagTypeSelect
		}
	}
	e.setSize = n
	e.chooseMany = s.Selection != nil
	var cursor *node
	buildList(c, e, e, &s.rows, rows, func(r int) {
		row := coreRow(c).Gap(gap).PaddingY(gap / 2).AlignItems(Stretch).Role(RoleNone)
		row.Children(func() {
			for i := r * cols; i < min(n, (r+1)*cols); i++ {
				if cell := s.cell(c, e, i, n, height, item); s.Selected != nil && i == *s.Selected {
					cursor = cell
				}
			}
			// Empty places keep the last row's items as wide as the others.
			for k := n; k < (r+1)*cols && r == rows-1; k++ {
				coreBox(c).Grow(1).Basis(0).MinWidth(0)
			}
		})
	}, gridList)
	// The list's own state is the rows': the grid takes the keys and tells
	// assistive technology about its items.
	e.rowsOf = nil
	e.gridFit = &gridFit{cols: cols, minW: minWidth, gap: gap}
	e.afterInput(func() { s.keys(e, n) })
	e.activeDescendant = cursor
	s.reorder(e, n, gap)
	if s.ItemMenu != nil {
		// The menu key and Shift+F10 act on the items chosen.
		e.ContextMenu(func(m *Menu) {
			if items := s.chosenItems(n); len(items) > 0 {
				s.ItemMenu(m, items)
			}
		})
	}
	if s.Copy != nil {
		e.HandleInput(copyInput(c.rt, func() []int { return s.chosenItems(n) }, s.Copy))
	}
	s.cells, s.built = s.built, s.cells
	clear(s.built)
	return e
}

// itemDrag is items of a grid being dragged to another place in it
// (GridState.Reorder).
type itemDrag struct {
	s     *GridState
	items []int
}

func (d *itemDrag) dragCount() int { return len(d.items) }

// dragged returns the items a drag from item i moves: those chosen when it
// is one of them, else item i.
func (s *GridState) dragged(i, n int) []int {
	if s.Selection == nil || !s.chosen(i) {
		return []int{i}
	}
	return s.chosenItems(n)
}

// reorder makes the grid take its own items dragged over it: it shows where
// they would go, and moves them there as they drop.
func (s *GridState) reorder(e *node, n int, gap float32) {
	if s.Reorder == nil {
		return
	}
	e.flags |= flagValueDrop
	e.st.accepts = func(v any) bool { d, ok := v.(*itemDrag); return ok && d.s == s }
	rt := e.c.rt
	var d *itemDrag
	dropped, x, y := false, rt.pointerX, rt.pointerY
	switch {
	case e.st.hasDropped:
		// Where the items dropped, which the pointer may have left since.
		d, _ = e.st.droppedValue.(*itemDrag)
		e.st.hasDropped, dropped, x, y = false, true, e.st.dropX, e.st.dropY
	case rt.drag != nil && !rt.drag.canceled && rt.drag.over == e.id:
		d, _ = rt.drag.value.(*itemDrag)
	}
	if d == nil || d.s != s {
		return
	}
	x, y = e.st.local(x, y)
	x += e.st.x
	y += e.st.y
	to, at, after := s.dropAt(x, y, n)
	if dropped {
		if !(len(d.items) == 1 && (to == d.items[0] || to == d.items[0]+1)) {
			s.Reorder(d.items, to)
			rt.consumed = true
		}
		return
	}
	t := e.c.theme
	e.DrawOver(func(p *Painter, r Rect) {
		cell := rt.states[at]
		if cell == nil {
			return
		}
		x := cell.x - gap/2
		if after != cell.rtl {
			x = cell.x + cell.w + gap/2
		}
		p.Fill(Rect{x - 1, cell.y + 4, 2, cell.h - 8}, t.Accent, 1)
	})
}

// dropAt returns the item before which items dropped at (x, y) go, and
// the cell of the last frame the line showing it goes beside, and on which
// side: the cell under the pointer, or the nearest in its row.
func (s *GridState) dropAt(x, y float32, n int) (to int, cell uint64, after bool) {
	best, dist := -1, float32(-1)
	for id, i := range s.cells {
		st := s.cellState(id)
		if st == nil || y < st.y || y >= st.y+st.h {
			continue
		}
		d := abs(x - (st.x + st.w/2))
		if best < 0 || d < dist {
			best, dist, cell = i, d, id
		}
	}
	if best < 0 {
		// Below the cells: past the last.
		for id, i := range s.cells {
			if i == n-1 {
				return n, id, true
			}
		}
		return n, 0, false
	}
	st := s.cellState(cell)
	// After a cell is past its middle: to its left right to left.
	if (x >= st.x+st.w/2) != st.rtl {
		return best + 1, cell, true
	}
	return best, cell, false
}

// cellState returns the state of the cell of the last frame whose element
// has id.
func (s *GridState) cellState(id uint64) *state { return s.rt.states[id] }

// cell builds item i of a grid, choosing it when clicked.
func (s *GridState) cell(c *context, grid *node, i, n int, height float32, item func(i int)) *node {
	t := c.theme
	// A border, clear unless chosen, keeps the content in place as it is.
	k := s.key(i)
	cell := coreColumn(c).Key(k).Grow(1).Basis(0).MinWidth(0).Height(height).Radius(t.Radius).Border(2, Color{}).Role(RoleListItem)
	cell.setPos, cell.setSize = i+1, n
	name := ""
	if s.Label != nil {
		name = s.Label(i)
		cell.Label(name)
	}
	if s.built == nil {
		s.built = map[uint64]int{}
	}
	s.built[cell.id], s.rt = i, c.rt
	if s.Reorder != nil {
		cell.dragFrom(func() any { return &itemDrag{s: s, items: s.dragged(i, n)} })
		if cell.Dragging() {
			cell.Opacity(0.4)
		}
	}
	if s.Selected != nil {
		cell.flags |= flagClickable | flagHover | flagChoosable
		cell.afterInput(func() {
			if cell.Clicked() {
				s.click(grid, i, cell.ClickModifiers())
				grid.Focus()
			}
			if cell.DoubleClicked() {
				grid.st.markSubmitted()
			}

		})
		chosen := s.chosen(i)
		cell.checked = 1 + int8(b2f(chosen))
		switch {
		case chosen && s.AccentChoice:
			cell.Selected(true)
		case chosen && grid.Focused():
			cell.Background(t.Accent.Alpha(0.2)).Border(2, t.Accent)
		case chosen:
			cell.Background(t.SurfacePressed).Border(2, t.Border)
		default:
			cell.styleFn = func(cell *node) {
				if cell.Hovered() {
					cell.bg = t.SurfaceHover
				}
			}
		}
		if i == *s.Selected && s.Selection != nil && !chosen {
			// Where the keys moved without choosing.
			cell.DrawOver(func(p *Painter, r Rect) {
				if grid.FocusVisible() {
					p.FocusRing(Rect{r.X + 3, r.Y + 3, r.W - 6, r.H - 6}, cell.radius)
				}
			})
		}
	}
	if s.ItemMenu != nil {
		cell.ContextMenu(func(m *Menu) { s.ItemMenu(m, s.menuItems(grid, i, n)) })
	}
	cell.Children(func() {
		item(i)
		if s.Checked != nil {
			s.checkBox(c, grid, cell, k, name)
		}
	})
	return cell
}

// checkBox builds the checkbox of item i at the start corner of its cell.
// It is built for every cell, so that assistive technology can act on it,
// and shows when the item is checked, when the pointer is over the cell, or
// when any item is: a hidden one lets the pointer through to the cell.
func (s *GridState) checkBox(c *context, grid, cell *node, k any, name string) {
	t := c.theme
	label := "Check"
	if name != "" {
		label += " " + name
	}
	box, on := itemCheckbox(c, grid, s.Checked, k, label)
	box.Absolute().Top(t.Space(1)).Start(t.Space(1))
	if s.PlainChecks {
		box.Padding(t.Space(1))
	} else {
		// A background keeps it visible over a light or a dark image.
		box.Padding(t.Space(1)).Radius(t.Radius).Background(t.Surface.Alpha(0.85))
	}
	if !on && !cell.Hovered() && s.Checked.size() == 0 {
		passThroughAll(box.Opacity(0))
	}
}

// passThroughAll lets the pointer through e and everything inside it, which
// PassThrough alone does not: the square of a checkbox would take the click.
func passThroughAll(e *node) {
	e.PassThrough()
	for ch := e.first; ch != nil; ch = ch.next {
		passThroughAll(ch)
	}
}

// chosenItems returns the items chosen, in order, or nil when none is.
func (s *GridState) chosenItems(n int) []int {
	switch {
	case s.Selection != nil:
		if s.Selection.size() == 0 {
			return nil
		}
		var items []int
		for j := range n {
			if s.Selection.has(s.key(j)) {
				items = append(items, j)
				if len(items) == s.Selection.size() {
					break // the rest of a large grid has none
				}
			}
		}
		return items
	case s.Selected != nil && *s.Selected >= 0 && *s.Selected < n:
		return []int{*s.Selected}
	}
	return nil
}

// menuItems returns the items a context menu on item i acts on: the items
// chosen, choosing item i first when it is not, or item i in a grid that
// does not choose.
func (s *GridState) menuItems(grid *node, i, n int) []int {
	if s.Selected == nil {
		return []int{i}
	}
	if !s.chosen(i) {
		s.choose(grid, i)
	}
	return s.chosenItems(n)
}

// chosen reports whether item i is chosen.
func (s *GridState) chosen(i int) bool {
	if s.Selection != nil {
		return s.Selection.has(s.key(i))
	}
	return s.Selected != nil && *s.Selected == i
}

// click chooses item i as a click with mods does.
func (s *GridState) click(grid *node, i int, mods Modifiers) {
	switch {
	case s.Selection == nil:
		s.choose(grid, i)
	case mods&Shift != 0:
		s.extend(grid, i, mods&Cmd != 0, false)
	case mods&Cmd != 0:
		s.Selection.set(s.key(i), !s.Selection.has(s.key(i)))
		*s.Selected, s.pivot, s.span = i, i, 0
		s.except = exceptReset(s.except)
		grid.st.markChanged()
	default:
		s.choose(grid, i)
	}
}

// choose chooses item i alone.
func (s *GridState) choose(grid *node, i int) {
	changed := *s.Selected != i
	if sel := s.Selection; sel != nil {
		if sel.size() != 1 || !sel.has(s.key(i)) {
			sel.clear()
			sel.set(s.key(i), true)
			changed = true
		}
	}
	*s.Selected, s.pivot, s.span = i, i, 0
	s.except = exceptReset(s.except)
	if changed {
		grid.st.markChanged()
	}
}

// extend chooses the items from the pivot to i, in place of all others unless
// keep, and with shrink in place of those it last chose only (items chosen
// before it stay).
func (s *GridState) extend(grid *node, i int, keep, shrink bool) {
	p := s.pivot
	if p < 0 || p >= grid.setSize || s.Selection == nil {
		s.choose(grid, i)
		return
	}
	if s.span != 0 || len(s.except) > 0 {
		if end := max(0, min(p+s.span, grid.setSize-1)); s.key(end) != s.endKey || s.Selection.generation() != s.endGen {
			s.span, s.except = 0, exceptReset(s.except)
		}
	}
	// track says the items the extension reaches for the first time that were
	// chosen already are kept in except, which going back leaves chosen; the
	// items from oldLo to oldHi were reached before.
	track, oldLo, oldHi := true, 1, 0
	switch {
	case shrink:
		last := max(0, min(p+s.span, grid.setSize-1))
		oldLo, oldHi = min(p, last), max(p, last)
		for j := oldLo; j <= oldHi; j++ {
			if j < min(p, i) || j > max(p, i) {
				k := s.key(j)
				if _, ok := s.except[k]; !ok {
					s.Selection.set(k, false)
				}
			}
		}
	case keep:
		s.except = exceptReset(s.except)
	default:
		s.except, track = exceptReset(s.except), false
		s.Selection.clear()
	}
	s.span = i - p
	for j := min(p, i); j <= max(p, i); j++ {
		k := s.key(j)
		if !s.Selection.set(k, true) && track && (j < oldLo || j > oldHi) {
			s.except = exceptAdd(s.except, k)
		}
	}
	s.endKey, s.endGen = s.key(i), s.Selection.generation()
	*s.Selected = i
	grid.st.markChanged()
}

// page returns the item a page from item at, in the same column: the last row
// in view going down (the first going up), or a page of rows further from
// there.
func (s *GridState) page(at, d, n int) int {
	cols := max(s.cols, 1)
	first, last := s.rows.Visible()
	row, per := at/cols, max(1, last-first)
	switch {
	case d > 0 && row < last:
		row = last
	case d > 0:
		row += per
	case row > first:
		row = first
	default:
		row -= per
	}
	row = max(0, min(row, (n-1)/cols))
	return min(row*cols+at%cols, n-1)
}

// keys moves the choice with the arrows in two dimensions, Home, End and
// the first letters of an item, while the grid has the focus; with Shift (and
// Cmd, besides the items chosen before) they extend it from the pivot, as
// do Page Up and Page Down by the rows in view, except on macOS.
func (s *GridState) keys(grid *node, n int) {
	if s.Selected == nil || n == 0 {
		return
	}
	cols := max(s.cols, 1)
	at := *s.Selected
	in := at >= 0 && at < n
	// The items of a row go from the right right to left.
	back, forth := grid.arrows()
	moves := []struct {
		key Key
		d   int
	}{{forth, 1}, {back, -1}, {KeyDown, cols}, {KeyUp, -cols}}
	// extend chooses from the pivot to the item the key reaches, and keep
	// does it besides the items chosen before (Cmd+Shift).
	to, extend, keep := -1, false, false
	for _, m := range moves {
		plain, shift := grid.Shortcut(0, m.key), s.Selection != nil && grid.Shortcut(Shift, m.key)
		both := s.Selection != nil && grid.Shortcut(Cmd|Shift, m.key)
		if !plain && !shift && !both {
			continue
		}
		extend, keep = shift || both, both
		switch {
		case !in:
			to = 0
		case at+m.d >= 0 && at+m.d < n:
			to = at + m.d
		case m.d == cols && at/cols < (n-1)/cols:
			to = n - 1 // the last row, shorter
		}
	}
	for _, h := range [...]struct {
		key Key
		to  int
	}{{KeyHome, 0}, {KeyEnd, n - 1}} {
		// Every key is asked for in every frame, which keeps it.
		plain := grid.Shortcut(0, h.key)
		shift, both := grid.Shortcut(Shift, h.key), grid.Shortcut(Cmd|Shift, h.key)
		switch {
		case plain:
			to = h.to
		case (shift || both) && s.Selection != nil:
			to, extend, keep = h.to, true, both
			if !in {
				to = 0 // as the arrows do
			}
		}
	}
	// Page Down and Page Up scroll the grid, and on macOS always, as in a
	// list; elsewhere with Shift they extend the choice by the rows in view.
	if runtime.GOOS != "darwin" {
		for _, pg := range [...]struct {
			key Key
			d   int
		}{{KeyPageDown, 1}, {KeyPageUp, -1}} {
			shift, both := grid.Shortcut(Shift, pg.key), grid.Shortcut(Cmd|Shift, pg.key)
			if (shift || both) && s.Selection != nil && in {
				to, extend, keep = s.page(at, pg.d, n), true, both
			}
		}
	}
	if s.Selection != nil && grid.Shortcut(Cmd, KeyA) {
		s.span, s.except = 0, exceptReset(s.except)
		for j := range n {
			s.Selection.set(s.key(j), true)
		}
		grid.st.markChanged()
	}
	if s.Checked != nil && grid.Shortcut(0, KeySpace) && toggleChecked(s.Checked, s.chosenItems(n), s.key) {
		grid.st.markEvent(eventChecked)
	}
	if grid.Shortcut(0, KeyEnter) && in {
		grid.st.markSubmitted()
	}
	if st := grid.st; st.typing && st.typed != "" && s.Label != nil {
		for k := 1; k <= n; k++ {
			j := (max(at, 0) + k - b2i(len([]rune(st.typed)) > 1) + n) % n
			if strings.HasPrefix(strings.ToLower(s.Label(j)), strings.ToLower(st.typed)) {
				to = j
				break
			}
		}
	}
	if to < 0 {
		return
	}
	if extend {
		if s.pivot < 0 || !in {
			s.pivot, s.span = max(at, 0), 0
			s.except = exceptReset(s.except)
		}
		s.extend(grid, to, keep, keep)
	} else {
		s.choose(grid, to)
	}
	s.ScrollIntoView(to)
	grid.c.rt.consumed = true
}
