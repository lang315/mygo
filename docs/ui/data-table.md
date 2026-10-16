# Data table

`ui.DataTable` shows an app's rows, a slice of any type, in a table with a
checkbox column, a search bar and, if you ask, a `#` column. It keeps what an
app otherwise keeps beside its rows to do that: their order (a
[TableView](table.md)), the rows checked and chosen, the sort and the search.
The rows stay yours: you give the slice at each frame, and `DataTable` never
copies it.

```go
type app struct {
	accounts []account
	table    ui.DataTableState[account, int]
}

func (a *app) init() {
	a.table.Key = func(x *account) int { return x.ID }
	a.table.Index = true
	a.table.Columns = []ui.DataColumn[account]{
		{
			TableColumn: ui.TableColumn{Title: "UID", Width: 160},
			Text:        func(x *account) string { return x.UID },
			Compare:     func(a, b *account) int { return cmp.Compare(a.UID, b.UID) },
		},
		{TableColumn: ui.TableColumn{Title: "Status"}, Text: func(x *account) string { return x.Status }},
	}
}

func (a *app) build(c *ui.Context) {
	ui.DataTable(c, &a.table, a.accounts).Grow(1)
}
```

## The state

Set `Key` and `Columns` before the first frame; they cannot change after it
(a different number of columns panics). Do not copy the state after the first
frame, and use it on the main thread.

- `Key` is a row's identity. Checked and chosen rows are kept by it, so they
  survive a sort, a search and changes of the rows, also while a search hides
  them.
- A `DataColumn` is a `TableColumn` with `Text`, the cell's text, which the
  search and Copy read too. `Compare` makes the column sort, as a
  `TableView`'s comparators do; `Cell` builds a cell that is more than `Text`
  in one line; `NoSearch` leaves the column out of the search.
- `Sort` is the order shown; a click on a header changes it, and you can set
  it to start sorted.
- `Checked` and `Chosen` are `Selection`s of keys: the rows checked with the
  checkboxes or Space, and the rows highlighted. `Current()` is the row
  chosen last, `Len()` and `At(i)` the rows shown. `Version()` changes
  whenever the rows shown, or their order, do.
- `NoSearchBar` hides the search bar, for an app that puts its own where it
  wants, with its own texts, and calls `SetSearch`. No bar is shown either when
  no column is searchable.
- `Index` shows a `#` column after the checkbox column: the position of each
  row shown, so it follows the sort and the search. It does not sort, move or
  resize, and Copy leaves it out. Its ID is `ui.DataTable.index`, so a column
  titled `#` of yours does not collide with it.

## Changing the rows

Append to your slice and the next frame places the new rows. When you change a
row, say so with `Changed(i)`, the index in your slice; when you remove or
reorder rows, call `Reset()`. Both cost what the matching `TableView` calls
cost.

## Search

The bar over the table has a column to search (the first of those without
`NoSearch`), a box and a button. The search runs on Enter, the button, clearing
the box or choosing another column, not at each key: it goes through all the
rows. Text matches the cells containing it, case aside; a keyword with `|`
lists the values to match exactly (`UID1|UID2`). Of columns with the same
`Title`, the first is searched.

`SetSearch(column, keyword)` searches from code, and `Search()` returns the
search applied; with `NoSearchBar` they are all an app's own bar needs.

For rows to show apart from the search, such as the folder of a file manager,
set `Filter` and a `FilterKey` that changes when the filter does; the rows are
filtered again only then. A `FilterKey` that cannot be compared (a slice, a
map, a func) panics at the first frame.

## Menu and Copy

`RowMenu` fills a [context menu](context-menu.md) for the rows chosen, in the
order shown, and `Copy` returns the text of them for Cmd+C (Ctrl+C on Linux
and Windows). Both get pointers into your slice, valid during the call. By
default Copy is the `Text` of each column but `#`, tab-separated, a line a
row.
