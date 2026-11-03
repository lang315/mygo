package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An input method with a preedit (macOS Telex, TSF, fcitx) composes the
// word, and Type commits it: the draft is what was committed.
func TestEditCellPreeditComposition(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	for _, pre := range []string{"t", "ti", "tie", "tiê", "tiế", "tiến", "tiếng"} {
		tt.Compose(pre, len([]rune(pre)))
		tt.Frame()
	}
	tt.Type("tiếng")
	tt.Type(" Việt")
	tt.Frame()
	if d := grid.editDraft(); d != "tiếng Việt" {
		t.Fatalf("draft %q, want tiếng Việt", d)
	}
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "tiếng Việt" {
		t.Fatalf("row %+v, want the word written", r)
	}
}

// Input methods without a preedit (UniKey, EVKey) rewrite the word with
// Backspace and typed characters.
func TestEditCellBackspaceRewrite(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Name")
	settleEdit(tt)
	for _, step := range []struct {
		backspaces int
		text       string
	}{{0, "tie"}, {1, "ê"}, {1, "ế"}, {0, "ng"}, {0, " Việt"}} {
		for range step.backspaces {
			tt.Key(0, ui.KeyBackspace)
		}
		tt.Type(step.text)
		tt.Frame()
	}
	if d := grid.editDraft(); d != "tiếng Việt" {
		t.Fatalf("draft %q, want tiếng Việt", d)
	}
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if r := rowOf(t, grid, 1); r.name != "tiếng Việt" {
		t.Fatalf("row %+v, want the word written", r)
	}
}

// Begin, typing, cancel and a refused commit do not touch the source: the
// row's generation moves only when a commit wrote.
func TestEditingLeavesTheSourceAlone(t *testing.T) {
	tt, grid, _ := editGrid(t)
	before := grid.src.Version(1)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("zzz")
	tt.Frame()
	if err := grid.EndEdit(); err == nil { // refused: not a number
		t.Fatal("EndEdit accepted zzz as a score")
	}
	grid.CancelEdit()
	settleEdit(tt)
	if v := grid.src.Version(1); v != before {
		t.Fatalf("Version(1) moved from %d to %d without a commit", before, v)
	}
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("55")
	tt.Frame()
	if err := grid.EndEdit(); err != nil {
		t.Fatal(err)
	}
	if v := grid.src.Version(1); v == before {
		t.Error("a commit that wrote did not move the row's generation")
	}
	if r := rowOf(t, grid, 1); r.score != 55 {
		t.Errorf("row %+v, want score 55", r)
	}
}

// The editor is a text input: assistive technology reads its value, and the
// refusal is announced and given as the description.
func TestEditorIsATextInputAndTheRefusalIsAnnounced(t *testing.T) {
	tt, grid, _ := editGrid(t)
	grid.BeginEdit(1, "Score")
	settleEdit(tt)
	tt.Type("abc")
	tt.Frame()
	var input *ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTextField {
			n := n
			input = &n
		}
	}
	if input == nil {
		t.Fatalf("no text input in the tree while editing")
	}
	if input.Value != "abc" {
		t.Errorf("the editor's value is %q, want abc", input.Value)
	}
	_ = grid.EndEdit()
	settleEdit(tt)
	found := false
	for _, a := range tt.Announcements() {
		found = found || a == "Score must be a number"
	}
	if !found {
		t.Errorf("the refusal was not announced: %q", tt.Announcements())
	}
}
