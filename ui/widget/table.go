package widget

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Column is a column of a table of rows of type T. Its functions get a
// pointer to a row and run under the package's lock, while the table
// builds, sorts or searches: they read the row and call no widget method.
type Column[T any] struct {
	// Title heads the column; it is also its ID in a ui.TableLayout.
	Title string
	// Text is what a cell shows. It is required.
	Text func(*T) string
	// Compare sorts by the column, as cmp.Compare does; nil compares Text.
	Compare func(a, b *T) int
	// Width and MinWidth are in DIPs; a Width of 0 shares the room left.
	Width, MinWidth float32
	Align           Align
	Sortable        bool
	// Filterable lists the column in the bottom panel's search.
	Filterable bool
	// Filter keeps a row for a search's keyword; nil keeps the rows whose
	// Text holds the keyword, whatever its case.
	Filter func(row *T, keyword string) bool
	// Spans, when set, makes the cell draw the spans it returns, as
	// Label.Rich does, in place of Text, which still sorts, searches and
	// copies, and names the row; assistive technology reads the cell as
	// its spans' text. On a selected row the spans' text takes the
	// selection's color, but their backgrounds stay, as badges do; that
	// copies the spans of each cell drawn, so keep them with the rows if
	// you can. A nil or empty result draws Text. It runs under the
	// package's lock, once for each cell drawn.
	Spans func(*T) []ui.Span
	// Color is the color of the text; nil, or a zero Color, keeps the
	// theme's, as does a selected row.
	Color func(*T) ui.Color
	// Image shows before the text, 16 DIPs square; nil for none.
	Image func(*T) *ui.Bitmap
}

// DataGridViewWidget shows rows of type T, each told apart by a key of type K,
// in columns that sort, for as many as a million rows. The rows are its
// source's (DataGridViewOf shares one between tables; DataGridView makes
// one of its own). A table shows in one window at a time.
type DataGridViewWidget[T any, K comparable] struct {
	common[DataGridViewWidget[T, K]]
	src     *RowSource[T, K]
	key     func(*T) K // src's
	link    *tableLink // what src knows of the table
	cols    []Column[T]
	columns []ui.TableColumn // cols, after the index column if any
	view    *ui.TableView[T]
	filter  func(*T) bool // the view's, nil for none
	// frame holds the rows of the frame ui shows, and order their order
	// once kept, when a sync outside a build changed the view: ui may
	// still lay that frame out, or apply input to it. Setters never move
	// or remove a frame's rows: they make new slices. gone is set when
	// rows went since, so that commits look for their keys only then.
	frame  []T
	order  []int
	heads  ui.TableGroups // the view's groups, kept with order
	kept   bool
	gone   bool
	resets int // test hook: how often catchUp had the view start over
	asks   int // test hook: how often ui asked whether a line is a header
	// posOf is each row's place in the view, -1 if hidden, as of
	// posVersion: Selected and Checked build it on first use.
	posOf                                   []int32
	posVersion                              uint64
	sort                                    mirror[ui.SortOrder]
	chosen, checked                         keys[K]
	checks                                  bool // the check box column shows
	panel                                   *BoxWidget
	totalLabel, selectedLabel, checkedLabel *LabelWidget
	column                                  *ComboBoxWidget
	keyword                                 *TextBoxWidget
	searchButton                            *ButtonWidget
	text                                    PanelText
	textSet                                 bool // PanelText was called: the words are the app's, not the language's
	beforeSearch                            func(column, keyword string)
	onActivate                              func(K)
	onColumns                               func(ui.TableLayout)
	shortcuts                               []shortcut
	menu                                    []MenuItem
	menuFunc                                func([]K) []MenuItem // ContextMenuFunc's
	copyFn                                  func([]K) string     // CopyFunc's
	late                                    []MenuItem           // what it built last
	lateSeq                                 uint64               // its latest right-click
	layout                                  *ui.TableLayout      // SetColumnsLayout's, not handed to ui yet
	numbered                                bool                 // the index column shows

	// watch, when set by a ListView, gets the keys the user added to and took
	// out of the choice, or of the checks (checks).
	watch func(checks bool, on, off []K)

	// ui's, on the main thread:
	list   ui.ListState
	cursor int
}

// DataGridView makes a table of cols whose rows key tells apart. key runs under
// the package's lock, as the columns' functions do.
func DataGridView[T any, K comparable](key func(*T) K, cols ...Column[T]) *DataGridViewWidget[T, K] {
	return DataGridViewOf(NewRowSource(key), cols...)
}

// DataGridViewOf makes a table of cols that shows the rows of src, which
// other tables may show too: the table has its own sort, search, selection
// and checks. Its SetRows, AddRows, UpdateRow and DeleteRows change src.
func DataGridViewOf[T any, K comparable](src *RowSource[T, K], cols ...Column[T]) *DataGridViewWidget[T, K] {
	compare := map[string]func(a, b *T) int{}
	for _, cl := range cols {
		switch {
		case !cl.Sortable:
		case cl.Compare != nil:
			compare[cl.Title] = cl.Compare
		default:
			text := cl.Text
			compare[cl.Title] = func(a, b *T) int { return strings.Compare(text(a), text(b)) }
		}
	}
	t := &DataGridViewWidget[T, K]{src: src, key: src.key, link: &tableLink{}, cols: cols, view: ui.NewTableView(compare), cursor: -1}
	t.columns = t.uiColumns()
	t.list.Key, t.list.Label, t.list.Selected = t.keyAt, t.labelAt, &t.cursor
	t.list.Copy = t.copyRows
	t.init(t)
	t.link.id, t.link.prune = t.id, t.prune
	mu.Lock()
	src.attach(t.link)
	mu.Unlock()
	return t
}

// uiColumns returns the columns ui shows. Under mu, or before the table
// is shared.
func (t *DataGridViewWidget[T, K]) uiColumns() []ui.TableColumn {
	var cs []ui.TableColumn
	if t.numbered {
		cs = append(cs, ui.TableColumn{Title: "#", Width: 56, Align: ui.End, Fixed: true})
	}
	for _, cl := range t.cols {
		cs = append(cs, ui.TableColumn{Title: cl.Title, Width: cl.Width, MinWidth: cl.MinWidth,
			Align: cl.Align.ui(), Sortable: cl.Sortable})
	}
	return cs
}

// IndexColumn shows first the position of each row as shown, from 1.
func (t *DataGridViewWidget[T, K]) IndexColumn() *DataGridViewWidget[T, K] {
	t.change(func() { t.numbered = true; t.columns = t.uiColumns() })
	return t
}

// SetRows shows a copy of rows instead of the rows shown. Of rows with
// the same key, the last stays, where the first was. Like AddRows,
// UpdateRow and DeleteRows, it changes the table's source.
func (t *DataGridViewWidget[T, K]) SetRows(rows []T) *DataGridViewWidget[T, K] {
	t.src.SetRows(rows)
	return t
}

// AddRow shows row after the others, or instead of the row of its key.
func (t *DataGridViewWidget[T, K]) AddRow(row T) *DataGridViewWidget[T, K] {
	return t.AddRows([]T{row})
}

// AddRows shows rows after the others, each instead of the row of its
// key if one shows.
func (t *DataGridViewWidget[T, K]) AddRows(rows []T) *DataGridViewWidget[T, K] {
	t.src.AddRows(rows)
	return t
}

// UpdateRow runs fn on the row of key k, if one shows, to change it in
// place. fn runs under the package's lock: it changes the row but not its
// key, and calls no widget method.
func (t *DataGridViewWidget[T, K]) UpdateRow(k K, fn func(*T)) *DataGridViewWidget[T, K] {
	t.src.UpdateRow(k, fn)
	return t
}

// DeleteRows removes the rows of keys ks.
func (t *DataGridViewWidget[T, K]) DeleteRows(ks ...K) *DataGridViewWidget[T, K] {
	t.src.DeleteRows(ks...)
	return t
}

// Clear removes every row.
func (t *DataGridViewWidget[T, K]) Clear() *DataGridViewWidget[T, K] { return t.SetRows(nil) }

// catchUp applies the changes the source made since the table last read
// them: rows changed in place go to the view. A table that fell behind the
// log, or by more than it holds, has the view sort again. Rows that went
// were pruned as they did (prune). Under mu.
func (t *DataGridViewWidget[T, K]) catchUp() {
	l, s := t.link, t.src
	if l.seq == s.seq {
		return
	}
	if changes, ok := s.since(l.seq); !ok || len(changes) > s.bound() {
		t.view.Reset()
		t.resets++
	} else {
		for _, c := range changes {
			if c.row >= 0 {
				t.view.Changed(c.row)
			}
		}
	}
	l.seq = s.seq
}

// prune is the table's link's: rows went from the source. The view sorts
// again, the selection and the checks drop their keys, and commits look
// for the keys of rows gone until the table builds again. Under mu.
func (t *DataGridViewWidget[T, K]) prune() {
	t.view.Reset()
	t.gone = true
	t.chosen.keep(t.src.index)
	t.checked.keep(t.src.index)
}

// Row returns the row of key k.
func (t *DataGridViewWidget[T, K]) Row(k K) (T, bool) { return t.src.Row(k) }

// Len returns how many rows the table has, the hidden by a search too.
func (t *DataGridViewWidget[T, K]) Len() int { return t.src.Len() }

// ClearSort shows the rows in the order added, as before a header click.
func (t *DataGridViewWidget[T, K]) ClearSort() *DataGridViewWidget[T, K] {
	t.change(func() { t.sort.set(ui.SortOrder{}) })
	return t
}

// sync brings the view up to date with the rows, keeping first, outside
// a build, the order of the frame ui shows, once a frame. Under mu.
func (t *DataGridViewWidget[T, K]) sync() {
	t.catchUp()
	if !viewing.Load() && !t.kept {
		t.order = t.order[:0]
		for i := range t.view.Len() {
			t.order = append(t.order, t.view.Row(i))
		}
		t.heads = append(t.heads[:0], t.view.Groups()...)
		t.kept = true
	}
	t.view.Sync(t.src.rows, t.sort.value)
}

// rowAtPos returns the row at position pos of the view in the frame ui shows,
// whatever goroutines deleted or reordered since. Under mu.
func (t *DataGridViewWidget[T, K]) rowAtPos(pos int) *T {
	switch {
	case pos < 0:
		return nil
	case t.kept && pos < len(t.order):
		return &t.frame[t.order[pos]]
	case !t.kept && pos < t.view.Len():
		return &t.frame[t.view.Row(pos)]
	}
	return nil
}

// groupsOf returns the groups of the frame ui shows: none without groups.
func (t *DataGridViewWidget[T, K]) groupsOf() ui.TableGroups {
	if t.kept {
		return t.heads
	}
	return t.view.Groups()
}

// posAt returns the position in the view of the line ui shows at row, which
// counts the headers of the groups; -1 for a header. Under mu.
func (t *DataGridViewWidget[T, K]) posAt(row int) int {
	pos, _ := t.groupsOf().PosOf(row)
	return pos
}

// headAt returns the group a header at row heads in the frame ui shows. Under mu.
func (t *DataGridViewWidget[T, K]) headAt(row int) (ui.TableGroup, bool) {
	gs := t.groupsOf()
	if pos, g := gs.PosOf(row); pos < 0 && g >= 0 && g < len(gs) {
		return gs[g], true
	}
	return ui.TableGroup{}, false
}

// rowAt returns the row at row, a line ui shows, in the frame it shows;
// nil for a header. Under mu.
func (t *DataGridViewWidget[T, K]) rowAt(row int) *T { return t.rowAtPos(t.posAt(row)) }

// headerAt is ui's ListState.Header: a header is the line of a group. When the
// table is not grouped, ui is not given it.
func (t *DataGridViewWidget[T, K]) headerAt(row int) bool {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	t.asks++
	_, ok := t.headAt(row)
	return ok
}

// lockOutsideView takes mu unless a view is building, which holds it, and
// reports whether it did; see viewing.
func lockOutsideView() bool {
	if viewing.Load() {
		return false
	}
	mu.Lock()
	return true
}

// keyAt is ui's ListState.Key.
func (t *DataGridViewWidget[T, K]) keyAt(row int) any {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if g, ok := t.headAt(row); ok {
		return groupKey{g.Title}
	}
	return t.keyOfPos(t.posAt(row))
}

// keyAtPos is the key of the row at position pos of the view.
func (t *DataGridViewWidget[T, K]) keyAtPos(pos int) any {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	return t.keyOfPos(pos)
}

// keyOfPos is keyAtPos under mu.
func (t *DataGridViewWidget[T, K]) keyOfPos(pos int) any {
	if r := t.rowAtPos(pos); r != nil {
		return t.key(r)
	}
	var zero K
	return zero
}

// labelAt is ui's ListState.Label, the first column's text: it names a
// row's check box, and typing finds rows by it.
func (t *DataGridViewWidget[T, K]) labelAt(row int) string {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if g, ok := t.headAt(row); ok {
		return groupText(ListViewGroup{g.Title, g.Len})
	}
	return t.labelOfPos(t.posAt(row))
}

// labelAtPos is the label of the row at position pos of the view.
func (t *DataGridViewWidget[T, K]) labelAtPos(pos int) string {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	return t.labelOfPos(pos)
}

// labelOfPos is labelAtPos under mu.
func (t *DataGridViewWidget[T, K]) labelOfPos(pos int) string {
	if r := t.rowAtPos(pos); r != nil && len(t.cols) > 0 {
		return t.cols[0].Text(r)
	}
	return ""
}

func (t *DataGridViewWidget[T, K]) build(c *ui.Context) ui.Element {
	// It fills the room it is given, as ScrollPanel does: as tall as its
	// rows, it would not fit where nothing grows it, as in View.
	return ui.Column(c).Gap(8).Grow(1).Children(func() {
		t.table(c).Grow(1)
		if t.panel != nil {
			t.counts()
			render(c, t.panel)
		}
	})
}

// table builds the ui table. Under mu.
func (t *DataGridViewWidget[T, K]) table(c *ui.Context) ui.Element {
	s := &t.list
	s.Sort, _ = t.sort.bind()
	s.Selection, s.Checked, s.RowsOf = t.chosen.bind(), nil, t.chosenRows
	if t.checks {
		s.Checked, s.CheckedCount, s.CheckRows = t.checked.bind(), t.checkedCount, t.checkRows
	}
	t.sync()
	t.frame, t.kept, t.gone = t.src.rows, false, false
	s.Version = t.view.Version()
	s.Header = nil
	if t.view.Groups() != nil {
		s.Header = t.headerAt
	}
	t.handOver()
	e := ui.Table(c, s, t.columns, t.view.Places(), func(row, col int) { t.cell(c, row, col) }).
		OnSortChange(onChange(&t.sort, nil)).
		OnSelectionChange(func() { t.commit(&t.chosen) }).
		OnCheckedChange(func() { t.commit(&t.checked) })
	t.events(e)
	return e
}

// cell builds the cell of the row shown at row in column col, an index
// into t.columns: while the table builds, or when ui lays out rows it did
// not build then.
func (t *DataGridViewWidget[T, K]) cell(c *ui.Context, row, col int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if g, ok := t.headAt(row); ok {
		if col == 0 { // a header spans the columns, built as column 0
			groupHeader(c, ListViewGroup{g.Title, g.Len})
		}
		return
	}
	if t.numbered {
		if col == 0 {
			ui.Text(c, strconv.Itoa(t.posAt(row)+1)).SingleLine()
			return
		}
		col--
	}
	cl, r := &t.cols[col], t.rowAt(row)
	// A selected row's text keeps the selection's color, which reads on
	// the accent; a column's own may not.
	colored := cl.Color != nil && !c.RowSelected()
	if cl.Image == nil {
		cellText(c, cl, r, colored)
		return
	}
	ui.Row(c).Gap(4).Children(func() {
		ui.Image(c, cl.Image(r)).Size(16, 16)
		cellText(c, cl, r, colored)
	})
}

func cellText[T any](c *ui.Context, cl *Column[T], r *T, colored bool) {
	var e ui.Element
	if spans := spansOf(c, cl, r); spans != nil {
		e = ui.RichText(c, spans...).SingleLine()
	} else {
		e = ui.Text(c, cl.Text(r)).SingleLine()
	}
	if colored {
		if col := cl.Color(r); col != (ui.Color{}) {
			e.TextColor(col)
		}
	}
}

// spansOf returns the spans a cell draws, nil for its Text. A selected
// row's lose their colors, so the text reads on the selection.
func spansOf[T any](c *ui.Context, cl *Column[T], r *T) []ui.Span {
	if cl.Spans == nil {
		return nil
	}
	spans := cl.Spans(r)
	if len(spans) == 0 {
		return nil
	}
	if !c.RowSelected() {
		return spans
	}
	spans = slices.Clone(spans)
	for i := range spans {
		spans[i].Color = ui.Color{}
	}
	return spans
}

func (t *DataGridViewWidget[T, K]) fillsAcross() bool { return true }
