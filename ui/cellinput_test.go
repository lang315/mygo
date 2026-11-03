package ui

import (
	"testing"
	"time"
)

// EditableText still edits as it did when its editor became CellInput's: a
// double click starts it, typing and Enter write the value, Escape goes back.
// (The refactor was checked to leave the pixels of an editing EditableText as
// they were, on one machine; a pixel checksum is not portable across the
// fonts and rasterizers of the three systems, so none is pinned.)
func TestEditableTextEditsAsBefore(t *testing.T) {
	name := "report.txt"
	changed := 0
	tt := NewTester(func(c *Context) {
		Row(c).Padding(8).Children(func() {
			if EditableText(c, &name).Changed() {
				changed++
			}
		})
	}, 300, 80)
	tt.Frame()
	edit := func() {
		t.Helper()
		tt.Advance(time.Second) // clicks of an earlier edit are not part of this double click
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("the text %q is not shown: %q", name, tt.Texts())
		}
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		if active, _ := tt.TextInputActive(); !active {
			t.Fatalf("the double click on %q did not start editing", name)
		}
	}
	edit()
	tt.Key(Cmd, KeyA)
	tt.Type("notes.md")
	tt.Key(0, KeyEnter)
	tt.Frame()
	if name != "notes.md" || changed == 0 {
		t.Fatalf("after typing and Enter: name %q, changed %d; want notes.md and a change reported", name, changed)
	}
	edit()
	tt.Key(Cmd, KeyA)
	tt.Type("lost.txt")
	tt.Key(0, KeyEscape)
	tt.Frame()
	if name != "notes.md" {
		t.Fatalf("Escape kept %q, want notes.md", name)
	}
}

func TestCellInputStartsFocusedWithAllSelected(t *testing.T) {
	draft := "tiếng"
	tt := NewTester(func(c *Context) {
		CellInput(c, &draft, CellStart{Fresh: true, SelectAll: true})
	}, 300, 80)
	tt.Frame()
	tt.Frame()
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("the editor did not take the focus")
	}
	tt.Type("x")
	tt.Frame()
	if draft != "x" {
		t.Fatalf("typing over a selection of all: draft %q, want x", draft)
	}
}

func TestCellInputCaretAtPoint(t *testing.T) {
	draft := "abcdef"
	tt := NewTester(func(c *Context) { CellInput(c, &draft, CellStart{Fresh: true, HasAt: true, X: 0, Y: 20}) }, 300, 80)
	tt.Frame()
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if draft != "Xabcdef" { // a point at the left edge puts the caret before the text
		t.Fatalf("caret at the left edge: draft %q, want Xabcdef", draft)
	}
}

func TestCellInputEnterEscapeTab(t *testing.T) {
	draft, esc, tab := "a", 0, 0
	submitted := false
	tt := NewTester(func(c *Context) {
		e := CellInput(c, &draft, CellStart{Fresh: true})
		e.OnShortcut(0, KeyEscape, func() { esc++ })
		e.OnShortcut(0, KeyTab, func() { tab++ })
		if e.Submitted() {
			submitted = true
		}
	}, 300, 80)
	tt.Frame()
	tt.Frame()
	tt.Key(0, KeyEscape)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	tt.Frame()
	if esc != 1 || tab != 1 || !submitted {
		t.Fatalf("Escape %d, Tab %d, Enter submitted %v; want 1, 1, true", esc, tab, submitted)
	}
}

// A composition reaches the draft through the cell editor, as it does the
// text input's.
func TestCellInputComposition(t *testing.T) {
	draft := ""
	tt := NewTester(func(c *Context) { CellInput(c, &draft, CellStart{Fresh: true}) }, 300, 80)
	defer tt.rt.close()
	tt.Frame()
	tt.Frame()
	c := tt.h.ime.Client
	for _, pre := range []string{"t", "ti", "tie", "tiê", "tiế", "tiến", "tiếng"} {
		c.SetMarkedText(nil, pre, TextInputRange{})
		tt.Frame()
	}
	c.ReplaceText(nil, "tiếng")
	c.SetMarkedText(nil, "", TextInputRange{})
	tt.Frame()
	if draft != "tiếng" {
		t.Fatalf("draft %q, want tiếng", draft)
	}
}

// A point past the text puts the caret at its end.
func TestCellInputCaretAtPointRight(t *testing.T) {
	draft := "abcdef"
	tt := NewTester(func(c *Context) { CellInput(c, &draft, CellStart{Fresh: true, HasAt: true, X: 290, Y: 20}) }, 300, 80)
	tt.Frame()
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if draft != "abcdefX" {
		t.Fatalf("caret past the text: draft %q, want abcdefX", draft)
	}
}

// Fresh is for the first frame only, as the contract says: the point is
// applied once the editor has a layout, not in the frame that builds it.
func TestCellInputCaretAtPointWhenFreshOnlyOnTheFirstFrame(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want string
	}{{0, "Xabcdef"}, {290, "abcdefX"}} {
		draft, frames := "abcdef", 0
		tt := NewTester(func(c *Context) {
			CellInput(c, &draft, CellStart{Fresh: frames == 0, HasAt: true, X: tc.x, Y: 20})
			frames++
		}, 300, 80)
		for range 4 {
			tt.Frame()
		}
		tt.Type("X")
		tt.Frame()
		if draft != tc.want {
			t.Errorf("point x=%v, Fresh on the first frame only: draft %q, want %q", tc.x, draft, tc.want)
		}
	}
}
