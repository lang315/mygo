package ui

import (
	"bytes"
	"fmt"
	"testing"
)

// looksTable is a table of n rows "r0", "r1"… over s.
func looksTable(s *ListState, n int) func(c *Context) {
	cols := []TableColumn{{Title: "Name"}, {Title: "Status", Width: 80}}
	return func(c *Context) {
		Table(c, s, cols, n, func(r, col int) {
			if col == 0 {
				Textf(c, "r%d", r)
			} else {
				Textf(c, "s%d", r)
			}
		}).Grow(1)
	}
}

func TestTableRowHeightIsUniform(t *testing.T) {
	s := &ListState{RowHeight: 44}
	tt := NewTester(looksTable(s, 5), 400, 300)
	tt.Frame()
	for i := 1; i < 5; i++ {
		a, _ := tt.Find(fmt.Sprintf("r%d", i-1))
		b, _ := tt.Find(fmt.Sprintf("r%d", i))
		if d := b.Y - a.Y; d != 44 {
			t.Fatalf("rows %d and %d %v apart, want 44", i-1, i, d)
		}
	}
}

func TestTableGridLinesDraw(t *testing.T) {
	off, on := &ListState{}, &ListState{GridLines: true}
	a, b := NewTester(looksTable(off, 5), 400, 300), NewTester(looksTable(on, 5), 400, 300)
	a.Frame()
	b.Frame()
	if bytes.Equal(a.Image().Pix, b.Image().Pix) {
		t.Fatal("grid lines draw nothing")
	}
	pitch := func(tt *Tester) float32 {
		r0, _ := tt.Find("r0")
		r3, _ := tt.Find("r3")
		return r3.Y - r0.Y
	}
	if pa, pb := pitch(a), pitch(b); pa != pb {
		t.Fatalf("grid lines changed the rows' pitch: 3 rows are %v tall, were %v", pb, pa)
	}
}

// Row height and grid lines are what a table builds its rows with: turned on
// once rows are held, the kept rows draw as rows built anew.
func TestRetainLayoutTableLooksTurnedOnAfterRowsAreHeld(t *testing.T) {
	var states []*ListState
	p := newPair(t, 400, 300, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		states = append(states, s)
		return looksTable(s, 50)
	})
	p.idle(4)
	if p.kept.rt.rowsLaidKept == 0 {
		t.Fatal("no row held")
	}
	for _, s := range states {
		s.GridLines = true
	}
	p.frame("grid lines on")
	p.idle(2)
	for _, s := range states {
		s.RowHeight = 40
	}
	p.frame("row height 40")
	p.idle(2)
	for _, s := range states {
		s.GridLines, s.RowHeight = false, 0
	}
	p.frame("both off")
	p.idle(2)
}
