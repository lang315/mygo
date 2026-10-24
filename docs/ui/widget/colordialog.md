# ColorDialog

`ColorDialog()` asks the user for a color, as WinForms' `ColorDialog`: a color
picker with OK and Cancel. It is [`ui.ColorPicker`](../color-picker.md) in a
dialog drawn by ui, not the system's: a square of saturation and brightness,
sliders of hue and opacity, the color in hex, and swatches of common colors.

```go
cd := widget.ColorDialog().SetTitle("Tint").SetColor(ui.Hex("#2563eb"))
if cd.ShowDialog(form) {
	apply(cd.Color())
}
```

- `SetColor(c)` is the color it starts from; `Color()` is the one the last
  OK chose.
- `ShowDialog(parent)` blocks as [`Dialog.Run`](../widget.md#dialogs) does, and
  returns true for OK. Cancel, Escape and closing the window return false and
  leave `Color()` alone.
- `SetTitle(s)` titles the dialog.

WinForms' `AllowFullOpen`, `CustomColors` and `FullOpen` are not here.

In a test, build the dialog's `Dialog` with `TestDialog` (see
[`widget.Tester`](tester.md)): the package's own tests do so with the dialog the
type builds for `ShowDialog`.
