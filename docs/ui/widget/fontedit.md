# FontEdit

`FontEdit()` is a field that picks a font family, as DevExpress's
`FontEdit`. Its drop-down lists the families of `ui.FontFamilies`, each name
drawn in its own family, under a search box: typing filters them, the
families that start with what is typed first, then those that hold it, case
ignored, as `FontDialog`'s list matches.

```go
font := widget.FontEdit().SetFamily("Menlo").SetFixedPitchOnly(true).OnChange(func() {
	editor.SetFont(font.Family()) // the app's own
})
```

- `Family()` and `SetFamily(name)`; `SetFamily` does not run `OnChange`. A
  family the system lacks keeps its name in the field, which draws it in the
  default face; the drop-down then highlights no row.
- `SetFixedPitchOnly(true)` lists only monospaced families, as `FontDialog`'s.
  It does not change the family chosen.
- `OnChange(fn)` runs on the event goroutine after the user picks a family
  other than the one chosen, once a pick.
- It can be bound to a string: `widget.Bind(font, get, set)`. See
  [Binding](../widget.md#binding).

## Cost

The field draws the family in the interface's face and reads no list. The
families are read when the drop-down first opens, once (ui asks the system
the first time, tens of milliseconds with hundreds of fonts installed), and
only the rows in view are built, so only their faces load.

## Keyboard

A click, Enter, Space, Down or Alt+Down on the field opens the drop-down,
with the keyboard in its search box, on the family chosen. Up and Down move
the highlighted row, Enter picks it (with no row highlighted it does
nothing), Escape closes the drop-down and keeps the family. Typing
highlights the first match, so Enter picks it.

## Look

The frame takes the skin's look of a `TextBox` (`Field`), with the accent at
its edge while the field has the focus. The drop-down is at least as wide as
the field. In a right-to-left layout the family sits at the field's right
and the chevron at its left.

## Testing

Assistive technology sees a combo box whose value is the family, and in
the drop-down a text field and a list item for each row built:

```go
tt.Find(widget.ByRole(ui.RoleComboBox)).Click()
tt.Find(widget.ByRole(ui.RoleTextField)).Type("men")
tt.Find(widget.ByRole(ui.RoleListItem).Label("Menlo")).Click()
```
