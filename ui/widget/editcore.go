package widget

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// What follows is the part of in-cell editing that does not depend on the
// table's row type or key type: it is compiled once, not once for each
// DataGridViewWidget[T, K] a program uses.

// editCore is the state of an edit in progress that does not name T or K. The
// draft lives here, in the table, and never in the row or the source until the
// edit is written. Under mu.
type editCore struct {
	col              int    // the column's index in cols
	title            string // the column's title
	orig, draft      string // the text the editor started with, and the user's
	err              string // the last refusal, shown by the editor
	fresh, selectAll bool   // the next build gives the editor the focus; all selected
	at               struct {
		has  bool
		x, y float32
	}
	gen       uint64 // versions the edited row while the edit lasts
	focused   bool   // the editor held the focus in the last frame
	announced string // the refusal the user was told
	hadFocus  bool   // the editor has held the focus: losing it commits
	// end and cancel commit and cancel this edit, and only this one: the table
	// makes them when the edit begins.
	end, cancel func()
	// tab moves the edit to the next editable cell (dir 1) or the previous (-1);
	// canNext and canPrev say whether the table may have one, so that a Tab with
	// nowhere to go keeps moving the focus on.
	tab              func(dir int)
	canNext, canPrev bool
}

// buildEditor builds the editor of the cell being edited in place of its text.
// Under mu, while building.
func buildEditor(c *ui.Context, e *editCore) {
	start := ui.CellStart{Fresh: e.fresh, SelectAll: e.selectAll, HasAt: e.at.has, X: e.at.x, Y: e.at.y}
	if e.fresh && !c.Measuring() {
		c.Announce("Editing " + e.title)
		e.fresh = false
	}
	in := ui.CellInput(c, &e.draft, start)
	if now := in.Focused(); !c.Measuring() {
		e.focused = now
		switch {
		case now:
			e.hadFocus = true
		case e.hadFocus && !start.Fresh:
			// The focus went elsewhere: what was typed is written, or, when
			// Validate refuses it, the edit stays open with its error and the
			// focus stays where the user put it. Once per loss.
			e.hadFocus = false
			Post(e.end)
		}
	}
	if e.err != "" {
		in.Error(e.err)
	}
	in.OnShortcut(0, ui.KeyEscape, func() { Post(e.cancel) })
	if e.canNext {
		in.OnShortcut(0, ui.KeyTab, func() { Post(func() { e.tab(1) }) })
	}
	if e.canPrev {
		in.OnShortcut(ui.Shift, ui.KeyTab, func() { Post(func() { e.tab(-1) }) })
	}
	if in.Submitted() {
		Post(e.end)
	}
	if e.err != "" && e.announced != e.err && !c.Measuring() {
		e.announced = e.err
		c.Announce(e.err)
	}
}

// commitKind says what a commit did.
type commitKind int

const (
	committedNothing  commitKind = iota // nothing to write: the edit closes
	committedRefused                    // Validate refused: the edit stays open
	committedWrote                      // Set wrote the draft
	committedPanicked                   // Validate or Set panicked: the edit closes
)

// commitOutcome is the result of commitCore.
type commitOutcome struct {
	kind  commitKind
	err   error          // the refusal
	wins  []*mygo.Window // to redraw once the lock is released
	panic reported       // what Validate or Set panicked with
}

// reported is a panic of a column function, for OnPanic.
type reported struct {
	v     any
	stack []byte
}

// postReports hands recovered panics to OnPanic, on the event goroutine.
func postReports(ps []reported) {
	for _, p := range ps {
		Post(func() { report(p.v, p.stack) })
	}
}

// commitCore decides a commit of the draft of e. rowThere says the row still
// is; validate (nil for none) and write run the column's functions, under the
// lock, and write returns the windows to redraw. A draft equal to the text the
// editor started with writes nothing, so that Enter without typing does not
// overwrite a change another goroutine made meanwhile.
func commitCore(e *editCore, rowThere bool, validate func(draft string) error, write func(draft string) []*mygo.Window) commitOutcome {
	if e.draft == e.orig || !rowThere {
		return commitOutcome{}
	}
	if validate != nil {
		var refusal error
		if v, stack, panicked := runColumnFn(func() { refusal = validate(e.draft) }); panicked {
			return commitOutcome{kind: committedPanicked, panic: reported{v, stack}}
		}
		if refusal != nil {
			e.err = refusal.Error()
			return commitOutcome{kind: committedRefused, err: refusal}
		}
	}
	var wins []*mygo.Window
	if v, stack, panicked := runColumnFn(func() { wins = write(e.draft) }); panicked {
		return commitOutcome{kind: committedPanicked, panic: reported{v, stack}}
	}
	return commitOutcome{kind: committedWrote, wins: wins}
}
