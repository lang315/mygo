package widget

import (
	"slices"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// ColumnFilter keeps the rows of a DataGridView whose cell in one column
// passes it (SetColumnFilter). The zero value filters nothing.
type ColumnFilter struct {
	// Contains keeps the rows whose cell text holds it, whatever its case,
	// or, for a column with its own Filter, the rows Filter keeps for it, as
	// the bottom panel's search does. "" keeps every row.
	Contains string
	// Values, when not nil, keeps only the rows whose cell Text is one of
	// them; an empty, non-nil Values keeps none.
	Values []string
}

// isZero reports whether f filters nothing.
func (f ColumnFilter) isZero() bool { return f.Contains == "" && f.Values == nil }

// clone returns a copy of f whose Values the caller's slice does not share,
// nil staying nil and empty staying empty.
func (f ColumnFilter) clone() ColumnFilter {
	if f.Values != nil {
		f.Values = append(make([]string, 0, len(f.Values)), f.Values...)
	}
	return f
}

// sameFilter reports whether a and b filter the same rows.
func sameFilter(a, b ColumnFilter) bool {
	return a.Contains == b.Contains && (a.Values == nil) == (b.Values == nil) && slices.Equal(a.Values, b.Values)
}

// filterListCap is how many distinct values a filter drop-down lists.
const filterListCap = 1000

// gridFilters is a DataGridView's column filters and the state of the
// controls that edit them. The grid makes it at the first filter call, so
// that a grid without filters has none. Not generic, so that it links once.
type gridFilters struct {
	// cols are the filters by index in the grid's columns, zero for none;
	// off is set while the filter panel's check box turns them off; gen
	// counts their changes, for the panel's text. Under mu.
	cols     []ColumnFilter
	off      bool
	gen      uint64
	onChange func() // OnFilterChange's

	dropDowns, autoRow, panel bool

	// labels name each column's funnel and field ("Filter Name"), made once.
	labels []string
	// row are the auto filter row's texts by column; on is the panel's
	// check box: as ui shows them and as setters set them.
	row []mirror[string]
	on  mirror[bool]

	// The drop-down, ui's alone, on the drawing thread: whether it is open
	// and on which column (an index in the grid's columns), its values, which
	// are checked, its Contains, and whether the values were cut at
	// filterListCap.
	open     bool
	at       int
	vals     []string
	checks   []bool
	contains string
	capped   bool

	// The panel's text, as of textGen.
	text    string
	textGen uint64
	textOK  bool

	// The auto filter row's theme: base's, with a header tall enough for the
	// row of fields; extra is how much taller.
	base, theme *ui.Theme
	extra       float32
}

// filters returns the grid's filters, made at the first call. Under mu.
func (t *DataGridViewWidget[T, K]) filters() *gridFilters {
	if t.fl == nil {
		fl := &gridFilters{cols: make([]ColumnFilter, len(t.cols)), row: make([]mirror[string], len(t.cols)), at: -1}
		for i := range t.cols {
			fl.labels = append(fl.labels, "Filter "+t.cols[i].Title)
		}
		fl.on.set(true)
		t.fl = fl
	}
	return t.fl
}

// SetColumnFilter filters the rows by the column titled title, which needs
// not be Filterable: a row shows when it passes every column's filter and
// the bottom panel's search. ColumnFilter{} takes the column's filter away;
// a title no column has is ignored. It turns the filters on again if the
// filter panel's check box turned them off. Setting a filter does not run
// OnFilterChange.
func (t *DataGridViewWidget[T, K]) SetColumnFilter(title string, f ColumnFilter) *DataGridViewWidget[T, K] {
	t.change(func() {
		if i := slices.IndexFunc(t.cols, func(cl Column[T]) bool { return cl.Title == title }); i >= 0 {
			t.setFilter(i, f.clone(), true)
		}
	})
	return t
}

// ColumnFilter returns a copy of the filter of the column titled title; the
// zero value for a column without one, or a title no column has.
func (t *DataGridViewWidget[T, K]) ColumnFilter(title string) ColumnFilter {
	mu.Lock()
	defer mu.Unlock()
	i := slices.IndexFunc(t.cols, func(cl Column[T]) bool { return cl.Title == title })
	if i < 0 || t.fl == nil {
		return ColumnFilter{}
	}
	return t.fl.cols[i].clone()
}

// ClearFilters takes away every column's filter. The bottom panel's search
// stays.
func (t *DataGridViewWidget[T, K]) ClearFilters() *DataGridViewWidget[T, K] {
	t.change(func() { t.clearFilters() })
	return t
}

// OnFilterChange runs fn on the event goroutine after the user changed a
// filter: in a drop-down, the auto filter row or the filter panel. A
// program's SetColumnFilter and ClearFilters do not run it.
func (t *DataGridViewWidget[T, K]) OnFilterChange(fn func()) *DataGridViewWidget[T, K] {
	t.change(func() { t.filters().onChange = fn })
	return t
}

// SetFilterDropDowns gives the header of each Filterable column a funnel,
// filled while the column has a filter. A click on it opens a drop-down: a
// Contains field, the column's distinct Texts as a check list, sorted, and
// "Select all", which checks them all, and "Clear", which takes the
// column's filter away. Each change applies at once. The list holds the
// values of the rows the grid has (those loaded so far, from a RowSource
// still loading), at most 1000 of them, and says so when there are more.
func (t *DataGridViewWidget[T, K]) SetFilterDropDowns(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		fl := t.filters()
		fl.dropDowns = on
		if !on {
			fl.open = false
		}
		t.hookFilterHeader()
	})
	return t
}

// SetAutoFilterRow puts a row of text fields under the header's titles, one
// for each Filterable column, each bound to its column's Contains: typing
// filters at once. The fields scroll sideways with their columns, and a
// Frozen column's stays pinned with it.
func (t *DataGridViewWidget[T, K]) SetAutoFilterRow(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.filters().autoRow = on
		t.hookFilterHeader()
		t.filterWrap = nil
		if on {
			t.filterWrap = t.wrapAutoRow
		}
	})
	return t
}

// SetFilterPanel shows a strip under the table while a column has a filter:
// the filters as text (Name contains "an" and Kind is Doc, Sheet), a check
// box that turns them all off and on again, keeping them, and "Clear", as
// ClearFilters.
func (t *DataGridViewWidget[T, K]) SetFilterPanel(on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		t.filters().panel = on
		t.filterStrip = nil
		if on {
			t.filterStrip = t.buildFilterPanel
		}
	})
	return t
}

// hookFilterHeader gives headerCell the filters' part of the header while a
// drop-down or the auto filter row is on. Under mu.
func (t *DataGridViewWidget[T, K]) hookFilterHeader() {
	t.filterHead = nil
	if fl := t.fl; fl != nil && (fl.dropDowns || fl.autoRow) {
		t.filterHead = t.filterHeader
	}
}

// setFilter sets the filter of column i, turns the filters on and filters
// the rows again; sync gives the auto filter row's field f's Contains. Under
// mu.
func (t *DataGridViewWidget[T, K]) setFilter(i int, f ColumnFilter, sync bool) {
	fl := t.filters()
	fl.cols[i] = f
	if sync {
		fl.row[i].set(f.Contains)
	}
	if fl.off {
		fl.off = false
		fl.on.set(true)
	}
	fl.gen++
	t.refilter()
}

// clearFilters takes every column's filter away, and reports whether one
// was set. Under mu.
func (t *DataGridViewWidget[T, K]) clearFilters() bool {
	fl := t.fl
	if fl == nil {
		return false
	}
	had := false
	for i := range fl.cols {
		had = had || !fl.cols[i].isZero()
		fl.cols[i] = ColumnFilter{}
		fl.row[i].set("")
	}
	fl.gen++
	t.refilter()
	return had
}

// refilter gives the view the rows' filter: the search's and the columns'
// together, nil when none filters. OnSelectionChanged runs when that hides
// or shows selected rows. Under mu.
func (t *DataGridViewWidget[T, K]) refilter() {
	var before []K
	t.selSeen = nil // the shown selection is taken anew under the new filter
	watch := t.onSelChanged != nil && t.chosen.value.Len() > 0
	if watch {
		before = t.shown(&t.chosen.value)
	}
	t.filter = t.search
	if fl := t.fl; fl != nil && !fl.off {
		t.filter = t.withColumns(t.search)
	}
	t.view.SetFilter(t.filter)
	if watch && !sameKeys(before, t.shown(&t.chosen.value)) {
		t.selectionChanged()
	}
}

// withColumns returns search, nil for none, with the columns' filters:
// search alone when no column has one. Under mu.
func (t *DataGridViewWidget[T, K]) withColumns(search func(*T) bool) func(*T) bool {
	var keeps []func(*T) bool
	for i, f := range t.fl.cols {
		if k := t.keepFilter(i, f); k != nil {
			keeps = append(keeps, k)
		}
	}
	if len(keeps) == 0 {
		return search
	}
	if search != nil {
		keeps = append(keeps, search)
	}
	return func(r *T) bool {
		for _, k := range keeps {
			if !k(r) {
				return false
			}
		}
		return true
	}
}

// keepFilter returns the filter of f on column i, nil when it keeps every
// row. Under mu.
func (t *DataGridViewWidget[T, K]) keepFilter(i int, f ColumnFilter) func(*T) bool {
	contains := t.keepColumn(i, f.Contains)
	if f.Values == nil {
		return contains
	}
	in := make(map[string]struct{}, len(f.Values))
	for _, v := range f.Values {
		in[v] = struct{}{}
	}
	text := t.cols[i].Text
	return func(r *T) bool {
		if _, ok := in[text(r)]; !ok {
			return false
		}
		return contains == nil || contains(r)
	}
}

// filterHeader is the filters' part of the header cell of ui's column col:
// a Filterable column's funnel and its drop-down, and the auto filter row's
// field under the title. Under mu, while the table builds.
func (t *DataGridViewWidget[T, K]) filterHeader(cell ui.Element, col int) {
	fl, c := t.fl, t.headCtx
	th := c.Theme()
	if fl.autoRow {
		// The title keeps the top of the cell, the field the room below.
		sp := th.Space(2.5)
		cell.Padding(0, sp, fl.extra, sp)
	}
	k := col
	if t.numbered {
		k--
	}
	if k < 0 || k >= len(t.visCols) || !t.cols[t.visCols[k]].Filterable {
		return
	}
	ci := t.visCols[k]
	cell.Label(t.cols[ci].Title) // the title alone, not its funnel's and field's names
	if fl.dropDowns {
		t.funnel(c, ci)
	}
	if fl.autoRow {
		m := &fl.row[ci]
		p, _ := m.bind()
		field := ui.TextInput(c, p).Label(fl.labels[ci]).Absolute().Bottom(2).Start(2).End(2).Height(fl.extra - 4)
		if field.Changed() {
			f := fl.cols[ci].clone()
			f.Contains = *p
			Post(func() { t.userFilter(ci, f, false) })
		}
	}
}

// funnelIcons are the funnel, empty and filled, parsed at their first use.
var funnelIcons = sync.OnceValue(func() [2]*ui.SVG {
	const d = `d="M10 20a1 1 0 0 0 .553.895l2 1A1 1 0 0 0 14 21v-7a2 2 0 0 1 .517-1.341L21.74 4.67A1 1 0 0 0 21 3H3a1 1 0 0 0-.742 1.67l7.225 7.989A2 2 0 0 1 10 14z"`
	return [2]*ui.SVG{lucide(`<path ` + d + `/>`), lucide(`<path fill="currentColor" ` + d + `/>`)}
})

// funnel builds the funnel of column ci (an index in t.cols), and its
// drop-down while open. Under mu, while the table builds.
func (t *DataGridViewWidget[T, K]) funnel(c *ui.Context, ci int) {
	fl := t.fl
	th := c.Theme()
	filtered := !fl.cols[ci].isZero()
	icon := funnelIcons()[0]
	if filtered {
		icon = funnelIcons()[1]
	}
	b := ui.ButtonBase(c).Label(fl.labels[ci]).Checked(filtered).Padding(th.Space(0.5)).Radius(th.Radius)
	if b.Hovered() {
		b.Background(th.Text.Alpha(0.12))
	}
	side := 12 * c.Preferences().TextScale
	b.Children(func() { ui.Icon(c, icon).FontSize(side) })
	if b.Clicked() {
		t.openDropDown(ci)
	}
	if fl.open && fl.at == ci {
		ui.Popover(c, b, &fl.open, func() { t.dropDown(c, ci) })
	}
}

// openDropDown opens the drop-down of column ci with the distinct Texts of
// the rows the grid has, sorted, at most filterListCap. Under mu.
func (t *DataGridViewWidget[T, K]) openDropDown(ci int) {
	fl := t.fl
	text := t.cols[ci].Text
	seen := make(map[string]struct{})
	var vals []string
	for i := range t.src.rows {
		s := text(&t.src.rows[i])
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			vals = append(vals, s)
		}
	}
	slices.Sort(vals)
	fl.capped = len(vals) > filterListCap
	if fl.capped {
		vals = vals[:filterListCap]
	}
	f := fl.cols[ci]
	fl.checks = make([]bool, len(vals))
	for i, v := range vals {
		fl.checks[i] = f.Values == nil || slices.Contains(f.Values, v)
	}
	fl.vals, fl.contains, fl.at, fl.open = vals, f.Contains, ci, true
}

// dropDown builds the drop-down of column ci. A change applies on the event
// goroutine, as the user's. Under mu, while the table builds.
func (t *DataGridViewWidget[T, K]) dropDown(c *ui.Context, ci int) {
	fl := t.fl
	th := c.Theme()
	ui.Column(c).Gap(th.Space(1.5)).Padding(th.Space(2)).Children(func() {
		if ui.TextInput(c, &fl.contains).Label("Contains").Placeholder("Contains").Width(220).Changed() {
			f := fl.cols[ci].clone()
			f.Contains = fl.contains
			Post(func() { t.userFilter(ci, f, true) })
		}
		ui.Row(c).Gap(th.Space(1.5)).Children(func() {
			if ui.Button(c, "Select all").Clicked() {
				for i := range fl.checks {
					fl.checks[i] = true
				}
				f := fl.cols[ci].clone()
				f.Values = nil
				Post(func() { t.userFilter(ci, f, true) })
			}
			if ui.Button(c, "Clear").Clicked() {
				for i := range fl.checks {
					fl.checks[i] = true
				}
				fl.contains = ""
				Post(func() { t.userFilter(ci, ColumnFilter{}, true) })
			}
		})
		ui.Scroll(c).MaxHeight(240).Children(func() {
			ui.Column(c).Gap(th.Space(1)).Children(func() {
				changed := false
				for i, v := range fl.vals {
					changed = ui.Checkbox(c, &fl.checks[i], v).Changed() || changed
				}
				if changed {
					f := fl.cols[ci].clone()
					f.Values = fl.checkedValues()
					Post(func() { t.userFilter(ci, f, true) })
				}
			})
		})
		if fl.capped {
			ui.Text(c, "Only the first 1000 values are listed").TextColor(th.TextMuted)
		}
	})
}

// checkedValues returns the values checked in the drop-down: nil when all
// are, so that rows loaded later show.
func (fl *gridFilters) checkedValues() []string {
	if !slices.Contains(fl.checks, false) {
		return nil
	}
	vals := []string{}
	for i, on := range fl.checks {
		if on {
			vals = append(vals, fl.vals[i])
		}
	}
	return vals
}

// userFilter sets the filter of column ci as the user changed it, and runs
// OnFilterChange if that changed it; sync gives the auto filter row's field
// its Contains (the field's own edits have it). On the event goroutine.
func (t *DataGridViewWidget[T, K]) userFilter(ci int, f ColumnFilter, sync bool) {
	var fire func()
	t.change(func() {
		fl := t.filters()
		if sameFilter(fl.cols[ci], f) && !fl.off {
			return
		}
		t.setFilter(ci, f, sync)
		fire = fl.onChange
	})
	if fire != nil {
		fire()
	}
}

// userFilters turns the filters on or off, or clears them (clear), as the
// filter panel's user did, and runs OnFilterChange if that changed what is
// filtered. On the event goroutine.
func (t *DataGridViewWidget[T, K]) userFilters(on, clear bool) {
	var fire func()
	t.change(func() {
		fl := t.filters()
		switch {
		case clear:
			if !t.clearFilters() {
				return
			}
		case fl.off != on:
			return
		default:
			fl.off = !on
			fl.gen++
			t.refilter()
		}
		fire = fl.onChange
	})
	if fire != nil {
		fire()
	}
}

// wrapAutoRow builds the table in a theme whose header is tall enough for
// the auto filter row, made again only when the window's theme changes.
// Under mu, while building.
func (t *DataGridViewWidget[T, K]) wrapAutoRow(c *ui.Context) ui.Element {
	fl := t.fl
	if base := c.Theme(); fl.base != base {
		th := *base
		row := th.Metrics.RowHeight
		if row <= 0 {
			row = th.Space(8)
		}
		head := th.Metrics.HeaderHeight
		if head <= 0 {
			head = row
		}
		fl.extra = row
		th.Metrics.HeaderHeight = head + row
		fl.base, fl.theme = base, &th
	}
	var e ui.Element
	ui.Themed(c, fl.theme).Background(ui.Transparent).Grow(1).Children(func() { e = t.uiTable(c) })
	return e
}

// headExtra is how much taller than the theme's the header is: the auto
// filter row's height, 0 without it. Under mu, while building.
func (t *DataGridViewWidget[T, K]) headExtra() float32 {
	if t.filterWrap == nil {
		return 0
	}
	return t.fl.extra
}

// buildFilterPanel builds the strip of the filters under the table while a
// column has one. Under mu, while building.
func (t *DataGridViewWidget[T, K]) buildFilterPanel(c *ui.Context) {
	fl := t.fl
	if !slices.ContainsFunc(fl.cols, func(f ColumnFilter) bool { return !f.isZero() }) {
		return
	}
	if !fl.textOK || fl.textGen != fl.gen {
		fl.text, fl.textGen, fl.textOK = t.filterText(), fl.gen, true
	}
	th := c.Theme()
	ui.Row(c).Gap(th.Space(3)).AlignItems(ui.Center).Shrink(0).Children(func() {
		p, _ := fl.on.bind()
		if ui.Checkbox(c, p, fl.text).Changed() {
			on := *p
			Post(func() { t.userFilters(on, false) })
		}
		ui.Row(c).Grow(1)
		if ui.Button(c, "Clear").Label("Clear filters").Clicked() {
			Post(func() { t.userFilters(true, true) })
		}
	})
}

// filterText is the filters as text, in the columns' order: Name contains
// "an" and Kind is Doc, Sheet. Under mu.
func (t *DataGridViewWidget[T, K]) filterText() string {
	var parts []string
	for i, f := range t.fl.cols {
		title := t.cols[i].Title
		if f.Contains != "" {
			parts = append(parts, title+` contains "`+f.Contains+`"`)
		}
		if f.Values != nil {
			vals := "(none)"
			if len(f.Values) > 0 {
				vals = strings.Join(f.Values, ", ")
			}
			parts = append(parts, title+" is "+vals)
		}
	}
	return strings.Join(parts, " and ")
}
