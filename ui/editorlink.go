package ui

// The engine reaches the editor's layout, painting and keys through these
// functions, which newEditor sets: a node holds an editor only after
// newEditor made it, in the same process, so every call below is made after
// linkEditor ran. Apps with no text input and no selectable text never call
// newEditor, and do not link the editor's code.
var (
	layoutInputFn      func(e *node, cw, ch float32)
	inputHeightFn      func(e *node, cw float32) float32
	paintInputFn       func(e *node, p *Painter)
	pressFn            func(ed *editor, x, y float32, clicks, button int)
	wantsFn            func(ed *editor, k keyEvent) bool
	textKeyFn          func(ed *editor, k keyEvent) bool
	caretRectFn        func(ed *editor, st *state) Rect
	textSelectionKeyFn func(rt *engine, s *state, k keyEvent) bool
	textUnitFn         func(s *state, at, unit int) (int, int)
	placeMenuCaretFn   func(s *state, x, y float32)
)

// linkEditor sets the functions above. It is called by newEditor, the one
// way an editor is made.
func linkEditor() {
	layoutInputFn = (*node).layoutInput
	inputHeightFn = (*node).inputHeight
	paintInputFn = (*node).paintInput
	pressFn = (*editor).press
	wantsFn = (*editor).wants
	textKeyFn = (*editor).textKey
	caretRectFn = (*editor).caretRect
	textSelectionKeyFn = (*engine).textSelectionKey
	textUnitFn = textUnit
	placeMenuCaretFn = placeMenuCaret
}

// placeMenuCaret puts the caret of the editor of s where a context menu was
// asked for, unless that is in the selection.
func placeMenuCaret(s *state, x, y float32) {
	if ed := s.editor; ed.laidOut() {
		lx, ly := s.local(x, y)
		i := ed.hit(lx, ly)
		if a, b := ed.selection(); a == b || i < a || i > b {
			ed.commitCompose()
			ed.move(i, false)
		}
	}
}
