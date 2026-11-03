package ui

import (
	"hash/crc32"
	"testing"
)

// editableFrame returns the checksum of the frame of an EditableText being
// edited, after "!" was typed.
func editableFrame(t *testing.T) uint32 {
	t.Helper()
	name := "report.txt"
	tt := NewTester(func(c *Context) {
		Row(c).Padding(8).Children(func() { EditableText(c, &name) })
	}, 300, 80)
	tt.Frame()
	r, ok := tt.Find("report.txt")
	if !ok {
		t.Fatalf("the text is not shown: %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("the double click did not start editing")
	}
	tt.Type("!")
	tt.Frame()
	return crc32.ChecksumIEEE(tt.Image().Pix)
}

// EditableText keeps its look: the refactor that builds its editor with the
// code CellInput shares must not change a frame.
func TestEditableTextEditorUnchanged(t *testing.T) {
	const want = 0xf4cdd1ed
	if got := editableFrame(t); got != want {
		t.Fatalf("frame checksum %#x, want %#x", got, want)
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
