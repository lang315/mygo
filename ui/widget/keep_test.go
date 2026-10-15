package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An entry moved to another box keeps its caret: typing goes where the
// caret was, not to the end.
func TestReparentedWidgetKeepsItsState(t *testing.T) {
	e := Entry().SetText("abcdef")
	left, right := VBox(0).Add(e), VBox(0)
	tt := tester(t, HBox(0).Add(left, right))
	e.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	left.Remove(e)
	right.Add(e)
	tt.Frame()
	e.Focus()
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if got := e.Text(); got != "abXcdef" {
		t.Fatalf("text %q after moving the entry and typing X, want abXcdef", got)
	}
}
