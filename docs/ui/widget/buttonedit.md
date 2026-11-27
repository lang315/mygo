# ButtonEdit

`ButtonEdit(buttons...)` is a one-line text field with buttons at its end,
inside its frame, as DevExpress's `ButtonEdit`. Each `EditButton` has a
`Kind`, which says what it shows, an optional `Tooltip` and an `OnClick`,
posted to the event goroutine as a `Button`'s is.

```go
var path *widget.ButtonEditWidget
path = widget.ButtonEdit(
	widget.EditButton{Kind: widget.EditClear},
	widget.EditButton{Kind: widget.EditEllipsis, Tooltip: "Browse", OnClick: func() {
		path.SetText(chooseFile()) // the app's own dialog
	}},
).SetPlaceholder("Choose a file")
```

The kinds:

| Kind | Shows | Does |
|---|---|---|
| `EditEllipsis` | … | its `OnClick` (browse, open a dialog) |
| `EditClear` | × | clears the text, then its `OnClick`; shows only while there is text |
| `EditSearch` | a magnifying glass | its `OnClick` |
| `EditPlus`, `EditMinus` | + and − | their `OnClick` |
| `EditDropDown` | a chevron down | its `OnClick` (open a list of your own) |
| `EditGlyph` | its `Image`, 16 DIPs square (the skin's icon size) | its `OnClick` |

- `Text()` and `SetText(s)`; `SetText` does not run `OnChange`.
- `SetPlaceholder(s)` shows `s` while the field is empty.
- `SetReadOnly(true)` keeps the user from typing; the buttons still work,
  `EditClear` included.
- `OnChange(fn)` runs on the event goroutine after each edit of the user.
  `EditClear` is one: it empties the text, runs `OnChange` once, then the
  button's own `OnClick`.
- `SetButtonVisible(i, on)` shows or hides the button of index `i`; a hidden
  one takes no room, and an index out of range is ignored.
- It can be bound to a string: `widget.Bind(path, get, set)`. A clear is an
  edit of the user, so `set` gets `""`. See [Binding](../widget.md#binding).

## Keyboard

The buttons are no stops of Tab, as DevExpress's are not: a click on one
leaves the keyboard in the text, and Tab goes from the text to the next
control. Ctrl+Enter in the text clicks the first `EditEllipsis` or
`EditDropDown` button shown; with neither it does nothing. Enter clicks the
form's accept button, as in a `TextBox`.

## Look

The frame takes the skin's look of a `TextBox` (`Field`), with the accent at
its edge while the text has the focus. In a right-to-left layout the buttons
sit at the field's left, the first one nearest the text. The icons take the
theme's muted text color.

## Testing

Assistive technology sees a text field and, after it, a button for each
button shown, named by its `Tooltip`, or else by its kind (More, Clear,
Search, Open, + and −; a glyph has no name without a tooltip):

```go
tt.Find(widget.ByRole(ui.RoleTextField)).Type("report.pdf")
tt.Find(widget.ByRole(ui.RoleButton).Label("Browse")).Click()
```
