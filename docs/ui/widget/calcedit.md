# CalcEdit

`CalcEdit()` is a number field with a calculator in its drop-down, as
DevExpress's `CalcEdit`. The field takes typed numbers as a `NumericUpDown`
does. The button at its end opens the calculator, which starts from the
field's value.

```go
var amount *widget.CalcEditWidget
amount = widget.CalcEdit().SetDecimals(0).SetValue(1200).
	OnChange(func() { save(amount.Value()) })
```

- `Value()` and `SetValue(v)`; `SetValue` does not run `OnChange`.
- `SetDecimals(n)` sets how many decimals the field and the calculator show,
  2 by default; below 0 is 0. The value keeps all of its own: after 1 ÷ 3
  with 3 decimals the field shows 0.333, and `Value()` is 1/3.
- `OnChange(fn)` runs on the event goroutine after each change of the user:
  a typed number, or the calculator's result. A result equal to the value
  runs nothing.
- `ShowPopup()` opens the calculator and `ClosePopup()` closes it, keeping
  the value. Neither runs `OnChange`. A disabled CalcEdit opens none, and
  disabling it closes it.
- It can be bound to a `float64`: `widget.Bind(amount, get, set)`. See
  [Binding](../widget.md#binding).

## The field

What is typed is the value as soon as it parses as a number (`12`, `-3.5`,
`1e3`); "nan", "inf" and other text are not. Leaving the field shows the
value with its decimals, or the last value where the text did not parse.
Enter clicks the form's accept button, as in a `TextBox`.

## The calculator

It has the keys of a desk calculator: the digits, the point, + − × ÷, =, %,
± (the sign), ⌫ (Backspace), C (clear all) and CE (clear the number being
typed). It chains left to right, as a desk calculator does: 12 + 3 × 2 = is
30, not 18.

- **Enter or =** puts the result in the field and closes the calculator.
- **Escape**, a click outside it or on the button closes it and keeps the
  value.
- **%** ends the operation at once, as on a desk calculator:

  | Keys | Result |
  |---|---|
  | a + b % | a + a×b/100 |
  | a − b % | a − a×b/100 |
  | a × b % | a×b/100 |
  | a ÷ b % | a÷b×100 |
  | b % | b/100 |

- A division by zero shows **E** in the theme's danger color and changes
  nothing: the keys, Enter and = are ignored until C.
- Results show rounded to the decimals; what is being typed shows as typed.

## Keyboard

Alt+Down or Ctrl+Enter in the field opens the calculator, which then takes
the keyboard: the digits, `.` or `,`, `+`, `-`, `*` (or `x`), `/`, `%` and `=`
as typed (the keypad's too), Enter (=), Escape (close), Backspace (⌫) and
Delete (CE). In a form, its Enter and Escape are the calculator's, not the
accept and cancel buttons'. ± and C are clicks only. Its keys are no stops of
Tab. Once it closes by Enter, = or Escape, the keyboard is back in the field.

## Look

The field takes the skin's look of a `TextBox` (`Field`); the drop-down that
of a `ComboBox`'s. In a right-to-left layout the button sits at the field's
left, and the keys' rows run right to left.

## Testing

Assistive technology sees a text field, a button named Open after it, and,
while the calculator shows, a button for each key named by its label (`7`,
`+`, `×`, `÷`, `−`, `%`, `±`, `⌫`, `C`, `CE`, `.`, `=`):

```go
tt.Find(widget.ByRole(ui.RoleButton).Label("Open")).Click()
tt.Type("12+3*2=")
```
