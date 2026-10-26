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
| `ViewLargeIcon` | an icon (48 DIPs by default, `SetIconSize`) over the text, in a grid | [`ui.GridView`](../grid-view.md) |
| `ViewSmallIcon` | a 16 DIPs icon before the text, in a grid | `ui.GridView` |
| `ViewList` | a 16 DIPs icon before the text, in one column | `ui.GridView` of one column |

`ViewDetails` is the zero value, so it is the view of a list view that never
called `SetView`. WinForms' default is large icons: call `SetView` for that.
`ViewList` is one column; WinForms' `List` flows its items into several columns
top to bottom, which this does not.

The rows chosen show in the accent in every view, as a table's do. The text of a
row in the icon views and the list is the text of the first column.

`SetImage(fn)` gives the icon of a row (nil for none), in every view; without it
the first column's `Image` is used. `SetImageList(il, key)` takes the image of an
[`ImageList`](imagelist.md) under `key(row)`, drawn at the list's size in the
icon views and the list (Details draws 16 DIPs). The check box of a small item
is in the row of its text; the large icons' is at the corner of the picture, on a
panel that keeps it readable when the item has a picture. `SetView` fires no
event.

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
  `RowSource.AddRowsInBatches` loads a big slice in batches and reports
  progress for a `ProgressBar` or `StatusProgress` (see `docs/ui/widget.md`).
- Rows are chosen as a table's are: a click, Cmd-click (Ctrl-click on Linux and
  Windows), Shift-click, and the keys, with Cmd+A for all. In the icon views the
  arrows move in two dimensions.
- Ctrl+Shift-click (Cmd+Shift on macOS) adds the range from the row last chosen
  to those chosen; a Shift-click alone takes their place.
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

## Groups

`SetGroupBy(fn)` groups the rows of every view, Details too, as a WinForms
`ListView`'s `ListViewGroup`s: `fn` returns the title of a row's group, and each
group shows under a header "Title (n)", which is not chosen. In Details a header
spans the columns and stays at the top while the rows of its group scroll under
it. The rows of a group keep the
order of the table, so its sort and its `Search` apply inside the groups, and a
group whose rows are all hidden by a search shows no header. `SetGroupOrder`
names the groups in the order to show them; a group it does not name comes after
them, by title as a text column of the table sorts, so sorting the table does
not move the groups. `Groups()` returns the groups the last
frame showed, with their counts. `SetGroupBy(nil)` shows no groups.

```go
view.SetGroupBy(func(f *file) string { return f.kind }).
	SetGroupOrder("Document", "Image")

for _, g := range view.Groups() {
	fmt.Println(g.Title, g.Count) // "Document 3", then "Image 3", then the others
}
```

In Details the table's view does the grouping (`ui.TableView.SetGroupBy`, see
[Table](../table.md#groups)): the group is the first key the rows sort by, `fn`
is asked once for each row when the rows are placed and for each row changed in
place, and the headers cost no pass over the rows. The icon views make their
groups themselves, with one call of `fn` for each row shown and a pass over the
rows, about 2.5 ms for 100,000 rows, again whenever a row changes in place
(`UpdateRow` and the like), not only when the rows or their order change.

**Order of the keys.** With groups, `Selected()` and `Checked()` return the keys
in the order shown, which is grouped in every view: the rows of the first group,
then those of the second, and so on. Before `SetGroupBy`, and in a list view
without it, that is the table's order. Adding `SetGroupBy` to an existing list
view therefore changes the order of what these return, in Details too.

The arrows, Home, End, Shift and Cmd+A pass the headers by, a click on one
chooses nothing, and the choice, the cursor and the row Shift extends from still
cross a switch of the view by key. `fn` runs under the package's lock, once for
each row shown when the rows, their order or the groups change, not in every
frame, and calls no widget method.

## Accessibility

Assistive technology sees a table in Details, and a list of items, each named by
its text, in the other views; a group's header is a row (an item in the icon
views) named "Title (n)", counted in the position and the size of the set, and
not chosen.
