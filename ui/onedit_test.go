package ui

import (
	"strings"
	"testing"
	"time"
)

// upperDigitsOnly keeps digits and letters in capitals: an edit's result is
// the text with the replaced runes swapped for what was typed, filtered.
func upperFilter(e TextEdit) (string, int, bool) {
	r := []rune(e.Text)
	ins := []rune(strings.ToUpper(strings.Map(func(c rune) rune {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			return c
		}
		return -1
	}, e.Insert)))
	out := append(append(append([]rune(nil), r[:e.Start]...), ins...), r[e.End:]...)
	return string(out), e.Start + len(ins), true
}

func TestTextInputOnEditFiltersTypingPasteAndDeletes(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("code").OnEdit(upperFilter)
	}, 300, 100)
	tt.Click("code")
	tt.Type("ab-c1")
	if text != "ABC1" {
		t.Fatalf("typed: %q", text)
	}
	tt.Key(Cmd, KeyA)
	tt.SetClipboard("x y-z9")
	tt.Key(Cmd, KeyV)
	if text != "XYZ9" {
		t.Fatalf("pasted: %q", text)
	}
	tt.Key(0, KeyHome)
	tt.Key(0, KeyRight)
	tt.Type("q") // in the middle: the caret follows the filter's
	tt.Type("w")
	if text != "XQWYZ9" {
		t.Fatalf("typed in the middle: %q", text)
	}
	tt.Key(0, KeyBackspace)
	if text != "XQYZ9" {
		t.Fatalf("backspace: %q", text)
	}
}

func TestTextInputOnEditCanRefuseAnEditAndMoveTheCaret(t *testing.T) {
	text := "12"
	changes := 0
	tt := NewTester(func(c *Context) {
		if TextInput(c, &text).Width(200).Label("n").OnEdit(func(e TextEdit) (string, int, bool) {
			if strings.ContainsFunc(e.Insert, func(r rune) bool { return r < '0' || r > '9' }) {
				return e.Text, len([]rune(e.Text)), true // refused, the caret at the end
			}
			return "", 0, false // the default
		}).Changed() {
			changes++
		}
	}, 300, 100)
	tt.Click("n")
	tt.Key(0, KeyEnd)
	tt.Type("x")
	if text != "12" || changes != 0 {
		t.Fatalf("a refused edit: %q (%d changes)", text, changes)
	}
	tt.Type("3")
	if text != "123" || changes != 1 {
		t.Fatalf("an edit left to the default: %q (%d changes)", text, changes)
	}
}

func TestTextInputOnEditIsOneStepOfUndo(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("code").OnEdit(upperFilter)
	}, 300, 100)
	tt.Click("code")
	tt.Type("a")
	tt.Key(Cmd, KeyA)
	tt.Type("b") // replaces the selection: its own step
	if text != "B" {
		t.Fatalf("text %q", text)
	}
	tt.Key(Cmd, KeyZ)
	if text != "A" {
		t.Fatalf("undo gave %q, want the text before the replacement", text)
	}
}

// An input method's composition that starts with no selection replaces
// nothing: fn must not see it as an edit, which a mask would take for a
// Backspace.
func TestTextInputOnEditNeverSeesAnEditThatChangesNothing(t *testing.T) {
	text := "abc"
	var empty []TextEdit
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("code").OnEdit(func(e TextEdit) (string, int, bool) {
			if e.Start == e.End && e.Insert == "" {
				empty = append(empty, e)
			}
			return upperFilter(e)
		})
	}, 300, 100)
	tt.Click("code")
	tt.Key(0, KeyEnd)
	tt.Compose("d", 1)
	tt.Compose("e", 1)
	tt.Type("e")
	if len(empty) != 0 {
		t.Errorf("fn saw an edit that replaces nothing with nothing: %+v", empty)
	}
	if text != "abcE" {
		t.Errorf("text %q, want the composition committed after abc", text)
	}
}

// An edit fn refuses leaves no step of undo: the first undo takes back the
// last edit that changed something.
func TestTextInputOnEditRefusedEditLeavesNoStepOfUndo(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("code").OnEdit(func(e TextEdit) (string, int, bool) {
			if e.Insert == "x" {
				return e.Text, e.Caret, true
			}
			return upperFilter(e)
		})
	}, 300, 100)
	tt.Click("code")
	tt.Type("1")
	tt.Advance(2 * time.Second)
	tt.Type("x")
	tt.Key(Cmd, KeyZ)
	if text != "" {
		t.Errorf("after one undo %q: the refused edit took a step", text)
	}
}
