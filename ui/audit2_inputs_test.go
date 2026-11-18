package ui

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/i18n"
)

// The app sets a value while an input method composes over a selection: the
// step of undo takes the text from before the composition, not the text with
// the selection hidden (R2-5: the removal slice went out of range).
func TestOnEditSetTextDuringCompositionOverASelection(t *testing.T) {
	s := ""
	var edits []TextEdit
	tt := onEditTester(&s, recordEdits(&edits))
	defer tt.rt.close()
	tt.Type("abc")
	tt.Key(Cmd, KeyA)
	tt.Frame()
	c := tt.h.ime.Client
	c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
	tt.Frame()
	s = "new value"
	tt.Frame()
	if s != "new value" {
		t.Fatalf("text %q", s)
	}
	tt.Key(Cmd, KeyZ)
	if s != "" {
		t.Errorf("undo gives %q, want the empty text from before the typing", s)
	}
}

// ReplaceText during a composition drops the base before it edits: the
// commit then adds to the replaced text (R2-12).
func TestReplaceTextDuringCompositionSurvivesCommit(t *testing.T) {
	s := "hello"
	var edits []TextEdit
	var in *node
	tt := coreNewTester(func(c *context) {
		in = textInput(c, &s, false).Fill().AutoFocus().OnEdit(recordEdits(&edits))
	}, 300, 120)
	defer tt.rt.close()
	tt.Frame()
	c := tt.h.ime.Client
	c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
	tt.Frame()
	in.ReplaceText(0, math.MaxInt, "REPLACED")
	tt.Frame()
	c.ReplaceText(nil, "y")
	tt.Frame()
	if s != "REPLACEDy" {
		t.Fatalf("text %q, want REPLACEDy", s)
	}
	tt.Key(Cmd, KeyZ)
	if s != "REPLACED" {
		t.Errorf("first undo %q, want REPLACED", s)
	}
	tt.Key(Cmd, KeyZ)
	if s != "hello" {
		t.Errorf("second undo %q, want hello", s)
	}
}

// The row chosen last leaves with its row, whose key the app removes from
// Chosen as the docs say: Current is nil, not the row that took its place
// (R2-19).
func TestDataTableCurrentAfterDeletingTheChosenRow(t *testing.T) {
	s, rows := dtState(), dtRows()
	tt := dtTester(s, &rows)
	defer tt.rt.close()
	tt.Click("alpha")
	if cur := s.Current(); cur == nil || cur.Name != "alpha" {
		t.Fatalf("chosen: %+v", cur)
	}
	rows = slices.Delete(rows, 1, 2)
	s.Chosen.Remove(2)
	s.Reset()
	tt.Frame()
	if cur := s.Current(); cur != nil {
		t.Errorf("Current after deleting its row: %+v, want nil", cur)
	}
}

// Space on one row chosen in a list of a million rows asks the row's key
// of a few rows, not of every row to find the one (R3-6).
func TestChosenRowsOneRowIsConstant(t *testing.T) {
	var chosen, checked Selection[int]
	calls := 0
	s := ListState{Selection: &chosen, Checked: &checked, Key: func(i int) any { calls++; return i }}
	tt := NewTester(func(c *Context) {
		List(c, &s, 1_000_000, func(i int) { Textf(c, "Row %d", i) }).Grow(1)
	}, 300, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Row 3"); err != nil {
		t.Fatal(err)
	}
	calls = 0
	tt.Key(0, KeySpace)
	if !checked.Has(3) {
		t.Fatalf("Space did not check the row chosen: %d checked", checked.Len())
	}
	if calls > 1000 {
		t.Errorf("Key was called %d times for one Space on a row", calls)
	}
}

// An input without OnEdit composing over a selection is as it was: the commit
// replaces the selection, and one undo gives it back.
func TestPlainInputCompositionOverASelectionIsUnchanged(t *testing.T) {
	s := "abc"
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.Key(Cmd, KeyA)
	c := tt.h.ime.Client
	c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
	tt.Frame()
	c.ReplaceText(nil, "y")
	tt.Frame()
	if s != "y" {
		t.Fatalf("text %q, want y", s)
	}
	tt.Key(Cmd, KeyZ)
	if s != "abc" {
		t.Errorf("undo gives %q, want abc", s)
	}
}

// A single-line input lays out a window of its line around the caret: after
// a paste over 524,288 runes, cut there, a keystroke lays out the window, not
// the line (R3-5). A line break becomes a space, as before.
func TestTextInputHugeValueKeystrokeCost(t *testing.T) {
	s := ""
	tt := NewTester(func(c *Context) { TextInput(c, &s).Fill().AutoFocus() }, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	paste := (1<<19 + 3072) / 3 * 3
	tt.SetClipboard(strings.Repeat("ab\n", paste/3))
	tt.Key(Cmd, KeyV)
	if laid := laidBy(tt, func() { tt.Type("x") }); laid > windowBound {
		t.Errorf("a keystroke laid out %d runes, more than %d", laid, windowBound)
	}
	if n := utf8.RuneCountInString(s); n != 1<<19+1 || strings.Contains(s, "\n") || !strings.HasPrefix(s, "ab ab ") || !strings.HasSuffix(s, "x") {
		t.Errorf("value of %d runes, want the paste of %d cut at %d with its breaks as spaces, then the x", n, paste, 1<<19)
	}
}

// The fast path of one chosen row is the row chosen, not whatever the
// cursor is on: Space checks the row chosen.
func TestChosenRowsOneRowIsNotTheCursorRow(t *testing.T) {
	var chosen, checked Selection[int]
	s := ListState{Selection: &chosen, Checked: &checked, Key: func(i int) any { return i }}
	tt := NewTester(func(c *Context) {
		List(c, &s, 100, func(i int) { Textf(c, "Row %d", i) }).Grow(1)
	}, 300, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Row 3"); err != nil {
		t.Fatal(err)
	}
	s.lead = 5 // the cursor moved to another row, which is not chosen
	tt.Frame()
	tt.Key(0, KeySpace)
	if !checked.Has(3) || checked.Has(5) || checked.Len() != 1 {
		t.Errorf("Space checked %d rows, has 3: %v, has 5: %v; want only row 3", checked.Len(), checked.Has(3), checked.Has(5))
	}
}

// Up from 01:30 on the night the clocks go forward steps to 03:30, the hour
// that follows, not back to the same hour (R3-20).
func TestTimeInputStepsOverSpringForward(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	alarm := time.Date(2026, 3, 8, 1, 30, 0, 0, ny)
	tt := coreNewTester(func(c *context) { coreTimeInput(c, &alarm).Label("Alarm") }, 300, 100)
	defer tt.rt.close()
	tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
	tt.Click("01")
	tt.Key(0, KeyUp)
	if alarm.Hour() != 3 || alarm.Minute() != 30 {
		t.Errorf("Up: %v, want 03:30", alarm)
	}
}

// A number typed beyond its range is clamped when the field is left, and the
// field shows the clamped number (R3-13: the old value stayed, silently).
func TestNumberInputOutOfRangeClampsOnCommit(t *testing.T) {
	v := 5.0
	tt := NewTester(func(c *Context) {
		NumberInput(c, &v, 1, 99, 1).Label("n")
		Button(c, "next")
	}, 300, 100)
	defer tt.rt.close()
	tt.Key(0, KeyTab)
	ed := tt.rt.states[tt.rt.focused].editor
	tt.Key(Cmd, KeyA)
	tt.Type("150")
	if v != 5 {
		t.Fatalf("value %v while typing, want it kept at 5 until the field is left", v)
	}
	tt.Key(0, KeyTab)
	if v != 99 {
		t.Errorf("value %v, want 99", v)
	}
	if ed.buf.s != "99" {
		t.Errorf("text %q, want 99", ed.buf.s)
	}
}

// Enter confirms a number beyond the range as the nearest end of it too.
func TestNumberInputOutOfRangeClampsOnEnter(t *testing.T) {
	v := 5.0
	tt := NewTester(func(c *Context) { NumberInput(c, &v, 1, 99, 1).Label("n") }, 300, 100)
	defer tt.rt.close()
	tt.Key(0, KeyTab)
	ed := tt.rt.states[tt.rt.focused].editor
	tt.Key(Cmd, KeyA)
	tt.Type("150")
	tt.Key(0, KeyEnter)
	if v != 99 || ed.buf.s != "99" {
		t.Errorf("value %v, text %q, want 99 in both", v, ed.buf.s)
	}
}
