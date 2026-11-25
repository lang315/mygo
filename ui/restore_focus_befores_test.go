package ui

import "testing"

// A dialog opened from a dialog that closes first gives the focus, as it
// closes, to the element the first dialog was opened from (befores), not to
// the first element of the dialog under it: the element it was itself opened
// from went with the dialog that closed first. arrangeFocus takes befores:
// restoreFocus would too, but too late, after the dialog under has moved the
// focus to its first element.
func TestRestoreFocusThroughADialogGoneFirst(t *testing.T) {
	var second, third bool
	first := true
	var opener *node
	tt := coreNewTester(func(c *context) {
		coreModal(c, &first, func() {
			coreTextInputBase(c, new(string)) // the first element of the dialog
			opener = coreTextInputBase(c, new(string))
		})
		// Keyed: the third keeps its elements, and its record of where it
		// opened from, when the second goes.
		if second {
			c.nextKey, c.keySet, c.keyGlobal = "second", true, false
		}
		coreModal(c, &second, func() { coreTextInputBase(c, new(string)) })
		if third {
			c.nextKey, c.keySet, c.keyGlobal = "third", true, false
		}
		coreModal(c, &third, func() { coreTextInputBase(c, new(string)) })
	}, 400, 400)
	tt.Frame()
	opener.Focus()
	tt.Frame()
	if tt.rt.focused != opener.id {
		t.Fatal("the opener did not take the focus")
	}
	second = true
	tt.Frame()
	tt.Frame()
	inSecond := tt.rt.focused
	if inSecond == opener.id {
		t.Fatal("the second dialog did not take the focus")
	}
	third = true
	tt.Frame()
	tt.Frame()
	inThird := tt.rt.focused
	if inThird == inSecond {
		t.Fatal("the third dialog did not take the focus")
	}
	// The second dialog goes first, the third stays on top with the focus.
	second = false
	tt.Frame()
	tt.Frame()
	if tt.rt.focused != inThird {
		t.Fatalf("the third dialog lost the focus as the second closed: %d, want %d", tt.rt.focused, inThird)
	}
	third = false
	tt.Frame()
	tt.Frame()
	if tt.rt.focused != opener.id {
		t.Errorf("focus %d, want the first dialog's opener %d", tt.rt.focused, opener.id)
	}
}
