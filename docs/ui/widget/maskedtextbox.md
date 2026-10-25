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
`192-345_`). An insert that would push a character off the end of the mask is
refused whole (typing 9 first in `123-4567`): the text stays, and
`OnMaskInputRejected` hears its first character once. A character no slot takes
is refused and reported to `OnMaskInputRejected`. A character equal to the
literal at the caret steps over it, as WinForms' `SkipLiterals` does, so typing
`-` before the dash moves the caret past it, and pasting `(012) 345-6789` into
`(000) 000-0000` is no rejection. Backspace and Delete clear a slot
and close the gap; over a literal, Backspace clears the slot before it, and
Delete the one after. A character typed with the caret on a literal goes to the
next slot, and the caret waits before the literal after the last slot of a
group. Each edit is one step of undo, and an edit the mask refuses is none. A
paste is filtered through the mask, except over a selection: if one of its
characters is refused there, nothing changes (the selection stays), as in
WinForms. While an input method composes, its
preedit text shows as it comes, unfiltered, and a selection it replaces only
looks gone; the mask gets one edit at the commit, against the text before the
composition, and if it refuses it the selection stays. After an undo or a redo
a typed character equal to the prompt reads back as an empty slot (`&&&&`: a, `_`, b, c, undo:
`Text()` is `abc`).

## The text

- `Text()`: the characters typed, with no literals and no prompts (`1234567`).
- `TextWithLiterals()`: the text with the literals, a space for each empty slot
  (`123-45  `).
- `SetText(s)`: fills the slots with the characters of `s` from the first, or
  with the text as `TextWithLiterals` returns it (`123-4567`, a space for an
  empty slot), and reports whether each fitted (nothing changes if not). It
  does not run `OnChange`.
- `MaskCompleted()`: every required slot is filled; `MaskFull()`: every slot is.
- `SetMask(m)` puts the text, as `Text()` returns it, in the new mask, what its
  slots do not take dropped; it does not run `OnChange`. `SetPromptChar(r)`
  (default `_`) changes the prompt and keeps the text: a slot may hold the
  prompt character itself, the box keeps which slots are filled.

Unlike WinForms, the caret never goes after the first empty slot that follows
what was typed (the first slot, when the box is empty): a click in the empty
part of the box, the End key and the right arrow stop there, so that what is
typed next finds room. A click inside what was typed keeps its place, and so
does a selection.

It is not bindable (`Bind` takes a TextBox, a number, a combo...); use `OnChange`.
Not here: `TextMaskFormat`, `ResetOnPrompt` and `HidePromptOnLeave`.
