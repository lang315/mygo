# CheckButton

`CheckButton(text)` is a button that stays pressed while it is checked, as
Bold in an editor's toolbar: DevExpress's CheckButton. A click presses it or
lets it go, as does Space while it has the keyboard focus. It is
[`ui.Toggle`](../toggle.md) underneath, as `Toggle` is; `CheckButton` has
`CheckBox`'s names (`Checked`, `SetChecked`), an icon, and a binding.

```go
var bold *widget.CheckButtonWidget
bold = widget.CheckButton("Bold").SetIcon(boldIcon).OnChange(func() { fmt.Println("bold", bold.Checked()) })
wrap := widget.CheckButton("Wrap lines").SetChecked(true)
```

- `Checked()`; `SetChecked(on)` does not run `OnChange`.
- `SetIcon(svg)` shows an icon in place of the text, in the color of the text,
  as `ButtonWidget.SetIcon`; the text still names the button for assistive
  technology and a `Tester`. nil removes it.
- `OnChange(fn)` runs on the event goroutine after the user pressed the button
  or let it go.
- It can be bound to a bool: `widget.Bind(bold, get, set)`. See
  [Binding](../widget.md#binding).
- Under a skin, a checked button takes the skin's accent, as `Toggle` does.

DevExpress's `GroupIndex`, which makes check buttons one choice, is
[`Segmented`](segmented.md).

Assistive technology sees a toggle button, pressed while checked; in a test,
`tt.Find(widget.ByLabel("Bold")).Checked()`.
