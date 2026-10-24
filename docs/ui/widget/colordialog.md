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

- `SetColor(c)` is the color the next `ShowDialog` starts from (it does not
  change one that is open); `Color()` is the one the last OK chose. Like
  WinForms' colors it has no transparency: the picker's opacity slider and an
  alpha in the hex are ignored, and `Color()` is opaque.
- `ShowDialog(parent)` blocks as [`Dialog.Run`](../widget.md#dialogs) does (and panics on the main
  thread), and returns true for OK. Cancel, Escape and closing the window return false and
  leave `Color()` alone.
- `SetTitle(s)` titles the dialog.

WinForms' `AllowFullOpen`, `CustomColors` and `FullOpen` are not here.
