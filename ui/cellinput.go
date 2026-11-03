package ui

// CellStart says how a CellInput starts. On the first frame the app builds
// the editor (Fresh), it takes the focus; the text is all selected
// (SelectAll), or the caret is at the point (X, Y) of the window (HasAt), or
// at the end.
type CellStart struct {
	Fresh, SelectAll, HasAt bool
	X, Y                    float32
}

// cellEditor builds the text input that EditableText and CellInput show in
// place of a cell's text: where the text was, within its padding and border.
func cellEditor(c *context, draft *string) *node {
	t := c.theme
	pad := t.Space(0.5)
	return coreTextInputBase(c, draft).Grow(1).MinWidth(t.Space(10)).Padding(pad, pad*2).
		Margin(-pad-1, -pad*2-1).Radius(t.Space(1)).Background(t.Background).Border(1, c.accent(t)).
		TextColor(t.Text)
}

// CellInput is a text input that fills the cell of a Table, drawn as an
// EditableText's editor is. The caller owns draft and decides when the
// input shows; Submitted reports Enter, and OnShortcut Escape and Tab.
// Error(msg) marks it not valid: red border, msg read with it.
//
//	if editing {
//		ui.CellInput(c, &draft, ui.CellStart{Fresh: first, SelectAll: true}).
//			OnShortcut(0, ui.KeyEscape, cancel)
//	}
func coreCellInput(c *context, draft *string, start CellStart) *node {
	in := cellEditor(c, draft)
	// The start is asked on the first frame, but a caret at a point needs the
	// editor's layout, which a frame later has: keep it until then.
	st := coreLocal(in, "cellstart", func() cellStarting { return cellStarting{} })
	if start.Fresh {
		*st = cellStarting{pending: true, start: start}
	}
	if st.pending {
		in.afterInput(func() {
			ed := in.st.editor
			if !st.focused {
				in.Focus()
				st.focused = true
			}
			s := st.start
			if s.HasAt && !s.SelectAll && ed.layout == nil {
				c.rt.requestFrame()
				return
			}
			at := ed.buf.n
			switch {
			case s.SelectAll:
				ed.anchor, ed.caret = 0, ed.buf.n
				st.pending = false
				return
			case s.HasAt:
				lx, ly := in.st.local(s.X, s.Y) // the point is in the window, hit takes the element's
				at = ed.hit(lx, ly)
			}
			ed.anchor, ed.caret = at, at
			st.pending = false
		})
	}
	return in
}

// cellStarting is a CellInput's start not yet carried out.
type cellStarting struct {
	pending, focused bool
	start            CellStart
}
