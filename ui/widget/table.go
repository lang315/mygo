package widget

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unsafe"

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
	// Hidden leaves the column out of the table at first, as WinForms'
	// Visible = false: SetColumnVisible shows it. A hidden column still
	// sorts the rows once they are sorted by it, and its Filterable
	// search works; it is not drawn, read by assistive technology or copied.
	Hidden bool

	// edit is the column's *CellEditor[T] (EditColumn), held as an any so that
	// Column does not name the type.
	edit any
}

// IntColumn makes a sortable column of an int, right-aligned, whose Compare
// is by value (so 9 sorts before 10): the cell shows get(row) in decimal. Set
// its other fields, as Width is, on the Column it returns.
func IntColumn[T any](title string, width float32, get func(*T) int) Column[T] {
	return Column[T]{
		Title: title, Width: width, Align: End, Sortable: true,
		Text:    func(r *T) string { return strconv.Itoa(get(r)) },
		Compare: func(a, b *T) int { return cmp.Compare(get(a), get(b)) },
	}
}

// DataGridViewWidget shows rows of type T, each told apart by a key of type K,
// in columns that sort, for as many as a million rows. The rows are its
// source's (DataGridViewOf shares one between tables; DataGridView makes
// one of its own). A table shows in one window at a time.
type DataGridViewWidget[T any, K comparable] struct {
	keys keyBox[K] // the boxed keys of the rows in view (ui's ListState.Key)
	common[DataGridViewWidget[T, K]]
	src     *RowSource[T, K]
	key     func(*T) K // src's
	link    *tableLink // what src knows of the table
	cols    []Column[T]
	columns []ui.TableColumn // the cols shown, after the index column if any
	visCols []int            // the indexes in cols of the columns shown: never changed, replaced
	full    []string         // the last order of all the columns' titles, hidden ones too
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
	near   int // the group of the line asked last: see locate
	// posOf is each row's place in the view, -1 if hidden, as of
	// posVersion: Selected and Checked build it on first use.
	posOf                                   []int32
	posVersion                              uint64
	sort                                    mirror[ui.SortOrder]
	chosen, checked                         keys[K]
	checks                                  bool // the check box column shows
	panel                                   *BoxWidget
	totalLabel, selectedLabel, checkedLabel *LabelWidget
	words                                   panelWords // what the panel's texts were worked out from
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
	bySource                                bool                 // it numbers the rows by their place in the source, and sorts by it
	noCopy                                  bool                 // SetClipboardCopy(false)
	onSelChanged                            func()               // OnSelectionChanged's
	selPending                              bool                 // a selection change is posted, not run yet
	// selSeen are the keys of the rows selected that a search showed, as of
	// the view's version selVersion and the selection's generation selGen;
	// nil when not known (see watchSearch).
	selSeen            map[K]struct{}
	selVersion, selGen uint64

	// In-cell editing (tableedit.go): its state is edAny, a *tableEditing, and the
	// parts of editing the table calls are the func fields below, all nil until
	// editing is turned on (enableEdit). The table names none of editing's types,
	// so a program that never edits does not link them.
	edAny       any
	cellHook    func(c *ui.Context, col int, r any) bool // builds the editor in the cell being edited (r is a *T)
	versionHook func(r any) uint64                       // the edited row's version (r is a *T)
	pruneHook   func()                                   // rows went
	prepareEdit func()                                   // before the table builds
	afterTable  func(e ui.Element, c *ui.Context)        // after it built
	editCell    func(row, col int, how ui.EditHow) bool
	startCell   func(row, col int, x, y float32, how ui.EditHow)

	// watch, when set by a ListView, gets the keys the user added to and took
	// out of the choice, or of the checks (checks).
	watch func(checks bool, on, off []K)

	// look versions how rows look (the columns, a column's image), which the
	// rows' own generations (RowSource.Version) do not: a change of it
	// builds every row again. Under mu.
	look uint64
	// images, when a ListView shows the first column's image from an image
	// list, is that list: its generation is part of the version of each row.
	images *ImageListWidget
	// follow is FollowEnd, and toEnd a ScrollToEnd to make at the next build;
	// the list is ui's, on the main thread, so the setters keep them here.
	follow, toEnd bool

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
	compare := map[string]func(a, b *T) int{
		// The view compares rows of one slice, the source's: their addresses
		// are in the source's order.
		"#": func(a, b *T) int { return cmp.Compare(uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b))) },
	}
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
	t := &DataGridViewWidget[T, K]{src: src, key: src.key, link: &tableLink{}, cols: slices.Clone(cols), view: ui.NewTableView(compare), cursor: -1}
	t.columns = t.uiColumns()
	t.list.Key, t.list.Label, t.list.Selected = t.keyAt, t.labelAt, &t.cursor
	t.list.RowVersion = t.rowVersion
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
		cs = append(cs, ui.TableColumn{Title: "#", Width: 56, Align: ui.End, Fixed: true, Sortable: t.bySource})
	}
	vis := make([]int, 0, len(t.cols)) // a new slice: rows ui builds late hold the old one
	for i, cl := range t.cols {
		if cl.Hidden {
			continue
		}
		vis = append(vis, i)
		cs = append(cs, ui.TableColumn{Title: cl.Title, Width: cl.Width, MinWidth: cl.MinWidth,
			Align: cl.Align.ui(), Sortable: cl.Sortable})
	}
	t.visCols = vis
	return cs
}

// SetColumnVisible shows or hides the column titled title, in the place its
// position and the columns' layout give it; a title no column has is ignored.
// Titles tell the columns apart, in layouts and in ui, so they must differ.
func (t *DataGridViewWidget[T, K]) SetColumnVisible(title string, on bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		changed := false
		for i := range t.cols {
			if t.cols[i].Title == title && t.cols[i].Hidden == on {
				t.cols[i].Hidden, changed = !on, true
			}
		}
		if changed {
			t.columns = t.uiColumns() // ui builds the rows it kept again when its columns change
		}
	})
	return t
}

// ColumnVisible reports whether a column titled title shows; false for a
// title no column has.
func (t *DataGridViewWidget[T, K]) ColumnVisible(title string) bool {
	mu.Lock()
	defer mu.Unlock()
	return slices.ContainsFunc(t.cols, func(cl Column[T]) bool { return cl.Title == title && !cl.Hidden })
}

// IndexColumn shows first the position of each row as shown, from 1.
func (t *DataGridViewWidget[T, K]) IndexColumn() *DataGridViewWidget[T, K] {
	t.change(func() { t.numbered, t.bySource = true, false; t.columns = t.uiColumns(); t.look++ })
	return t
}

// IndexColumnBySource shows first the place of each row in the source, from
// 1: the order rows show in unsorted, which sorting and searching do not
// change (inserting and deleting rows do). A click on its header, titled
// "#", sorts by it, as Sort("#", descending) does.
func (t *DataGridViewWidget[T, K]) IndexColumnBySource() *DataGridViewWidget[T, K] {
	t.change(func() { t.numbered, t.bySource = true, true; t.columns = t.uiColumns(); t.look++ })
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

// InsertRows shows rows before the row at i of the source, the order the
// rows show in unsorted, from 0, each instead of the row of its key if one
// is held (a row a search hides too); see RowSource.InsertRows.
func (t *DataGridViewWidget[T, K]) InsertRows(i int, rows ...T) *DataGridViewWidget[T, K] {
	t.src.InsertRows(i, rows...)
	return t
}

// Rows returns a copy of the rows, the hidden by a search too, in the
// source's order.
func (t *DataGridViewWidget[T, K]) Rows() []T { return t.src.Rows() }

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
	n := t.chosen.value.Len()
	t.chosen.keep(t.src.index)
	if t.chosen.value.Len() != n {
		t.selectionChanged()
	}
	t.checked.keep(t.src.index)
	if t.pruneHook != nil {
		t.pruneHook()
	}
}

// Row returns the row of key k.
func (t *DataGridViewWidget[T, K]) Row(k K) (T, bool) { return t.src.Row(k) }

// Len returns how many rows the table has, the hidden by a search too.
func (t *DataGridViewWidget[T, K]) Len() int { return t.src.Len() }

// ClearSort shows the rows in the source's order (as added, and inserted),
// as before a header click.
func (t *DataGridViewWidget[T, K]) ClearSort() *DataGridViewWidget[T, K] {
	t.change(func() { t.sort.set(ui.SortOrder{}) })
	return t
}

// Sort sorts the rows by the column titled column, as a click on its header
// does, and the header shows the direction. A title of no Sortable column
// is ignored.
func (t *DataGridViewWidget[T, K]) Sort(column string, descending bool) *DataGridViewWidget[T, K] {
	t.change(func() {
		if column == "#" && t.bySource || slices.ContainsFunc(t.cols, func(cl Column[T]) bool { return cl.Sortable && cl.Title == column }) {
			t.sort.set(ui.SortOrder{Column: column, Descending: descending})
		}
	})
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
	if t.onSelChanged != nil && t.filter != nil {
		t.watchSearch()
	} else {
		t.selSeen = nil
	}
}

// watchSearch fires OnSelectionChanged when the rows changed so that the
// search hides or shows other selected rows: the view's version moved, the
// selection did not (the user's changes, and deletes, fire by themselves). It
// costs a pass over the rows selected when the view changes. Under mu, after
// the view synced.
func (t *DataGridViewWidget[T, K]) watchSearch() {
	v, gen := t.view.Version(), t.chosen.gen
	if t.selSeen != nil && v == t.selVersion && gen == t.selGen {
		return
	}
	known := t.selSeen != nil && gen == t.selGen
	same, n := t.selSeen != nil, 0
	for k := range t.chosen.value.All() {
		if i, ok := t.src.index[k]; ok && t.filter(&t.src.rows[i]) {
			n++
			if _, was := t.selSeen[k]; !was {
				same = false
			}
		}
	}
	if same = same && n == len(t.selSeen); !same {
		if t.selSeen == nil {
			t.selSeen = make(map[K]struct{}, n)
		}
		clear(t.selSeen)
		for k := range t.chosen.value.All() {
			if i, ok := t.src.index[k]; ok && t.filter(&t.src.rows[i]) {
				t.selSeen[k] = struct{}{}
			}
		}
		if known {
			t.selectionChanged()
		}
	}
	t.selVersion, t.selGen = v, gen
}

// rowAtPos returns the row at position pos of the view in the frame ui shows,
// whatever goroutines deleted or reordered since. Under mu.
func (t *DataGridViewWidget[T, K]) rowAtPos(pos int) *T {
	if i := t.indexAtPos(pos); i >= 0 {
		return &t.frame[i]
	}
	return nil
}

// indexAtPos returns the index in the frame ui shows of the row at position
// pos of the view, -1 for none. Under mu.
func (t *DataGridViewWidget[T, K]) indexAtPos(pos int) int {
	switch {
	case pos < 0:
		return -1
	case t.kept && pos < len(t.order):
		return t.order[pos]
	case !t.kept && pos < t.view.Len():
		return t.view.Row(pos)
	}
	return -1
}

// numberAt is what the index column shows of the row at position pos of the
// view. Under mu.
func (t *DataGridViewWidget[T, K]) numberAt(pos int) int {
	if t.bySource {
		return t.indexAtPos(pos) + 1
	}
	return pos + 1
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
	pos, _ := t.locate(row)
	return pos
}

// locate is PosOf of the frame ui shows. A frame asks for its lines in order,
// so it tries the group of the line asked last before it searches. Under mu.
func (t *DataGridViewWidget[T, K]) locate(row int) (pos, group int) {
	gs := t.groupsOf()
	if k := t.near; k < len(gs) && gs[k].Start+k <= row && (k+1 == len(gs) || row < gs[k+1].Start+k+1) {
		if row == gs[k].Start+k {
			return -1, k
		}
		return row - k - 1, k
	}
	pos, group = gs.PosOf(row)
	t.near = max(group, 0)
	return pos, group
}

// headAt returns the group a header at row heads in the frame ui shows. Under mu.
func (t *DataGridViewWidget[T, K]) headAt(row int) (ui.TableGroup, bool) {
	gs := t.groupsOf()
	if pos, g := t.locate(row); pos < 0 && g >= 0 && g < len(gs) {
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

// rowVersion is ui's ListState.RowVersion: what the build of the row shown at
// row reads of the table, for a list that keeps rows it need not build: the
// generation of its row in the source, how rows look, and its place among the
// rows shown (a numbered column prints it). A header has none.
func (t *DataGridViewWidget[T, K]) rowVersion(row int) uint64 {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if _, ok := t.headAt(row); ok {
		return 0
	}
	pos := t.posAt(row)
	r := t.rowAtPos(pos)
	if r == nil {
		return 0
	}
	place := 0
	if t.numbered {
		place = t.numberAt(pos)
	}
	look := foldLook(t.look, t.images.generation())
	if t.versionHook != nil {
		look ^= t.versionHook(r) * 0x9ddfea08eb382d69 // the row of the edit builds again when the edit begins or ends
	}
	return versionOf(t.src.gens.of(t.key(r)), look, place)
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
		return t.keys.of(pos, t.key(r))
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
	if r := t.rowAtPos(pos); r != nil && len(t.visCols) > 0 {
		return t.cols[t.visCols[0]].Text(r)
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
	if t.noCopy {
		s.Copy = nil
	} else if s.Copy == nil {
		s.Copy = t.copyRows
	}
	if t.prepareEdit != nil {
		t.prepareEdit()
	}
	s.Version = t.view.Version()
	s.Header = nil
	if t.view.Groups() != nil {
		s.Header = t.headerAt
	}
	t.handOver()
	s.FollowEnd = t.follow
	if t.toEnd && !c.Measuring() { // the request is for the real window
		t.toEnd = false
		s.ScrollToEnd()
	}
	vis := t.visCols // the columns ui is given now, for the cells it builds late
	t.keepHidden(&s.Columns)
	e := ui.Table(c, s, t.columns, t.view.Places(), func(row, col int) { t.cell(c, vis, row, col) }).
		OnSortChange(onChange(&t.sort, nil)).
		OnSelectionChange(func() { t.commit(&t.chosen) }).
		OnCheckedChange(func() { t.commit(&t.checked) })
	if t.afterTable != nil {
		t.afterTable(e, c)
	}
	t.events(e)
	return e
}

// cell builds the cell of the row shown at row in column col, an index
// into t.columns, whose cols are vis: while the table builds, or when ui lays out rows it did
// not build then.
func (t *DataGridViewWidget[T, K]) cell(c *ui.Context, vis []int, row, col int) {
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
			ui.Text(c, strconv.Itoa(t.numberAt(t.posAt(row)))).SingleLine()
			return
		}
		col--
	}
	cl, r := &t.cols[vis[col]], t.rowAt(row)
	if t.cellHook != nil && t.cellHook(c, vis[col], r) {
		return
	}
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
