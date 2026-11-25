package widget

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// TokenEditWidget is a field of tokens, as of tags: each shows as a chip
// with a button taking it out, before an input where Enter or a comma adds
// what was typed, and Backspace while it is empty takes out the last.
type TokenEditWidget struct {
	common[TokenEditWidget]
	// tokens is the widget's, under mu; shown is the copy ui edits, which only
	// the drawing thread touches. They never share an array: ui appends to
	// shown and deletes from it in place. given is what shown held as ui got
	// it, which tells what the user added: it shares tokens' array, which is
	// replaced, never written.
	tokens, shown, given []string
	dirty                bool // a setter changed tokens since the last build
	only                 bool // SetSuggestionsOnly
	suggestions          []string
	placeholder          string
	onChange             func()
	edit                 func()            // commit, made once
	paste                func(string) bool // pasted, made once
}

// TokenEdit makes an empty token field; the suggestions containing what is
// typed show below it, those not already tokens.
func TokenEdit(suggestions ...string) *TokenEditWidget {
	t := &TokenEditWidget{suggestions: suggestions}
	t.init(t)
	t.lookAs(skinhook.KindField)
	return t
}

// Tokens returns the tokens, in order: a copy.
func (t *TokenEditWidget) Tokens() []string {
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(t.tokens)
}

// SetTokens replaces the tokens, each trimmed, leaving out blanks and
// repeats, as the user's typing does. It does not run OnChange.
func (t *TokenEditWidget) SetTokens(tokens ...string) *TokenEditWidget {
	var v []string
	for _, s := range tokens {
		if s = strings.TrimSpace(s); s != "" && !slices.Contains(v, s) {
			v = append(v, s)
		}
	}
	t.change(func() { t.tokens, t.dirty = v, true })
	return t
}

// SetSuggestions replaces the suggestions.
func (t *TokenEditWidget) SetSuggestions(s ...string) *TokenEditWidget {
	t.change(func() { t.suggestions = s })
	return t
}

// SetSuggestionsOnly makes the field take only tokens that are suggestions.
func (t *TokenEditWidget) SetSuggestionsOnly(on bool) *TokenEditWidget {
	t.change(func() { t.only = on })
	return t
}

// SetPlaceholder sets the text shown while there is no token and no text.
func (t *TokenEditWidget) SetPlaceholder(s string) *TokenEditWidget {
	t.change(func() { t.placeholder = s })
	return t
}

// OnChange runs fn after each change of the tokens by the user.
func (t *TokenEditWidget) OnChange(fn func()) *TokenEditWidget {
	t.change(func() { t.onChange = fn })
	return t
}

// commit takes the tokens the user changed, in ui's OnChange callback, and
// posts OnChange, then the binding's write. The user wins over a setter that
// ran since the last build, as mirror.commit: tokens become what ui shows,
// and a build still to apply the setter copies them back.
func (t *TokenEditWidget) commit() {
	mu.Lock()
	if t.only {
		t.keepSuggestions()
	}
	changed := !slices.Equal(t.tokens, t.shown)
	if changed {
		t.tokens = slices.Clone(t.shown)
	}
	t.given = t.tokens
	h, write := t.onChange, t.bound.write
	var v []string
	if changed && write != nil {
		v = slices.Clone(t.tokens) // set's own: tokens' array is given's too
	}
	mu.Unlock()
	if !changed || (h == nil && write == nil) {
		return
	}
	Post(func() {
		if h != nil {
			h()
		}
		if write != nil {
			write(v)
		}
	})
}

// keepSuggestions turns what the user added to shown (what given has not)
// into the suggestion it equals, in its spelling, the very spelling first,
// then ignoring case, and drops what is none or then a repeat. In place, on
// the drawing thread, under mu.
func (t *TokenEditWidget) keepSuggestions() {
	out := t.shown[:0]
	for _, s := range t.shown {
		if !slices.Contains(t.given, s) {
			i := slices.Index(t.suggestions, s)
			if i < 0 {
				i = slices.IndexFunc(t.suggestions, func(g string) bool { return strings.EqualFold(g, s) })
			}
			if i < 0 || slices.Contains(out, t.suggestions[i]) {
				continue
			}
			s = t.suggestions[i]
		}
		out = append(out, s)
	}
	t.shown = out
}

// pasted takes a paste of several lines, as a list of IDs, as a token a line,
// in ui's OnPaste, on the drawing thread: each line trimmed, blanks and
// repeats left out. A paste of one line is text, as typed.
func (t *TokenEditWidget) pasted(s string) bool {
	s = strings.ReplaceAll(s, "\r", "\n") // CRLF too: the empty line it leaves is dropped
	if !strings.Contains(s, "\n") {
		return false
	}
	for line := range strings.SplitSeq(s, "\n") {
		if line = strings.TrimSpace(line); line != "" && !slices.Contains(t.shown, line) {
			t.shown = append(t.shown, line)
		}
	}
	t.commit()
	return true
}

func (t *TokenEditWidget) build(c *ui.Context) ui.Element {
	if t.dirty {
		t.shown, t.given, t.dirty = slices.Clone(t.tokens), t.tokens, false
	}
	el := ui.TokenField(c, &t.shown, t.suggestions)
	// ui shows a placeholder whenever the input is empty, tokens or not.
	if len(t.shown) == 0 && t.placeholder != "" {
		el.Placeholder(t.placeholder)
		if tk, _ := skinOf(c); tk != nil {
			el.PlaceholderColor(tk.TextPlaceholder)
		}
	} else {
		el.Placeholder("")
	}
	if t.edit == nil {
		t.edit, t.paste = t.commit, t.pasted
	}
	el.OnChange(t.edit)
	el.OnPaste(t.paste)
	return el
}
