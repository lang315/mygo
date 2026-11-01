# widget.Tester

`widget.Tester` tests widgets without a window. It is a `ui.Tester` that finds
elements as assistive technology does, by label or role, and acts on them as
the user would. It adds no reflection and opens no window.

```go
func TestSignIn(t *testing.T) {
	name, pass := widget.TextBox(), widget.TextBox()
	var signedIn bool
	form := widget.VBox(8).Add(name, pass, widget.Button("Sign in").OnClick(func() { signedIn = true }))

	tt := widget.NewTester(t, form, 0, 0) // 0, 0 is 600 by 800 DIPs
	tt.Find(widget.ByRole(ui.RoleTextField).Nth(0)).Type("ada")
	tt.Find(widget.ByLabel("Sign in")).Click()
	if !signedIn || name.Text() != "ada" {
		t.Fatal("not signed in")
	}
}
```

`NewTester(t, control, w, h)` draws one control; `TestForm(t, form)` draws a
form's view, with its header, tool strip and status strip; `TestDialog(t, d)`
is a `TestForm` whose `Run` blocks without a window (see `Dialog`). `t` is
any `widget.TB`: `*testing.T` and `*testing.B`.

## Finding

| | |
|---|---|
| `ByLabel(s)` | elements named exactly `s`, case-sensitive: a button's text, a field's label, a text shown |
| `ByRole(r)` | elements of a `ui.Role` (`ui.RoleButton`, `ui.RoleTextField`, ...) |
| `q.Label(s)`, `q.Role(r)` | narrow a query |
| `q.Nth(i)` | the i-th match (from 0), in tree order |
| `tt.Find(q)` | the one match; the test fails where none or several match |
| `tt.FindAll(q)`, `tt.Has(q)` | every match; whether any matches |

A failing `Find` lists what it found, so a misspelled label is quick to see:

```
no button labeled "OK" (found: button "Ok", button "Cancel")
2 elements match button (found: button "Ok", button "Cancel")
```

## Acting and reading

A `*Node` looks itself up again by identity in the last frame each time, so it
follows what the app did to it. Identities of the parts of a widget are those
of their places, so the node must also still have the role and the label it was
found with; otherwise the test fails with `no longer shown (now button "C")`.
A button that renames itself is found again: `Label()` reads what the element
shows now, and any other use of the old node fails. An action settles
afterwards (see below) and returns the node.

- `Click()`, `DoubleClick()`, `RightClick()`, `Focus()`, `Type(s)`,
  `Key(mods, key)`.
- `Label()`, `Value()`, `Role()`, `Bounds()`, `Checked()`, `Selected()`,
  `Disabled()`, `Focused()`, `Min()`, `Max()`, `Now()`.

`Bounds()` is the part of the element the window shows: what the clipping and
scrolling areas around it leave. A click goes to the middle of that part, and
an element scrolled out of view is scrolled to first, as assistive technology
does. An action fails the test, saying why, where a user could not do it: the
element is disabled (`cannot click button "OK": it is disabled`), no longer
shown, hidden, or outside the window (whose current size counts).

`Focus()` gives the keyboard focus the way assistive technology does, without
pressing: a button or a check box is not activated by it. `Type(s)` and
`Key(...)` focus first, so `Find(ByLabel("OK")).Key(0, ui.KeyEnter)` presses OK
once. An element that cannot take the keyboard focus (a label) fails the test with
`cannot take the keyboard focus`, instead of typing into whatever had it. A text
field takes the focus with the caret at the end of its text, which is where
`Type` inserts.

`tt.Settle()` runs what was posted to the event goroutine and draws a frame,
again and again until a round posts nothing of its own, so what a handler
posts runs too; after 100 rounds it fails the test (`still posting`). It
counts every `Post` of the process, so a program that posts nonstop should
`Frame` instead. Call it
for a change made from the test, never from a handler. The embedded `*ui.Tester` (`tt.Tester`) gives
`Frame`, `Key`, `Texts`, `Advance` and the rest; its `Nodes()` returns a copy
of the last frame's elements.

**What `Texts` holds.** `tt.Texts()` and `tt.HasText` are the texts the frame
shows, labels and cells and tabs, once each, in tree order. They leave out the
text a field holds (a `TextBox`'s, a `MultilineTextBox`'s, a number's): read it
with the node, `tt.Find(widget.ByRole(ui.RoleTextField)).Value()`, or with the
widget's own getter. `tt.Nodes()` lists the raw elements, as assistive
technology sees them.

`DialogTester` embeds this `Tester`, so its `Find` takes a `Query`; the
`Find(string)` of `ui.Tester` (`tt.Tester.Tester.Find`) is shadowed.
