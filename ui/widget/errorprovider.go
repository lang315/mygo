package widget

import (
	"slices"
	"weak"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

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
	marked map[weak.Pointer[core]]struct{} // under mu; weak, so a control the app dropped goes
}

// ErrorProvider makes a provider with no errors.
func ErrorProvider() *ErrorProviderWidget {
	return &ErrorProviderWidget{marked: map[weak.Pointer[core]]struct{}{}}
}

// SetError marks w with the message msg, or takes its mark off for "".
func (p *ErrorProviderWidget) SetError(w Control, msg string) *ErrorProviderWidget {
	b := w.base()
	b.change(func() {
		if msg == "" {
			if b.errBy == p {
				b.errMsg, b.errBy = "", nil
			}
			delete(p.marked, weak.Make(b))
			return
		}
		if old := b.errBy; old != nil && old != p {
			delete(old.marked, weak.Make(b))
		}
		b.errMsg, b.errBy = msg, p
		p.marked[weak.Make(b)] = struct{}{}
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
	for k := range p.marked {
		if k.Value() != nil {
			return true
		}
		delete(p.marked, k)
	}
	return false
}

// FocusFirst gives the keyboard focus, at the next frame, to the first control
// of f that this provider marks, in the order Tab visits them (the order
// ValidateChildren runs the handlers in, not the order the errors were set),
// and reports whether there is one. A hidden or disabled control, and one in a
// hidden or disabled container, is passed over. A control on a page of a
// TabControl or a Sidebar that is not showing is reached: its pages show first
// (ValidateChildren, which validates every page, does not switch them). It
// changes nothing when no control can take the focus. Call it after a
// ValidateChildren that returned false, so that the user lands on the first
// field to fix:
//
//	if !form.ValidateChildren() {
//		errors.FocusFirst(form)
//	}
//
// It is safe from any goroutine: the request is kept under the package's lock
// and made when the control builds.
func (p *ErrorProviderWidget) FocusFirst(f *Form) bool {
	mu.Lock()
	var target *core
	var pages []func()
	seen := map[*core]bool{}
	var walk func(c Control, path []func(), off bool) bool
	walk = func(c Control, path []func(), off bool) bool {
		if c == nil {
			return false
		}
		b := c.base()
		if b.hidden || seen[b] {
			return false
		}
		seen[b] = true
		off = off || b.disabled
		if !off && b.errBy == p {
			target, pages = b, slices.Clone(path)
			return true
		}
		switch w := c.(type) {
		case *TabControlWidget:
			for i, k := range w.pages {
				if walk(k, append(path, func() { w.SetPage(i) }), off) {
					return true
				}
			}
		case *SidebarWidget:
			for i, k := range w.pages {
				if walk(k, append(path, func() { w.Select(w.names[i]) }), off) {
					return true
				}
			}
		default:
			if h, ok := c.(interface{ children() []Control }); ok {
				for _, k := range h.children() {
					if walk(k, path, off) {
						return true
					}
				}
			}
		}
		return false
	}
	_ = (f.header != nil && walk(f.header, nil, false)) ||
		(f.tools != nil && walk(f.tools, nil, false)) ||
		walk(f.root, nil, false) ||
		(f.status != nil && walk(f.status, nil, false))
	mu.Unlock()
	if target == nil {
		return false
	}
	for _, show := range pages {
		show()
	}
	target.change(func() { target.wantFocus = true })
	return true
}

// Clear takes every mark off.
func (p *ErrorProviderWidget) Clear() *ErrorProviderWidget {
	mu.Lock()
	cs := make([]*core, 0, len(p.marked))
	for k := range p.marked {
		if b := k.Value(); b != nil {
			cs = append(cs, b)
		}
	}
	clear(p.marked)
	mu.Unlock()
	for _, b := range cs {
		b.change(func() {
			if b.errBy == p {
				b.errMsg, b.errBy = "", nil
			}
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
// holds, in the order they are laid out, and reports whether all returned
// true: every handler runs, so every error is set. That is every control in
// the form's widget tree, those on pages of a TabControl or Sidebar that are
// not showing and in a card whose body is hidden too; only a control made
// invisible (SetVisible(false)), and what is inside it, is skipped. A disabled
// control is validated, as in WinForms. A handler that panics ends the
// validation: the panic propagates and the handlers after it do not run.
//
// When it returns false, the first failing control's error message is
// announced to assistive technology, once. Call it from the event goroutine
// (a button's handler), not from a view.
func (w *Form) ValidateChildren() bool {
	mu.Lock()
	var cs []*core
	seen := map[*core]bool{}
	var walk func(Control)
	walk = func(c Control) {
		if c == nil {
			return
		}
		b := c.base()
		if b.hidden || seen[b] {
			return
		}
		seen[b] = true
		if b.validating != nil {
			cs = append(cs, b)
		}
		if h, ok := c.(interface{ children() []Control }); ok {
			for _, k := range h.children() {
				walk(k)
			}
		}
	}
	if h := w.header; h != nil {
		walk(h)
	}
	if w.tools != nil {
		walk(w.tools)
	}
	walk(w.root)
	if w.status != nil {
		walk(w.status)
	}
	fns := make([]func() bool, len(cs))
	for i, b := range cs {
		fns[i] = b.validating
	}
	mu.Unlock()
	ok, first := true, -1
	for i, fn := range fns {
		if !fn() {
			if ok {
				first = i
			}
			ok = false
		}
	}
	if !ok && first >= 0 {
		mu.Lock()
		msg := cs[first].errMsg
		if msg != "" {
			w.announce = msg
		}
		mu.Unlock()
		if msg != "" {
			w.do(func(n *mygo.Window) { n.Invalidate() })
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
