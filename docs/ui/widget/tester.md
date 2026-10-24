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
follows what the app did to it; the test fails if it is gone. An action
settles afterwards (it runs what was posted to the event goroutine, then draws
a frame) and returns the node.

- `Click()`, `DoubleClick()`, `RightClick()`, `Focus()`, `Type(s)`,
  `Key(mods, key)`.
- `Label()`, `Value()`, `Role()`, `Bounds()`, `Checked()`, `Selected()`,
  `Disabled()`, `Focused()`, `Min()`, `Max()`, `Now()`.

An action fails the test, saying why, where a user could not do it: the
element is disabled (`cannot click button "OK": it is disabled`), no longer
shown, or outside the window.

`tt.Settle()` does the same wait by itself, for a change made from the test;
do not call it from a handler. The embedded `*ui.Tester` (`tt.Tester`) gives
`Frame`, `Key`, `Texts`, `Advance` and the rest; its `Nodes()` returns a copy
of the last frame's elements.
