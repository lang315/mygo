# ColorEdit

`ColorEdit()` is a swatch of a color inline in a form, which opens a color
picker below it, as DevExpress's `ColorPickEdit`: a click, Enter or Space
opens it, and Escape or a click outside closes it. It is
[`ui.ColorWell`](../color-picker.md) underneath. [`ColorDialog`](colordialog.md)
is the picker in a dialog, with the same `ui.Color`.

```go
var tint *widget.ColorEditWidget
tint = widget.ColorEdit().SetColor(ui.Hex("#3b82f6")).OnChange(func() { theme.SetTint(tint.Color()) })
```

- `Color()` and `SetColor(c)`; `SetColor` does not run `OnChange`. A new
  color edit shows the zero `ui.Color` until `SetColor`.
- `OnChange(fn)` runs on the event goroutine after the user picked a color.

Assistive technology sees a color well whose value is the color in hex; in a
test, `tt.Find(widget.ByRole(ui.RoleColorWell)).Value()` reads it.
