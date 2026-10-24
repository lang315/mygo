package ui

import "testing"

func TestTextInputMaxLength(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("name").MaxLength(5)
	}, 300, 100)
	tt.Click("name")

	tt.Type("abcdefgh") // typing past the limit is cut at it
	if text != "abcde" {
		t.Fatalf("typed past 5: %q", text)
	}

	tt.Key(Cmd, KeyA)
	tt.SetClipboard("héllo wörld") // runes, not bytes
	tt.Key(Cmd, KeyV)
	if text != "héllo" {
		t.Fatalf("pasted past 5: %q", text)
	}

	tt.Key(0, KeyHome)
	tt.Key(0, KeyRight)
	tt.Type("Z") // in the middle at the limit: the typed rune is refused
	if text != "héllo" {
		t.Fatalf("typed at the limit: %q", text)
	}

	tt.Key(0, KeyBackspace)
	tt.Type("ZZ") // one place free
	if text != "Zéllo" {
		t.Fatalf("typed with one free: %q", text)
	}
}

func TestTextInputNoMaxLengthIsNoLimit(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("name").MaxLength(0)
	}, 300, 100)
	tt.Click("name")
	tt.Type("abcdefghij")
	if text != "abcdefghij" {
		t.Fatalf("typed %q", text)
	}
}

// limitedEditor returns the editor of the one text input the tester shows.
func limitedEditor(t *testing.T, tt *Tester) *editor {
	t.Helper()
	for _, s := range tt.rt.states {
		if s.editor != nil {
			return s.editor
		}
	}
	t.Fatal("no text input")
	return nil
}

func limited(limit int) (*string, *Tester) {
	text := new(string)
	tt := NewTester(func(c *Context) {
		TextInput(c, text).Width(200).Label("name").MaxLength(limit)
	}, 300, 100)
	tt.Click("name")
	return text, tt
}

func TestMaxLengthCutsAComposition(t *testing.T) {
	text, tt := limited(5)
	tt.Type("abc")
	tt.Compose("việt", 4) // Telex: only two places are free
	tt.Frame()
	if got := limitedEditor(t, tt).compose; got != "vi" {
		t.Fatalf("composition shown %q, want vi: what shows is what is stored", got)
	}
	tt.Type("vi") // the input method commits it
	if *text != "abcvi" {
		t.Fatalf("stored %q", *text)
	}
}

func TestMaxLengthCutsAtAGraphemeBoundary(t *testing.T) {
	text, tt := limited(5)
	tt.Type("abcd")
	tt.SetClipboard("é") // é as two runes: one place free
	tt.Key(Cmd, KeyV)
	if *text != "abcd" {
		t.Fatalf("stored %q: the accent must not be dropped from its letter", *text)
	}
}

func TestMaxLengthRefusedPasteLeavesNoUndoStep(t *testing.T) {
	text, tt := limited(3)
	tt.Type("abc")
	tt.SetClipboard("x")
	tt.Key(Cmd, KeyV) // refused: no room
	tt.Key(Cmd, KeyZ) // the first undo takes back the typing, not an empty step
	if *text != "" {
		t.Fatalf("after one undo %q, want the typing undone", *text)
	}
}
