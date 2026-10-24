# MaskedTextBox

`MaskedTextBox(mask)` is a one-line input that takes only what its mask allows,
as WinForms' `MaskedTextBox`: `"000-0000"` is seven digits with a dash, shown as
the template `___-____` the user fills.

```go
var phone *widget.MaskedTextBoxWidget
phone = widget.MaskedTextBox("(000) 000-0000").
	OnChange(func() { savePhone(phone.Text()) }).
	OnMaskInputRejected(func(r rune) { beep() })
if phone.MaskCompleted() { call(phone.TextWithLiterals()) }
```

## The mask

| | |
|---|---|
| `0` / `9` | a digit, required / optional |
| `#` | a digit, space, `+` or `-`, optional |
| `L` / `?` | a letter, required / optional |
| `A` / `a` | a letter or digit, required / optional |
| `&` / `C` | any character, required / optional |
| `>` `<` `\|` | letters typed after it are upper case / lower case / as typed |
| `\` | the next character is a literal |

Every other character is a literal, including `.`, `,`, `:`, `/` and `$`: the
culture-aware placeholders of WinForms are not here.

## Typing

The box always shows the template. The characters typed, pasted or committed by
an input method go into the slots, skipping literals, and the characters after
make room as WinForms' insert mode does (`123-45__`, typing 9 after the 1:
`192-345_`; what is pushed off the end is dropped). A character no slot takes is
refused and reported to `OnMaskInputRejected`. Backspace and Delete clear a slot
and close the gap; over a literal, Backspace clears the slot before it, and
Delete the one after. A character typed with the caret on a literal goes to the
next slot, and the caret waits before the literal after the last slot of a
group. Each edit is one step of undo. A paste is filtered through the mask.

## The text

- `Text()`: the characters typed, with no literals and no prompts (`1234567`).
- `TextWithLiterals()`: the text with the literals, a space for each empty slot
  (`123-45  `).
- `SetText(s)`: fills the slots with the characters of `s` from the first, and
  reports whether each fitted (nothing changes if not). It does not run
  `OnChange`.
- `MaskCompleted()`: every required slot is filled; `MaskFull()`: every slot is.
- `SetMask(m)` empties the box; `SetPromptChar(r)` (default `_`) changes the
  prompt and keeps the text.

It is not bindable (`Bind` takes a TextBox, a number, a combo...); use `OnChange`.
Not here: `TextMaskFormat`, `ResetOnPrompt` and `HidePromptOnLeave`.
