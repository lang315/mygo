package ui

import (
	"hash/maphash"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
)

// TableColumn describes a column of a Table.
type TableColumn struct {
	// Title heads the column. ID identifies it among the others, for the
	// order and the widths the user gives the columns (ListState.Columns)
	// and for the sort (ListState.Sort and SortBy): the Title when empty.
	Title, ID string
	// Width is the column's width in DIPs, or 0 to share the room the
	// others leave. The user resizes the column by dragging the right edge
	// of its header, between MinWidth and MaxWidth (none for 0), and fits
	// it to its cells with a double click there.
	Width, MinWidth, MaxWidth float32
	// Align aligns the content of its cells.
	Align Align
	// Sortable lets a click on the column's header sort the rows by it,
	// with ListState.Sort or SortBy. Tab reaches the header too, which
	// sorts on Enter or Space as on a click.
	Sortable bool
	// Fixed keeps the column where it is and as wide as it is: the user
	// neither moves nor resizes it. It does not pin the column when the
	// table scrolls sideways; that is Frozen.
	Fixed bool
	// RowHeader marks a column of row headers, as a spreadsheet's row
	// numbers: in a table choosing cells (ListState.CellSelect), a click or
	// a drag on its cells chooses whole rows, with every column, and a
	// click on its header chooses every row. It does not sort, and is
	// never Cells.Cursor.
	RowHeader bool
	// Frozen pins the column at the start of the table while the others
	// scroll sideways under it, and nothing else: unlike Fixed, the user
	// still moves and resizes it. Only the columns from the first in a row
	// are pinned, each with a Width (or one the user resized): a Frozen
	// column after one that is not, or sharing the room left, scrolls as
	// the others. What is pinned follows the order of the columns, so
	// moving a Frozen column pins the one that takes its place if that one
	// is Frozen too, and not otherwise. Give them room: the columns that
	// scroll can only be seen in what the pinned ones leave.
	Frozen bool
}

// stationary reports whether the user cannot move the column. Frozen has
// no say: it is about scrolling only.
func (col *TableColumn) stationary() bool { return col.Fixed }

// sortable reports whether a click on the column's header sorts by it: a
// RowHeader column does not.
func (col *TableColumn) sortable() bool { return col.Sortable && !col.RowHeader }

func (col *TableColumn) id() string {
	if col.ID != "" {
		return col.ID
	}
	return col.Title
}

// SortOrder is how a table's rows are sorted: by the column whose ID is
// Column, in ascending order unless Descending.
type SortOrder struct {
	Column     string
	Descending bool
}

// TableLayout is the order and the widths the user gave the columns of a
// Table, by their IDs. Columns it does not name keep the order and the
// widths the table gives them.
type TableLayout struct {
	Order  []string
	Widths map[string]float32
}

// arrange returns the indexes of columns in the order the layout gives
// them: those it names in its order, the others after them in theirs, and
// fixed columns where they are.
func (l *TableLayout) arrange(columns []TableColumn) []int {
	order := make([]int, 0, len(columns))
	var named []int
	for _, id := range l.Order {
		if j := slices.IndexFunc(columns, func(col TableColumn) bool { return col.id() == id }); j >= 0 && !columns[j].stationary() && !slices.Contains(named, j) {
			named = append(named, j)
		}
	}
	// Fixed columns at their place, the others in the layout's order, then
	// in theirs.
	movable := append([]int(nil), named...)
	for j := range columns {
		if !columns[j].stationary() && !slices.Contains(named, j) {
			movable = append(movable, j)
		}
	}
	for j := range columns {
		if columns[j].stationary() {
			order = append(order, j)
		} else {
			order = append(order, movable[0])
			movable = movable[1:]
		}
	}
	return order
}

// width returns the width of a column: the user's, or the table's, or
// false for one sharing the room left.
func (l *TableLayout) width(col *TableColumn) (float32, bool) {
	w := col.Width
	if v, ok := l.Widths[col.id()]; ok && !col.Fixed {
		w = v
	}
	if w <= 0 {
		return 0, false
	}
	return clampWidth(col, w), true
}

func clampWidth(col *TableColumn, w float32) float32 {
	if col.MaxWidth > 0 {
		w = min(w, col.MaxWidth)
	}
	return max(w, col.MinWidth, 16)
}

// equal reports whether l gives the columns the order and the widths o
// does.
func (l *TableLayout) equal(o *TableLayout) bool {
	return slices.Equal(l.Order, o.Order) && maps.Equal(l.Widths, o.Widths)
}

// tableDrag is a press on a table's header: the column pressed, by ID,
// how far the pointer moved since, and whether it moves the column.
type tableDrag struct {
	id     string
	dx     float32
	moving bool
	// resizing is the column whose edge is pressed, by ID.
	resizing string
	// from is the layout as the move or the resize began, for its end to
	// tell whether it changed it. The table gives the layout a new slice
	// or map as it changes, so these are the ones it had then.
	from TableLayout
	// fit are the columns to fit to their cells in this frame, by ID: the
	// one whose edge was double-clicked, or those of FitColumns. fitted
	// reports that a fit changed a width, for the next build to report.
	fit    []string
	fitted bool
}

// Table creates a table of n rows under a header of columns, whose rows
// are a List's: it builds only those in view, cell building the content
// of a row's column, usually a Text. Rows are as high as their tallest
// cell, and at least as high as the header.
//
// s keeps the place of the rows and says how they behave, as a List's, or
// nil: with s.Selected, a click chooses a row, as Up, Down, Home and End
// do while the table has the keyboard focus, Changed reports a new choice,
// and Submitted a double click or Enter on it; with s.Selection, the user
// chooses several. A row s.Header names spans
// every column, cell building it as column 0, and stays at the top while
// the rows of its section scroll under it.
//
// The user resizes columns by dragging the edges of their headers, and
// moves them by dragging the headers, which s.Columns keeps; Changed
// reports the new layout once the user lets go, or once a double click
// fits a column to its cells, for the app to save. Columns wider
// than the table scroll sideways, the header with them. With s.Sort, a
// click on the header of a Sortable column sorts the rows by it, as Enter
// and Space do on the header Tab reached; with
// s.SortBy, a Cmd-click or a Shift-click sorts by it after the others;
// with s.HeaderMenu, the header has a context menu.
//
//	cols := []ui.TableColumn{{Title: "Name", Sortable: true}, {Title: "Size", Width: 90, Align: ui.End, Sortable: true}}
//	app.files.Selected, app.files.Sort = &app.file, &app.sort
//	if ui.Table(c, &app.files, cols, len(files), func(row, col int) {
//		f := files[row]
//		switch col {
//		case 0:
//			ui.Text(c, f.Name).SingleLine()
//		case 1:
//			ui.Text(c, f.Size())
//		}
//	}).Grow(1).Submitted() {
//		app.open(files[app.file])
//	}
func coreTable(c *context, s *ListState, columns []TableColumn, n int, cell func(row, col int)) *node {
	return table(c, s, columns, n, cell, tableList)
}

// table creates a Table, or with kind treeTableList an OutlineTable.
func table(c *context, s *ListState, columns []TableColumn, n int, cell func(row, col int), kind listKind) *node {
	t := c.theme
	// The height of the header, and the least of the rows.
	tableRow := metricOr(t.Metrics.RowHeight, t.Space(8))
	headRow := metricOr(t.Metrics.HeaderHeight, tableRow)
	table := coreColumn(c).Role(RoleTable).Focusable().Clip().ColumnCount(len(columns))
	table.widget = "Table"
	if kind == treeTableList {
		table.widget, table.role = "OutlineTable", RoleTree
	}
	table.flags |= flagOwnRing
	if s == nil {
		s = coreLocal(table, "rows", func() ListState { return ListState{} })
	}
	// With RowHeight the rows are as high as it says, every one, and
	// their cells hold their content in the middle without the padding
	// above and below that would keep a short row from shrinking.
	cellPadY := t.Space(1.5)
	if s.RowHeight > 0 {
		cellPadY = 0
	}
	drag := coreLocal(table, "drag", func() tableDrag { return tableDrag{} })
	if drag.fitted {
		// The last frame fitted a column to its cells as it laid out.
		drag.fitted = false
		table.st.markEvent(eventColumns)
		table.st.markChanged()
	}
	s.syncSort()
	layout := &s.Columns
	order := layout.arrange(columns)
	// The columns FitColumns asked for: those given that the user can fit.
	for _, id := range s.fits {
		if j := slices.IndexFunc(columns, func(col TableColumn) bool { return col.id() == id }); j >= 0 && !columns[j].Fixed {
			drag.fit = append(drag.fit, id)
		}
	}
	s.fits = s.fits[:0]
	// The block of cells chosen (CellSelect): its columns' places, from lo
	// to hi, and the Cursor's.
	cellSel := s.tracksCells()
	lo, hi, cur := 0, -1, -1
	s.tableCols = s.tableCols[:0]
	if cellSel {
		if c.rt.cellClick == nil {
			c.rt.cellClick, c.rt.cellInBlock, c.rt.cellColumn = (*listFrame).clickCell, (*listFrame).cellInBlockAt, (*listFrame).clickColumn
			c.rt.cellAccess = &cellAccessFuncs
		}
		for _, j := range order {
			s.tableCols = append(s.tableCols, tableCol{id: columns[j].id(), col: j, rowHeader: columns[j].RowHeader, title: columns[j].Title})
		}
		s.fixCells()
		lo, hi = s.block()
		cur = s.colIndex(s.Cells.Cursor)
	}
	// The least width of the rows: the columns', those sharing the room
	// left at their least.
	flexMin := t.Space(15)
	var least float32
	for _, j := range order {
		if w, ok := layout.width(&columns[j]); ok {
			least += w
		} else {
			least += max(columns[j].MinWidth, flexMin)
		}
	}
	checkW := t.Space(8)
	var checks *tableChecks
	if s.Checked != nil {
		least += checkW
		checks = coreLocal(table, "checks", func() tableChecks { return tableChecks{} })
	}
	// Where each column starts, in the row, and how many lead pinned
	// (Frozen); known only when the columns have widths.
	starts, frozenN, frozenW := tableGeometry(layout, columns, order)
	if s.Checked != nil {
		// The checkbox's cell comes first in each row, and pins with the
		// columns pinned after it (pinCheck).
		for i := range starts {
			starts[i] += checkW
		}
		if frozenN > 0 {
			frozenW += checkW
		}
	}
	s.frozenN, s.frozenW = frozenN, frozenW
	// pinCheck pins the checkbox's cell of a row or of the header at the
	// start, before the pinned columns, which would leave a strip the
	// columns scroll under where it was.
	pinCheck := func(box *node) {
		if frozenN > 0 {
			box.Sticky().Start(0)
			box.styleFn = frozenStyle(t, true)
		}
	}
	s.focusCol, s.focusColOn = s.focusColNext, s.focusColNextOn
	s.focusColNext, s.focusColNextOn = "", false
	// fitting are the cells of the columns fitting their cells, by their
	// places in drag.fit.
	var fitting [][]*node
	// The columns' keys, boxed once: a cell is keyed by its column, and a
	// key boxed in each cell of each row allocates for every one of them.
	colKeys := make([]any, len(columns))
	for j := range columns {
		colKeys[j] = columns[j].id()
	}
	var list *node
	// window returns the columns a row builds, from lo to hi (not the
	// pinned ones, which are always built), or false for all: the rows
	// of columns that all have widths, with VirtualColumns, once the list
	// has a width, in the scroll it has now (a row built as the list lays
	// out sees the scroll of that frame).
	// The rows of a frame see the same scroll and width unless the list
	// is laying out and moves them, so what was found for them is kept.
	var win struct {
		x, w   float32
		lo, hi int
		ok     bool
	}
	window := func() (vlo, vhi int, ok bool) {
		if !s.VirtualColumns || starts == nil || list == nil || list.st.w <= 0 {
			return 0, 0, false
		}
		x, w := float32(list.st.scrollX), list.st.w
		if win.ok && win.x == x && win.w == w {
			return win.lo, win.hi, true
		}
		margin := w / 2
		x0, x1 := x-margin, x+w+margin
		n := len(order)
		vlo = frozenN + sort.Search(n-frozenN, func(i int) bool { return starts[frozenN+i+1] > x0 })
		vhi = frozenN + sort.Search(n-frozenN, func(i int) bool { return starts[frozenN+i] >= x1 })
		win.x, win.w, win.lo, win.hi, win.ok = x, w, vlo, vhi, true
		return vlo, vhi, true
	}
	// kept are the columns a row builds out of the window, as a List keeps
	// the rows it needs built: the cell of the Cursor, the one a drag of
	// the choice started in, and the one holding the keyboard, which would
	// lose it, or be skipped by Tab, were it to go.
	kept := [3]int{-1, -1, -1}
	if cellSel {
		kept[0] = cur
		if s.dragSel.on {
			kept[1] = s.colIndex(s.dragSel.fromCol)
		}
	}
	if s.focusColOn {
		kept[2] = slices.IndexFunc(order, func(j int) bool { return columns[j].id() == s.focusCol })
	}
	keep := func(k int) bool { return k >= 0 && (k == kept[0] || k == kept[1] || k == kept[2]) }
	// cells builds a row's cells, with fill building the content of each;
	// a row of the body, windowed, builds only the columns in view, and
	// blocks as wide as the others stand for them. It returns the cells of
	// the header, which tableHeader uses, and nothing for the rows, which
	// would allocate a slice of them for nothing.
	cells := func(role Role, fill func(col int), windowed bool) []*node {
		var boxes []*node
		if role == RoleColumnHeader {
			boxes = make([]*node, 0, len(order))
		}
		vlo, vhi, virt := 0, len(order), false
		if windowed {
			if l, h, ok := window(); ok {
				vlo, vhi, virt = l, h, true
			}
		}
		// A block as wide as a run of columns not built stands for it, where
		// the run ends. Keyed by its side of the window and its place among
		// those of the side, it stays the same element as the window moves.
		run := -1
		var gaps [2]int
		flush := func(to int) {
			if run >= 0 {
				side := 0
				if run >= vlo {
					side = 1
				}
				coreBox(c).Key(gapKeys[side][gaps[side]]).Width(starts[to] - starts[run]).Shrink(0).Role(RoleNone)
				gaps[side]++
				run = -1
			}
		}
		for k, j := range order {
			if virt && k >= frozenN && (k < vlo || k >= vhi) && !keep(k) {
				if run < 0 {
					run = k
				}
				continue
			}
			flush(k)
			col := &columns[j]
			// Keyed by its column, a cell keeps its state as the columns
			// move: a header being dragged keeps the press. Keyed as it
			// is made, as a list's rows are.
			c.nextKey, c.keySet, c.keyGlobal = colKeys[j], true, false
			padY := t.Space(1.5)
			if role != RoleColumnHeader {
				padY = cellPadY
			}
			box := coreRow(c).Padding(padY, t.Space(2.5)).AlignItems(Center).Shrink(0).Clip().Role(role)
			if role == RoleColumnHeader && t.Metrics.HeaderHeight > 0 || role == RoleCell && t.Metrics.RowHeight > 0 {
				box.PaddingY(0) // the metric is the row's height, its text centered in it
			}
			if role == RoleCell && col.RowHeader {
				box.Role(RoleRowHeader) // naming its row
			}
			// Its place among all the columns, for assistive technology to
			// pair it with its header as the ones not built leave a gap.
			box.ColumnIndex(k + 1)
			if w, ok := layout.width(col); ok {
				box.Width(w)
			} else {
				box.Grow(1).Shrink(1).MinWidth(max(col.MinWidth, flexMin))
			}
			if k < frozenN {
				// Pinned at its place from the start, over the columns
				// scrolling under it, which it hides with the color its row
				// shows.
				box.Sticky().Start(starts[k])
				// The first takes the row's start corners, unless the
				// checkbox's cell, pinned too, is before it.
				box.styleFn = frozenStyle(t, k == 0 && s.Checked == nil)
			}
			switch col.Align {
			case Center:
				box.Justify(Center)
			case End:
				box.Justify(End)
			}
			box.Children(func() {
				fill(j)
				if role == RoleColumnHeader && s.HeaderCell != nil {
					s.HeaderCell(wrapElement(box), j)
				}
			})
			if s.GridLines && role == RoleCell {
				box.BorderEndWidth(1).BorderColor(t.Border)
			}
			if menu := s.CellMenu; menu != nil && role == RoleCell && c.row != nil {
				row := c.row.i
				box.ContextMenu(func(m *Menu) { menu(m, row, j) })
			}
			if cellSel && role == RoleCell && c.row != nil {
				// A cell of the block shows as a chosen row does, and a row
				// header with its whole row; the Cursor's, in the row the
				// keys move from, has a border while the table has the
				// focus. A cell holding the focus, as an editor in it, shows
				// what it holds. CellsStyle paints the block in its own
				// colors instead.
				block := s.CellSelect // CurrentCell only marks the cursor's cell
				on := block && c.row.chosen && k >= lo && k <= hi && (!col.RowHeader || s.Cells.All)
				at := k == cur && *s.cursor() == c.row.i
				if block {
					box.checked = 1 + int8(b2f(on))
				}
				style := s.CellsStyle
				bg, fg := c.rowSelected(t)
				if style.Cell != Transparent {
					bg = style.Cell
				}
				if style.CellText != Transparent {
					fg = style.CellText
				}
				switch {
				case !on && !at, box.FocusWithin():
				case on && at:
					// The border in the text's color, on the cell's own.
					box.Background(bg).TextColor(fg).DrawOver(cursorBorder(table, fg))
				case on:
					box.Background(bg).TextColor(fg)
				case at && c.row.chosen: // the row's own tint is the accent
					box.DrawOver(cursorBorder(table, fg))
				case at:
					// The border in the block's own color, the accent
					// without one.
					border := style.Cell
					if border == Transparent {
						border = t.Accent
					}
					box.DrawOver(cursorBorder(table, border))
				}
			}
			if s.VirtualColumns && role == RoleCell && box.FocusWithin() {
				s.focusColNext, s.focusColNextOn = col.id(), true
			}
			if len(drag.fit) > 0 {
				if f := slices.Index(drag.fit, col.id()); f >= 0 {
					for len(fitting) <= f {
						fitting = append(fitting, nil)
					}
					fitting[f] = append(fitting[f], box)
				}
			}
			if boxes != nil {
				boxes = append(boxes, box)
			}
		}
		flush(len(order))
		return boxes
	}
	table.Children(func() {
		// The header, which scrolls sideways with the rows.
		head := coreRow(c).Height(headRow).Shrink(0).AlignItems(Stretch).Role(RoleRow).Clip()
		if menu := s.HeaderMenu; menu != nil {
			// The part of the header past the columns, as AppKit's
			// clickedColumn -1.
			head.ContextMenu(func(m *Menu) { menu(m, -1) })
		}
		head.Children(func() {
			if s.Checked != nil {
				// Not a column header, a leaf role whose checkbox would not show.
				pinCheck(checkCell(c, checkW, func() {
					count, total := checks.counted(s, n)
					all := total > 0 && count == total
					box := checkboxOf(c, &all, "", count > 0, false).Label("Check all")
					// Not focusable, as the rows' are: a click leaves the focus on the table
					// (Copy, the keys), and Tab leaves it in one step; Space checks.
					box.flags &^= flagFocusable
					box.outOfName = true
					table.afterInput(func() {
						if box.Changed() {
							changed := false
							if s.CheckRows != nil && s.Skip == nil {
								changed = s.CheckRows(0, n-1, all)
							} else {
								for i := range n {
									if !s.notChosen(i) {
										changed = s.Checked.set(s.keyOf(i), all) || changed
									}
								}
							}
							if changed {
								table.st.markEvent(eventChecked)
							}
						}
					})
				}))
			}
			heads := cells(RoleColumnHeader, func(j int) {
				col := &columns[j]
				// The header is named by its title, which needs no node
				// of its own beside it.
				coreText(c, col.Title).SingleLine().FontWeight(600).TextColor(t.TextMuted).Role(RoleNone)
				if at, descending := s.sortPlace(col); at >= 0 {
					place := 0
					if s.SortBy != nil && len(*s.SortBy) > 1 {
						place = at + 1
					}
					sortArrow(c, descending, place)
				}
			}, false)
			for k, j := range order {
				if cellSel {
					s.tableCols[k].head = heads[k].id
				}
				tableHeader(c, table, heads[k], j, s, drag, heads, order, columns)
			}
		})
		coreDivider(c)
		// The rows are a List's, whose choice the table takes the focus
		// and the keys for.
		list = coreScroll(c).Grow(1).Role(RoleNone)
		list.widget = "List"
		list.flags |= flagScrollX
		list.rowMinW = least
		// The table's Scrollbars and KeepScrollbars are its rows'.
		list.barsFrom = table
		head.followX = list
		// Only a block of cells (CellSelect) is part of what the rows were built
		// with: a current cell is drawn by the row that has the cursor, which
		// the list never keeps.
		slo, shi, scur := lo, hi, cur
		if !s.CellSelect {
			slo, shi, scur = 0, -1, -1
		}
		// Rows of VirtualColumns build the columns the list's scroll and
		// width show (window), and the one holding the focus: none of which
		// keep compares, so they are not kept.
		s.rowSig, s.rowNoKeep = rowSignature(columns, order, layout, slo, shi, scur, checkW, s), len(drag.fit) > 0 || s.VirtualColumns
		buildList(c, list, table, s, n, func(i int) {
			row := coreRow(c).MinHeight(tableRow).AlignItems(Stretch)
			if s.RowHeight > 0 {
				row.Height(s.RowHeight).MinHeight(0)
			}
			if s.GridLines {
				row.BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
			}
			// The list's element holding the row is the row.
			row.Role(RoleNone)
			if s.Header != nil && s.Header(i) {
				row.Padding(cellPadY, t.Space(2.5)).AlignItems(Center).Background(t.Surface).FontWeight(600)
				if t.Metrics.RowHeight > 0 {
					row.PaddingY(0)
				}
				row.Children(func() { cell(i, 0) })
				return
			}
			if s.cursor() == nil {
				// Chosen rows show the pointer over them already.
				row.flags |= flagHover
				if s.RoundRows {
					row.Radius(t.Radius)
				}
				row.builtWith(t)
				setStyle(row, tableRowStyle, staticTableRow)
				row.flags |= flagClickable
				row.afterInputNode(tableRowActivate)
			}
			if s.RowMenu != nil {
				row.ContextMenu(func(m *Menu) { s.RowMenu(m, s.frame.menuRows(i)) })
			}
			row.Children(func() {
				if s.Checked != nil {
					box := checkCell(c, checkW, func() {
						label := "Check"
						if s.Label != nil {
							// The list's element holding the row has it already.
							label += " " + row.parent.label
						}
						itemCheckbox(c, table, s.Checked, s.keyIn(i), label)
					})
					pinCheck(box)
					if s.GridLines {
						// Ends in a line as the cells after it do.
						box.BorderEndWidth(1).BorderColor(t.Border)
					}
				}
				cells(RoleCell, func(j int) { cell(i, j) }, true)
			})
		}, kind)
	})
	if s.RowMenu != nil {
		// The menu key, Shift+F10 and a click outside the rows act on the
		// rows chosen.
		table.ContextMenu(func(m *Menu) {
			if rows := s.frame.chosenRows(); len(rows) > 0 {
				s.RowMenu(m, rows)
			}
		})
	}
	if s.Copy != nil {
		table.HandleInput(copyInput(c.rt, func() []int { return s.frame.chosenRows() }, s.Copy))
	}
	if len(drag.fit) > 0 {
		table.colFit = &tableFit{ids: drag.fit, cells: fitting, layout: layout, drag: drag, rt: c.rt}
		table.special = true // it fits its columns as it lays out
	}
	table.DrawOver(func(p *Painter, r Rect) {
		if table.FocusVisible() {
			p.focusRingOf(table, r, [4]float32{})
		}
	})
	return table
}

// tableHeader makes the header of columns[j] sort the rows when clicked,
// move the column when dragged, resize it from its end edge (the right,
// or the left right to left), and open s.HeaderMenu.
func tableHeader(c *context, table, h *node, j int, s *ListState, drag *tableDrag, heads []*node, order []int, columns []TableColumn) {
	t := c.theme
	col := &columns[j]
	id := col.id()
	if menu := s.HeaderMenu; menu != nil {
		h.ContextMenu(func(m *Menu) { menu(m, j) })
	}
	sorts, choosesAll := s.sorts() && col.sortable(), col.RowHeader && s.choosesCells()
	if sorts || choosesAll {
		h.flags |= flagClickable | flagHover
		if pinned := h.styleFn; pinned != nil {
			// The hover first: a pinned header then composites it, if it is
			// translucent, over what shows behind.
			h.styleFn = func(h *node) {
				if h.Hovered() {
					h.bg = t.SurfaceHover
				}
				pinned(h)
			}
		} else {
			h.builtWith(t)
			setStyle(h, hoverSurface, staticHoverSurface)
		}
	}
	if choosesAll {
		h.afterInput(func() {
			if f := &s.frame; h.Clicked() {
				f.chooseAll()
				f.chooseAllColumns()
				table.focus(false)
			}
		})
	}
	if sorts {
		// A header that sorts is a button to the keyboard too, as WAI-ARIA's
		// sortable table has it: Tab reaches it, and Enter or Space sorts
		// by it as a click does, which the table's Enter leaves to it. The
		// ring goes inside it, which the header row clips. A click leaves
		// the focus with the table, as it always did, for the keys moving
		// the choice and typing a row's name.
		h.Focusable().FocusRingOffset(-t.focusRingWidth())
		if rt := c.rt; rt.focused == h.id && !rt.focusVisible {
			table.focus(false)
		}
		h.sort = 0
		if at, descending := s.sortPlace(col); at >= 0 {
			h.sort = 1
			if descending {
				h.sort = 2
			}
		}
		h.Label(col.Title) // the title alone, not the place number beside it
	}
	// Moving the column, once the pointer went a few DIPs.
	if dx, _, held := h.Dragged(); held && !col.stationary() && !s.FixedColumnOrder {
		if drag.id != id {
			*drag = tableDrag{id: id, from: s.Columns}
		}
		drag.dx += dx
		if !drag.moving && abs(drag.dx) > 4 {
			drag.moving = true
		}
		if drag.moving {
			if moveColumn(c, s, drag, heads, order, columns) {
				table.st.markEvent(eventColumns)
			}
			// A pinned header is held by its sticky inset, which Left would
			// replace: it follows the pointer by a transform instead.
			if h.sticky {
				h.Translate(drag.dx, 0)
			} else {
				h.Left(drag.dx)
			}
			h.Background(t.SurfacePressed).Opacity(0.9)
		}
	} else if drag.id == id && !held {
		// Released: a click, unless it moved the column, which the
		// table reports if it took another place.
		moved, from := drag.moving, drag.from
		*drag = tableDrag{fit: drag.fit, fitted: drag.fitted}
		if moved {
			h.st.clicks = 0
			if !s.Columns.equal(&from) {
				table.st.markChanged()
			}
		}
	}
	h.afterInput(func() {
		if sorts && h.Clicked() {
			if s.SortBy != nil {
				// Cmd-click as Finder, Shift-click as web grids (TanStack
				// Table, AG Grid, MUI).
				*s.SortBy = sortClicked(*s.SortBy, id, h.ClickModifiers()&(Cmd|Shift) != 0)
				s.syncSort()
			} else {
				*s.Sort = sortClicked([]SortOrder{*s.Sort}, id, false)[0]
			}
			table.st.markChanged()
			table.st.markEvent(eventSort)
		}
	})
	if col.Fixed {
		return
	}
	if _, ok := s.Columns.width(col); !ok && j == order[len(order)-1] {
		// The last column sharing the room left ends at the table's end:
		// an edge there would resize nothing, as AppKit's has none.
		return
	}
	// The edge, which resizes the column, and fits it to its cells on a
	// double click.
	grip := t.Space(2)
	h.Children(func() {
		edge := coreBox(c).Absolute().Top(0).Bottom(0).End(0).Width(grip).Cursor(CursorResizeEW).Role(RoleNone)
		rtl := edge.rtl()
		edge.flags |= flagHover
		if edge.DoubleClicked() {
			drag.fit = append(drag.fit, id)
			c.rt.consumed = true
		}
		dx, _, held := edge.Dragged()
		switch {
		case held && drag.resizing != id:
			drag.resizing, drag.from = id, s.Columns
		case !held && drag.resizing == id:
			// Released: the table reports a new width.
			if !s.Columns.equal(&drag.from) {
				table.st.markChanged()
			}
			drag.resizing, drag.from = "", TableLayout{}
		}
		if held && dx != 0 {
			if rtl {
				// The edge is on the left: dragging it left widens.
				dx = -dx
			}
			// The columns before it sharing the room left keep their
			// widths, for the edge to follow the pointer: the room the
			// column takes or gives comes from the columns after it, or
			// scrolls.
			for k, j := range order {
				if heads[k] == h {
					break
				}
				if _, ok := s.Columns.width(&columns[j]); !ok {
					s.setWidth(&columns[j], heads[k].Bounds().W)
				}
			}
			w, ok := s.Columns.width(col)
			if !ok {
				w = h.Bounds().W // sharing the room left, as wide as it was
			}
			s.setWidth(col, w+dx)
			table.st.markEvent(eventColumns)
		}
		edge.Draw(func(p *Painter, r Rect) {
			x := r.X + r.W - 0.5
			if rtl {
				x = r.X + 0.5
			}
			color := t.Border
			if edge.Hovered() || edge.Pressed() {
				color = t.Accent
			}
			p.Fill(Rect{x - 0.5, r.Y + r.H/4, 1, r.H / 2}, color, 0)
		})
	})
}

// setWidth gives a column a width the user chose.
func (s *ListState) setWidth(col *TableColumn, w float32) {
	if s.Columns.Widths == nil {
		s.Columns.Widths = map[string]float32{}
	} else {
		// A copy, as the app may hold the map it restored.
		s.Columns.Widths = maps.Clone(s.Columns.Widths)
	}
	s.Columns.Widths[col.id()] = clampWidth(col, w)
}

// moveColumn moves the column being dragged past a neighbor whose middle
// it passed, keeping it under the pointer.
func moveColumn(c *context, s *ListState, drag *tableDrag, heads []*node, order []int, columns []TableColumn) bool {
	at := slices.IndexFunc(order, func(j int) bool { return columns[j].id() == drag.id })
	if at < 0 {
		return false
	}
	to := at
	// The columns after it are to its left right to left.
	sign := float32(1)
	if heads[at].rtl() {
		sign = -1
	}
	dx := drag.dx * sign
	switch {
	case dx > 0 && at+1 < len(order):
		if next := heads[at+1].Bounds(); next.W > 0 && dx > next.W/2 {
			to = at + 1
		}
	case dx < 0 && at > 0:
		if prev := heads[at-1].Bounds(); prev.W > 0 && -dx > prev.W/2 {
			to = at - 1
		}
	}
	if to == at || columns[order[to]].stationary() {
		return false
	}
	// Under the pointer still: the column's place moved by the other's
	// width.
	if to > at {
		drag.dx -= sign * heads[to].Bounds().W
	} else {
		drag.dx += sign * heads[to].Bounds().W
	}
	ids := make([]string, len(order))
	for k, j := range order {
		ids[k] = columns[j].id()
	}
	ids[at], ids[to] = ids[to], ids[at]
	s.Columns.Order = ids
	c.rt.consumed = true
	return true
}

// tableFit fits columns of a table to their cells as the table lays out,
// the cells built in the frame and the header: those of ids[i] are
// cells[i].
type tableFit struct {
	ids    []string
	cells  [][]*node
	layout *TableLayout
	drag   *tableDrag
	rt     *engine
	done   bool
}

// apply sets each column's width to the widest of its cells, once, before
// the table is measured or laid out. An ID named twice is fitted once: its
// cells are with the first.
func (tf *tableFit) apply() {
	if tf.done {
		return
	}
	tf.done, tf.drag.fit = true, nil
	for i, id := range tf.ids {
		if slices.Index(tf.ids, id) != i {
			continue
		}
		var cells []*node
		if i < len(tf.cells) {
			cells = tf.cells[i]
		}
		tf.fit(id, cells)
	}
}

// fit sets the width of the column whose ID is id to the widest of its
// cells.
func (tf *tableFit) fit(id string, cells []*node) {
	var w float32
	for _, cell := range cells {
		rt := cell.c.rt
		journalKeep(rt, &cell.width)
		journalKeep(rt, &cell.grow)
		journalKeep(rt, &cell.shrink)
		cell.width = length{}
		w = max(w, intrinsic(cell, true))
	}
	for _, cell := range cells {
		cell.width, cell.grow, cell.shrink = px(w), 0, 0
		cell.parent.forgetIntrinsic()
	}
	if old, ok := tf.layout.Widths[id]; ok && old == w {
		return
	}
	if tf.layout.Widths == nil {
		tf.layout.Widths = map[string]float32{}
	} else {
		tf.layout.Widths = maps.Clone(tf.layout.Widths)
	}
	tf.layout.Widths[id] = w
	tf.drag.fitted = true
	if tf.drag.resizing == id {
		tf.drag.from = *tf.layout
	}
	// Layout runs inside a frame, where requestFrame does nothing: ask for the
	// next one as animations do.
	tf.rt.animating = true
}

// tableChecks is how many of a table's rows shown are checked, and how many
// it shows, counted from what they were counted from, which the header's
// checkbox reads in every frame without counting them again.
type tableChecks struct {
	sel             Selector
	gen, version    uint64
	n, count, total int
}

// counted returns how many of the n rows shown are checked, and how many
// can be: those that are not headers.
func (tc *tableChecks) counted(s *ListState, n int) (count, total int) {
	sel := s.Checked
	if tc.sel == sel && tc.gen == sel.generation() && tc.n == n && tc.version == s.Version {
		return tc.count, tc.total
	}
	if s.CheckedCount != nil {
		count, total = s.CheckedCount()
	} else if s.Header == nil && s.Skip == nil && sel.size() == 0 {
		total = n
	} else {
		for i := range n {
			if s.notChosen(i) {
				continue
			}
			total++
			if sel.size() > 0 && sel.has(s.keyIn(i)) {
				count++
			}
		}
	}
	*tc = tableChecks{sel: sel, gen: sel.generation(), version: s.Version, n: n, count: count, total: total}
	return count, total
}

// copyInput returns the input handler of a table or a grid with a Copy: on
// Cmd+C (Ctrl+C) and the Edit menu's Copy it puts the text copy returns for
// the rows or items chosen on the clipboard. Nothing chosen leaves the key to
// others, "" leaves the clipboard, and a repeat of the key copies nothing more.
func copyInput(rt *engine, chosen func() []int, copy func([]int) string) func(InputEvent) bool {
	return func(ev InputEvent) bool {
		isCopy := ev.Kind == InputCommand && ev.Text == "copy" || ev.Kind == InputKeyDown && ev.Key == KeyC && ev.Mods == Cmd
		if !isCopy {
			return false
		}
		rows := chosen()
		if len(rows) == 0 {
			return false
		}
		if ev.Repeat {
			return true
		}
		if text := copy(rows); text != "" {
			rt.host.writeClipboard(text)
		}
		return true
	}
}

// tableRowStyle shows the pointer over a row.
func tableRowStyle(row *node) {
	if row.Hovered() {
		row.bg = row.builtTheme().rowHover()
	}
}

// rowSignature hashes what a Table's rows are built with besides their own
// data, which a list that keeps rows compares (ListState.RowVersion versions
// the data): the columns in the order shown, how wide each is, how its text
// aligns, which are pinned (Frozen), the block of cells chosen, whether the rows have check boxes, their
// height and grid lines.
func rowSignature(columns []TableColumn, order []int, layout *TableLayout, lo, hi, cur int, checkW float32, s *ListState) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v uint64) { h = (h ^ v) * 1099511628211 }
	for _, j := range order {
		col := &columns[j]
		w, ok := layout.width(col)
		mix(uint64(j))
		mix(uint64(math.Float32bits(w)))
		mix(uint64(math.Float32bits(col.MinWidth)))
		mix(uint64(col.Align))
		mix(maphash.String(keySeed, col.id()))
		if ok {
			mix(1)
		}
		if col.RowHeader {
			mix(2)
		}
		if col.Frozen {
			// Which columns are pinned, and where they stick: frozenN,
			// starts and frozenStyle, which the rows' cells read.
			mix(6)
		}
	}
	mix(uint64(lo))
	mix(uint64(hi))
	mix(uint64(cur))
	mix(uint64(math.Float32bits(checkW)))
	if s.Checked != nil {
		mix(3)
	}
	if s.RoundRows {
		mix(4)
	}
	if s.GridLines {
		mix(5)
	}
	mix(uint64(math.Float32bits(s.RowHeight)))
	return h
}

// tableRowActivate activates a row that is double-clicked. The element
// holding the row has its index, and the list the table is the owner of.
func tableRowActivate(row *node) {
	if !row.DoubleClicked() {
		return
	}
	holder := row.parent
	for n := holder; n != nil; n = n.parent {
		if f := n.list; f != nil {
			f.owner.st.activate(holder.rowIndex)
			return
		}
	}
}

// itemCheckbox builds the checkbox named label of the row or item whose key
// is k, which checks it in checked and tells owner, the table or grid. It
// returns the checkbox and whether k is checked. It takes no keyboard focus: a
// click leaves it on owner, whose keys and Copy keep working, and Space checks.
func itemCheckbox(c *context, owner *node, checked Selector, k any, label string) (box *node, on bool) {
	ic := c.rt.itemChecks.next()
	ic.on = checked.has(k)
	box = checkboxOf(c, &ic.on, "", false, false).Label(label)
	box.flags &^= flagFocusable
	box.outOfName = true
	ic.owner, ic.box, ic.checked, ic.k = owner, box, checked, k
	owner.afterInputCheck(ic)
	return box, ic.on
}

// itemCheck is the check box of a row or item, in the frame it is built in:
// on is what the checkbox sets through its pointer. They live in a slab
// that frames reuse, rather than being made, with a closure, for each row.
type itemCheck struct {
	owner, box *node
	checked    Selector
	k          any
	on         bool
	// link is a Link's action instead (afterInputLink): it runs on box with
	// url and router. A Link's data lives here, in the slab, so that its
	// action is no bigger and finalizeInputs names no link code.
	link   func(*node, string, *Router)
	url    string
	router *Router
}

func (ic *itemCheck) run() {
	if ic.link != nil {
		ic.link(ic.box, ic.url, ic.router)
		return
	}
	if ic.box.Changed() && ic.checked.set(ic.k, ic.on) {
		ic.owner.st.markEvent(eventChecked)
	}
}

func (n *node) afterInputCheck(ic *itemCheck) {
	n.c.rt.afterInputs = append(n.c.rt.afterInputs, inputAction{node: n, check: ic, building: n.c.rt.building})
}

// itemCheckSlab hands out itemChecks whose addresses stay put for the
// frame (the checkboxes point at them), in chunks that the next frame reuses.
type itemCheckSlab struct {
	chunks []*[itemCheckChunk]itemCheck
	n      int
}

const itemCheckChunk = 64

func (s *itemCheckSlab) next() *itemCheck {
	i, j := s.n/itemCheckChunk, s.n%itemCheckChunk
	if i == len(s.chunks) {
		s.chunks = append(s.chunks, new([itemCheckChunk]itemCheck))
	}
	s.n++
	return &s.chunks[i][j]
}

// reset forgets the frame's itemChecks and what they hold.
func (s *itemCheckSlab) reset() {
	for i := 0; i*itemCheckChunk < s.n; i++ {
		clear(s.chunks[i][:min(itemCheckChunk, s.n-i*itemCheckChunk)])
	}
	s.n = 0
}

// checkCell builds the checkbox cell of a row or of the header, which
// comes first in it, so its ID stays apart from the columns' keyed cells.
func checkCell(c *context, w float32, fill func()) *node {
	return coreRow(c).Width(w).Shrink(0).AlignItems(Center).Justify(Center).Role(RoleCell).Children(fill)
}

// syncSort keeps Sort, when set with SortBy, the first column SortBy sorts
// by, or the zero SortOrder when it sorts by none.
func (s *ListState) syncSort() {
	if s.Sort == nil || s.SortBy == nil {
		return
	}
	var first SortOrder
	if by := *s.SortBy; len(by) > 0 {
		first = by[0]
	}
	*s.Sort = first
}

// sorts reports whether a Table's headers sort its rows.
func (s *ListState) sorts() bool { return s.Sort != nil || s.SortBy != nil }

// sortPlace returns the place of col among the columns the rows are sorted
// by, from 0, and whether it sorts them descending, or -1.
func (s *ListState) sortPlace(col *TableColumn) (int, bool) {
	if !col.sortable() {
		return -1, false
	}
	id := col.id()
	if s.SortBy != nil {
		for i, o := range *s.SortBy {
			if o.Column == id {
				return i, o.Descending
			}
		}
		return -1, false
	}
	if s.Sort != nil && s.Sort.Column == id {
		return 0, s.Sort.Descending
	}
	return -1, false
}

// sortClicked returns, as a new slice, the columns sorted by after a click
// on the header of the column whose ID is id, a Cmd-click when add.
func sortClicked(by []SortOrder, id string, add bool) []SortOrder {
	i := slices.IndexFunc(by, func(o SortOrder) bool { return o.Column == id })
	if !add {
		if i == 0 {
			return []SortOrder{{Column: id, Descending: !by[0].Descending}}
		}
		return []SortOrder{{Column: id}}
	}
	by = slices.Clone(by)
	switch {
	case i < 0:
		return append(by, SortOrder{Column: id})
	case !by[i].Descending:
		by[i].Descending = true
		return by
	default:
		return slices.Delete(by, i, i+1)
	}
}

// sortArrow draws the arrow of a column the rows are sorted by, after its
// place among them from 1, or alone for 0.
func sortArrow(c *context, descending bool, place int) {
	t := c.theme
	if place > 0 {
		coreText(c, strconv.Itoa(place)).FontSize(t.FontSize * 0.75).TextColor(t.TextMuted).Shrink(0).MarginStart(t.Space(1)).Role(RoleNone)
	}
	coreBox(c).Size(t.Space(2.5), t.Space(2.5)).Shrink(0).MarginStart(t.Space(1)).Draw(func(p *Painter, r Rect) {
		var path Path
		if descending {
			path.MoveTo(r.X+r.W*0.1, r.Y+r.H*0.3).LineTo(r.X+r.W*0.5, r.Y+r.H*0.7).LineTo(r.X+r.W*0.9, r.Y+r.H*0.3)
		} else {
			path.MoveTo(r.X+r.W*0.1, r.Y+r.H*0.7).LineTo(r.X+r.W*0.5, r.Y+r.H*0.3).LineTo(r.X+r.W*0.9, r.Y+r.H*0.7)
		}
		p.StrokePath(&path, 1.5, t.TextMuted)
	})
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// tableEvent is something that happened to a table (or a list) in a build
// pass, which its queries and callbacks report apart.
type tableEvent uint8

const (
	eventSelection tableEvent = 1 << iota
	eventSort
	eventColumns
	eventChecked
	eventActivated
)

// markEvent reports ev in this build pass.
func (s *state) markEvent(ev tableEvent) {
	if s.rt != nil && !s.rt.inFrame {
		// As a right-click choosing a row, between frames: the next
		// frame's build reports it.
		s.pendingEvents |= ev
		s.touch()
		return
	}
	s.events |= ev
	s.touch()
	s.eventsDelivered &^= ev
	s.notice()
}

// activate reports row double-clicked, or Entered.
func (s *state) activate(row int) {
	s.activated = row
	s.markEvent(eventActivated)
}

// Selected shows the element as chosen among its siblings, in the accent
// color, as a row of a list, and tells assistive technology it is.
func (e *node) Selected(on bool) *node {
	if !on {
		e.checked = 1
		return e
	}
	t := e.theme()
	e.checked = 2
	bg, fg := e.c.rowSelected(t)
	e.Background(bg).TextColor(fg)
	return e
}

// gapKeys are the keys of the blocks standing for the columns not built, by
// the side of the window they are on and their place there: at most three
// columns are kept built out of it, which split a side into four runs.
var gapKeys = [2][4]string{
	{"\x00gap0", "\x00gap1", "\x00gap2", "\x00gap3"},
	{"\x00gap4", "\x00gap5", "\x00gap6", "\x00gap7"},
}

// tableGeometry returns where each column of order starts in a row, with
// one more for the row's end, or nil unless every column has a width; and
// how many columns at the start are pinned (Frozen) and how wide they are
// together.
func tableGeometry(layout *TableLayout, columns []TableColumn, order []int) (starts []float32, frozenN int, frozenW float32) {
	starts = make([]float32, 1, len(order)+1)
	for _, j := range order {
		w, ok := layout.width(&columns[j])
		if !ok {
			starts = nil
			break
		}
		starts = append(starts, starts[len(starts)-1]+w)
	}
	// Only a run of Frozen columns with widths, from the first of order, is
	// pinned: the user's moves change which.
	for k, j := range order {
		w, ok := layout.width(&columns[j])
		if !columns[j].Frozen || !ok || k != frozenN {
			break
		}
		frozenN++
		frozenW += w
	}
	return starts, frozenN, frozenW
}

// frozenStyle is the styleFn of a pinned cell: it takes the background of
// the nearest element around it that has one, or the theme's, for the cells
// scrolling under it not to show through. A background that is itself
// translucent (a selected or hovered row) is composited over the ones
// behind it, down to an opaque one: taken as it is, it would let the cells
// under the pinned one show through. The first of them, at the start of its
// row, takes the row's corners there (RoundRows), which its own square
// background would cover.
func frozenStyle(t *Theme, first bool) func(*node) {
	return func(e *node) {
		if first && e.parent != nil {
			r := e.parent.radius
			if e.rtl() {
				e.radius[1], e.radius[2] = r[1], r[2]
			} else {
				e.radius[0], e.radius[3] = r[0], r[3]
			}
		}
		if e.bg != Transparent && e.bg.A == 255 {
			return
		}
		var buf [4]Color
		layers := buf[:0]
		if e.bg != Transparent {
			layers = append(layers, e.bg)
		}
		base := t.Background
		for p := e.parent; p != nil; p = p.parent {
			if p.bg == Transparent {
				continue
			}
			if p.bg.A == 255 {
				base = p.bg
				break
			}
			layers = append(layers, p.bg)
		}
		for i := len(layers) - 1; i >= 0; i-- {
			base = layers[i].Over(base)
		}
		e.bg = base
	}
}
