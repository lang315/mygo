package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// prof is a row of the grid of the groups' and the summaries' cases: a
// profile of a manager of many.
type prof struct {
	ID           int
	Name, Status string
	Runs         int
}

// profRows are the 10,000 rows of the cases: 312 "running" (each 32nd from
// the second), 7 "failed" (rows 7, 1007, ..., 6007) and the others "idle",
// each with one run. The source's order is not that of the groups: row 0 is
// idle, row 1 running, row 7 failed.
func profRows() []prof {
	rows := make([]prof, 10_000)
	for i := range rows {
		status := "idle"
		switch {
		case i%32 == 1 && i < 9984:
			status = "running"
		case i%1000 == 7 && i < 7000:
			status = "failed"
		}
		rows[i] = prof{ID: i, Name: fmt.Sprintf("p%05d", i), Status: status, Runs: 1}
	}
	return rows
}

// profGrid is the grid of the cases, of cols, holding profRows.
func profGrid(cols ...Column[prof]) *DataGridViewWidget[prof, int] {
	if len(cols) == 0 {
		cols = []Column[prof]{
			{Title: "Name", Text: func(p *prof) string { return p.Name }, Width: 200, Sortable: true},
			{Title: "Status", Text: func(p *prof) string { return p.Status }, Width: 200},
		}
	}
	return DataGridView(func(p *prof) int { return p.ID }, cols...).SetRows(profRows()).Expand()
}

// profTester shows g alone in a window of 900 by 600 DIPs.
func profTester(g Control) *ui.Tester {
	tt := ui.NewTester(content(g), 900, 600)
	tt.Frame()
	return tt
}

// groupLines returns the labels of the lines that head groups, in order,
// as the last frame showed them.
func groupLines[T any, K comparable](g *DataGridViewWidget[T, K]) []string {
	mu.Lock()
	gs := slices.Clone(g.groupsOf())
	mu.Unlock()
	var heads []string
	for k, gr := range gs {
		heads = append(heads, g.labelAt(gr.Start+k))
	}
	return heads
}

// 8. SetGroupBy groups the rows under headers "Title (n)", the groups by
// title; SetGroupOrder puts the groups it names first, in its order;
// SetGroupBy(nil) shows no headers.
func TestGridGroupsByStatus(t *testing.T) {
	g := profGrid()
	tt := profTester(g)
	g.SetGroupBy(func(p *prof) string { return p.Status })
	tt.Frame()
	if got, want := groupLines(g), []string{"failed (7)", "idle (9681)", "running (312)"}; !slices.Equal(got, want) {
		t.Fatalf("headers %q, want %q", got, want)
	}
	if !shows(tt, "failed (7)") {
		t.Fatalf("the first header does not show: %q", tt.Texts()[:8])
	}
	// The header of idle follows the seven failed rows: line 8.
	if got := g.labelAt(8); got != "idle (9681)" {
		t.Errorf("line 8 is %q, want the header idle (9681)", got)
	}

	g.SetGroupOrder("failed", "running")
	tt.Frame()
	if got, want := groupLines(g), []string{"failed (7)", "running (312)", "idle (9681)"}; !slices.Equal(got, want) {
		t.Fatalf("after SetGroupOrder(failed, running): headers %q, want %q", got, want)
	}

	g.SetGroupBy(nil)
	tt.Frame()
	if got := groupLines(g); len(got) != 0 {
		t.Fatalf("after SetGroupBy(nil): headers %q", got)
	}
	if shows(tt, "failed (7)") || shows(tt, "idle (9681)") {
		t.Fatalf("after SetGroupBy(nil) a header shows: %q", tt.Texts()[:8])
	}
	if got := g.labelAt(0); got != "p00000" {
		t.Errorf("after SetGroupBy(nil) line 0 is %q, want the row p00000", got)
	}
}

// SetGroupOrder before SetGroupBy keeps its order.
func TestGridGroupOrderBeforeGroupBy(t *testing.T) {
	g := profGrid().SetGroupOrder("running").SetGroupBy(func(p *prof) string { return p.Status })
	profTester(g)
	if got, want := groupLines(g), []string{"running (312)", "failed (7)", "idle (9681)"}; !slices.Equal(got, want) {
		t.Fatalf("headers %q, want %q", got, want)
	}
}

// 9. With groups, Selected returns the keys in the order shown, grouped; a
// click on a group's header selects nothing.
func TestGridGroupsSelectedInOrderShown(t *testing.T) {
	g := profGrid().SetGroupBy(func(p *prof) string { return p.Status })
	tt := profTester(g)
	g.SetSelected(0, 1, 7) // idle, running, failed
	tt.Frame()
	if got, want := g.Selected(), []int{7, 0, 1}; !slices.Equal(got, want) {
		t.Fatalf("Selected() = %v, want %v: failed, idle, running", got, want)
	}
	g.SetSelected()
	tt.Frame()
	click(t, tt, "failed (7)")
	if got := g.Selected(); len(got) != 0 {
		t.Fatalf("a click on the header failed (7) selected %v", got)
	}
}
