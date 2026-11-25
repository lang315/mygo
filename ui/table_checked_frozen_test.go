package ui

import (
	"bytes"
	"strconv"
	"testing"
)

// A Table with checkboxes and pinned columns pins the checkboxes' cells
// too, at its start, before the pinned columns: a sideways scroll leaves
// that strip as it was, and a column the cursor reveals lands after the
// pinned ones, the checkboxes' cells counted.
func TestTableCheckedFrozenPinsTheCheckCell(t *testing.T) {
	var checked Selection[int]
	s := &ListState{Checked: &checked, CellSelect: true}
	s.Selection = &Selection[int]{}
	cols := []TableColumn{{Title: "A", Width: 100, Frozen: true}}
	for k := range 30 {
		cols = append(cols, TableColumn{Title: "C" + strconv.Itoa(k), Width: 100})
	}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 5, func(row, col int) {
			Textf(c, "r%dc%d", row, col)
		}).Grow(1)
	}, 500, 300)
	tt.Frame()
	checkW := int(LightTheme().Space(8))
	// strip is the pixels of the checkboxes' column, from the header down to
	// above where a sideways scroll bar shows as the table scrolls.
	strip := func() []byte {
		img := tt.Image()
		var out []byte
		for y := img.Rect.Min.Y; y < img.Rect.Min.Y+img.Rect.Dy()*2/3; y++ {
			i := img.PixOffset(img.Rect.Min.X, y)
			out = append(out, img.Pix[i:i+4*checkW]...)
		}
		return out
	}
	// The pointer where the scroll comes, so that the row it hovers is the
	// same before and after.
	tt.Move(250, 150)
	tt.Frame()
	before := strip()
	pinned0, _ := tt.Find("r2c0")
	tt.Scroll(250, 150, 700, 0)
	tt.Frame()
	if !bytes.Equal(strip(), before) {
		t.Error("the checkboxes' column changed as the table scrolled sideways: their cells scrolled away, or cells showed there")
	}
	if pinned, _ := tt.Find("r2c0"); pinned.X != pinned0.X {
		t.Errorf("the pinned column moved from %v to %v", pinned0.X, pinned.X)
	}
	// The cursor goes right, then back left: the column it comes back to
	// shows after the pinned one, not under it by the checkboxes' width.
	tt.Click("r0c0")
	for range 20 {
		tt.Key(0, KeyRight)
		tt.Frame()
	}
	for range 19 {
		tt.Key(0, KeyLeft)
		tt.Frame()
	}
	if s.Cells.Cursor != "C0" {
		t.Fatalf("the cursor is in %q", s.Cells.Cursor)
	}
	cell, ok := tt.Find("r0c1")
	pinned, _ := tt.Find("r0c0")
	if !ok || cell.X < pinned.X+100 {
		t.Errorf("the revealed cell's text at %v (%v), under the pinned one at %v", cell.X, ok, pinned.X)
	}
}
