# Mnemonics

A label can name a key with `&`, as in WinForms: `Button("&Save")` shows
**S**ave with the S underlined, and Alt+S clicks it; `Fields().Row("&Name",
nameBox)` underlines the N and Alt+N gives `nameBox` the focus. `&&` is a
literal `&`. The first single `&` that has a character after it marks the key
(letters of the Latin alphabet and digits); an `&` at the end stays as it is.

- **Where:** the text of a `Button` (without an icon) and the labels of
  `Fields` rows, in the form that shows them. The key works while the control
  is shown (a hidden one, or one on a tab page that is not showing, does not
  answer), a disabled button is not clicked, and the same key twice goes to the
  first control laid out.
- **macOS:** mnemonics are off. The `&` is taken out of the text and nothing is
  underlined; the native menu bar owns Alt (Option types characters there).
- **Not yet:** check boxes, radio buttons, menu items and plain labels keep an
  `&` as they are; a plain `Label` with a mnemonic does not move the focus.
- Alt+letter does not reach the form while a text box has the keyboard focus
  on macOS (mnemonics are off there); on Windows and Linux it does.
