# Number input

<!-- preview number-input 160 -->

`ui.NumberInput` creates a text input editing a `*float64` as a number
between two bounds, which Up and Down, and the buttons beside it, change by
a step. What is typed applies as soon as it is a number in range, and shows
rounded to the decimals of the step once the input loses the focus.
`Changed` reports a new value and `Submitted` an Enter in it; `Focus` and
`AutoFocus` give its text field the keyboard.

```go
ui.NumberInput(c, &app.copies, 1, 99, 1).Label("Copies")
ui.NumberInput(c, &app.opacity, 0, 1, 0.05).Label("Opacity")
```

For a value the user only steps, use a [stepper](stepper.md); for one
chosen roughly, a [slider](slider.md); for an amount of money, which a
`float64` cannot hold exactly, a [decimal input](decimal-input.md).

## Spin box

`ui.SpinBox` takes the same arguments and is a number input in the look
of a classic spin box, as WinForms' `NumericUpDown`: one box holding the
text and, joined at its end, an up arrow over a down one. It edits,
steps, reports and is named as `NumberInput` is; its arrows are the
buttons "Increase" and "Decrease", in that order, to assistive
technology. They are no stops of Tab: a press on one steps, again as it
is held, faster after a while, and leaves the keyboard in the text, as
WinForms' do. They stay at the end of the box, its left right to
left, whatever its padding. A negative step steps by its
absolute value.

```go
ui.SpinBox(c, &app.copies, 1, 99, 1).Label("Copies")
```

## Accessibility

Assistive technology sees a spin button of the value, named by its `Label`
or its [field](form.md), which it increments and decrements by the step.
The text field takes the `Label`, and the two buttons take it before their own
names ("Copies Increase", "Copies Decrease"), so that two numbers side by side
are told apart.
