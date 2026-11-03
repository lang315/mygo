package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A double click opens the editor of a cell with the caret where the click
// was, in the text it keeps: at the start, between two letters, past the end.
func TestCellEditorCaretAtTheDoubleClick(t *testing.T) {
	for _, c := range []struct {
		name string
		dx   float32 // from the left of "ann", which is 22 DIPs wide
		want string
	}{
		{"start", 0.5, "Xann"},
		{"between the letters", 7, "aXnn"},
		{"past the end", 40, "annX"},
	} {
		t.Run(c.name, func(t *testing.T) {
			tt, grid, _ := editGrid(t)
			r := find(t, tt, "ann")
			x, y := r.X+c.dx, r.Y+r.H/2
			tt.ClickAt(x, y)
			tt.ClickAt(x, y)
			settleEdit(tt) // the editor builds in the frame after the Post that invalidates
			tt.Frame()     // its caret at a point needs its layout
			if !grid.IsEditing() {
				t.Fatal("a double click did not start an edit")
			}
			tt.Type("X")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			settleEdit(tt)
			if got := rowOf(t, grid, 1).name; got != c.want {
				t.Errorf("name %q, want %q", got, c.want)
			}
		})
	}
}
