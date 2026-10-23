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

// copyRows is ui's ListState.Copy: the rows shown at rows, a line each,
// their cells apart by tabs. On the main thread, without mu.
func (t *DataGridViewWidget[T, K]) copyRows(rows []int) string {
	mu.Lock()
	defer mu.Unlock()
	var lines []string
	for _, row := range rows {
		r := t.rowAt(row)
		if r == nil {
			continue
		}
		cells := make([]string, len(t.cols))
		for j := range t.cols {
			cells[j] = t.cols[j].Text(r)
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
	if menu := t.menu; len(menu) > 0 {
		s.RowMenu = func(m *ui.Menu, _ []int) { buildMenu(m, menu) }
	}
}

// events adds the table's handlers to e, the ui table. Under mu.
func (t *DataGridViewWidget[T, K]) events(e ui.Element) {
	s := &t.list
	if fn := t.onActivate; fn != nil {
		e.OnActivate(func(row int) {
			k := t.keyAt(row).(K)
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
