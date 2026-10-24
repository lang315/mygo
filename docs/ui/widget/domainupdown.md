# DomainUpDown

`DomainUpDown(items...)` chooses one of a list of texts with a pair of arrows,
as WinForms' `DomainUpDown`: a field shows the selected item, and Up goes to
the item **before** it, Down to the next one, as the list reads from top to
bottom. The arrows are a [`ui.Stepper`](../stepper.md): a click, Up and Down
while it has the keyboard focus, and holding an arrow keeps stepping.

```go
var day *widget.DomainUpDownWidget
day = widget.DomainUpDown("Mon", "Tue", "Wed", "Thu", "Fri").
	SetWrap(true).
	OnChange(func() { plan(day.SelectedItem()) })
```

- `SelectedIndex()` (-1 when there are no items) and `SelectedItem()`.
- `SetSelectedIndex(i)` clamps to the list; `SetSelectedItem(s)` ignores a text
  that is not an item; neither runs `OnChange`.
- `SetItems(items...)` keeps the selection if the item at its index is the same,
  else follows the selected item if it is still there, else selects the first.
- `SetWrap(true)`: Up at the first item leads to the last, Down at the last to
  the first. Without it the arrows stop at the ends.
- `OnChange(fn)` runs on the event goroutine after the user chose another item.
- It can be bound to the selected text: `widget.Bind(day, get, set)`; see
  [Binding](../widget.md#binding). A text that is not an item is ignored.

There is no typing into the field: WinForms' `ReadOnly = false` (where typing
selects the item that starts with the text) is not supported.

Assistive technology sees a spin button whose value is the selected item's text (its numeric range is the internal, reversed position, which is not meant to be read). In a test,
`tt.Find(widget.ByRole(ui.RoleStepper)).Key(0, ui.KeyDown)` steps it
(see [`widget.Tester`](tester.md)).
