package widget

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// LookUpEditWidget is a field that picks one record of a list by its key:
// its drop-down shows the records in columns, which typing filters, and
// the field shows the chosen record's text (DevExpress's LookUpEdit).
type LookUpEditWidget[T any, K comparable] struct {
	common[LookUpEditWidget[T, K]]
	key      func(*T) K
	text     func(*T) string
	cols     []Column[T] // the columns shown: those not Hidden
	heads    bool        // a column has a title: the header shows
	items    []T
	ids      []string // each record's value in ui's combobox: gen, a dot and its index, made when the items are set
	gen      int      // how many times the items were set: a choice of ids made before is stale
	sel      int      // the index of the chosen record, -1 for none
	input    mirror[string]
	onSelect func(k K, ok bool)
	// What only the drawing thread touches, under mu while building: the
	// context and the combobox of the build, the records the popup shows,
	// and the functions handed to ui, made once.
	c        *ui.Context
	cb       ui.ComboboxParts
	shown    []int
	children func()
	popup    func(panel ui.Element)
	search   *lookSearch // the search panel's, nil while it is off
}

// lookSearch is the state of a LookUpEdit's search panel that does not name
// T or K. Under mu.
type lookSearch struct {
	open      bool // ui's Popover writes it while the view builds
	seenHides int  // the hides the last build saw: a hide closes the drop-down
	query     string
	matched   bool   // shown holds the matches of seenQuery among the records of seenGen
	seenQuery string // the query shown matches
	seenGen   int
	hl        int // the index in shown of the row highlighted, -1 for none
	list      ui.ListState
	c         *ui.Context // the context of the last build, for the rows ui lays out after it
	in        ui.Element  // the search box of the last build, whose active descendant the highlighted row is
	text      string      // what the field shows
	// The functions handed to ui, made once.
	field func()
	panel func(panel ui.Element)
	row   func(i int)
}

// LookUpEdit creates a LookUpEdit: key names a record, text is what the
// field shows of the chosen one, and cols are the drop-down's columns, of
// which it reads Title, Text, Width, MinWidth, Align and Hidden. With no
// column the drop-down shows text alone.
func LookUpEdit[T any, K comparable](key func(*T) K, text func(*T) string, cols ...Column[T]) *LookUpEditWidget[T, K] {
	l := &LookUpEditWidget[T, K]{key: key, text: text, sel: -1}
	for _, c := range cols {
		if !c.Hidden {
			l.cols = append(l.cols, c)
			l.heads = l.heads || c.Title != ""
		}
	}
	l.children = l.buildField
	l.popup = l.buildPopup
	l.init(l)
	l.lookAs(skinhook.KindField) // typed into, as a TextBox
	return l
}

// SetItems sets the records, keeping the selection if its key is still
// one of them.
func (l *LookUpEditWidget[T, K]) SetItems(items []T) *LookUpEditWidget[T, K] {
	items = slices.Clone(items) // the caller's slice may change
	l.change(func() {
		was, had := "", l.sel >= 0
		var k K
		if had {
			k, was = l.key(&l.items[l.sel]), l.text(&l.items[l.sel])
		}
		l.items, l.sel = items, -1
		l.gen++
		g := strconv.Itoa(l.gen) + "."
		l.ids = make([]string, len(items)) // new strings: ui may still hold the old ones
		for i := range l.ids {
			l.ids[i] = g + strconv.Itoa(i)
		}
		if had {
			l.sel = l.index(k)
		}
		switch {
		case l.sel >= 0:
			if now := l.text(&l.items[l.sel]); now != was && l.input.value == was {
				l.input.set(now)
			}
		case had:
			l.input.set("")
		}
	})
	return l
}

// index returns the index of the record of key k, -1 for none. Under mu.
func (l *LookUpEditWidget[T, K]) index(k K) int {
	for i := range l.items {
		if l.key(&l.items[i]) == k {
			return i
		}
	}
	return -1
}

// Selected returns the key of the chosen record, and whether one is.
func (l *LookUpEditWidget[T, K]) Selected() (K, bool) {
	mu.Lock()
	defer mu.Unlock()
	var k K
	if l.sel < 0 {
		return k, false
	}
	return l.key(&l.items[l.sel]), true
}

// SetSelected chooses the record of key k, firing no event; a key that is
// not a record is ignored.
func (l *LookUpEditWidget[T, K]) SetSelected(k K) *LookUpEditWidget[T, K] {
	l.change(func() {
		if i := l.index(k); i >= 0 {
			l.sel = i
			l.input.set(l.text(&l.items[i]))
		}
	})
	return l
}

// Unselect chooses no record, firing no event.
func (l *LookUpEditWidget[T, K]) Unselect() *LookUpEditWidget[T, K] {
	l.change(func() {
		l.sel = -1
		l.input.set("")
	})
	return l
}

// OnSelect runs fn on the event goroutine when the user chooses a record
// (ok true) or empties the field (ok false). A setter does not run it.
func (l *LookUpEditWidget[T, K]) OnSelect(fn func(k K, ok bool)) *LookUpEditWidget[T, K] {
	l.change(func() { l.onSelect = fn })
	return l
}

// SetSearchPanel turns the search panel on or off (off by default). On, the
// field takes no typing and shows the chosen record's text; its drop-down
// has a search box over the records, which shows those of which a column
// shown (the text, with no column) holds what is typed, case ignored, as
// the DataGridView's search does. Up and Down in the search box move the
// highlighted record, Enter chooses it, Escape closes the drop-down. Only
// the records in view are built, however many there are (DevExpress's
// SearchLookUpEdit).
func (l *LookUpEditWidget[T, K]) SetSearchPanel(on bool) *LookUpEditWidget[T, K] {
	l.change(func() {
		switch {
		case !on:
			l.search, l.children = nil, l.buildField
		case l.search == nil:
			// The search's path is reached only from here: a LookUpEdit
			// without it builds as it did.
			s := &lookSearch{hl: -1, seenHides: l.hides}
			s.field = l.buildSearchText
			s.panel = l.buildSearchPopup
			s.row = func(i int) { l.buildSearchRow(s, i) } // s, as l.search may be nil by the time ui lays the rows out
			l.search, l.children = s, l.buildSearchField
		}
	})
	return l
}

// show puts s in the field, as the user's own text. Under mu, building.
func (l *LookUpEditWidget[T, K]) show(s string) {
	l.input.value, l.input.shown, l.input.dirty = s, s, false
}

// pick chooses the record of index i, as the user did, or none for -1, and
// runs OnSelect if that changed the choice. Under mu, building.
func (l *LookUpEditWidget[T, K]) pick(i int) {
	s := ""
	if i >= 0 {
		s = l.text(&l.items[i])
	}
	l.show(s)
	if i == l.sel {
		return
	}
	l.sel = i
	fn := l.onSelect
	if fn == nil {
		return
	}
	var k K
	if i >= 0 {
		k = l.key(&l.items[i])
	}
	Post(func() { fn(k, i >= 0) })
}

func (l *LookUpEditWidget[T, K]) build(c *ui.Context) ui.Element {
	t := c.Theme()
	l.c = c
	frame := ui.Row(c).Radius(t.Radius).Border(1, t.Outline()).Background(t.FieldFill()).
		AlignItems(ui.Center).MinWidth(t.Space(35))
	if h := t.Metrics.ControlHeight; h > 0 {
		frame.MinHeight(h)
	}
	frame.Children(l.children)
	l.c, l.cb = nil, ui.ComboboxParts{}
	return frame
}

// buildField builds the input, its popup and the chevron. Under mu,
// building.
func (l *LookUpEditWidget[T, K]) buildField() {
	c, t := l.c, l.c.Theme()
	p, _ := l.input.bind()
	l.cb = ui.ComboboxBase(c, p)
	in := l.cb.Input
	in.Grow(1).Padding(t.Space(1.5), t.Space(3)).AutoComplete(ui.AutoCompleteList)
	l.inner = in
	if h := formAccept; h != nil {
		in.OnSubmit(h) // Enter clicks the form's accept button
	}
	l.cb.Popup(l.popup)
	if v, ok := l.cb.Chosen(); ok {
		// ui may choose among the options of the last frame: one of items
		// set since names a record that moved or went, and chooses nothing.
		g, idx, _ := strings.Cut(v, ".")
		if gen, _ := strconv.Atoi(g); gen == l.gen {
			i, _ := strconv.Atoi(idx)
			l.pick(i)
		}
	}
	// Leaving the field puts the chosen record's text back: a free text is
	// never a value. Leaving it emptied chooses none.
	if in.FocusLeft() {
		switch {
		case strings.TrimSpace(*p) == "":
			l.pick(-1)
		case l.sel >= 0:
			l.show(l.text(&l.items[l.sel]))
		default:
			l.show("")
		}
	}
	ui.Icon(c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted).Margin(0, t.Space(3), 0, 0)
}

// buildPopup builds the drop-down: the header, then the records matching
// what was typed, those whose text starts with it first. Under mu,
// building.
func (l *LookUpEditWidget[T, K]) buildPopup(panel ui.Element) {
	c, t := l.c, l.c.Theme()
	comboPanel(c, panel)
	scale := c.Preferences().TextScale
	pad, gap := t.Space(2.5), t.Space(2)
	if len(l.cols) > 0 {
		panel.Width(l.panelWidth(t, scale)) // a long text clips in its cell, rather than widening every row
	}

	l.shown = l.matching(strings.ToLower(strings.TrimSpace(l.input.shown))) // the input's text, as ui edits it
	if len(l.shown) == 0 {
		ui.Text(c, l10n.NoResults()).TextColor(t.TextMuted).Padding(t.Space(1.5), pad)
		return
	}
	l.header(scale)
	for _, i := range l.shown {
		r := &l.items[i]
		item := l.cb.Item(l.ids[i]).Gap(gap).Padding(t.Space(1.5), pad).Radius(t.Radius)
		if h := t.Metrics.RowHeight; h > 0 {
			item.MinHeight(h).PaddingY(0).AlignItems(ui.Center)
		}
		if item.Highlighted() {
			bg, fg := highlightOf(t)
			item.Background(bg).TextColor(fg)
		}
		item.Children(func() {
			if len(l.cols) == 0 {
				ui.Text(c, l.text(r)).SingleLine()
				return
			}
			for _, col := range l.cols {
				l.cell(col, scale, func() { ui.Text(c, col.Text(r)).SingleLine() })
			}
		})
	}
}

// cell builds a cell of col, of its width, holding what fn builds: the
// header's and every row's alike, so that they line up.
func (l *LookUpEditWidget[T, K]) cell(col Column[T], scale float32, fn func()) {
	box := ui.Row(l.c).AlignItems(ui.Center)
	if col.Width > 0 {
		box.Width(col.Width * scale).Shrink(0)
	} else {
		box.Grow(1) // the panel's width holds its MinWidth
	}
	switch col.Align {
	case Center:
		box.Justify(ui.Center)
	case End:
		box.Justify(ui.End)
	}
	box.Children(fn)
}

// matching returns the indexes of the records of which text or a shown
// column holds q (lower case), those whose text starts with it first; all
// of them for an empty q, or while the popup shows them all (opened by a
// click or Down). Under mu, building.
func (l *LookUpEditWidget[T, K]) matching(q string) []int {
	out := l.shown[:0]
	if q == "" || !l.cb.Filtering() {
		for i := range l.items {
			out = append(out, i)
		}
		return out
	}
	holds := func(s string) bool { return strings.Contains(strings.ToLower(s), q) }
	for i := range l.items {
		if strings.HasPrefix(strings.ToLower(l.text(&l.items[i])), q) {
			out = append(out, i)
		}
	}
	for i := range l.items {
		r := &l.items[i]
		if strings.HasPrefix(strings.ToLower(l.text(r)), q) {
			continue
		}
		ok := holds(l.text(r))
		for _, col := range l.cols {
			ok = ok || holds(col.Text(r))
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

// panelWidth is how wide the drop-down's columns need it. Under mu.
func (l *LookUpEditWidget[T, K]) panelWidth(t *ui.Theme, scale float32) float32 {
	pad, gap, flexMin := t.Space(2.5), t.Space(2), t.Space(15) // flexMin: the grid's least width of a shared column
	// As wide as its columns need, and at least as the field (ui's MinWidth,
	// which wins over Width).
	need := 2*(t.Space(1)+pad) + 2
	for i, col := range l.cols {
		if i > 0 {
			need += gap
		}
		if col.Width > 0 {
			need += col.Width * scale
		} else {
			need += max(col.MinWidth*scale, flexMin)
		}
	}
	return need
}

// header builds the drop-down's header, when a column has a title. Under
// mu, building.
func (l *LookUpEditWidget[T, K]) header(scale float32) {
	if !l.heads {
		return
	}
	c, t := l.c, l.c.Theme()
	ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2.5)).Children(func() {
		for _, col := range l.cols {
			l.cell(col, scale, func() { ui.Text(c, col.Title).SingleLine().FontWeight(600) })
		}
	})
}

// lookSearchRows is how many rows the search panel's list shows at most.
const lookSearchRows = 10

// buildSearchField builds the field of the search panel: the chosen
// record's text, which a click, Enter, Space, Down or Alt+Down opens the
// drop-down from. Under mu, building.
func (l *LookUpEditWidget[T, K]) buildSearchField() {
	c, t, s := l.c, l.c.Theme(), l.search
	s.c = c
	if s.seenHides != l.hides { // out of sight since the last build: it closes
		s.open, s.seenHides = false, l.hides
	}
	s.text = ""
	if l.sel >= 0 {
		s.text = l.text(&l.items[l.sel])
	}
	field := ui.ButtonBase(c).Role(ui.RoleComboBox).Value(s.text).Grow(1).
		Padding(t.Space(1.5), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.SpaceBetween)
	l.inner = field
	field.Children(s.field)
	opened := false
	if field.Clicked() {
		s.open = !s.open // a press on the field is not outside the drop-down: it toggles
		opened = s.open
	} else if !s.open && (field.Shortcut(ui.Alt, ui.KeyDown) || field.Shortcut(0, ui.KeyDown)) {
		s.open, opened = true, true
	}
	if opened {
		s.query, s.hl, s.matched, s.list = "", -1, false, ui.ListState{}
	}
	field.Expanded(s.open)
	if s.open {
		ui.PopoverBase(c, field, &s.open, s.panel)
	}
}

// buildSearchText builds what the search panel's field holds. Under mu,
// building.
func (l *LookUpEditWidget[T, K]) buildSearchText() {
	c := l.search.c
	t := c.Theme()
	ui.Text(c, l.search.text).SingleLine().AccessHidden(true) // the field's value, as the text field's is without the panel
	ui.Icon(c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
}

// buildSearchPopup builds the search panel's drop-down: the search box,
// the header, and a list of the matching records, which builds only those
// in view. Under mu, building.
func (l *LookUpEditWidget[T, K]) buildSearchPopup(panel ui.Element) {
	c, t, s := l.c, l.c.Theme(), l.search
	comboPanel(c, panel)
	scale := c.Preferences().TextScale
	panel.Gap(t.Space(1)).Width(max(l.panelWidth(t, scale), t.Space(60)))
	in := ui.TextInput(c, &s.query).AutoFocus().Label(l10n.Search()).Placeholder(l10n.Search())
	s.in = in
	if !s.matched || s.query != s.seenQuery || s.seenGen != l.gen {
		l.shown = l.searchMatching(strings.ToLower(s.query))
		s.matched, s.seenQuery, s.seenGen, s.hl = true, s.query, l.gen, -1
		s.list.ScrollTo(0, ui.Start)
	}
	n := len(l.shown)
	if in.Shortcut(0, ui.KeyDown) && s.hl < n-1 {
		s.hl++
		s.list.ScrollIntoView(s.hl)
	}
	if in.Shortcut(0, ui.KeyUp) && s.hl > 0 {
		s.hl--
		s.list.ScrollIntoView(s.hl)
	}
	if in.Submitted() && s.hl >= 0 { // hl < n: the matches reset it
		l.pick(l.shown[s.hl])
		s.open = false
		return
	}
	if n == 0 {
		ui.Text(c, l10n.NoResults()).TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2.5))
		return
	}
	l.header(scale)
	rh := t.Metrics.RowHeight
	if rh <= 0 {
		rh = t.Space(7)
	}
	ui.List(c, &s.list, n, s.row).Height(float32(min(n, lookSearchRows)) * rh)
}

// buildSearchRow builds the row of the search panel's list at i: while the
// list builds, or when ui lays out rows it did not build then.
func (l *LookUpEditWidget[T, K]) buildSearchRow(s *lookSearch, i int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	c, t := s.c, s.c.Theme()
	if i >= len(l.shown) || l.shown[i] >= len(l.items) { // the records changed under the list
		ui.Text(c, "")
		return
	}
	was := l.c
	l.c = c // cell's
	r := &l.items[l.shown[i]]
	row := ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius)
	if h := t.Metrics.RowHeight; h > 0 {
		row.MinHeight(h).PaddingY(0).AlignItems(ui.Center)
	}
	if i == s.hl {
		s.in.ActiveDescendant(row) // assistive technology reads the row the arrows are on
	}
	if i == s.hl || row.Hovered() {
		bg, fg := highlightOf(t)
		row.Background(bg).TextColor(fg)
	}
	if row.Clicked() {
		l.pick(l.shown[i])
		s.open = false
	}
	scale := c.Preferences().TextScale
	row.Children(func() {
		if len(l.cols) == 0 {
			ui.Text(c, l.text(r)).SingleLine()
			return
		}
		for _, col := range l.cols {
			l.cell(col, scale, func() { ui.Text(c, col.Text(r)).SingleLine() })
		}
	})
	l.c = was
}

// searchMatching returns the indexes of the records of which a column shown
// (the text, with no column) holds q (lower case), in their order; all of
// them for an empty q. Under mu.
func (l *LookUpEditWidget[T, K]) searchMatching(q string) []int {
	out := l.shown[:0]
	holds := func(s string) bool { return strings.Contains(strings.ToLower(s), q) }
	for i := range l.items {
		r := &l.items[i]
		ok := q == ""
		if !ok && len(l.cols) == 0 {
			ok = holds(l.text(r))
		}
		for _, col := range l.cols {
			ok = ok || holds(col.Text(r))
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}
