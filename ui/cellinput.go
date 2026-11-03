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
		Margin(-pad-1, -pad*2-1).Radius(t.Space(1)).Background(t.Background).Border(1, t.Accent).
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
	if start.Fresh {
		in.afterInput(func() {
			in.Focus()
			ed := in.st.editor
			at := ed.buf.n
			switch {
			case start.SelectAll:
				ed.anchor, ed.caret = 0, ed.buf.n
				return
			case start.HasAt:
				at = ed.hit(start.X, start.Y)
			}
			ed.anchor, ed.caret = at, at
		})
	}
	return in
}
