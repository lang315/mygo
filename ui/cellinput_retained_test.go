package ui

import "testing"

// editScene is a table whose rows are kept between frames (RowVersion) and
// whose cell at (editRow, 1) shows a CellInput while it is being edited.
type editScene struct {
	tt      *Tester
	s       *ListState
	editRow int
	gen     uint64
	draft   string
}

func newEditScene(keep bool) *editScene {
	e := &editScene{editRow: -1}
	e.s = &ListState{CurrentCell: true, Selection: &Selection[int]{}, Key: func(i int) any { return i }}
	e.s.EditCell = func(row, col int, how EditHow) bool { return col == 1 }
	e.s.OnEditCell = func(row, col int, x, y float32, how EditHow) {
		e.editRow, e.draft = row, "name"+string(rune('0'+row))
		e.gen++
		e.tt.rt.requestFrame()
	}
	e.s.RowVersion = func(i int) uint64 {
		if i == e.editRow {
			return 1000 + e.gen
		}
		return uint64(i) + 1
	}
	cols := []TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 160}}
	e.tt = coreNewTester(func(c *context) {
		coreTable(c, e.s, cols, 30, func(row, col int) {
			if row == e.editRow && col == 1 {
				in := wrapElement(coreCellInput(c, &e.draft, CellStart{Fresh: e.gen > 0 && e.editRow == row, SelectAll: true}))
				if in.Submitted() {
					e.editRow = -1
					e.gen++
				}
				return
			}
			coreText(c, "r"+string(rune('a'+row%26))+string(rune('0'+col)))
		}).Grow(1)
	}, 400, 300)
	e.tt.rt.noRetain = !keep
	e.tt.Frame()
	return e
}

// While an edit is open the frames of a table that keeps its rows are those of
// one that builds them all: the editor's row is built each frame and the
// others are kept, with the same pictures, boxes and focus.
func TestCellEditRetainedRowsEqualBuiltRows(t *testing.T) {
	kept, built := newEditScene(true), newEditScene(false)
	both := func(what string, f func(e *editScene)) {
		t.Helper()
		f(kept)
		f(built)
		kept.tt.Frame()
		built.tt.Frame()
		if d := sameFrames(kept.tt, built.tt, true); d != "" {
			t.Fatalf("%s: %s", what, d)
		}
	}
	both("first frame", func(e *editScene) {})
	both("click a cell", func(e *editScene) { e.tt.Click("rc1") })
	both("F2", func(e *editScene) { e.tt.Key(0, KeyF2) })
	if kept.editRow != 2 || built.editRow != 2 {
		t.Fatalf("the edit did not start in both: %d and %d", kept.editRow, built.editRow)
	}
	both("type", func(e *editScene) { e.tt.Type("xyz") })
	both("idle", func(e *editScene) {})
	both("Enter", func(e *editScene) { e.tt.Key(0, KeyEnter) })
	both("idle after", func(e *editScene) {})
}
