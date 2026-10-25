---
name: mygo-widget-testing
description: Write tests for a MyGo app's ui/widget forms, tables and dialogs without opening a window, with widget.Tester (find controls by label or role, click, type, press keys, read state), widget.TestForm and widget.TestDialog. Use when adding or fixing widget behavior, reproducing a UI bug, or checking a screen after a change.
---

# Testing widgets without a window

`widget.Tester` draws widgets in memory, finds them as assistive technology does (by label or role), and acts on them as a user would. It opens no window and uses no reflection. Check the app's pinned MyGo version and read `docs/ui/widget/tester.md` in its source (`go list -m -f '{{.Dir}}' github.com/egoist/mygo`); confirm each API you use against that source. For raw `ui` views use `ui.NewTester` (`docs/ui/testing.md`).

## A first test

```go
package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestSignIn(t *testing.T) {
	name, pass := widget.TextBox(), widget.TextBox()
	signedIn := false
	form := widget.VBox(8).Add(name, pass, widget.Button("Sign in").OnClick(func() { signedIn = true }))

	tt := widget.NewTester(t, form, 0, 0) // 0, 0 is 600 by 800 DIPs
	tt.Find(widget.ByRole(ui.RoleTextField).Nth(0)).Type("ada")
	tt.Find(widget.ByLabel("Sign in")).Click()

	if !signedIn || name.Text() != "ada" {
		t.Fatalf("signed in %v, name %q", signedIn, name.Text())
	}
}
```

- `widget.NewTester(t, control, w, h)` draws one control. `widget.TestForm(t, form)` draws a form's view with its header, tool strip and status strip. `t` is a `*testing.T` or `*testing.B`.
- Make the controls, then the tester. Assert on the controls' getters (`widget.TextBoxWidget.Text`) as well as on what the tester reads.

## Finding

- `widget.ByLabel(s)` matches the elements named exactly `s`, case-sensitive: a button's text, a field's label, a text shown. `widget.ByRole(r)` matches a `ui.Role` (`ui.RoleButton`, `ui.RoleTextField`, `ui.RoleCheckBox`, …). `widget.Query.Label`, `widget.Query.Role` and `widget.Query.Nth(i)` narrow a query.
- `widget.Tester.Find(q)` returns the one match and fails the test when none or several match, listing what it found (`no button labeled "OK" (found: button "Ok", button "Cancel")`). Add `Nth(i)` for the i-th of several, in tree order. `widget.Tester.FindAll(q)` and `widget.Tester.Has(q)` do not fail.
- A control with no text needs a label for the test and for screen readers: give it one in the app rather than selecting it by position.

## Acting and reading

- Actions find the element again by identity in the last frame each time: `widget.Node.Click`, `widget.Node.DoubleClick`, `widget.Node.RightClick`, `widget.Node.Focus`, `widget.Node.Type(s)`, `widget.Node.Key(mods, key)`. Each settles afterwards and returns the node.
- An action fails the test, saying why, where a user could not do it: the element is disabled, hidden, outside the window, or cannot take the keyboard focus. Assert that a button is disabled with `widget.Node.Disabled`, do not click it.
- `Type` and `Key` focus first. A text field takes the focus with the caret at the end of its text, which is where `Type` inserts.
- Read with `widget.Node.Label`, `Value`, `Checked`, `Selected`, `Disabled`, `Focused`, `Bounds`, `Min`, `Max` and `Now`. Every read but `Label` fails the test (`no longer shown`) when the node no longer has the role and label it was found with; find it again. `Label` checks the role only, so you can read the new text of a control that renamed itself.

## Settling

`widget.Tester.Settle` runs what was posted to the event goroutine and draws frames until a round posts nothing of its own. Every action settles, so a handler's `Post`, `BeginInvoke` or `Invoke` has run when `Click` returns. Call `Settle` after a change the test made itself (a setter, a `Post`), never from a handler. A program that posts without end fails it with `still posting`: use the embedded `ui.Tester`'s `Frame` for that.

## Work on other goroutines

Never pass or fail a test on the clock: no `time.Sleep`, no timeout that decides the result.

- Start the work, wait for the event it ends with (a channel the app or the test closes, `sync.WaitGroup`), then `Settle` and assert.
- Make the app's clock and timers injectable and advance a fake clock, instead of waiting for a real one.
- Run `go test -race -shuffle=on`: handlers run on the event goroutine, and a race between a handler and the test is a real bug. A test that fails once in a while has a race to fix in the test or the app, not a timeout to raise.

## Dialogs

`widget.TestDialog(t, d)` makes `widget.Dialog.Run` block until the dialog closes, as with a window, but show none. Call it before `Run`, run the dialog on another goroutine, act through the tester, and read `widget.DialogTester.Closed`. The test's end closes the dialog.

```go
package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestRenameDialog(t *testing.T) {
	name := widget.TextBox()
	result := ""
	var d *widget.Dialog
	d = widget.NewDialog(nil, "Rename", "", 300, 0, widget.VBox(8).Add(
		name,
		widget.Button("OK").Suggested().OnClick(func() { result = name.Text(); d.Close() }),
	))

	dt := widget.TestDialog(t, d)
	done := make(chan struct{})
	go func() { d.Run(); close(done) }()

	dt.Find(widget.ByRole(ui.RoleTextField)).Type("Ada")
	dt.Find(widget.ByLabel("OK")).Click()
	<-done

	if result != "Ada" || !dt.Closed() {
		t.Fatalf("result %q, closed %v", result, dt.Closed())
	}
}
```

- `widget.DialogTester.Escape` presses Escape as a window does: an open drop-down or a shortcut takes it, else it asks `OnClose` (on the event goroutine: wait for it before `Closed`) and closes unless that says no or the dialog is not deletable.
- A `DialogTester` embeds the `Tester`, so its `Find` takes a `widget.Query`; `ui.Tester`'s `Find(string)` is shadowed.
- Only a dialog you made with `widget.NewDialog` and hold in the test can be driven by `widget.TestDialog`. `widget.Alert`, `widget.Confirm`, `widget.InputString` and the other input and file dialogs make their own dialog and show a real window, so a test cannot reach them and one that calls them hangs or opens a window. Have the app call them through a function variable or an interface that the test replaces.

## Tables

```go
package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

type Account struct {
	ID   uint
	Name string
}

func TestAccountsTable(t *testing.T) {
	table := widget.DataGridView(func(a *Account) uint { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }},
	).CheckboxColumn().Expand()
	table.AddRows([]Account{{1, "Ada"}, {2, "Grace"}})

	tt := widget.NewTester(t, table, 0, 0)
	tt.Find(widget.ByRole(ui.RoleRow).Label("Grace")).Click()
	if got := table.Selected(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("selected %v, want [2]", got)
	}

	tt.Find(widget.ByLabel("Check Ada")).Click()
	if got := table.Checked(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("checked %v, want [1]", got)
	}
}
```

- A row is named by the text of its **first column**, and its text is a `ui.RoleText` node of its own, so `widget.ByLabel("Ada")` matches both. Find the row with `widget.ByRole(ui.RoleRow).Label("Ada")` and a text in a cell with `widget.ByRole(ui.RoleText).Label("ada@x")`: the `ui.RoleCell` nodes have no label. The header is a row too, named by its titles joined (`Name Mail`), so `widget.ByRole(ui.RoleRow).Nth(0)` is the header; its titles are `ui.RoleColumnHeader` nodes.
- The check boxes of a `CheckboxColumn` are named `Check <first column's text>` and `Check all`. A table shows what is in view: scroll it (`widget.Node.Click` scrolls an element into view first) or size the window to the rows.
- Read rows from the table, not from the screen: `widget.DataGridViewWidget.Row`, `Selected`, `Checked`, `Len`.

## A bug is a failing test first

Reproduce it with a tester, watch it fail for the reason the bug gives, then fix it. Exercise the frames in between: a control that disappears and returns, focus moving, typing, shortcuts. Run the app's whole suite with `-race` before you finish.
