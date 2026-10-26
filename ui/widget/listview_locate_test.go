package widget

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// locate, which tries the group of the line asked last, answers as PosOf does
// whatever order the lines are asked in, and after the groups changed under
// the hint.
func TestListViewLocateIsPosOf(t *testing.T) {
	rows := make([]vitem, 300)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Row ", i), fmt.Sprint("g", i%7)}
	}
	lv := groupedView(ViewDetails).SetRows(rows)
	tt := tester(t, lv)
	d := lv.Details()
	check := func(when string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		gs := d.groupsOf()
		r := rand.New(rand.NewPCG(1, 2))
		lines := d.view.Places()
		for range 3000 {
			row := r.IntN(lines + 4) // a few past the end
			if r.IntN(3) == 0 && row > 0 {
				row-- // near the last one asked, as a frame does
			}
			gotPos, gotGroup := d.locate(row)
			wantPos, wantGroup := gs.PosOf(row)
			if gotPos != wantPos || (wantPos < 0 && gotGroup != wantGroup) {
				t.Fatalf("%s: locate(%d) = %d, %d; PosOf = %d, %d", when, row, gotPos, gotGroup, wantPos, wantGroup)
			}
		}
	}
	check("first")
	// Fewer groups than the hint: every row goes to one group.
	lv.SetGroupBy(func(v *vitem) string { return "all" })
	tt.Frame()
	tt.Frame()
	check("one group")
	lv.SetGroupBy(kindOf)
	tt.Frame()
	tt.Frame()
	check("regrouped")
}

// Unchecking all with groups is one Clear, as without them: it does not go
// through the lines asking for each row's key.
func TestListViewDetailsUncheckAllClearsWithGroups(t *testing.T) {
	keys := 0
	rows := make([]vitem, 100)
	for i := range rows {
		rows[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Row ", i), fmt.Sprint("g", i%5)}
	}
	lv := ListView(func(v *vitem) string { keys++; return v.id },
		Column[vitem]{Title: "Name", Text: func(v *vitem) string { return v.name }, Width: 200}).
		SetRows(rows).SetView(ViewDetails).SetGroupBy(kindOf).CheckBoxes()
	tt := tester(t, lv)
	click(t, tt, "Check all")
	if n := len(lv.Checked()); n != len(rows) {
		t.Fatalf("Check all checked %d rows", n)
	}
	d := lv.Details()
	mu.Lock()
	lines := d.view.Places()
	mu.Unlock()
	keys = 0
	d.checkRows(0, lines-1, false) // takes the lock itself
	got := keys
	if got != 0 {
		t.Fatalf("unchecking all asked for %d keys, want a Clear", got)
	}
	if n := len(lv.Checked()); n != 0 {
		t.Fatalf("%d rows still checked", n)
	}
}
