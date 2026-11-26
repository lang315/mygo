package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// suggestionsOf reads the box's suggestions.
func suggestionsOf(e *TextBoxWidget) []string {
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(e.suggestions)
}

// focusedBox shows e, focused.
func focusedBox(t *testing.T, e *TextBoxWidget) *Tester {
	t.Helper()
	tt := NewTester(t, e.Focus(), 400, 400)
	tt.Settle()
	return tt
}

// submit sets the box's text and presses Enter in it, as the user does.
func submit(tt *Tester, e *TextBoxWidget, s string) {
	e.SetText(s)
	tt.Settle()
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
}

func TestTextBoxSuggestions(t *testing.T) {
	e := TextBox().SetSuggestions("Hanoi", "Hue")
	tt := focusedBox(t, e)
	tt.Tester.Type("H")
	tt.Settle()
	if !tt.HasText("Hanoi") || !tt.HasText("Hue") {
		t.Fatalf("typing H suggests %q", tt.Texts())
	}
	tt.Tester.Key(0, ui.KeyDown)
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	if e.Text() != "Hanoi" {
		t.Fatalf("Down, Enter took %q, want Hanoi", e.Text())
	}
}

func TestTextBoxMRU(t *testing.T) {
	e := TextBox().SetMRU(2)
	tt := focusedBox(t, e)
	for _, s := range []string{"a", "b", "a"} {
		submit(tt, e, s)
	}
	if got := suggestionsOf(e); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("after a, b, a the suggestions are %q, want [a b]", got)
	}
	submit(tt, e, "c")
	if got := suggestionsOf(e); !slices.Equal(got, []string{"c", "a"}) {
		t.Fatalf("after c the suggestions are %q, want [c a]", got)
	}
}

func TestTextBoxPasswordIgnoresSuggestions(t *testing.T) {
	for _, e := range []*TextBoxWidget{
		TextBox().Password().SetSuggestions("secret").SetMRU(3),
		TextBox().SetSuggestions("secret").SetMRU(3).Password(),
	} {
		tt := focusedBox(t, e)
		tt.Tester.Type("s")
		tt.Settle()
		if tt.HasText("secret") {
			t.Fatalf("a password box suggests: %q", tt.Texts())
		}
		submit(tt, e, "hunter2")
		for _, s := range suggestionsOf(e) {
			if s == "hunter2" {
				t.Fatalf("a password box remembered what was typed: %q", suggestionsOf(e))
			}
		}
	}
}

// Added: with room to spare, an entry submitted again moves to the front
// and is not kept twice (TestTextBoxMRU's max of 2 cuts a duplicate off).
func TestTextBoxMRUNoDuplicates(t *testing.T) {
	e := TextBox().SetMRU(3)
	tt := focusedBox(t, e)
	for _, s := range []string{"a", "b", "a"} {
		submit(tt, e, s)
	}
	if got := suggestionsOf(e); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("after a, b, a the suggestions are %q, want [a b]", got)
	}
}

// Added: the first Enter of an MRU box turns its input into an
// Autocomplete; the box keeps the keyboard focus, so the next entry types
// into it.
func TestTextBoxMRUKeepsFocus(t *testing.T) {
	e := TextBox().SetMRU(3)
	tt := focusedBox(t, e)
	tt.Tester.Type("x")
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	tt.Tester.Type("y")
	tt.Settle()
	if e.Text() != "xy" {
		t.Fatalf("Text %q after x, Enter, y: the box lost the focus", e.Text())
	}
}

// Added: Enter in an MRU box still runs OnEnter.
func TestTextBoxMRUStillRunsOnEnter(t *testing.T) {
	enters := make(chan struct{}, 4)
	e := TextBox().SetMRU(3).OnEnter(func() { enters <- struct{}{} })
	tt := focusedBox(t, e)
	submit(tt, e, "a")
	submit(tt, e, "b")
	if len(enters) != 2 {
		t.Fatalf("two Enters ran OnEnter %d times", len(enters))
	}
}

// Added: SetMRU(0) turns it off, and a negative max is 0, not a panic.
func TestTextBoxMRUOff(t *testing.T) {
	for _, n := range []int{0, -1} {
		e := TextBox().SetMRU(n)
		tt := focusedBox(t, e)
		submit(tt, e, "a")
		if got := suggestionsOf(e); len(got) != 0 {
			t.Fatalf("SetMRU(%d) remembered %q", n, got)
		}
	}
}

// Added: the suggestions a caller passes are copied.
func TestTextBoxSuggestionsCopied(t *testing.T) {
	items := []string{"Hanoi", "Hue"}
	e := TextBox().SetSuggestions(items...)
	items[0] = "Changed"
	if got := suggestionsOf(e); !slices.Equal(got, []string{"Hanoi", "Hue"}) {
		t.Fatalf("suggestions %q changed with the caller's slice", got)
	}
}
