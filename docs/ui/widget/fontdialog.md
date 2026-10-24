# FontDialog

`FontDialog()` asks the user for a font, as WinForms' `FontDialog`: a list of
families, a list of styles (Regular, Bold, Italic, Bold Italic), a size and a
sample in the chosen font, with OK and Cancel. It is a dialog drawn by ui, not
the system's, and lists the families of [`ui.FontFamilies`](../window.md).

```go
fd := widget.FontDialog().SetTitle("Editor font").SetFont(ui.Font{Family: "Menlo", Size: 13})
if fd.ShowDialog(form) {
	apply(fd.Font())
}
```

- `SetFont(f)` is the font the next `ShowDialog` starts from (it does not
  change one that is open); a size of 0 starts at 14. A family the system does
  not have is listed anyway. `Font()` is the one the last OK chose.
- OK sets `Family`, `Size` and `Italic` of the font, and `Weight` to 400 or 700
  when the user changed between regular and bold, else to the weight it had.
  `Features`, `Lang` and the rest are what `SetFont` gave.
- `SetMinSize(v)` and `SetMaxSize(v)` bound the size, in DIPs (1 and 999 by
  default). A size typed outside them is not taken, and a size `SetFont` gave
  is brought inside. As in WinForms, setting one past the other moves the
  other to match. The size is a number field: there is no list of preset
  sizes.
- `SetFixedPitchOnly(true)` lists only monospaced families. A font that is not
  one is not added: the dialog starts on the first fixed-pitch family.
- `SetSampleText(s)` is the text shown in the chosen font, `AaBbYyZz` as
  WinForms'; give one in the language of your app (`Nguyễn Văn Đức`). The
  sample shows what OK returns: the exact weight (a 300 stays 300 unless the
  user picks another style), the `Features` and `Lang` of `SetFont`.
- The family list opens scrolled to the current family. Typing in the box
  above it jumps to the first family that starts with what is typed, without
  regard to case. A family `SetFont` gave in another case than the system's
  is listed once, in the system's spelling, which `Font()` returns.
- `ShowDialog(parent)` blocks as [`Dialog.Run`](../widget.md#dialogs) does (and
  panics on the main thread), and returns true for OK. Cancel, Escape and
  closing the window return false and leave `Font()` alone.
- `SetTitle(s)` titles the dialog.

WinForms' `ShowColor`, `ShowEffects` (strikeout, underline), `ShowApply`,
`ShowHelp`, `FontMustExist` and `AllowScriptChange` are not here.
