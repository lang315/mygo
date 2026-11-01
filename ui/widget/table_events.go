package widget

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

type shortcut struct {
	key  ui.Key
	mods ui.Modifiers
	fn   func()
}

// OnActivate runs fn with the key of a row double-clicked, or chosen when
// Enter is pressed.
func (t *DataGridViewWidget[T, K]) OnActivate(fn func(k K)) *DataGridViewWidget[T, K] {
	t.change(func() { t.onActivate = fn })
	return t
}

// OnKey runs fn when key is pressed with mods while the table has the
// keyboard focus. Each call adds a key.
func (t *DataGridViewWidget[T, K]) OnKey(key ui.Key, mods ui.Modifiers, fn func()) *DataGridViewWidget[T, K] {
	t.change(func() { t.shortcuts = append(t.shortcuts, shortcut{key, mods, fn}) })
	return t
}

// ContextMenu shows items when rows are right-clicked, or the menu key is
// pressed on them, and only then. An item's OnClick reads the rows from
// Selected: ui builds menus as it draws, under the package's lock, so the
// items cannot be made from the rows then.
func (t *DataGridViewWidget[T, K]) ContextMenu(items ...MenuItem) *DataGridViewWidget[T, K] {
	t.change(func() { t.menu = items })
	return t
}

// ContextMenuFunc shows the items fn makes when rows are right-clicked, or
// the menu key is pressed on them: fn gets the keys of the rows selected at
// that moment, in the order shown. fn runs on the event goroutine, after
// the click and without the package's lock, so it may read the table
// (Selected, CountSelected, Row), take its time, or query a database; the
// menu opens where the user clicked when fn returns. It does not open if
// the user pressed a key or a button meanwhile, if another right-click
// came (the latest wins), or if fn returns no items. An item's OnClick
// runs on the event goroutine, as for ContextMenu, which ContextMenuFunc
// replaces when both are set.
func (t *DataGridViewWidget[T, K]) ContextMenuFunc(fn func(selected []K) []MenuItem) *DataGridViewWidget[T, K] {
	t.change(func() { t.menuFunc = fn })
	return t
}

// lateMenu is ui's RowMenu for ContextMenuFunc. The build that opens the
// menu posts fn, with the keys of the rows; the build that follows fn's
// return (Menu.Later) and the one after an item was chosen make the items.
// Under mu, while building.
func (t *DataGridViewWidget[T, K]) lateMenu(m *ui.Menu, rows []int) {
	if open := m.Later(); open != nil {
		ks := make([]K, 0, len(rows))
		for _, row := range rows {
			if r := t.rowAt(row); r != nil {
				ks = append(ks, t.key(r))
			}
		}
		t.lateSeq++
		seq, fn := t.lateSeq, t.menuFunc
		Post(func() {
			items := fn(ks)
			mu.Lock()
			latest := seq == t.lateSeq
			if latest {
				t.late = items
			}
			mu.Unlock()
			if latest && len(items) > 0 {
				open()
			}
		})
		return
	}
	buildMenu(m, t.late)
}

// OnColumnsChange runs fn with the columns' layout after the user
// resizes, moves or fits a column: once a frame while a column is dragged.
// Keep it to give SetColumnsLayout.
func (t *DataGridViewWidget[T, K]) OnColumnsChange(fn func(ui.TableLayout)) *DataGridViewWidget[T, K] {
	t.change(func() { t.onColumns = fn })
	return t
}

// SetColumnsLayout arranges the columns as l, which OnColumnsChange gave:
// their order and widths by title ("#" for the index column).
func (t *DataGridViewWidget[T, K]) SetColumnsLayout(l ui.TableLayout) *DataGridViewWidget[T, K] {
	t.change(func() { t.layout = &l })
	return t
}

// CopyFunc sets what the table copies: fn gets the keys of the rows
// selected, in the order shown, and returns the text. Both Ctrl+C (Cmd+C)
// and the Edit menu's Copy use it, so they agree; without it, or with nil,
// they copy the cells of the rows, a line each, tab-separated. fn runs on
// the main thread, off the package's lock: it may call the table's getters,
// but should be quick.
func (t *DataGridViewWidget[T, K]) CopyFunc(fn func(selected []K) string) *DataGridViewWidget[T, K] {
	t.change(func() { t.copyFn = fn })
	return t
}

// copyRows is ui's ListState.Copy: the rows shown at the lines rows, a text
// each, their cells apart by tabs, or what CopyFunc makes of their keys. On
// the main thread, without mu.
func (t *DataGridViewWidget[T, K]) copyRows(rows []int) string {
	mu.Lock()
	if fn := t.copyFn; fn != nil {
		// The keys are read under mu, in a closure so that a panic of key
		// unlocks it; fn runs off the lock.
		keys := func() []K {
			defer mu.Unlock()
			keys := make([]K, 0, len(rows))
			for _, row := range rows {
				if r := t.rowAt(row); r != nil {
					keys = append(keys, t.key(r))
				}
			}
			return keys
		}()
		return fn(keys)
	}
	defer mu.Unlock()
	var lines []string
	for _, row := range rows {
		r := t.rowAt(row)
		if r == nil {
			continue
		}
		cells := make([]string, len(t.visCols))
		for j, i := range t.visCols {
			cells[j] = t.cols[i].Text(r)
		}
		lines = append(lines, strings.Join(cells, "\t"))
	}
	return strings.Join(lines, "\n")
}

// handOver gives ui the layout set and the row menu, before the table
// builds. Under mu.
func (t *DataGridViewWidget[T, K]) handOver() {
	s := &t.list
	if t.layout != nil {
		s.Columns, t.layout = *t.layout, nil
	}
	s.RowMenu = nil
	if t.menuFunc != nil {
		s.RowMenu = func(m *ui.Menu, rows []int) { t.lateMenu(m, rows) }
	} else if menu := t.menu; len(menu) > 0 {
		s.RowMenu = func(m *ui.Menu, _ []int) { buildMenu(m, menu) }
	}
}

// events adds the table's handlers to e, the ui table. Under mu.
func (t *DataGridViewWidget[T, K]) events(e ui.Element) {
	s := &t.list
	if fn := t.onActivate; fn != nil {
		e.OnActivate(func(row int) {
			k, ok := t.keyAt(row).(K)
			if !ok { // a header
				return
			}
			Post(func() { fn(k) })
		})
	}
	if fn := t.onColumns; fn != nil {
		e.OnColumnsChange(func() {
			l := s.Columns
			Post(func() { fn(l) })
		})
	}
	for _, sc := range t.shortcuts {
		fn := sc.fn
		e.OnShortcut(sc.mods, sc.key, func() { Post(fn) })
	}
}
