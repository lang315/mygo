# ListView

`ListView(key, columns...)` shows rows in the view the app or the user picks,
as WinForms' `ListView` does: in columns (`ViewDetails`), as large icons or small
icons in a grid, or as a list. The rows, their choice and their checks are the
same in every view, told apart by a key you choose, so switching the view at run
time keeps what was chosen and checked.

```go
type file struct{ path, name, kind string }

view := widget.ListView(
	func(f *file) string { return f.path },
	widget.Column[file]{Title: "Name", Text: func(f *file) string { return f.name }, Width: 240, Sortable: true, Filterable: true},
	widget.Column[file]{Title: "Kind", Text: func(f *file) string { return f.kind }, Width: 100},
).SetRows(files).CheckBoxes().
	SetImage(func(f *file) *ui.Bitmap { return icons[f.kind] })

view.OnSelect(func(selected []string) {
	status.SetText(fmt.Sprint(len(selected), " chosen"))
}).OnActivate(func(path string) {
	open(path)
})

viewMode.OnChange(func() { view.SetView(widget.ViewLargeIcon) })

view.Details().Search("Name", "report") // filters every view
```

## Views

| `ViewMode` | Shows | Built on |
|---|---|---|
| `ViewDetails` | the columns, which the user sorts, resizes and moves | the [DataGridView](../widget.md#datagridview) underneath |
| `ViewLargeIcon` | an icon (48 DIPs by default, `SetIconSize`) over the text, in a grid | [`ui.GridView`](../gridview.md) |
| `ViewSmallIcon` | a 16 DIPs icon before the text, in a grid | `ui.GridView` |
| `ViewList` | a 16 DIPs icon before the text, in one column | `ui.GridView` of one column |

`ViewDetails` is the zero value, so it is the view of a list view that never
called `SetView`. WinForms' default is large icons: call `SetView` for that.
`ViewList` is one column; WinForms' `List` flows its items into several columns
top to bottom, which this does not.

The rows chosen show in the accent in every view, as a table's do. The text of a
row in the icon views and the list is the text of the first column. `SetImage(fn)` gives the icon of a row (nil for none); without it the
first column's `Image` is used. `SetView` fires no event.

## Details

`Details()` returns the `DataGridView` underneath. Its options apply in every
view, since the other views show its rows: its sort and its `Search` decide the
order and which rows show, the icons follow them, and a change keeps the choice
and the checks by key, with the keys moving on from the row chosen. Use it for
`SetColumnsLayout`, `OnColumnsChange`, `BottomPanel`, `ClearSort`, `Search` and
the rest. Its `OnKey` keys work in every view, and so does its bottom panel, under
the icons, with the counts and the search box: a search that hides rows shows in
the panel in every view. (`SetColumnsLayout` and `OnColumnsChange` are about the
columns, which only Details shows.) Do not add it to a window yourself: the list
view shows it.

## Rows, choosing and checks

- `SetRows`, `AddRow`, `AddRows`, `UpdateRow`, `DeleteRows`, `Clear`, `Row` and
  `Len` are the `DataGridView`'s; `ListViewOf(src, columns...)` shares a
  `RowSource` with other tables.
- Rows are chosen as a table's are: a click, Cmd-click (Ctrl-click on Linux and
  Windows), Shift-click, and the keys, with Cmd+A for all. In the icon views the
  arrows move in two dimensions.
- `Selected()` returns the keys of the rows chosen that show, in the order shown;
  `SetSelected(ks...)` chooses rows, and the last is where the keys move from.
- `OnSelect(fn)` runs `fn` with the keys chosen, on the event goroutine, after
  the user changes the choice in any view (WinForms' `SelectedIndexChanged`);
  `OnActivate(fn)` after a double click or Enter.
- `CheckBoxes()` shows a check box on each row in every view: a column in
  Details, a box on the item in the others. `Checked()`, `IsChecked(k)`,
  `SetChecked(ks, on)` are the `DataGridView`'s. `OnCheck(fn)` is the list
  view's own (WinForms' `ItemCheck`; the `DataGridView` has none): it runs for each
  row the user checks or unchecks, with Space, a box, or the header's "Check all" in
  Details. Space checks the rows chosen, or unchecks them when all are checked.
- Setters, and switching the view, fire no event. A deleted row leaves the
  choice and the checks without one, and a row added again starts clean. That
  differs from WinForms, whose `SelectedIndexChanged` runs when the row chosen is
  removed: here `DeleteRows` is the program's change, and `OnSelect` is the user's.
- The cursor (the row the keys move from), the row Shift extends from, and the
  keyboard focus cross a switch of the view by key: after switching, the arrows
  go on from the row that was chosen, and Shift-click extends from it.
- `ContextMenu`, `ContextMenuFunc` and `CopyFunc` work in every view, as the
  `DataGridView`'s do.

## Size

The choice and the checks are the table's sets of keys, so a click costs what the
items in view cost, not the number of rows, in Details and in the icon views alike.
The test of `ListView` clicks in 200,000 rows and checks that a click allocates what
it does in 2,000.

Not yet there: groups, which need section headers in the icon views.

## Accessibility

Assistive technology sees a table in Details, and a list of items, each named by
its text, in the other views.
