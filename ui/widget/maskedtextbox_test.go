package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMaskEngineShowsATemplateAndFillsSlotsSkippingLiterals(t *testing.T) {
	m := newMask("000-0000", '_')
	if got := m.display(); got != "___-____" {
		t.Fatalf("empty: %q", got)
	}
	text, caret, _ := m.edit(m.display(), 0, 0, "1234567")
	if text != "123-4567" || caret != 8 {
		t.Fatalf("typed: %q caret %d", text, caret)
	}
	m = newMask("000-0000", '_')
	text, caret, _ = m.edit(m.display(), 0, 0, "12")
	if text != "12_-____" || caret != 2 {
		t.Fatalf("two digits: %q caret %d", text, caret)
	}
	text, caret, _ = m.edit(text, 2, 2, "3")
	if text != "123-____" || caret != 3 {
		t.Fatalf("third: %q caret %d (the caret waits before the literal)", text, caret)
	}
	text, caret, _ = m.edit(text, 3, 3, "4") // the caret is on the literal: the digit goes to the next slot
	if text != "123-4___" || caret != 5 {
		t.Fatalf("fourth over the literal: %q caret %d", text, caret)
	}
}

func TestMaskEngineRejectsWhatASlotDoesNotTake(t *testing.T) {
	m := newMask("000", '_')
	text, caret, _ := m.edit(m.display(), 0, 0, "1a2b3")
	if text != "123" || caret != 3 || len(m.rejected) != 2 {
		t.Fatalf("%q caret %d rejected %q", text, caret, string(m.rejected))
	}
	m = newMask("LL", '_')
	text, _, _ = m.edit(m.display(), 0, 0, "a1b")
	if text != "ab" {
		t.Fatalf("letters: %q", text)
	}
	m = newMask(">LL<LL", '_') // upper, then lower
	text, _, _ = m.edit(m.display(), 0, 0, "abcd")
	if text != "ABcd" {
		t.Fatalf("case: %q", text)
	}
}

func TestMaskEngineDeletesShiftingLaterCharactersLeft(t *testing.T) {
	m := newMask("000-0000", '_')
	text := "123-4567"
	// Backspace after the 3rd digit.
	text, caret, _ := m.edit(text, 2, 3, "")
	if text != "124-567_" || caret != 2 {
		t.Fatalf("delete: %q caret %d", text, caret)
	}
	// Backspace on a literal removes the editable char before it.
	text, caret, _ = m.edit("123-4567", 3, 4, "", 4)
	if text != "124-567_" || caret != 2 {
		t.Fatalf("backspace over the literal: %q caret %d", text, caret)
	}
	// Delete (forward) with the caret before a literal removes the next digit.
	text, caret, _ = m.edit("123-4567", 3, 4, "", 3)
	if text != "123-567_" || caret != 4 {
		t.Fatalf("delete over the literal: %q caret %d", text, caret)
	}
}

func TestMaskEngineInsertsShiftingLaterCharactersRight(t *testing.T) {
	m := newMask("000-0000", '_')
	text, caret, _ := m.edit("123-45__", 1, 1, "9")
	if text != "192-345_" || caret != 2 {
		t.Fatalf("insert: %q caret %d", text, caret)
	}
	// A full mask refuses what would push a character off its end.
	text, _, _ = m.edit("123-4567", 0, 0, "0")
	if text != "123-4567" {
		t.Fatalf("full: %q", text)
	}
}

func TestMaskEngineReplacesASelectionAndOptionalSlots(t *testing.T) {
	m := newMask("000-0000", '_')
	text, caret, _ := m.edit("123-4567", 1, 6, "9")
	if text != "196-7___" {
		t.Fatalf("selection replaced: %q caret %d", text, caret)
	}
	o := newMask("00-99", '_') // two required digits, two optional
	if o.completed("12-__") != true || o.full("12-__") {
		t.Fatalf("completed %v full %v", o.completed("12-__"), o.full("12-__"))
	}
	if o.completed("1_-__") {
		t.Fatal("completed with a required slot empty")
	}
}

func TestMaskEngineLiteralsEscapeAndText(t *testing.T) {
	m := newMask(`(000) \0-00`, '_')
	if got := m.display(); got != "(___) 0-__" {
		t.Fatalf("display %q", got)
	}
	text, _, _ := m.edit(m.display(), 0, 0, "12345")
	if text != "(123) 0-45" {
		t.Fatalf("text %q", text)
	}
	if got := m.value(text); got != "12345" {
		t.Fatalf("value (no literals, no prompts) %q", got)
	}
}

func TestMaskedTextBoxTypesThroughTheMask(t *testing.T) {
	mt := MaskedTextBox("000-0000")
	tt := NewTester(t, mt, 400, 120)
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	if box.Value() != "___-____" {
		t.Fatalf("shows %q", box.Value())
	}
	box.Key(ui.Cmd, ui.KeyHome)
	box.Type("12a34567")
	if mt.TextWithLiterals() != "123-4567" || mt.Text() != "1234567" || !mt.MaskCompleted() || !mt.MaskFull() {
		t.Fatalf("TextWithLiterals %q Text %q completed %v", mt.TextWithLiterals(), mt.Text(), mt.MaskCompleted())
	}
}

func TestMaskedTextBoxSetTextPasteAndEvents(t *testing.T) {
	changed := make(chan struct{}, 8)
	rejected := make(chan rune, 8)
	mt := MaskedTextBox("00/00").OnChange(func() { changed <- struct{}{} }).OnMaskInputRejected(func(r rune) { rejected <- r })
	tt := NewTester(t, mt, 400, 120)
	if !mt.SetText("1231") {
		t.Fatal("SetText refused valid characters")
	}
	tt.Settle()
	if mt.TextWithLiterals() != "12/31" || mt.MaskCompleted() != true {
		t.Fatalf("SetText: %q", mt.TextWithLiterals())
	}
	select {
	case <-changed:
		t.Fatal("SetText ran OnChange")
	default:
	}
	if mt.SetText("12x1") {
		t.Fatal("SetText accepted a letter in a digit mask")
	}
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyA)
	tt.SetClipboard("9-8/7z6")
	box.Key(ui.Cmd, ui.KeyV)
	if got := mt.TextWithLiterals(); got != "98/76" {
		t.Fatalf("paste filtered through the mask: %q", got)
	}
	received(t, changed)
	if r := received(t, rejected); r != '-' && r != '/' && r != 'z' {
		t.Fatalf("rejected %q", r)
	}
}

func TestMaskedTextBoxPromptCharAndEmpty(t *testing.T) {
	mt := MaskedTextBox("000").SetPromptChar('*')
	tt := NewTester(t, mt, 400, 120)
	if v := tt.Find(ByRole(ui.RoleTextField)).Value(); v != "***" {
		t.Fatalf("shows %q", v)
	}
	if mt.Text() != "" || mt.MaskCompleted() || mt.MaskFull() {
		t.Fatal("an empty mask reads as filled")
	}
	mt.SetMask("00-00")
	tt.Settle()
	if v := tt.Find(ByRole(ui.RoleTextField)).Value(); v != "**-**" {
		t.Fatalf("after SetMask shows %q", v)
	}
}

func TestMaskedTextBoxCaretCrossesLiteralsForBackspaceAndDelete(t *testing.T) {
	mt := MaskedTextBox("000-0000")
	tt := NewTester(t, mt, 400, 120)
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyHome)
	box.Type("1234567")
	for range 4 {
		box.Key(0, ui.KeyLeft) // the caret is right after the dash
	}
	box.Key(0, ui.KeyBackspace)
	if got := mt.TextWithLiterals(); got != "124-567 " {
		t.Fatalf("Backspace over the dash: %q", got)
	}
	box.Key(ui.Cmd, ui.KeyHome)
	for range 3 {
		box.Key(0, ui.KeyRight) // right before the dash
	}
	box.Key(0, ui.KeyDelete)
	if got := mt.TextWithLiterals(); got != "124-67  " {
		t.Fatalf("Delete before the dash: %q", got)
	}
	box.Type("9") // the caret sits before the dash: the digit goes to the next slot
	if got := mt.TextWithLiterals(); got != "124-967 " {
		t.Fatalf("typed before the dash: %q", got)
	}
}

func TestMaskedTextBoxUndoIsOneStepPerEdit(t *testing.T) {
	mt := MaskedTextBox("00-00")
	tt := NewTester(t, mt, 400, 120)
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyHome)
	box.Type("12")
	box.Key(ui.Cmd, ui.KeyA)
	box.Type("9")
	box.Key(ui.Cmd, ui.KeyZ)
	if got := mt.TextWithLiterals(); got != "12-  " {
		t.Fatalf("after undo: %q", got)
	}
}

// An edit that replaces nothing with nothing, as an input method's
// composition starting, is not a Backspace.
func TestMaskEngineAnEditThatChangesNothingChangesNothing(t *testing.T) {
	m := newMask("000-0000", '_')
	for _, c := range []struct {
		text  string
		caret int
	}{{"12_-____", 2}, {"123-4___", 5}} {
		if got, caret, _ := m.edit(c.text, c.caret, c.caret, "", c.caret); got != c.text || caret != c.caret {
			t.Errorf("empty edit at %d of %q -> %q caret %d", c.caret, c.text, got, caret)
		}
	}
}

// An input method composing letters into a mask keeps the letters before.
func TestMaskedTextBoxKeepsWhatWasTypedWhenAnInputMethodComposes(t *testing.T) {
	mt := MaskedTextBox(">LLL")
	tt := NewTester(t, mt, 400, 120)
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyHome)
	for _, w := range []string{"Đ", "ứ", "c"} {
		tt.Tester.Compose(w, 1)
		tt.Tester.Type(w)
	}
	if got := mt.Text(); got != "ĐỨC" {
		t.Errorf("Text = %q, want ĐỨC", got)
	}
}

// WinForms refuses an insert whose shifted characters no longer fit.
func TestMaskEngineRefusesAnInsertThatPushesCharactersOut(t *testing.T) {
	m := newMask("000-0000", '_')
	got, caret, _ := m.edit("123-4567", 0, 0, "9", 0)
	if got != "123-4567" || caret != 0 || string(m.rejected) != "9" {
		t.Errorf("full mask: %q caret %d rejected %q", got, caret, string(m.rejected))
	}
	m = newMask("LL00", '_')
	got, _, _ = m.edit("AB12", 0, 0, "C", 0)
	if got != "AB12" || string(m.rejected) != "C" {
		t.Errorf("LL00: %q rejected %q", got, string(m.rejected))
	}
	m = newMask("000-0000", '_')
	got, _, _ = m.edit("123-45__", 0, 0, "9", 0) // room for it: the rest moves
	if got != "912-345_" || len(m.rejected) != 0 {
		t.Errorf("room left: %q rejected %q", got, string(m.rejected))
	}
	got, _, _ = m.edit("123-4567", 0, 0, "98", 0)
	if got != "123-4567" || string(m.rejected) != "9" {
		t.Errorf("a paste is refused whole, and reports once: %q rejected %q", got, string(m.rejected))
	}
}

// WinForms' SkipLiterals: a character equal to the literal at the caret steps
// over it.
func TestMaskEngineSkipsLiteralsATypedCharacterEquals(t *testing.T) {
	m := newMask("(000) 000-0000", '_')
	got, caret, _ := m.edit(m.display(), 0, 14, "(012) 345-6789", 0)
	if got != "(012) 345-6789" || caret != 14 || len(m.rejected) != 0 {
		t.Errorf("formatted paste: %q caret %d rejected %q", got, caret, string(m.rejected))
	}
	m = newMask("000-0000", '_')
	got, caret, _ = m.edit("123-____", 3, 3, "-", 3)
	if got != "123-____" || caret != 4 || len(m.rejected) != 0 {
		t.Errorf("typing the dash: %q caret %d rejected %q", got, caret, string(m.rejected))
	}
	got, _, _ = m.edit("12_-____", 2, 2, "-", 2) // the slot before the dash is empty: '-' is no digit
	if string(m.rejected) != "-" {
		t.Errorf("a literal that is not at the caret is rejected: %q rejected %q", got, string(m.rejected))
	}
}

// The prompt is a display; a character equal to it still fills its slot.
func TestMaskedTextBoxKeepsATypedCharacterEqualToThePrompt(t *testing.T) {
	mt := MaskedTextBox("CCC")
	tt := NewTester(t, mt, 400, 120)
	box := tt.Find(ByRole(ui.RoleTextField)).Focus()
	box.Key(ui.Cmd, ui.KeyHome)
	box.Type("a_b")
	if got := mt.Text(); got != "a_b" {
		t.Errorf("a typed _ in CCC: Text = %q, want a_b", got)
	}
	mt.SetPromptChar('-')
	tt.Settle()
	if got := mt.Text(); got != "a_b" {
		t.Errorf("SetPromptChar: Text = %q, want a_b", got)
	}
	box.Key(ui.Cmd, ui.KeyHome)
	box.Key(ui.Cmd, ui.KeyA)
	box.Type("a-b")
	if got := mt.Text(); got != "a-b" || !mt.MaskFull() {
		t.Errorf("typing the prompt: Text = %q full %v, want a-b", got, mt.MaskFull())
	}
}

// WinForms keeps the text when the mask changes.
func TestMaskedTextBoxSetMaskReappliesTheText(t *testing.T) {
	changed := make(chan struct{}, 4)
	mt := MaskedTextBox("000-0000").OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, mt, 400, 120)
	mt.SetText("1234567")
	mt.SetMask("00-00")
	tt.Settle()
	if got := mt.TextWithLiterals(); got != "12-34" {
		t.Errorf("after SetMask: %q, want 12-34 (what does not fit is dropped)", got)
	}
	mt.SetMask("LL")
	tt.Settle()
	if got := mt.TextWithLiterals(); got != "  " || mt.Text() != "" {
		t.Errorf("digits in a letter mask: %q", got)
	}
	select {
	case <-changed:
		t.Error("SetMask ran OnChange")
	default:
	}
}

// WinForms' Text takes the literals too.
func TestMaskedTextBoxSetTextTakesLiteralsToo(t *testing.T) {
	mt := MaskedTextBox("000-0000")
	NewTester(t, mt, 400, 120)
	if !mt.SetText("123-4567") || mt.Text() != "1234567" {
		t.Errorf("SetText with the dash: %q", mt.Text())
	}
	if !mt.SetText("12 -    ") || mt.Text() != "12" {
		t.Errorf("SetText with blanks: %q", mt.Text())
	}
	if mt.SetText("123+4567") {
		t.Error("SetText took a wrong literal")
	}
	if mt.Text() != "12" {
		t.Errorf("a refused SetText changed the text: %q", mt.Text())
	}
}
