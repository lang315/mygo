package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// ListBoxWidget shows items in a scrolling list, as WinForms' ListBox, and
// with check boxes as its CheckedListBox. An item, of type T, is told apart
// by a key of type K, so that its choice and its check follow it as items
// are added, removed and replaced above it. The list builds only the rows in
// view, and a click costs what the rows shown cost, not the number of items.
//
// The functions the list calls back (key, text, SetImage's, and
// UpdateItem's) run under the package's lock: they call no widget method.
type ListBoxWidget[T any, K comparable] struct {
	common[ListBoxWidget[T, K]]
	key   func(*T) K
	text  func(*T) string
	image func(*T) *ui.Bitmap

	// items are never moved or removed in place: a setter that does makes
	// a new slice, so that frame, the items of the frame ui shows, stays
	// as ui lays it out.
	items []T
	index map[K]int // an item's place in items
	frame []T

	checks, multi bool
	// chosen and checked hold the keys chosen and checked: value is the
	// list's, and ui's copy of each is folded back at the start of the next
	// build, which reports what the user changed.
	chosen, checked keys[K]
	// anchor is the item the keys move from: the last one chosen. fixed says
	// a setter chose it, not ui's cursor.
	anchor    K
	hasAnchor bool
	fixed     bool

	// reveal is an item to scroll into view at the next build (EnsureVisible).
	reveal    K
	hasReveal bool

	onSelect   func([]K)
	onCheck    func(K, bool)
	onActivate func(K)

	// ui's, on the main thread:
	list   ui.ListState
	cursor int
}

// ListBox makes a list of items, of type T, that key tells apart by a key of
// type K, and text shows. key and text run under the package's lock.
func ListBox[T any, K comparable](key func(*T) K, text func(*T) string) *ListBoxWidget[T, K] {
	l := &ListBoxWidget[T, K]{key: key, text: text, index: map[K]int{}, cursor: -1}
	l.init(l)
	return l
}

// CheckedListBox makes a ListBox with a check box before each item.
func CheckedListBox[T any, K comparable](key func(*T) K, text func(*T) string) *ListBoxWidget[T, K] {
	return ListBox(key, text).CheckBoxes()
}

// CheckBoxes shows a check box before each item: a CheckedListBox is a
// ListBox that has it.
func (l *ListBoxWidget[T, K]) CheckBoxes() *ListBoxWidget[T, K] {
	l.change(func() { l.checks = true })
	return l
}

// Multi lets the user choose several items: a click chooses one, Cmd-click
// (Ctrl-click on Linux and Windows) adds an item or takes it out, and
// Shift-click chooses the items from the last one chosen. Without it one
// item is chosen, the one clicked last.
func (l *ListBoxWidget[T, K]) Multi() *ListBoxWidget[T, K] {
	l.change(func() { l.multi = true })
	return l
}

// SetImage shows an image before the text of each item, 16 DIPs square; nil,
// or a nil image, for none. fn runs under the package's lock.
func (l *ListBoxWidget[T, K]) SetImage(fn func(*T) *ui.Bitmap) *ListBoxWidget[T, K] {
	l.change(func() { l.image = fn })
	return l
}

// dedupe returns a copy of items of which, for the same key, the last stays,
// where the first was, with each item's place. Under mu.
func (l *ListBoxWidget[T, K]) dedupe(items []T) ([]T, map[K]int) {
	out := make([]T, 0, len(items))
	index := make(map[K]int, len(items))
	for i := range items {
		k := l.key(&items[i])
		if j, ok := index[k]; ok {
			out[j] = items[i]
			continue
		}
		index[k] = len(out)
		out = append(out, items[i])
	}
	return out, index
}

// set makes items the items, a new slice, and drops the keys of those gone
// from the choice and the checks. Under mu.
func (l *ListBoxWidget[T, K]) set(items []T, index map[K]int) {
	l.items, l.index = items, index
	l.chosen.keep(index)
	l.checked.keep(index)
	if _, ok := index[l.anchor]; !ok {
		l.hasAnchor = false
	}
}

// SetItems shows a copy of items instead of the items shown. Of items with
// the same key, the last stays, where the first was. An item whose key is
// among the new ones keeps its choice and its check; the others lose them,
// silently, so that a key added again starts clean.
func (l *ListBoxWidget[T, K]) SetItems(items []T) *ListBoxWidget[T, K] {
	l.change(func() { l.set(l.dedupe(items)) })
	return l
}

// AddItem shows item after the others, or instead of the item of its key.
func (l *ListBoxWidget[T, K]) AddItem(item T) *ListBoxWidget[T, K] { return l.AddItems([]T{item}) }

// AddItems shows items after the others, each instead of the item of its key
// if one shows, where it was.
func (l *ListBoxWidget[T, K]) AddItems(items []T) *ListBoxWidget[T, K] {
	l.change(func() {
		for i := range items {
			k := l.key(&items[i])
			if j, ok := l.index[k]; ok {
				l.items[j] = items[i]
				continue
			}
			l.index[k] = len(l.items)
			l.items = append(l.items, items[i])
		}
	})
	return l
}

// UpdateItem runs fn on the item of key k, if one shows, to change it in
// place. fn runs under the package's lock: it changes the item but not its
// key, and calls no widget method.
func (l *ListBoxWidget[T, K]) UpdateItem(k K, fn func(*T)) *ListBoxWidget[T, K] {
	l.change(func() {
		if i, ok := l.index[k]; ok {
			fn(&l.items[i])
		}
	})
	return l
}

// RemoveItems removes the items of keys ks, with their choice and check and
// without an event. A key that is not an item is ignored.
func (l *ListBoxWidget[T, K]) RemoveItems(ks ...K) *ListBoxWidget[T, K] {
	l.change(func() {
		gone := make(map[K]struct{}, len(ks))
		for _, k := range ks {
			if _, ok := l.index[k]; ok {
				gone[k] = struct{}{}
			}
		}
		if len(gone) == 0 {
			return
		}
		out := make([]T, 0, len(l.items)-len(gone))
		index := make(map[K]int, len(l.items)-len(gone))
		for i := range l.items {
			k := l.key(&l.items[i])
			if _, ok := gone[k]; ok {
				continue
			}
			index[k] = len(out)
			out = append(out, l.items[i])
		}
		l.set(out, index)
	})
	return l
}

// Clear removes every item.
func (l *ListBoxWidget[T, K]) Clear() *ListBoxWidget[T, K] { return l.SetItems(nil) }

// Item returns the item of key k.
func (l *ListBoxWidget[T, K]) Item(k K) (T, bool) {
	mu.Lock()
	defer mu.Unlock()
	if i, ok := l.index[k]; ok {
		return l.items[i], true
	}
	var zero T
	return zero, false
}

// Len returns how many items the list has.
func (l *ListBoxWidget[T, K]) Len() int {
	mu.Lock()
	defer mu.Unlock()
	return len(l.items)
}

// inOrder returns the keys of in, in the order of the list. Under mu.
func (l *ListBoxWidget[T, K]) inOrder(in *ui.Selection[K]) []K {
	n := in.Len()
	if n == 0 {
		return nil
	}
	ks := make([]K, 0, n)
	if n <= len(l.items)/8 {
		// Few keys: look each one's place up, instead of walking the list.
		for k := range in.All() {
			if _, ok := l.index[k]; ok {
				ks = append(ks, k)
			}
		}
		slices.SortFunc(ks, func(a, b K) int { return l.index[a] - l.index[b] })
		return ks
	}
	for i := range l.items {
		if k := l.key(&l.items[i]); in.Has(k) {
			if ks = append(ks, k); len(ks) == n {
				break
			}
		}
	}
	return ks
}

// Selected returns the keys of the items chosen, in the order of the list.
func (l *ListBoxWidget[T, K]) Selected() []K {
	mu.Lock()
	defer mu.Unlock()
	return l.inOrder(&l.chosen.value)
}

// CountSelected returns how many items are chosen.
func (l *ListBoxWidget[T, K]) CountSelected() int {
	mu.Lock()
	defer mu.Unlock()
	return l.chosen.value.Len()
}

// SetSelected chooses the items of keys ks, instead of those chosen, firing
// no event: the last, if the list is not Multi. A key that is not an item is
// ignored.
func (l *ListBoxWidget[T, K]) SetSelected(ks ...K) *ListBoxWidget[T, K] {
	l.change(func() {
		var known []K
		for _, k := range ks {
			if _, ok := l.index[k]; ok {
				known = append(known, k)
			}
		}
		if !l.multi && len(known) > 1 {
			known = known[len(known)-1:]
		}
		l.clearChosen()
		for _, k := range known {
			l.chosen.set(k, true)
		}
		if len(known) > 0 {
			l.anchor, l.hasAnchor, l.fixed = known[len(known)-1], true, true
		}
	})
	return l
}

// EnsureVisible scrolls the list as little as shows the item of key k whole,
// as WinForms' EnsureVisible does, at the next frame: not at all if it is in
// view. It chooses nothing. A key that is not an item is ignored.
func (l *ListBoxWidget[T, K]) EnsureVisible(k K) *ListBoxWidget[T, K] {
	l.change(func() { l.reveal, l.hasReveal = k, true })
	return l
}

// ClearSelection chooses no item, firing no event.
func (l *ListBoxWidget[T, K]) ClearSelection() *ListBoxWidget[T, K] {
	l.change(func() {
		l.clearChosen()
		l.hasAnchor, l.fixed = false, true
	})
	return l
}

// clearChosen takes every item out of the choice. Under mu.
func (l *ListBoxWidget[T, K]) clearChosen() {
	for _, k := range slices.Collect(l.chosen.value.All()) {
		l.chosen.set(k, false)
	}
}

// Checked returns the keys of the items checked (a CheckedListBox's
// CheckedItems), in the order of the list.
func (l *ListBoxWidget[T, K]) Checked() []K {
	mu.Lock()
	defer mu.Unlock()
	return l.inOrder(&l.checked.value)
}

// IsChecked reports whether the item of key k is checked.
func (l *ListBoxWidget[T, K]) IsChecked(k K) bool {
	mu.Lock()
	defer mu.Unlock()
	return l.checked.value.Has(k)
}

// SetChecked checks, or unchecks, the items of keys ks, firing no event. A
// key that is not an item is ignored.
func (l *ListBoxWidget[T, K]) SetChecked(ks []K, on bool) *ListBoxWidget[T, K] {
	l.change(func() {
		for _, k := range ks {
			if _, ok := l.index[k]; ok {
				l.checked.set(k, on)
			}
		}
	})
	return l
}

// OnSelect runs fn on the event goroutine with the keys of the items chosen,
// in the order of the list, when the user changes the choice (WinForms'
// SelectedIndexChanged). A setter does not run it.
func (l *ListBoxWidget[T, K]) OnSelect(fn func(selected []K)) *ListBoxWidget[T, K] {
	l.change(func() { l.onSelect = fn })
	return l
}

// OnCheck runs fn on the event goroutine with the key of an item the user
// checked or unchecked (WinForms' ItemCheck), and whether it is checked. A
// setter does not run it.
func (l *ListBoxWidget[T, K]) OnCheck(fn func(k K, on bool)) *ListBoxWidget[T, K] {
	l.change(func() { l.onCheck = fn })
	return l
}

// OnActivate runs fn on the event goroutine with the key of an item the user
// double-clicked, or chose with Enter.
func (l *ListBoxWidget[T, K]) OnActivate(fn func(k K)) *ListBoxWidget[T, K] {
	l.change(func() { l.onActivate = fn })
	return l
}

// takeUser folds what the user did since the last build into the list's sets
// and posts the events. Under mu, at the start of a build, while frame is
// still the frame ui showed.
func (l *ListBoxWidget[T, K]) takeUser() {
	valid := func(k K) bool { _, ok := l.index[k]; return ok }
	if l.multi {
		l.takeMulti(valid)
	} else {
		l.takeSingle(valid)
	}

	on, off := foldKeys(&l.checked, &l.checked.shown, valid)
	if h := l.onCheck; h != nil {
		for _, k := range on {
			Post(func() { h(k, true) })
		}
		for _, k := range off {
			Post(func() { h(k, false) })
		}
	}
}

// takeSingle takes the choice of a single list: ui holds one row, the cursor,
// which the user moved if its item is not the one chosen. A setter's choice
// stands over a cursor ui had not seen it in. Under mu.
func (l *ListBoxWidget[T, K]) takeSingle(valid func(K) bool) {
	if l.fixed {
		l.fixed = false
		return
	}
	c := l.cursor
	if c < 0 || c >= len(l.frame) {
		return
	}
	k := l.key(&l.frame[c])
	if !valid(k) || l.chosen.value.Has(k) {
		return
	}
	l.chosen.value.Clear()
	l.chosen.value.Add(k)
	if h := l.onSelect; h != nil {
		Post(func() { h([]K{k}) })
	}
}

// takeMulti takes the choices of a list that chooses several, which ui holds
// as a set. Under mu.
func (l *ListBoxWidget[T, K]) takeMulti(valid func(K) bool) {
	if l.fixed {
		l.fixed = false
	} else if c := l.cursor; c >= 0 && c < len(l.frame) {
		l.anchor, l.hasAnchor = l.key(&l.frame[c]), true
	}
	on, off := foldKeys(&l.chosen, &l.chosen.shown, valid)
	if len(on)+len(off) > 0 {
		if h := l.onSelect; h != nil {
			ks := l.inOrder(&l.chosen.value)
			Post(func() { h(ks) })
		}
	}
}

// keyAt is ui's ListState.Key.
func (l *ListBoxWidget[T, K]) keyAt(row int) any {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if row < len(l.frame) {
		return l.key(&l.frame[row])
	}
	var zero K
	return zero
}

// labelAt is ui's ListState.Label: typing the first letters of an item's text
// chooses it.
func (l *ListBoxWidget[T, K]) labelAt(row int) string {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if row < len(l.frame) {
		return l.text(&l.frame[row])
	}
	return ""
}

func (l *ListBoxWidget[T, K]) build(c *ui.Context) ui.Element {
	l.takeUser()
	l.frame = l.items
	s := &l.list
	s.Key, s.Label, s.Selected = l.keyAt, l.labelAt, &l.cursor
	l.cursor = -1
	if l.multi {
		if i, ok := l.index[l.anchor]; ok && l.hasAnchor {
			l.cursor = i
		}
		s.Selection = l.chosen.bind()
	} else {
		// ui chooses one row when it holds no Selection: Cmd+A, Cmd-click and
		// Shift-extend are for lists of several.
		s.Selection = nil
		for k := range l.chosen.value.All() {
			if i, ok := l.index[k]; ok {
				l.cursor = i
			}
			break
		}
	}
	if l.checks {
		l.checked.bind()
	}
	e := ui.List(c, s, len(l.frame), func(i int) { l.row(c, i) }).Grow(1)
	if l.hasReveal && !c.Measuring() { // the request is for the real window
		// Made once: ui keeps the request until the row is measured and shows,
		// however far it is, and a row in view moves nothing.
		l.hasReveal = false
		if i, ok := l.index[l.reveal]; ok {
			s.ScrollIntoView(i)
		}
	}
	if l.checks {
		e.OnShortcut(0, ui.KeySpace, l.toggleChecks)
	}
	if !l.multi {
		// A single list's choice moves with Shift too, as WinForms'.
		e.OnShortcut(ui.Shift, ui.KeyDown, func() { l.step(1) })
		e.OnShortcut(ui.Shift, ui.KeyUp, func() { l.step(-1) })
	}
	if fn := l.onActivate; fn != nil {
		e.OnActivate(func(row int) {
			if k, ok := l.keyAt(row).(K); ok {
				Post(func() { fn(k) })
			}
		})
	}
	return e
}

// step moves a single list's choice d rows, to the first or the last row when
// there is none, and stays at the ends. On the main thread.
func (l *ListBoxWidget[T, K]) step(d int) {
	n := len(l.frame)
	if n == 0 {
		return
	}
	to := l.cursor + d
	if l.cursor < 0 {
		to = 0
		if d < 0 {
			to = n - 1
		}
	}
	to = min(max(to, 0), n-1)
	if to != l.cursor {
		l.cursor = to
		l.list.ScrollIntoView(to)
	}
}

// toggleChecks checks the items chosen, or unchecks them when all are
// checked: the item the cursor is on in a single list. On the main thread.
func (l *ListBoxWidget[T, K]) toggleChecks() {
	var chosen []K
	if l.multi {
		chosen = slices.Collect(l.chosen.shown.All())
	} else if c := l.cursor; c >= 0 && c < len(l.frame) {
		chosen = []K{l.key(&l.frame[c])}
	}
	checked := &l.checked.shown
	all := len(chosen) > 0
	for _, k := range chosen {
		if !checked.Has(k) {
			all = false
			break
		}
	}
	for _, k := range chosen {
		if all {
			checked.Remove(k)
		} else {
			checked.Add(k)
		}
	}
}

// row builds the row of the item shown at i: while the list builds, or when
// ui lays out rows it did not build then.
func (l *ListBoxWidget[T, K]) row(c *ui.Context, i int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if i >= len(l.frame) {
		ui.Text(c, "")
		return
	}
	it := &l.frame[i]
	text := l.text(it)
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1)).Padding(t.Space(1), t.Space(2)).Children(func() {
		if l.checks {
			k := l.key(it)
			on := l.checked.shown.Has(k)
			ui.Checkbox(c, &on, "").Label(text).OnChange(func() {
				if on {
					l.checked.shown.Add(k)
				} else {
					l.checked.shown.Remove(k)
				}
			})
		}
		if l.image != nil {
			if bmp := l.image(it); bmp != nil {
				ui.Image(c, bmp).Size(16, 16)
			}
		}
		ui.Text(c, text).SingleLine()
	})
}

func (l *ListBoxWidget[T, K]) fillsAcross() bool { return true }
