# PopupEdit

`PopupEdit(controls...)` is a field showing a text whose drop-down holds any
controls, in a column, as DevExpress's PopupContainerEdit. A click on the field,
or Enter, Space or Alt+Down while it has the keyboard focus, opens the
drop-down; Escape, a click outside it, or a click on the field closes it. The
app reads its controls when the user closes it, and sets the field's text. It
is [`ui.Popover`](../popover.md) underneath, below a field with a `ComboBox`'s
look (and the skin's `ComboBox` look).

```go
cols := widget.CheckBoxGroup(false,
	widget.Option{Value: "name", Text: "Name"},
	widget.Option{Value: "city", Text: "City"},
)
var edit *widget.PopupEditWidget
edit = widget.PopupEdit(cols).SetText("No columns").OnClosed(func() {
	edit.SetText(strings.Join(cols.Values(), ", "))
})
```

- `SetText(s)` and `Text()`: the text the field shows.
- `ShowPopup()`, `ClosePopup()` and `IsOpen()` open, close and read the
  drop-down from the app. They do not run `OnClosed`.
- `OnClosed(fn)` runs on the event goroutine after the user closed the
  drop-down, once per close.
- Hiding the control (`SetVisible(false)`) closes its drop-down; a disabled
  one does not open.

Assistive technology sees a pop-up button whose value is the text, expanded
while the drop-down shows; in a test, `tt.Find(widget.ByRole(ui.RolePopUpButton))`
finds the field (see [`widget.Tester`](tester.md)).
