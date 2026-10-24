package widget

import "github.com/egoist/mygo/ui"

// formShortcut is a Form.Shortcut.
type formShortcut struct {
	mods ui.Modifiers
	key  ui.Key
	fn   func()
}

// Shortcut runs fn, on the event goroutine, when the user presses key with
// exactly mods, whichever control has the keyboard focus, unless that
// control takes the key itself: a focused button takes Enter and Space, a
// check box Space, an open drop-down Escape and the arrows, a text field
// what it types. The same mods and key again replace the handler; a nil fn
// removes it. A form behind a modal dialog gets no shortcuts, and a key
// of the menu strip's accelerators goes to the menu.
func (w *Form) Shortcut(mods ui.Modifiers, key ui.Key, fn func()) *Form {
	w.change(func() {
		for i, s := range w.shortcuts {
			if s.mods == mods && s.key == key {
				if fn == nil {
					w.shortcuts = append(w.shortcuts[:i:i], w.shortcuts[i+1:]...)
				} else {
					w.shortcuts[i].fn = fn
				}
				return
			}
		}
		if fn != nil {
			w.shortcuts = append(w.shortcuts, formShortcut{mods, key, fn})
		}
	})
	return w
}

// SetAcceptButton makes Enter click b, from wherever the focus is, as
// WinForms' AcceptButton: unless the control with the focus takes Enter
// (another button, a multiline text box, a text box with OnEnter, which runs
// instead). A disabled or hidden b gets no Enter. nil clears it.
func (w *Form) SetAcceptButton(b *ButtonWidget) *Form {
	w.change(func() { w.accept = b })
	return w
}

// SetCancelButton makes Escape click b, as WinForms' CancelButton: after an
// open drop-down took the key, and instead of a dialog's request to close
// (OnClose is not asked), so b's OnClick decides: one that does not close
// leaves the dialog open. A disabled or hidden b, or none, leaves Escape as
// it is. nil clears it.
func (w *Form) SetCancelButton(b *ButtonWidget) *Form {
	w.change(func() { w.cancel = b })
	return w
}

// formAccept is, while a form builds, what Enter in a text box without
// OnEnter does: click the form's accept button. Under mu.
var formAccept func()

// canClick reports whether b may get the form's Enter or Escape: it is
// enabled, has a handler, and this form built it in its last frame, which
// leaves out a hidden one (inside a hidden container too) and one of another
// form. Under mu.
func (w *Form) canClick(b *ButtonWidget) bool {
	return b != nil && !b.hidden && !b.disabled && w.keep.built[b.id]
}

// acceptClick returns what Enter in a text field without OnEnter does while
// the form builds: click the accept button if it can be clicked then. Under
// mu.
func (w *Form) acceptClick() func() {
	b := w.accept
	if b == nil {
		return nil
	}
	return func() {
		Post(func() {
			mu.Lock()
			ok, h := w.canClick(b), b.onClick
			mu.Unlock()
			if ok && h != nil {
				h()
			}
		})
	}
}

// formKeys asks for the form's shortcuts and posts the handlers of those
// pressed. Under mu, while building.
func (w *Form) formKeys(c *ui.Context) {
	for _, m := range w.mnemonics {
		if k, ok := mnemonicKey(m.key); ok && c.Shortcut(ui.Alt, k) {
			m.act()
		}
	}
	w.mnemonics = w.mnemonics[:0]
	for _, s := range w.shortcuts {
		if c.Shortcut(s.mods, s.key) {
			Post(s.fn)
		}
	}
	for _, k := range []struct {
		key ui.Key
		b   *ButtonWidget
	}{{ui.KeyEnter, w.accept}, {ui.KeyEscape, w.cancel}} {
		if b := k.b; w.canClick(b) && c.Shortcut(0, k.key) && b.onClick != nil {
			Post(b.onClick)
		}
	}
}
