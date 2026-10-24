# Mnemonics

A label can name a key with `&`, as in WinForms: `Button("&Save")` is Save with
the S as its access key, and Alt+S clicks it; `Fields().Row("&Name", nameBox)`
makes Alt+N give `nameBox` the focus. `&&` is a literal `&`. The first single
`&` that has a character after it marks the key. The key is the letter or digit
that types it, accents folded: `&Đóng` answers to Alt+D, `&Ẩn` to Alt+A; a
character no key types (`&@`) makes no mnemonic. An `&` at the end stays as it
is.

- **Where:** the text of a `Button` (without an icon), a `CheckBox` (Alt+key
  toggles it and gives it the focus), a `RadioButton` (selects it) and the
  labels of `Fields` rows (focus the row's control), in a form that shows them
  (not in `widget.View` or a `NewTester` of a control alone: there nothing
  answers and nothing is underlined). Alt+Shift+key answers as Alt+key.
- **Who answers:** a `Form.Shortcut` of the same key first; then, when several
  controls share a key, Alt+key moves the focus from one to the next and clicks
  none, as in WinForms. A control that is hidden, on a page that is not showing,
  or disabled (also by a container) does not answer.
- **Underline:** the access key is underlined in a button and a `Fields` label
  while Alt is held, as Windows and GTK show them; check boxes and radio buttons
  are not underlined (their label cannot be styled). The text is the same width
  either way.
- **macOS:** mnemonics are off: the `&` is taken out of the text, nothing is
  underlined and nothing is answered; the native menu bar owns Alt (Option types
  characters).
- **Windows:** an Alt+key that a shortcut or a mnemonic answers is taken, so the
  system does not enter its menu mode or beep. Menu items already take `&` as
  the platform's access keys on Windows (`MenuItem` text goes to the native
  menu); the menu bar is not yet consulted before control mnemonics.
- **Not yet:** plain labels (a `Label` with a mnemonic that focuses the next
  control), menu items on Linux, and an access-key property for assistive
  technology.
- Alt+letter does not reach the form while a text box has the keyboard focus on
  macOS (Option types there; mnemonics are off anyway); on Windows and Linux it
  does.
