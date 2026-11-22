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
	ids      []string // each record's value in ui's combobox: its index, made when the items are set
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
		for len(l.ids) < len(items) {
			l.ids = append(l.ids, strconv.Itoa(len(l.ids)))
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
	l.cb.Popup(l.popup)
	if v, ok := l.cb.Chosen(); ok {
		i, _ := strconv.Atoi(v) // one of l.ids: this build's Items
		l.pick(i)
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
	if len(l.cols) > 0 {
		panel.Width(need) // a long text clips in its cell, rather than widening every row
	}

	l.shown = l.matching(strings.ToLower(strings.TrimSpace(l.input.shown))) // the input's text, as ui edits it
	if len(l.shown) == 0 {
		ui.Text(c, l10n.NoResults()).TextColor(t.TextMuted).Padding(t.Space(1.5), pad)
		return
	}
	if l.heads {
		ui.Row(c).Gap(gap).Padding(t.Space(1.5), pad).Children(func() {
			for _, col := range l.cols {
				l.cell(col, scale, func() { ui.Text(c, col.Title).SingleLine().FontWeight(600) })
			}
		})
	}
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
	t := l.c.Theme()
	box := ui.Row(l.c).AlignItems(ui.Center).Clip()
	if col.Width > 0 {
		box.Width(col.Width * scale).Shrink(0)
	} else {
		box.Grow(1).Shrink(1).MinWidth(max(col.MinWidth*scale, t.Space(15)))
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
