package ui

import "testing"

// GTK's fcitx order (empty preedit before or after
// the commit) over a selection, in a password field, and in a multiline
// input: one undo takes the whole composition back, selection included.
func TestWidgetCompositionUndoesOnceVariants(t *testing.T) {
	for _, kind := range []string{"plain", "selected", "password", "passwordSelected", "multiline", "multilineSelected"} {
		for _, preeditAfter := range []bool{false, true} {
			s := "cafe"
			multiline := kind == "multiline" || kind == "multilineSelected"
			pw := kind == "password" || kind == "passwordSelected"
			tt := coreNewTester(func(c *context) {
				e := textInput(c, &s, multiline).Fill().AutoFocus()
				if pw {
					e.Password()
				}
			}, 300, 120)
			tt.Frame()
			ed := tt.rt.states[tt.rt.focused].editor
			if kind == "selected" || kind == "passwordSelected" || kind == "multilineSelected" {
				ed.anchor, ed.caret = 1, 3
			} else {
				ed.anchor, ed.caret = 4, 4
			}
			tt.Frame()
			composeOver(t, tt.h.ime.Client, "你好", preeditAfter)
			tt.Frame()
			after := s
			tt.Key(Cmd, KeyZ)
			tt.Frame()
			if s != "cafe" {
				t.Errorf("%s preeditAfter=%v: after commit %q, one undo gives %q, want %q", kind, preeditAfter, after, s, "cafe")
			}
			tt.rt.close()
		}
	}
}
