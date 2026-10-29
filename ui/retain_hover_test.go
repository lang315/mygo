package ui

import (
	"fmt"
	"testing"
)

// A hovered row whose callback never asks where the pointer is (its hover look
// is a style function, read at paint) is kept while it is hovered: it is not
// built, its picture is the one of a list that builds every row, at every step
// of a pointer that enters, moves, leaves and goes to another row.
func TestRetainAHoveredRowIsKeptIfItNeverAsksForTheHover(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	a.frame()
	same(t, "idle", a, b)
	var rows [3][2]float32
	for i, r := range []int{5, 6, 9} {
		rows[i][0], rows[i][1] = a.rowAt(r)
	}
	step := func(what string, x, y float32, row int) {
		t.Helper()
		before := slicesClone(a.builds)
		a.tt.Move(x, y)
		b.tt.Move(x, y)
		for range 2 {
			a.tt.Frame()
			b.tt.Frame()
		}
		if got := a.diff(before); len(got) != 0 {
			t.Errorf("%s: rows built %v, want none (row %d is hovered)", what, got, row)
		}
		same(t, what, a, b)
		if row >= 0 && a.list.frame.kept[row] == nil {
			t.Errorf("%s: row %d has no record while it is hovered", what, row)
		}
	}
	step("enter row 5", rows[0][0], rows[0][1], 5)
	step("move within row 5", rows[0][0]+3, rows[0][1], 5)
	step("on to row 6", rows[1][0], rows[1][1], 6)
	step("on to row 9", rows[2][0], rows[2][1], 9)
	step("leave to the header", rows[2][0], 5, -1)
	step("back on row 5", rows[0][0], rows[0][1], 5)
	// And a row hovered when something else changes (its version): built, as a
	// row whose version changed is.
	a.vers[5]++
	b.vers[5]++
	before := slicesClone(a.builds)
	a.tt.Frame()
	b.tt.Frame()
	if got := a.diff(before); got[5] == 0 {
		t.Errorf("the version of the hovered row changed: rows built %v, want row 5", got)
	}
	same(t, "version of the hovered row", a, b)
}

// The same for rows that ask Hovered in their callback (a row that shows
// something only while hovered): they are built while hovered, as before, and
// the record made when the row was not hovered is not linked. The call counts,
// whatever it returns.
func TestRetainARowThatAsksForTheHoverIsBuiltWhileHovered(t *testing.T) {
	a, b := newHoverScene(t, 200, true), newHoverScene(t, 200, false)
	a.frame()
	x, y := a.rowAt(5)
	if rb := a.list.frame.kept[5]; rb == nil || !rb.readsHover {
		t.Fatalf("the record of row 5 does not say that its callback asked for the hover (%v)", rb)
	}
	for step, move := range []func(){
		func() { a.tt.Move(x, y); b.tt.Move(x, y) },
		func() { a.tt.Move(x+2, y); b.tt.Move(x+2, y) },
	} {
		before := slicesClone(a.builds)
		move()
		a.tt.Frame()
		b.tt.Frame()
		got := a.diff(before)
		if got[5] == 0 {
			t.Errorf("step %d: rows built %v, want row 5 built while it is hovered", step, got)
		}
		same(t, fmt.Sprint("step ", step), a, b)
	}
}

// The rows that ui builds itself do not ask for the hover in their callback, so
// a hovered one is kept: a Table, a DataTable and an Outline. A row of them that
// began to ask would be built again while hovered (the cost this change took
// away), and this test says so. (The rows of ui/widget's ListBox, DataGridView
// and ListView were counted the same way: none asked.)
func TestRowsOfTheLibraryDoNotAskForTheHover(t *testing.T) {
	asked := func(f *listFrame) (n, rows int) {
		for _, r := range f.rows {
			rows++
			if r.readsHover {
				n++
			}
		}
		return
	}
	hover := func(tt *Tester) {
		for y := float32(40); y < 380; y += 40 {
			tt.Move(100, y)
			tt.Frame()
		}
	}
	t.Run("Table", func(t *testing.T) {
		s := newKeepScene(t, 100, true)
		hover(s.tt)
		if n, rows := asked(&s.list.frame); rows == 0 || n != 0 {
			t.Errorf("%d of %d rows of the Table asked for the hover", n, rows)
		}
	})
	t.Run("DataTable", func(t *testing.T) {
		rows := make([]dtRow, 60)
		for i := range rows {
			rows[i] = dtRow{ID: i, Name: fmt.Sprint("n", i), Mail: "x"}
		}
		s := dtState()
		tt := dtTester(s, &rows)
		tt.Frame()
		hover(tt)
		if n, got := asked(&s.list.frame); got == 0 || n != 0 {
			t.Errorf("%d of %d rows of the DataTable asked for the hover", n, got)
		}
	})
	t.Run("Outline", func(t *testing.T) {
		var s OutlineState[string]
		roots := []string{"src", "docs", "README.md"}
		tt := coreNewTester(func(c *context) {
			coreOutline(c, &s, roots, children, func(path string) { coreText(c, baseName(path)) }).Grow(1)
		}, 400, 400)
		tt.Frame()
		hover(tt)
		if n, got := asked(&s.List.frame); got == 0 || n != 0 {
			t.Errorf("%d of %d rows of the Outline asked for the hover", n, got)
		}
	})
}

// An element that does not exist (a nil one, which the header of a Table asks at
// paint) is not hovered, and asking notes nothing.
func TestHoveredOfAMissingElement(t *testing.T) {
	var e *node
	if e.Hovered() {
		t.Error("a nil element is hovered")
	}
	if _, _, over := e.PointerPosition(); over {
		t.Error("the pointer is over a nil element")
	}
}
