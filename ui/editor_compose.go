package ui

import "unicode/utf8"

// composeBase is the text and the selection before an input method's
// composition began, kept while the input has an OnEdit: the preedit and the
// removal of the selection it replaces are only shown, and the commit is one
// edit of the text as it was, which fn sees once. A cancelled composition
// restores the text and the selection.
type composeBase struct {
	text          string
	a, z          int // the selection the composition replaces
	caret, anchor int
	// raw is set once UnmarkText put the preedit in the text unfiltered,
	// from a to rawEnd; the commit that follows (GTK delivers it after the
	// unmark) replaces it.
	raw    bool
	rawEnd int
}

// beginBase starts a composition over a..z. The selection goes from the
// text without fn or a step of undo.
func (ed *editor) beginBase(a, z int) {
	ed.base = &composeBase{text: ed.buf.slice(0, ed.buf.n), a: a, z: z, caret: ed.caret, anchor: ed.anchor}
	if a != z {
		ed.edit(a, z, "")
	}
	ed.caret, ed.anchor = a, a
	ed.visualSelection, ed.selected = false, nil
}

// restoreBase gives back the text and the selection before the composition.
func (ed *editor) restoreBase() {
	b := ed.base
	if b == nil {
		return
	}
	ed.base = nil
	if ed.buf.slice(0, ed.buf.n) != b.text {
		ed.edit(0, ed.buf.n, b.text)
	}
	ed.caret, ed.anchor = min(b.caret, ed.buf.n), min(b.anchor, ed.buf.n)
	ed.visualSelection, ed.selected = false, nil
}

// commitBase makes the composition's result s one edit of the text before it.
func (ed *editor) commitBase(s string) {
	b := ed.base
	if b == nil {
		return
	}
	ed.restoreBase()
	if s == "" {
		return
	}
	ed.record(false)
	ed.replace(b.a, b.z, s)
}

// settleBase makes the preedit UnmarkText put in the text, when no commit
// followed, an edit as it is.
func (ed *editor) settleBase() {
	if b := ed.base; b != nil && b.raw {
		s := ed.buf.slice(b.a, b.rawEnd)
		ed.commitBase(s)
	}
}

// unmarkBase ends the preedit with the text it shows put in the text, which
// a commit that follows replaces.
func (ed *editor) unmarkBase() {
	b := ed.base
	s := ed.compose
	ed.compose = ""
	if s == "" {
		ed.restoreBase()
		return
	}
	ed.edit(b.a, b.a, s)
	ed.caret = b.a + utf8.RuneCountInString(s)
	ed.anchor = ed.caret
	b.raw, b.rawEnd = true, ed.caret
}
