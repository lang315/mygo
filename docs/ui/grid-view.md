# Grid view

<!-- preview grid-view 310 -->

`ui.GridView` shows items in a grid, as Photos' library and Finder's icon
view: as many columns as fit items at least as wide as asked, sharing the
width left, in rows of a height, built only as they show, so grids of
millions of items are as fast as small ones. `Gap` spaces the items.

```go
app.photos.Selected = &app.photo
ui.GridView(c, &app.photos, len(photos), 140, 120, func(i int) {
	ui.Image(c, thumbs[i]).Fit(ui.Contain).Grow(1)
	ui.Text(c, photos[i].Name).SingleLine()
}).Grow(1)
```

## Grid state

A `GridState` keeps the grid's place and says how items are chosen, as a
`ListState` does rows, or `nil`:

- `Selected` lets a click choose an item, as the arrows do in both
  directions while the grid has the keyboard focus, with Home, End, and
  Enter (`Submitted`).
- `Selection` lets the user choose several, by `Key`: Cmd-click adds an
  item or takes it out, Shift-click and Shift with the arrows choose those
  from the item last chosen, in place of the others (with Cmd held for the
  click, besides them), and Cmd+A chooses all.
  Shift with Home, End, Page Up and Page Down extends the choice to the first
  or last item, or by the rows in view (Page Up and Page Down alone scroll),
  and Cmd+Shift with the arrows and those keys (Ctrl+Shift on Linux and
  Windows) extends it besides the items chosen before; going back takes out the
  items it added and left, and those chosen before stay. With nothing chosen,
  Shift with Home or End chooses the first item, as the arrows do. Page Up and
  Page Down only scroll on macOS, as in a list.
  `Pivot()` returns the item Shift extends from, -1 for none (as at the start:
  a Shift-click as the first click chooses that item alone), and
  `SetPivot(item)` sets it without changing the choice, for an app that shows
  the same items in another view and keeps the pivot by key; an item out of
  range is ignored by Shift-click.
- `AccentChoice` shows the items chosen in the accent with its text, focused
  or not, as a table's rows are, instead of the tint (a photo library's) or,
  unfocused, a grey.
- `Label` returns an item's text: typing its first letters chooses it, and
  assistive technology reads it as the item's name.
- `Header` reports the items that head a group, as `ListState.Header` does
  rows: a header is an item that takes a row of its own, at the width of the
  grid, built by the same item function, and the items after it start a row,
  the last row of a group being short. Headers are not chosen or checked: the
  arrows, Home, End, Page Up and Page Down pass them by (Up and Down keep the
  column inside a group and go on to the next, to the last item of a shorter
  row), Shift and Cmd+Shift extend over them, and Cmd+A and typing leave them
  out. They are not dragged by `Reorder`; items dropped on the upper half of one
  go before it, on the lower half after it (first in its group), the line
  showing it across the header.
  Assistive technology sees a header as an item of the list, named by `Label`,
  counted in the position and the size of the set. Give headers a background;
  they do not stay at the top while their group scrolls.

  The grid asks `Header` for every item when the number of items changes, and
  again when a header it knew is none, so a header that moves is found; it is
  one call per item (about 2.5 ms for a million). Resizing does not ask. Call
  `ReloadHeaders()` when an item that was not a header becomes one and the
  number of items stays, as when a group is split without adding a header.
  Each frame asks only for the headers it knew.

  An app that knows where its headers are says so with `SetHeaders(items, n)`,
  and the grid asks `Header` for none of them (not for every item, nor for the
  headers it knew in each frame) until the number of items is not `n`,
  `ReloadHeaders()` is called, or `SetHeaders` is called again: call it whenever
  a header moves. Items must be in ascending order, each less than `n`. This
  spares the layout's scan only: `Header` is still required and answers for the
  items the grid builds, and the keys, the choice and drag still ask it per
  item. A list view with groups does, so a million rows with a few groups cost
  the groups.
- `Reorder` lets the user drag items to another place, as a list's rows:
  see [drag and drop](drag-and-drop.md).
- `Checked` shows a checkbox on each item, by `Key`, apart from the items
  chosen, as a table's `Checked` does: it shows on an item checked, on the
  one under the pointer, and on all of them while any is checked. A click on
  it checks the item and leaves the choice, Space checks the items chosen
  (or unchecks them when all are checked) instead of scrolling a page, and
  `CheckedChanged` reports a change by the user. Assistive technology reads
  it as "Check" and the item's `Label`. There is no "check all": Cmd+A, then
  Space.
  `PlainChecks` draws the boxes without the panel behind them that keeps them
  readable over a picture, for items that have none.
- `ItemMenu` gives the items a [context menu](context-menu.md), filled for
  `items`, the items chosen in order. A right click on an item not chosen
  chooses it first, and a grid that does not choose gets the item clicked.
  The Menu key, Shift+F10 and a right click between the items open it on the
  items chosen, and nothing opens when none is.
- `Copy` returns the text for the same `items`: it goes to the clipboard on
  Cmd+C (Ctrl+C on Linux and Windows), or Copy in the Edit menu, while the
  grid has the keyboard focus. Nothing is copied when no item is chosen,
  and `""` leaves the clipboard as it was.

```go
app.photos.Checked = &app.checked
if ui.GridView(c, &app.photos, len(photos), 140, 120, item).Grow(1).CheckedChanged() {
	app.updateToolbar()
}

app.photos.ItemMenu = func(m *ui.Menu, items []int) {
	if m.Item("Delete").Chosen() {
		app.delete(items)
	}
}
app.photos.Copy = func(items []int) string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = photos[item].Name
	}
	return strings.Join(names, "\n")
}
```

## Accessibility

Assistive technology reads the grid as a list of items, each saying which
of all it is, built or not.
