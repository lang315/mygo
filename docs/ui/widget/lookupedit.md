# LookUpEdit

`LookUpEdit(key, text, cols...)` is a field that picks one record of a list
by its key. Its drop-down shows the records in columns under a header, and
typing in the field filters them; the field shows the chosen record's text.
It is DevExpress's LookUpEdit: a data-entry form picks a customer, a product
or an account while the user reads several of its fields.

```go
type Customer struct {
	ID               int
	Code, Name, City string
}

customer := widget.LookUpEdit(
	func(c *Customer) int { return c.ID },
	func(c *Customer) string { return c.Name },
	widget.Column[Customer]{Title: "Code", Text: func(c *Customer) string { return c.Code }, Width: 60},
	widget.Column[Customer]{Title: "Name", Text: func(c *Customer) string { return c.Name }},
	widget.Column[Customer]{Title: "City", Text: func(c *Customer) string { return c.City }, Width: 80},
).SetItems(customers).
	SetSelected(order.CustomerID).
	OnSelect(func(id int, ok bool) { order.SetCustomer(id, ok) })

form := widget.Fields().Row("&Customer", customer)
```

- `key` names a record and `text` is what the field shows once it is
  chosen. Records are of any type `T`, keys of any comparable type `K`, as a
  `ListBox`'s and a `DataGridView`'s.
- The columns are a `DataGridView`'s `Column[T]`, of which the LookUpEdit
  reads `Title`, `Text`, `Width`, `MinWidth`, `Align` and `Hidden`; it
  ignores the others. A column of `Width` 0 shares the room left, at least
  its `MinWidth` (or the grid's least, 15 theme units); widths are DIPs times the
  text size the user chose. Every row and the header take the same widths,
  so the cells line up, and a long text clips in its cell. The header shows
  the titles in semibold, only when a column has a title. A `Hidden` column
  neither shows nor filters. With no column the drop-down shows `text`
  alone, with no header.
- The drop-down is as wide as its columns need, and at least as wide as the
  field; of many records it scrolls. A click in the field or Down opens it
  on every record; typing opens it on those of which `text` or a shown
  column's `Text` holds what was typed, whatever its case, those whose `text`
  starts with it first. When none does, it says "No results".
- Up and Down move among the records, Enter or a click chooses one, and
  Escape closes the drop-down. Choosing a record puts its `text` in the
  field and runs `OnSelect(fn)` with its key and `true`, on the event
  goroutine, once per change: choosing the chosen record again runs
  nothing. Enter with the drop-down closed clicks the form's accept button
  (`SetAcceptButton`), as in a `TextBox`.
- A free text is never a value: leaving the field with a text that is not
  the chosen record's puts that record's text back (none: empty). Leaving it
  emptied chooses none, and runs `OnSelect` with the zero key and `false` if
  a record was chosen.
- `Selected()` returns the chosen record's key and whether one is.
  `SetSelected(k)` chooses the record of key `k`, and `Unselect()` none,
  without running `OnSelect`, as every setter; a key that is not a record
  is ignored. `SetItems(items)` replaces the records, copying the slice, and
  keeps the choice while its key is still a record (showing its new text),
  choosing none otherwise.
- The field takes the `Field` look of the skin, as a `TextBox` and an
  editable `ComboBox`; the drop-down the theme's popup surface, border and
  shadow, the record the arrows are on the theme's `RowSelected`.
  `Focus()`, and the label of a `Fields` row with its access key
  (`Row("&Customer", customer)`), reach the field's input, which assistive
  technology sees as a combo box whose options match what was typed.

## Search panel

`SetSearchPanel(true)` gives the drop-down a search box over the records, as
DevExpress's SearchLookUpEdit, for lists too long to scan by eye. It is off
by default, and a LookUpEdit without it behaves as described above.

```go
product := widget.LookUpEdit(
	func(p *Product) int { return p.ID },
	func(p *Product) string { return p.Name },
	widget.Column[Product]{Title: "Code", Text: func(p *Product) string { return p.Code }, Width: 70},
	widget.Column[Product]{Title: "Name", Text: func(p *Product) string { return p.Name }},
).SetItems(products).SetSearchPanel(true)
```

- The field takes no typing: it shows the chosen record's `text`, as a
  `ComboBox` does. A click, Enter, Space, Down or Alt+Down opens the
  drop-down with the focus in its search box; a click on the field closes it.
- A record shows when a shown column's `Text` (`text`, with no column) holds
  what is typed in the search box, whatever its case, as the
  `DataGridView`'s search does, in the records' order. A `Hidden` column
  does not match. When none does, it says "No results".
- Up and Down in the search box move the highlighted record, from none when
  the drop-down opens or the search changes; Enter or a click chooses it and
  closes the drop-down, running `OnSelect` as above. Escape or a click
  outside closes it, the choice as it was. The highlighted record is the
  search box's active descendant, so assistive technology reads it as the
  arrows move, with the focus staying in the search box.
- The list builds only the records in view (ten rows high), with ui's
  `List`: 10,000 records open and search as fast as ten.

`text` and the columns' functions run under the package's lock, as a
`DataGridView`'s: they read the record and call no widget method. The
LookUpEdit is opt-in: an app that makes none does not link it, and the
search panel's code runs only once `SetSearchPanel(true)` is called. A grid
in the drop-down, with sorting (DevExpress's GridLookUpEdit), is not part of
it: use a `DataGridView` in a dialog for that.
