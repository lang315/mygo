package widget_test

import (
	"iter"
	"strconv"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of the totals for each group in docs/ui/widget.md, which must
// compile.
func ExampleColumn_groupSummary() {
	type Profile struct {
		ID           int
		Name, Status string
		Runs         int
	}
	key := func(p *Profile) int { return p.ID }
	name := widget.Column[Profile]{Title: "Name", Text: func(p *Profile) string { return p.Name }}
	status := widget.Column[Profile]{Title: "Status", Text: func(p *Profile) string { return p.Status }}

	runs := widget.IntColumn("Runs", 80, func(p *Profile) int { return p.Runs })
	runs.GroupSummary = func(rows iter.Seq[*Profile]) string { // one group's rows shown
		total := 0
		for p := range rows {
			total += p.Runs
		}
		return strconv.Itoa(total)
	}
	t := widget.DataGridView(key, name, status, runs).
		SetGroupBy(func(p *Profile) string { return p.Status }) // "running (312)" … "4210" under Runs
	_ = t
}
