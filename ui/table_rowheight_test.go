package ui

import "testing"

// A table whose rows are kept (RowVersion) takes a new RowHeight at once:
// the rows' height and their cells' padding come from it, so it is part of
// what the rows were built with (rowSignature).
func TestTableRowHeightRebuildsKeptRows(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80, Align: End}}
	s := ListState{RowVersion: func(int) uint64 { return 1 }}
	tt := coreNewTester(tableView(&s, cols, names), 500, 300)
	tt.Frame()
	r0, _ := rowBox(tt, &s, 1)
	if want := LightTheme().Space(8); r0.H != want {
		t.Fatalf("a row by default is %v high, want %v", r0.H, want)
	}
	s.RowHeight = 22
	tt.Frame()
	for i := range names {
		if r, _ := rowBox(tt, &s, i); r.H != 22 {
			t.Errorf("row %d is %v high, want 22", i, r.H)
		}
	}
	r, _ := rowBox(tt, &s, 1)
	b, ok := tt.Find("b.txt")
	if !ok || b.Y+b.H/2 < r.Y+r.H/2-1 || b.Y+b.H/2 > r.Y+r.H/2+1 {
		t.Errorf("b.txt at %v in its row at %v: want it in the middle", b, r)
	}
}
