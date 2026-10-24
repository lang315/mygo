package ui

import (
	"github.com/egoist/mygo/internal/platform"
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

// splice applies an edit as it is, recording it.
func recordEdits(into *[]TextEdit) func(TextEdit) (string, int, bool) {
	return func(e TextEdit) (string, int, bool) {
		*into = append(*into, e)
		r := []rune(e.Text)
		out := append(append(append([]rune(nil), r[:e.Start]...), []rune(e.Insert)...), r[e.End:]...)
		return string(out), e.Start + len([]rune(e.Insert)), true
	}
}

func onEditTester(text *string, fn func(TextEdit) (string, int, bool)) *Tester {
	return coreNewTester(func(c *context) { textInput(c, text, false).Fill().AutoFocus().OnEdit(fn) }, 300, 120)
}

// An input method composing over a selection hands fn one edit, at the
// commit, against the text before the composition.
func TestTextInputOnEditSeesOneEditForACompositionOverASelection(t *testing.T) {
	for _, via := range []string{"tester", "client"} {
		s := "12345"
		var edits []TextEdit
		tt := onEditTester(&s, recordEdits(&edits))
		ed := tt.rt.states[tt.rt.focused].editor
		ed.anchor, ed.caret = 1, 3
		tt.Frame()
		switch via {
		case "tester":
			tt.Compose("x", 1)
			tt.Type("X")
		default:
			c := tt.h.ime.Client
			c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
			tt.Frame()
			c.ReplaceText(nil, "X")
		}
		tt.Frame()
		want := []TextEdit{{Text: "12345", Start: 1, End: 3, Insert: "X", Caret: 3}}
		if len(edits) != 1 || edits[0].Text != want[0].Text || edits[0].Start != 1 || edits[0].End != 3 || edits[0].Insert != "X" {
			t.Errorf("%s: fn saw %+v, want one edit %+v", via, edits, want)
		}
		if s != "1X45" {
			t.Errorf("%s: text %q", via, s)
		}
		tt.rt.close()
	}
}

// Cancelling a composition over a selection restores the text and the selection.
func TestTextInputOnEditCancelledCompositionRestoresTheSelection(t *testing.T) {
	for _, via := range []string{"tester", "client"} {
		s := "12345"
		var edits []TextEdit
		tt := onEditTester(&s, recordEdits(&edits))
		ed := tt.rt.states[tt.rt.focused].editor
		ed.anchor, ed.caret = 1, 3
		tt.Frame()
		if via == "tester" {
			tt.Compose("x", 1)
			tt.Compose("", 0)
		} else {
			c := tt.h.ime.Client
			c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
			c.SetMarkedText(nil, "", TextInputRange{})
		}
		tt.Frame()
		a, z := ed.selection()
		if s != "12345" || a != 1 || z != 3 || len(edits) != 0 {
			t.Errorf("%s: text %q selection %d..%d, fn saw %+v", via, s, a, z, edits)
		}
		tt.Key(Cmd, KeyZ) // nothing to undo: the cancelled composition took no step
		if s != "12345" {
			t.Errorf("%s: undo changed %q", via, s)
		}
		tt.rt.close()
	}
}

// An input method that ends the preedit before it commits (GTK) still hands
// fn one edit, against the text before the composition.
func TestTextInputOnEditEndBeforeCommitIsOneEdit(t *testing.T) {
	for _, selected := range []bool{false, true} {
		s := "12345"
		var edits []TextEdit
		tt := onEditTester(&s, recordEdits(&edits))
		ed := tt.rt.states[tt.rt.focused].editor
		ed.anchor, ed.caret = 1, 1
		if selected {
			ed.caret = 3
		}
		tt.Frame()
		c := tt.h.ime.Client
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		var order platform.InputComposition
		order.End(c)
		order.Replace(c, nil, "一")
		tt.Frame()
		end := 1
		if selected {
			end = 3
		}
		if len(edits) != 1 || edits[0].Text != "12345" || edits[0].Start != 1 || edits[0].End != end || edits[0].Insert != "一" {
			t.Errorf("selected=%v: fn saw %+v, want one edit 1..%d \"一\" of 12345", selected, edits, end)
		}
		tt.rt.close()
	}
}

// An edit fn refused keeps what undo took back for redo.
func TestTextInputOnEditRefusedEditKeepsRedo(t *testing.T) {
	text := ""
	tt := onEditTester(&text, func(e TextEdit) (string, int, bool) {
		if e.Insert == "x" {
			return e.Text, e.Caret, true
		}
		return upperFilter(e)
	})
	defer tt.rt.close()
	tt.Type("1")
	tt.Advance(2 * time.Second)
	tt.Type("2")
	tt.Key(Cmd, KeyZ)
	tt.Key(Cmd, KeyZ)
	tt.Type("x")
	tt.Key(Cmd|Shift, KeyZ)
	if text != "1" {
		t.Errorf("redo after a refused edit: %q, want 1", text)
	}
}

// An applied edit takes what undo took back.
func TestTextInputOnEditAnAppliedEditClearsRedo(t *testing.T) {
	text := ""
	tt := onEditTester(&text, upperFilter)
	defer tt.rt.close()
	tt.Type("1")
	tt.Advance(2 * time.Second)
	tt.Type("2")
	tt.Key(Cmd, KeyZ)
	tt.Type("3")
	tt.Key(Cmd|Shift, KeyZ)
	if text != "13" {
		t.Errorf("redo after a new edit: %q, want 13", text)
	}
}

// An input method that unmarks without a commit after it (AppKit) commits
// the preedit as it is: one edit.
func TestTextInputOnEditUnmarkWithoutACommitIsOneEdit(t *testing.T) {
	s := "12345"
	var edits []TextEdit
	tt := onEditTester(&s, recordEdits(&edits))
	defer tt.rt.close()
	ed := tt.rt.states[tt.rt.focused].editor
	ed.anchor, ed.caret = 1, 3
	tt.Frame()
	c := tt.h.ime.Client
	c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
	tt.Frame()
	c.UnmarkText()
	tt.Frame()
	if len(edits) != 1 || edits[0].Text != "12345" || edits[0].Start != 1 || edits[0].End != 3 || edits[0].Insert != "yi" || s != "1yi45" {
		t.Errorf("fn saw %+v, text %q", edits, s)
	}
}

// Input that reaches the editor as events, not through the input client
// (the window does not have the keyboard), is one edit too.
func TestTextInputOnEditCompositionEventsAreOneEditToo(t *testing.T) {
	s := "12345"
	var edits []TextEdit
	tt := onEditTester(&s, recordEdits(&edits))
	defer tt.rt.close()
	ed := tt.rt.states[tt.rt.focused].editor
	ed.anchor, ed.caret = 1, 3
	tt.Frame()
	tt.SetFocused(false)
	tt.Compose("x", 1)
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if len(edits) != 1 || edits[0].Text != "12345" || edits[0].Start != 1 || edits[0].End != 3 || edits[0].Insert != "X" {
		t.Errorf("fn saw %+v, text %q", edits, s)
	}
	tt.Compose("y", 1)
	tt.Compose("", 0)
	tt.Frame()
	if s != "1X45" {
		t.Errorf("a cancelled composition changed the text: %q", s)
	}
}
