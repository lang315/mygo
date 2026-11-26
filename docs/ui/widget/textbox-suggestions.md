# TextBox suggestions and MRU

A `TextBox` can suggest texts below it as the user types, and remember what
the user entered, as DevExpress's `MRUEdit`.

```go
city := widget.TextBox().SetSuggestions("Hanoi", "Hue", "Da Nang")
recent := widget.TextBox().SetMRU(10) // remembers the last 10 entries
```

- `SetSuggestions(items...)` suggests those of the items containing what was
  typed, those starting with it first. Up and Down move among them, and Enter
  or a click takes one, which runs `OnChange`. While there are suggestions
  the box is a [`ui.Autocomplete`](../autocomplete.md); with none it is a
  plain text box. The items are copied.
- `SetMRU(max)`: after Enter, the text moves to the front of the
  suggestions, without duplicates, keeping at most `max` of them. 0 turns it
  off. Enter still runs `OnEnter`, or the form's accept button. Enter on a
  highlighted suggestion takes it and remembers nothing.
- A `Password()` box ignores both: it suggests nothing and remembers nothing.

An app that calls neither does not link `ui.Autocomplete`.
