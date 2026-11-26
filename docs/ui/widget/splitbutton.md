# SplitButton

`SplitButton(text, items...)` is a button joined to an arrow, as Save beside
Save As…: a click on the button runs `OnClick`, and the arrow opens a menu of
`items`, the same `MenuItem`s as `DropDownButton`'s. It is
[`ui.SplitButton`](../split-button.md) underneath.

```go
save := widget.SplitButton("Save",
	widget.MenuItem{Text: "Save As…", OnClick: func() { fmt.Println("save as") }},
	widget.MenuItem{Text: "Save All", OnClick: func() { fmt.Println("save all") }},
).Suggested().OnClick(func() { fmt.Println("save") })
```

- `OnClick(fn)` runs on the event goroutine when the button is clicked, or
  Enter or Space pressed on it; the arrow does not run it. Choosing an item
  runs the item's `OnClick`.
- The arrow opens the menu on a click, and Enter, Space, Up or Down while it
  has the focus; Alt+Down opens it from the button too.
- `Suggested()` draws it in the accent color, as `ui.PrimarySplitButton`.
- `SetIcon(svg)` shows an icon in place of the text, in the color of the text,
  as `ButtonWidget.SetIcon`; the text still names the button.
- `SetDisabled(true)` disables the button and the arrow.

Assistive technology sees a group of a button, named by the text, and a menu
button, "More options"; in a test, `tt.Find(widget.ByLabel("More options")).Click()`
opens the menu, which `tt.Menu()` lists and `tt.ChooseMenuItem` chooses from.
