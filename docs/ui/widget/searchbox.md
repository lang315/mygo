# SearchBox

`SearchBox()` is a field for searching, as DevExpress's `SearchControl`: a
magnifying glass, the text, and a button clearing it while it holds any, as
Escape does then. It shows Search while empty, or what `SetPlaceholder` sets.
It is [`ui.SearchField`](../search-field.md) underneath, and takes the skin's
look of a `TextBox`.

```go
var search *widget.SearchBoxWidget
search = widget.SearchBox().SetPlaceholder("Find a customer").
	OnChange(func() { list.Filter(search.Text()) })
```

- `Text()` and `SetText(s)`; `SetText` does not run `OnChange`.
- `OnChange(fn)` runs on the event goroutine after each edit of the user,
  the clear button and Escape included.
- `OnSubmit(fn)` runs on the event goroutine when the user presses Enter.
- It can be bound to a string: `widget.Bind(search, get, set)`. See
  [Binding](../widget.md#binding).

Assistive technology sees a text field whose placeholder is the
placeholder; in a test, `tt.Find(widget.ByRole(ui.RoleTextField))` types into
it (see [`widget.Tester`](tester.md)).
