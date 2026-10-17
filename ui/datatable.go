package ui

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A DataColumn is a column of a DataTable: a TableColumn, and how to get its
// cells from a row.
type DataColumn[T any] struct {
	TableColumn
	// Text is the cell's text. The search and the default Copy read it too.
	// It is required.
	Text func(x *T) string
	// Compare sorts by the column when set, as a TableView's comparator does:
	// negative, zero or positive as a sorts before, with or after b. Sortable
	// is set for you.
	Compare func(a, b *T) int
	// Cell builds the cell, when it is more than Text(x) in one line.
	Cell func(c *Context, x *T)
	// NoSearch leaves the column out of the search's columns.
	NoSearch bool
	// Match, when set, decides which rows a search of the column keeps,
	// given the keyword trimmed, in place of the cells containing it (and
	// of the | list of values).
	Match func(x *T, keyword string) bool
}

// A DataTableState is what a DataTable keeps for an app's rows: their order
// (a TableView), the rows checked and chosen, the sort and the search. The
// rows stay the app's slice, which it gives DataTable at each frame. Keep it
// as a field, set Key and Columns before the first frame, use it on the main
// thread, and do not copy it after the first frame.
type DataTableState[T any, K comparable] struct {
	// Key is a row's identity: the checked and chosen rows are kept by it
	// across a sort, a search and changes of the rows. It is required.
	Key func(x *T) K
	// Columns are the table's columns, as many as at the first frame.
	Columns []DataColumn[T]
	// Index shows a # column, the position of each row shown.
	Index bool
	// Sort is the order shown. A click on a header of a column with a
	// Compare changes it; set it to start sorted.
	Sort SortOrder
	// MultiSort sorts by several columns, SortBy holding them, the first
	// first: a click on a header sorts by it alone, a Cmd-click (Ctrl-click
	// on Linux and Windows) adds it, as ListState.SortBy does. Sort is left
	// alone while it is set. Set it before the first frame.
	MultiSort bool
	SortBy    []SortOrder
	// Filter keeps the rows it returns true for, apart from the search.
	// FilterKey tells when Filter changed: the rows are filtered again when
	// it differs from the last frame's. It must be comparable.
	Filter    func(x *T) bool
	FilterKey any
	// RowMenu, when set, is the rows' context menu, filled for the rows
	// chosen, in the order shown. The rows point into the app's slice and
	// are valid during the call.
	RowMenu func(m *Menu, rows []*T)
	// Copy returns the text for the rows chosen on Cmd+C, as above. By
	// default it is the Text of each column but #, tab-separated, a line a
	// row.
	Copy func(rows []*T) string
	// NoSearchBar hides the search bar, for an app that puts its own where
	// it wants, with its own texts, and searches with SetSearch. No bar is
	// shown either when no column is searchable.
	NoSearchBar bool

	// Checked are the rows checked, Chosen the rows highlighted.
	Checked, Chosen Selection[K]

	rows   []T
	view   *TableView[T]
	list   ListState
	cursor int
	ncols  int
	multi  bool // MultiSort at the first frame

	// titles are the columns the search offers; column and keyword are what
	// the select and the box show, applied what the rows are filtered by,
	// and match keeps the rows it finds, nil for none.
	titles          []string
	searchable      []int // the index in Columns of each of titles
	column, keyword string
	applied         dataSearch
	match           func(x *T) bool
	keep            func(x *T) bool
	tcols           []TableColumn
	menu            func(m *Menu, rows []int)
}

// start builds what the state keeps, once, from Key and Columns.
func (s *DataTableState[T, K]) start() {
	if s.view != nil {
		return
	}
	if s.Key == nil {
		panic("ui: DataTableState.Key is nil")
	}
	if len(s.Columns) == 0 {
		panic("ui: a DataTable needs a column")
	}
	compare := map[string]func(a, b *T) int{}
	for i := range s.Columns {
		col := &s.Columns[i]
		if col.Text == nil {
			panic(fmt.Sprintf("ui: DataTable column %q has no Text", col.Title))
		}
		if col.Compare != nil {
			compare[col.id()] = col.Compare
		}
		if !col.NoSearch {
			s.titles, s.searchable = append(s.titles, col.Title), append(s.searchable, i)
		}
	}
	if len(s.titles) > 0 {
		s.column = s.titles[0]
	}
	s.ncols, s.cursor = len(s.Columns), -1
	s.view = NewTableView(compare)
	s.list.Selected, s.list.Selection, s.list.Checked = &s.cursor, &s.Chosen, &s.Checked
	s.multi = s.MultiSort
	if s.multi {
		s.list.SortBy = &s.SortBy
	} else {
		s.list.Sort = &s.Sort
	}
	s.list.Key = func(row int) any { return s.Key(s.At(row)) }
	s.list.Label = func(row int) string { return s.Columns[0].Text(s.At(row)) }
	s.list.Copy = func(rows []int) string { return s.copyRows(s.pick(rows)) }
	s.menu = func(m *Menu, rows []int) { s.RowMenu(m, s.pick(rows)) }
	s.keep = func(x *T) bool {
		return (s.Filter == nil || s.Filter(x)) && (s.match == nil || s.match(x))
	}
}

// Len returns how many rows are shown.
func (s *DataTableState[T, K]) Len() int {
	s.start()
	return s.view.Len()
}

// At returns the i-th row shown, a pointer into the app's slice.
func (s *DataTableState[T, K]) At(i int) *T {
	s.start()
	return &s.rows[s.view.Row(i)]
}

// Current returns the row chosen last, or nil when none is shown.
func (s *DataTableState[T, K]) Current() *T {
	s.start()
	if s.cursor < 0 || s.cursor >= s.view.Len() {
		return nil
	}
	return s.At(s.cursor)
}

// Version changes whenever the rows shown, or their order, change: a sort, a
// search, a filter, rows placed again. It is the version of the TableView:
// 0 before the first frame, and it moves at the frame that applies a change,
// not at the call that asks for it, as SetSearch.
func (s *DataTableState[T, K]) Version() uint64 {
	s.start()
	return s.view.Version()
}

// Changed says that the app changed the rows of its slice at these indexes,
// so the next frame places them again, as TableView.Changed does. Rows
// appended need no call.
func (s *DataTableState[T, K]) Changed(rows ...int) {
	s.start()
	s.view.Changed(rows...)
}

// Reset says that the app removed or reordered rows: the next frame places
// them all again, as TableView.Reset does.
func (s *DataTableState[T, K]) Reset() {
	s.start()
	s.view.Reset()
}

// DataTable shows rows, an app's slice, in a table with a checkbox column
// and, with s.Index, a # column. It keeps their order, the rows checked and
// chosen, the sort and the search in s: call it once a frame with the
// current rows, and tell s about rows that changed (Changed, Reset).
//
//	app.table.Key = func(a *account) int { return a.ID }
//	app.table.Columns = columns
//	ui.DataTable(c, &app.table, app.accounts).Grow(1)
func DataTable[T any, K comparable](c *Context, s *DataTableState[T, K], rows []T) Element {
	if c.build() == nil {
		return Element{}
	}
	s.start()
	if len(s.Columns) != s.ncols {
		panic("ui: DataTable columns changed after the first frame")
	}
	if s.MultiSort != s.multi {
		panic("ui: DataTable MultiSort changed after the first frame")
	}
	s.rows = rows
	return Column(c).Gap(8).Children(func() {
		if !s.NoSearchBar && len(s.titles) > 0 {
			s.searchBar(c)
		}
		if k := s.FilterKey; k != nil && !reflect.ValueOf(k).Comparable() {
			panic(fmt.Sprintf("ui: DataTable FilterKey %T is not comparable", k))
		}
		s.view.SetFilterKey(dataFilterKey{s.FilterKey, s.applied}, s.keep)
		if s.multi {
			s.view.Sync(rows, s.SortBy...)
		} else {
			s.view.Sync(rows, s.Sort)
		}
		s.list.Version = s.view.Version()
		if s.RowMenu != nil {
			s.list.RowMenu = s.menu
		} else {
			s.list.RowMenu = nil
		}
		Table(c, &s.list, s.tableColumns(c), s.view.Len(), func(row, col int) {
			s.cell(c, row, col)
		}).Grow(1)
	})
}

// dataIndexID is the ID of the # column, one no column of an app has.
const dataIndexID = "ui.DataTable.index"

// tableColumns are the columns of the Table: a # column when s.Index, wide
// enough for the number of the last row (and for two digits), then those of
// s, sortable when they have a Compare. The slice is reused from frame to
// frame.
func (s *DataTableState[T, K]) tableColumns(c *Context) []TableColumn {
	s.tcols = s.tcols[:0]
	if s.Index {
		digits := max(2, len(strconv.Itoa(s.view.Len())))
		w, _ := c.MeasureText(0, Span{Text: strings.Repeat("9", digits)})
		t := c.Theme()
		s.tcols = append(s.tcols, TableColumn{ID: dataIndexID, Title: "#", Fixed: true, Align: End, Width: w + 2*t.Space(2.5)})
	}
	for i := range s.Columns {
		col := s.Columns[i].TableColumn
		col.Sortable = s.Columns[i].Compare != nil
		s.tcols = append(s.tcols, col)
	}
	return s.tcols
}

// cell builds the cell of a row shown, in column col of the Table.
func (s *DataTableState[T, K]) cell(c *Context, row, col int) {
	if s.Index {
		if col == 0 {
			Text(c, strconv.Itoa(row+1)).SingleLine()
			return
		}
		col--
	}
	x := s.At(row)
	if f := s.Columns[col].Cell; f != nil {
		f(c, x)
		return
	}
	Text(c, s.Columns[col].Text(x)).SingleLine()
}

// dataSearch is a search applied: a column's title, and the trimmed keyword;
// both empty for none.
type dataSearch struct{ column, keyword string }

// dataFilterKey is what decides the rows shown, to tell when they change.
type dataFilterKey struct {
	user   any
	search dataSearch
}

// Search returns the search applied: the title of the column and the
// keyword, both empty when none is.
func (s *DataTableState[T, K]) Search() (column, keyword string) {
	s.start()
	return s.applied.column, s.applied.keyword
}

// SetSearch searches column, by its Title, for keyword, as a search typed in
// the bar does. An empty keyword shows all rows. It panics for a column that
// does not exist or has NoSearch; of columns with the same Title, the first
// is searched.
func (s *DataTableState[T, K]) SetSearch(column, keyword string) {
	s.start()
	if !slices.Contains(s.titles, column) {
		panic(fmt.Sprintf("ui: DataTable has no searchable column %q", column))
	}
	s.column, s.keyword = column, keyword
	s.apply()
}

// apply makes the column and keyword shown the search of the rows. An empty
// keyword leaves the column out, so choosing another column with nothing to
// find filters nothing again.
func (s *DataTableState[T, K]) apply() {
	applied := dataSearch{s.column, strings.TrimSpace(s.keyword)}
	if applied.keyword == "" {
		applied.column = ""
	}
	if applied == s.applied {
		return
	}
	s.applied, s.match = applied, nil
	if applied.column != "" {
		col := &s.Columns[s.searchable[slices.Index(s.titles, applied.column)]]
		if match, kw := col.Match, applied.keyword; match != nil {
			s.match = func(x *T) bool { return match(x, kw) }
		} else {
			s.match = dataMatcher(col.Text, applied.keyword)
		}
	}
}

// searchBar is the row of the search: the column, the keyword, and a button.
// The search runs on Enter, a click on the button, clearing the box or
// choosing another column; at each key it would go through all the rows.
func (s *DataTableState[T, K]) searchBar(c *Context) {
	raw := c.build()
	Row(c).Gap(8).AlignItems(Center).Children(func() {
		column := Select(c, &s.column, s.titles).Label(raw.texts().Column()).Width(170).Changed()
		input := TextInput(c, &s.keyword).Label(raw.texts().Search()).Placeholder(raw.texts().Search()).Grow(1)
		cleared := input.Changed() && s.keyword == ""
		submitted := input.Submitted()
		// Built before the others are tested: a click is read as the button is.
		clicked := Button(c, raw.texts().Search()).Clicked()
		if clicked || submitted || column || cleared {
			s.apply()
			c.Invalidate()
		}
	})
}

// dataMatcher returns what keeps the rows whose text matches keyword: a
// keyword with | lists the values to match exactly, as fhtool's UID1|UID2
// search; another matches the texts containing it, case aside. It returns
// nil for a keyword that keeps all rows, as one listing nothing.
func dataMatcher[T any](text func(x *T) string, keyword string) func(x *T) bool {
	if strings.Contains(keyword, "|") {
		set := map[string]bool{}
		for v := range strings.SplitSeq(keyword, "|") {
			if v = strings.TrimSpace(v); v != "" {
				set[v] = true
			}
		}
		if len(set) == 0 {
			return nil
		}
		return func(x *T) bool { return set[text(x)] }
	}
	kw := []byte(strings.ToLower(keyword))
	// The filter runs on one goroutine, so one buffer holds each text lowered
	// in turn.
	var buf []byte
	return func(x *T) bool {
		buf = appendLower(buf[:0], text(x))
		return bytes.Contains(buf, kw)
	}
}

// appendLower appends s as strings.ToLower lowers it, without allocating for
// a buffer large enough.
func appendLower(buf []byte, s string) []byte {
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			buf = append(buf, c)
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		buf = utf8.AppendRune(buf, unicode.ToLower(r))
		i += n
	}
	return buf
}

// pick returns the rows shown at these indexes, pointing into the app's
// slice.
func (s *DataTableState[T, K]) pick(rows []int) []*T {
	out := make([]*T, len(rows))
	for i, row := range rows {
		out[i] = s.At(row)
	}
	return out
}

// copyRows is the text for the rows chosen: the app's Copy, or the Text of
// each column, tab-separated, a line a row.
func (s *DataTableState[T, K]) copyRows(rows []*T) string {
	if s.Copy != nil {
		return s.Copy(rows)
	}
	var b strings.Builder
	for i, x := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j := range s.Columns {
			if j > 0 {
				b.WriteByte('\t')
			}
			b.WriteString(s.Columns[j].Text(x))
		}
	}
	return b.String()
}
