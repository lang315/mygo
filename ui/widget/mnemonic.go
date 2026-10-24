package widget

import (
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/egoist/mygo/ui"
)

// parseMnemonic splits a label with a mnemonic, as WinForms writes them, into
// the text to show, the key that activates it (lower case, without accents:
// "&Đóng" answers to d) and its index in the text, in runes: "&Save" is
// "Save", 's', 0; "&&" is a literal "&". A label with no mnemonic, or whose
// marked character no key types, gives key 0 and index -1. The first single
// "&" that is followed by a character marks it; a "&" at the end stays as it
// is.
func parseMnemonic(s string) (display string, key rune, at int) {
	in := []rune(s)
	out := make([]rune, 0, len(in))
	at = -1
	marked := false
	for i := 0; i < len(in); i++ {
		r := in[i]
		if r == '&' && i+1 < len(in) {
			if in[i+1] == '&' {
				out = append(out, '&')
				i++
				continue
			}
			if !marked {
				marked = true
				if k := foldKey(in[i+1]); k != 0 {
					key, at = k, len(out)
				}
			}
			continue
		}
		out = append(out, r)
	}
	return string(out), key, at
}

// foldKey returns the letter or digit of the Latin alphabet that types r on a
// keyboard, without its accent, lower case; 0 for any other character.
func foldKey(r rune) rune {
	r = unicode.ToLower(r)
	if r == 'đ' {
		return 'd'
	}
	if r >= 0x80 {
		if d := []rune(norm.NFD.String(string(r))); len(d) > 0 {
			r = d[0]
		}
	}
	if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
		return r
	}
	return 0
}

// mnemonicKey returns the key that types r (a result of foldKey).
func mnemonicKey(r rune) (ui.Key, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return ui.KeyA + ui.Key(r-'a'), true
	case r >= '0' && r <= '9':
		return ui.Key0 + ui.Key(r-'0'), true
	}
	return 0, false
}

// mnemonic is a key the form being built answers with Alt. Its element is
// valid until the build ends, where formKeys asks whether it is enabled.
type mnemonic struct {
	key  rune
	el   ui.Element
	ctrl Control
	act  func() // under mu
}

// registerMnemonic makes Alt+key (also with Shift) run act while the form
// being built has the control, unless mnemonics are off (macOS, whose menu bar
// owns the key) or no form is building (widget.View). It reports whether it
// did, which is when the label is underlined. Under mu, from a build.
func registerMnemonic(key rune, el ui.Element, ctrl Control, act func()) bool {
	if key == 0 || macMenus || buildingForm == nil {
		return false
	}
	buildingForm.mnemonics = append(buildingForm.mnemonics, mnemonic{key, el, ctrl, act})
	return true
}

// answerMnemonics runs, for each key pressed, the enabled control that asked
// for it; with several for one key the focus moves among them, as in
// WinForms, and none acts. Keys a Form.Shortcut of the form takes are its.
// Under mu, at the end of a build.
func (w *Form) answerMnemonics(c *ui.Context) {
	ms := w.mnemonics
	w.mnemonics = nil
	defer func() { w.built = w.built[:0] }()
	done := map[rune]bool{}
	menuKeys := w.menuBarKeys() // the menu bar's access keys beat the controls'
	for _, m := range ms {
		if done[m.key] {
			continue
		}
		done[m.key] = true
		k, ok := mnemonicKey(m.key)
		if !ok || menuKeys[m.key] || w.hasShortcut(ui.Alt, k) || w.hasShortcut(ui.Alt|ui.Shift, k) {
			continue
		}
		pressed := c.Shortcut(ui.Alt, k)
		if c.Shortcut(ui.Alt|ui.Shift, k) {
			pressed = true
		}
		if !pressed {
			continue
		}
		var hit []mnemonic
		for _, o := range ms {
			if o.key == m.key && !o.el.IsDisabled() {
				hit = append(hit, o)
			}
		}
		switch len(hit) {
		case 0:
		case 1:
			hit[0].act()
		default:
			next := 0
			for i, o := range hit {
				if o.el.Focused() {
					next = (i + 1) % len(hit)
				}
			}
			focusLater(hit[next].ctrl)
		}
	}
}

// hasShortcut reports whether the form has a Shortcut with mods and key.
func (w *Form) hasShortcut(mods ui.Modifiers, key ui.Key) bool {
	for _, s := range w.shortcuts {
		if s.mods == mods && s.key == key {
			return true
		}
	}
	return false
}

// mnemonicSpans shows display with the character at index at underlined.
func mnemonicSpans(display string, at int) []ui.Span {
	r := []rune(display)
	spans := []ui.Span{}
	if at > 0 {
		spans = append(spans, ui.Span{Text: string(r[:at])})
	}
	spans = append(spans, ui.Span{Text: string(r[at]), Underline: true})
	if at+1 < len(r) {
		spans = append(spans, ui.Span{Text: string(r[at+1:])})
	}
	return spans
}

// focusLater asks for the keyboard focus for c, on the event goroutine.
func focusLater(c Control) {
	b := c.base()
	Post(func() { b.change(func() { b.wantFocus = true }) })
}

// menuBarKeys returns the access keys of the form's menu bar: the top items
// whose text marks one with "&". Where the bar is the system's (Windows,
// Linux) those keys open its menus, and a control that asked for the same key
// does not answer it. Under mu.
func (w *Form) menuBarKeys() map[rune]bool {
	keys := map[rune]bool{}
	for _, it := range w.menu {
		if _, k, _ := parseMnemonic(it.Text); k != 0 {
			keys[k] = true
		}
	}
	if w.strip != nil && w.strip.menu != nil {
		for _, it := range w.strip.menu.Items() {
			if _, k, _ := parseMnemonic(it.Label); k != 0 {
				keys[k] = true
			}
		}
	}
	return keys
}

// builtEl is a control a form built, and its element, valid until the build
// ends.
type builtEl struct {
	ctrl Control
	el   ui.Element
}

// focusNextAfter focuses the first control built after c that can take the
// focus, as WinForms' label mnemonic does. Under mu, at the end of a build.
func (w *Form) focusNextAfter(c Control) {
	after := false
	for _, b := range w.built {
		if after && b.el.CanFocus() {
			focusLater(b.ctrl)
			return
		}
		if b.ctrl == c {
			after = true
		}
	}
}
