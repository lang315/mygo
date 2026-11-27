package widget

import "slices"

// ShownCells returns what the table shows, for an export (package
// ui/widget/export writes it as CSV or XLSX): the titles of the columns
// shown, in the order the header shows them, and the rows as shown,
// filtered, sorted and grouped, without the groups' headers, each a cell's
// Text by column. The index column and the check box column are left out;
// a RowSource's grid gives the rows the source holds. The column funcs
// run under the package's lock.
func (t *DataGridViewWidget[T, K]) ShownCells() (header []string, rows [][]string) {
	mu.Lock()
	defer mu.Unlock()
	t.sync()
	cols := t.shownOrder()
	header = make([]string, len(cols))
	for j, ci := range cols {
		header[j] = t.cols[ci].Title
	}
	rows = make([][]string, t.view.Len())
	for i := range rows {
		r := &t.src.rows[t.view.Row(i)]
		cells := make([]string, len(cols))
		for j, ci := range cols {
			cells[j] = t.cols[ci].Text(r)
		}
		rows[i] = cells
	}
	return header, rows
}

// shownOrder returns the indexes in t.cols of the columns shown, in the
// order the header shows them, as ui's TableLayout arranges them: those
// the layout names, in its order, then the others in theirs. The layout is
// SetColumnsLayout's not handed to ui yet, or the last ui had (t.full),
// read under mu rather than ui's own, which the main thread changes.
// Under mu.
func (t *DataGridViewWidget[T, K]) shownOrder() []int {
	order := t.full
	if t.layout != nil {
		order = t.layout.Order
	}
	out := make([]int, 0, len(t.visCols))
	for _, title := range order {
		j := slices.IndexFunc(t.visCols, func(ci int) bool { return t.cols[ci].Title == title })
		if j >= 0 && !slices.Contains(out, t.visCols[j]) {
			out = append(out, t.visCols[j])
		}
	}
	for _, ci := range t.visCols {
		if !slices.Contains(out, ci) {
			out = append(out, ci)
		}
	}
	return out
}
