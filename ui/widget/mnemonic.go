package widget

import (
	"unicode"

	"github.com/egoist/mygo/ui"
)

// parseMnemonic splits a label with a mnemonic, as WinForms writes them, into
// the text to show, the key that activates it (lower case) and its index in
// the text, in runes: "&Save" is "Save", 's', 0; "&&" is a literal "&". A
// label with no mnemonic gives key 0 and index -1. The first single "&" that
// is followed by a character marks it; a "&" at the end stays as it is.
func parseMnemonic(s string) (display string, key rune, at int) {
	in := []rune(s)
	out := make([]rune, 0, len(in))
	at = -1
	for i := 0; i < len(in); i++ {
		r := in[i]
		if r == '&' && i+1 < len(in) {
			if in[i+1] == '&' {
				out = append(out, '&')
				i++
				continue
			}
			if key == 0 {
				key, at = unicode.ToLower(in[i+1]), len(out)
			}
			continue
		}
		out = append(out, r)
	}
	return string(out), key, at
}

// mnemonicKey returns the key that types r on a keyboard, for the letters of
// the Latin alphabet and the digits; false for any other character.
func mnemonicKey(r rune) (ui.Key, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return ui.KeyA + ui.Key(r-'a'), true
	case r >= '0' && r <= '9':
		return ui.Key0 + ui.Key(r-'0'), true
	}
	return 0, false
}

// mnemonic is a key the form being built answers with Alt: act runs under mu.
type mnemonic struct {
	key rune
	act func()
}

// registerMnemonic makes Alt+key run act while the form being built has the
// control that asked, unless mnemonics are off (macOS, whose menu bar owns the
// key). Under mu, from a build.
func registerMnemonic(key rune, act func()) {
	if key == 0 || macMenus || buildingForm == nil {
		return
	}
	buildingForm.mnemonics = append(buildingForm.mnemonics, mnemonic{key, act})
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
