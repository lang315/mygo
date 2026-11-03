package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// editName opens the editor of row 1's Name ("ann") as open says, plays
// keys, presses Enter and returns the name the grid wrote. The commit runs
// on the event goroutine: settleEdit waits for it.
func editName(t *testing.T, open func(tt *ui.Tester), keys func(tt *ui.Tester)) string {
	t.Helper()
	tt, grid, _ := editGrid(t)
	open(tt)
	settleEdit(tt) // the editor builds in the frame after the Post that invalidates
	tt.Frame()     // its caret at a point needs its layout
	if k, col, ok := grid.EditingCell(); !ok || k != 1 || col != "Name" {
		t.Fatalf("EditingCell = %v %q %v, want 1 Name", k, col, ok)
	}
	keys(tt)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if grid.IsEditing() {
		t.Fatal("Enter did not end the edit")
	}
	return rowOf(t, grid, 1).name
}

func openWithF2(t *testing.T) func(tt *ui.Tester) {
	return func(tt *ui.Tester) {
		if err := tt.Click("ann"); err != nil {
			t.Fatal(err)
		}
		tt.Key(0, ui.KeyF2)
	}
}

// The arrow keys, Home and Shift+Left move the caret and the selection of the
// editor opened with F2, where typing puts the text.
func TestCellEditorCaretKeysAfterF2(t *testing.T) {
	for _, c := range []struct {
		name string
		keys func(tt *ui.Tester)
		want string
	}{
		{"Left", func(tt *ui.Tester) { tt.Type("abc"); tt.Key(0, ui.KeyLeft); tt.Type("X") }, "abXc"},
		{"Home", func(tt *ui.Tester) { tt.Type("abc"); tt.Key(0, ui.KeyHome); tt.Type("X") }, "Xabc"},
		{"End after Home", func(tt *ui.Tester) { tt.Type("abc"); tt.Key(0, ui.KeyHome); tt.Key(0, ui.KeyEnd); tt.Type("X") }, "abcX"},
		{"Left, Left, Right", func(tt *ui.Tester) {
			tt.Type("abc")
			tt.Key(0, ui.KeyLeft)
			tt.Key(0, ui.KeyLeft)
			tt.Key(0, ui.KeyRight)
			tt.Type("X")
		}, "abXc"},
		{"Shift+Left selects", func(tt *ui.Tester) { tt.Type("abc"); tt.Key(ui.Shift, ui.KeyLeft); tt.Type("X") }, "abX"},
		{"Shift+Home selects", func(tt *ui.Tester) { tt.Type("abc"); tt.Key(ui.Shift, ui.KeyHome); tt.Type("X") }, "X"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := editName(t, openWithF2(t), c.keys); got != c.want {
				t.Errorf("name %q, want %q", got, c.want)
			}
		})
	}
}

// The grid does not take Left and Right while a cell is edited: the caret
// moves in the editor and the current cell stays.
func TestArrowKeysInTheEditorDoNotMoveTheCurrentCell(t *testing.T) {
	tt, grid, _ := editGrid(t)
	tt.Click("ann")
	k0, col0, ok := grid.CurrentCell()
	if !ok || k0 != 1 || col0 != "Name" {
		t.Fatalf("CurrentCell = %v %q %v, want 1 Name", k0, col0, ok)
	}
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Type("abc")
	for _, key := range []ui.Key{ui.KeyLeft, ui.KeyLeft, ui.KeyRight, ui.KeyRight, ui.KeyRight, ui.KeyLeft} {
		tt.Key(0, key)
		tt.Frame()
		if k, col, ok := grid.CurrentCell(); !ok || k != k0 || col != col0 {
			t.Fatalf("after %v CurrentCell = %v %q %v, want %v %q", key, k, col, ok, k0, col0)
		}
		if k, col, ok := grid.EditingCell(); !ok || k != k0 || col != col0 {
			t.Fatalf("after %v EditingCell = %v %q %v, want %v %q", key, k, col, ok, k0, col0)
		}
	}
	tt.Type("X")
	tt.Key(0, ui.KeyEnter)
	settleEdit(tt)
	if got := rowOf(t, grid, 1).name; got != "abXc" {
		t.Errorf("name %q, want abXc (the arrows moved the caret: right to the end, left once)", got)
	}
}
