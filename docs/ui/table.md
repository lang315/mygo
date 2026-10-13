# Table

`ui.Table` puts a [list](list.md)'s rows in columns, under a header of
`ui.TableColumn`s, each with a title, a width (or 0 to share the room the
others leave) and an alignment; `cell` builds the content of a row's
column, usually a `Text`. It builds only the rows in view, as a list does,
and rows are as high as their tallest cell, so text that wraps makes its
row taller.

```go
cols := []ui.TableColumn{{Title: "Name"}, {Title: "Size", Width: 90, Align: ui.End}}
app.files.Selected = &app.file
if ui.Table(c, &app.files, cols, len(files), func(row, col int) {
	switch col {
	case 0:
		ui.Text(c, files[row].Name).SingleLine()
	case 1:
		ui.Text(c, files[row].Size())
	}
}).Grow(1).Submitted() {
	app.open(files[app.file])
}
```

## Choosing rows

The table takes the same `ListState` as a list, or `nil`, and takes the
keyboard focus for it: with `Selected`, a click or the keys choose a row,
with `Selection` several, and `Submitted` reports a double click or Enter;
`Key`, `Label`, `Reorder` and the rest work as for a list. A row `Header`
names spans every column, which `cell` builds as column 0, and stays at the
top while its section's rows scroll under it.

Chosen and hovered rows are square from edge to edge, as in Finder's list
view; `RoundRows` rounds their corners at the theme's `Radius`, as a list's
rows are, for an app whose other lists are rounded.

## Arranging columns

The user arranges the columns, as in Finder:

- **Resizing.** Dragging the right edge of a column's header resizes the
  column, between its `MinWidth` and `MaxWidth`, and a double click there
  fits it to its cells. The columns before it that share the room left keep
  their widths, for the edge to follow the pointer.
- **Moving.** Dragging a header moves its column among the others.
- **Scrolling sideways.** Columns wider than the table scroll sideways, the
  header with them.
- **Fixed columns.** A `Fixed` column stays where it is, as wide as it is.

`ListState.Columns` keeps the order and the widths, by the columns' `ID`s
(their titles by default): set it from saved settings to restore them, and
save it as it changes.

## Sorting

With `ListState.Sort`, a click on the header of a `Sortable` column sorts
the rows by it, ascending, and a second click reverses the order: the header
shows an arrow, assistive technology reads the order, and the table's
`Changed` reports it. The table shows the rows in the order `cell` gets
them, so sort them as it says, keyed by `Key` for the choice to follow its
rows:

```go
cols := []ui.TableColumn{{Title: "Name", Sortable: true}, {Title: "Size", Width: 90, Align: ui.End, Sortable: true}}
app.files.Sort = &app.sort
sorted := sortFiles(app.files, app.sort) // by app.sort.Column, Descending or not
app.files.Key = func(i int) any { return sorted[i].ID }
```

## Renaming in place

An [editable text](editable-text.md) in a cell is renamed in place, as
Finder's file names: Return on macOS and F2 elsewhere edit the chosen row's
text, as does a click on it once the row is chosen.

```go
ui.Table(c, &app.files, cols, len(files), func(row, col int) {
	if col == 0 && ui.EditableText(c, &files[row].Name).Changed() {
		app.rename(files[row])
	}
})
```

## Checking rows

`ListState.Checked`, a `*ui.Selection[K]`, adds a column of checkboxes before
the others, apart from the rows chosen: a checkbox checks its row by the
row's `Key` (the index without one). The column is the table's own, so `cell`
still gets your columns as 0, 1, ... and the user cannot move or resize it.
The header's checkbox shows whether none, some or all of the rows shown are
checked, and a click checks all of them, or unchecks all of them when all are.
Assistive technology reads it as "Check all".

```go
app.files.Checked = &app.checked
if ui.Table(c, &app.files, cols, len(files), cell).Grow(1).CheckedChanged() {
	app.updateToolbar()
}
```

Rows not shown keep their keys, so rows a filter hides stay checked, and the
header counts and changes only the rows shown. An app that wants only the
rows shown checked clears the selection when the filter changes, with
`app.checked.Clear()` rather than by assigning a new `Selection`: the header
caches its count by the selection's changes.

`Checked.All()` includes rows that are hidden, so run an action on the rows
checked in the order shown by looping over them:

```go
for i := range app.view.Len() {
	a := &app.accounts[app.view.Row(i)]
	if app.checked.Has(a.ID) {
		run(a)
	}
}
```

Keys must be unique. The keys of rows removed stay in the selection until
you take them out.

## Row menus and copying

`ListState.RowMenu` gives the rows a [context menu](context-menu.md). It
fills the menu for `rows`, the rows chosen in the order shown. A right click
(Ctrl-click on macOS) on a row not chosen chooses it first, and a table that
does not choose rows gets the row clicked alone. The Menu key, Shift+F10 and
a right click on a section header, on the column header or below the rows
open the menu on the rows chosen, and nothing opens when none is chosen.

```go
app.files.RowMenu = func(m *ui.Menu, rows []int) {
	if m.Item("Open").Chosen() {
		app.open(rows)
	}
}
```

`ListState.Copy` returns the text for the same `rows`: it goes to the
clipboard on Cmd+C (Ctrl+C on Linux and Windows), or Copy in the Edit menu,
while the table has the keyboard focus. Nothing is copied when no row is
chosen, and a text being edited in a cell copies its own. Separate columns
with tabs, for a spreadsheet to take them in cells:

```go
app.files.Copy = func(rows []int) string {
	lines := make([]string, len(rows))
	for i, row := range rows {
		f := files[row]
		lines[i] = f.Name + "\t" + f.Size()
	}
	return strings.Join(lines, "\n")
}
```

## Events

A table reports what the user did with separate queries, each with a
callback that runs after input, before the view is built again:

| Query | Callback | Reports |
|---|---|---|
| `SelectionChanged()` | `OnSelectionChange(func())` | the rows chosen changed |
| `SortChanged()` | `OnSortChange(func())` | a click on a header changed `Sort` |
| `ColumnsChanged()` | `OnColumnsChange(func())` | a column was resized, moved or fitted: save `Columns` |
| `CheckedChanged()` | `OnCheckedChange(func())` | a checkbox changed `Checked` |
| `Activated()` (`row, ok`) | `OnActivate(func(row int))` | a double click on a row; Enter on the chosen row, with `Selected` or `Selection` |

```go
t := ui.Table(c, &app.files, cols, len(files), cell).Grow(1)
if t.SortChanged() {
	app.resort()
}
if t.ColumnsChanged() {
	app.save(app.files.Columns)
}
if row, ok := t.Activated(); ok {
	app.open(row)
}
```

A column fitted by a double click on its header's edge reports
`ColumnsChanged` in the frame after the fit. `Changed` still reports a new
choice or sort, and `Submitted` a double click or Enter, as before.

## Many rows

A table builds only the rows in view, but sorting and filtering 100,000 rows
in every frame would not do. A `ui.TableView[T]` keeps the order of your rows
between frames and updates it for the rows that changed. You keep the rows;
the view only says which row to show at each place. `NewTableView` takes a
comparator for each sortable column, by its `ID` (its title by default); a
column without one shows the rows in the slice's order. Use the view on the
main thread only.

Call `Sync` in every frame: it costs nothing when nothing changed, places
again only the rows changed or appended, and sorts all of them when the sort,
the filter or the slice changed. Give the table the view's `Len`, and read
rows through `Row`:

```go
app.view = ui.NewTableView(map[string]func(a, b Account) int{
	"Name":   func(a, b Account) int { return strings.Compare(a.Name, b.Name) },
	"Status": func(a, b Account) int { return cmp.Compare(a.Status, b.Status) },
})

app.view.Sync(app.accounts, app.sort)
app.list.Sort = &app.sort
app.list.Key = func(i int) any { return app.accounts[app.view.Row(i)].ID }
app.list.Version = app.view.Version()
ui.Table(c, &app.list, cols, app.view.Len(), func(r, col int) {
	a := &app.accounts[app.view.Row(r)]
	// ...
}).Grow(1)
```

A click on a header changes the sort, and `Sync` sorts again in the same
frame. Rows of equal value keep the slice's order.

`SetFilter` shows only the rows its function keeps. It places all rows again
whenever it is called, as functions cannot be compared, so call it when what
the filter keeps changes, not in every frame:

```go
if app.keyword != app.filtered {
	app.filtered = app.keyword
	kw := app.keyword
	app.view.SetFilter(func(a Account) bool { return strings.Contains(a.Name, kw) })
	app.checked.Clear()
}
```

After you change a row in place, tell the view with `Changed(index...)` for it
to place the row again. Rows appended to the slice need not be named. Call
`Reset` after you remove or move rows: `Sync` sees a shorter slice or another
one, but not rows removed and others appended back to the same length.

`ListState.Version` tells the table its rows changed order or membership
while their number stayed the same, as the view's `Version` does. The choice,
the Shift pivot and the focused row then follow their keys across all the
rows, wherever the sort sent them, and the table keeps its place by index.
Each change of `Version` costs a pass over all the rows' keys (the search for
the choice and the header checkbox's count), so it must change when the rows
do, never in every frame: the view's does.

Workers should not touch the rows or the view. Have them collect results
under a mutex, and apply a batch of them with `Window.Update` a few times a
second, for one redraw each:

```go
// In a goroutine: results come in, the view hears of them ten times a second.
for range time.Tick(100 * time.Millisecond) {
	batch := app.takeResults() // under the app's mutex
	win.Update(func() {
		for _, r := range batch {
			app.accounts[r.index].Status = r.status
			app.view.Changed(r.index)
		}
	})
}
```

## Accessibility

Assistive technology sees a table, named by its `Label`, of rows named by
the text inside them, each saying which of all the rows it is, with column
headers that say the order the rows are sorted in.
The header's checkbox is a plain cell, not a column header, so that
assistive technology finds "Check all".

For a tree of rows in columns, see [Outline](outline.md#in-columns).
