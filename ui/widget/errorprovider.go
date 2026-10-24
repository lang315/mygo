package widget

import "github.com/egoist/mygo/ui"

// ErrorProviderWidget shows why a control's value is not valid, as WinForms'
// ErrorProvider does: a red mark over the end of the control and the message
// as its tooltip, which assistive technology reads as its description. The mark is drawn over the
// control and takes no room, so showing it moves nothing. A provider holds
// the errors of the controls it was given; a control has one error, the last
// provider's.
//
// Nothing validates by itself: a handler set with OnValidating runs when the
// app calls Form.ValidateChildren, and sets or clears the errors.
type ErrorProviderWidget struct {
	marked map[*core]struct{} // under mu
}

// ErrorProvider makes a provider with no errors.
func ErrorProvider() *ErrorProviderWidget {
	return &ErrorProviderWidget{marked: map[*core]struct{}{}}
}

// SetError marks w with the message msg, or takes its mark off for "".
func (p *ErrorProviderWidget) SetError(w Control, msg string) *ErrorProviderWidget {
	b := w.base()
	b.change(func() {
		if msg == "" {
			if b.errBy == p {
				b.errMsg, b.errBy = "", nil
			}
			delete(p.marked, b)
			return
		}
		if old := b.errBy; old != nil && old != p {
			delete(old.marked, b)
		}
		b.errMsg, b.errBy = msg, p
		p.marked[b] = struct{}{}
	})
	return p
}

// Error returns the message w is marked with by this provider, "" for none.
func (p *ErrorProviderWidget) Error(w Control) string {
	mu.Lock()
	defer mu.Unlock()
	if b := w.base(); b.errBy == p {
		return b.errMsg
	}
	return ""
}

// HasErrors reports whether any control is marked.
func (p *ErrorProviderWidget) HasErrors() bool {
	mu.Lock()
	defer mu.Unlock()
	return len(p.marked) > 0
}

// Clear takes every mark off.
func (p *ErrorProviderWidget) Clear() *ErrorProviderWidget {
	mu.Lock()
	cs := make([]*core, 0, len(p.marked))
	for b := range p.marked {
		cs = append(cs, b)
	}
	mu.Unlock()
	for _, b := range cs {
		b.change(func() {
			if b.errBy == p {
				b.errMsg, b.errBy = "", nil
			}
			delete(p.marked, b)
		})
	}
	return p
}

// OnValidating sets fn to run when the form calls ValidateChildren; it
// returns false to say the control's value is not valid (and usually sets an
// ErrorProvider's error). It runs on the goroutine that called
// ValidateChildren, with no lock held.
func (c *common[W]) OnValidating(fn func() bool) *W {
	c.change(func() { c.validating = fn })
	return c.self
}

// ValidateChildren runs the OnValidating handler of every control the form
// shows, in the order they are laid out, and reports whether all returned
// true: every handler runs, so every error is set. Hidden controls, and ones
// the form never showed, are not validated. Call it from the event goroutine
// (a button's handler), not from a view.
func (w *Form) ValidateChildren() bool {
	mu.Lock()
	var fns []func() bool
	for _, p := range w.keep.order {
		if b := p.Value(); b != nil && b.validating != nil {
			fns = append(fns, b.validating)
		}
	}
	mu.Unlock()
	ok := true
	for _, fn := range fns {
		if !fn() {
			ok = false
		}
	}
	return ok
}

// errorMarkSize is the side of the error mark, in DIPs; errorMarkInset how
// far it is from the end of the control.
const (
	errorMarkSize  = 14
	errorMarkInset = 6
)

// paintErrorMark draws a red disc with a white exclamation mark at the end of
// r, centred across it.
func paintErrorMark(p *ui.Painter, r ui.Rect, rtl bool, danger ui.Color) {
	x := r.X + r.W - errorMarkInset - errorMarkSize
	if rtl {
		x = r.X + errorMarkInset
	}
	y := r.Y + (r.H-errorMarkSize)/2
	p.Fill(ui.Rect{X: x, Y: y, W: errorMarkSize, H: errorMarkSize}, danger, errorMarkSize/2)
	cx := x + errorMarkSize/2
	p.Line(cx, y+3, cx, y+errorMarkSize-6, 1.6, ui.RGB(255, 255, 255))
	p.Fill(ui.Rect{X: cx - 0.9, Y: y + errorMarkSize - 4.5, W: 1.8, H: 1.8}, ui.RGB(255, 255, 255), 0.9)
}
