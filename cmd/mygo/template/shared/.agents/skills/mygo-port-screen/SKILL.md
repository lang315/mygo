---
name: mygo-port-screen
description: Port a screen written for WinForms (or for gota, the GTK Go toolkit) to MyGo's ui/widget package, control by control, with its layout, events, data binding, threads and Tailwind-like classes. Use when moving a form, dialog or table from another desktop toolkit to a native MyGo app.
---

# Porting a screen to ui/widget

`ui/widget` takes WinForms' names where WinForms has a control, with Go-style methods. Port one screen at a time, test it, then the next. Check the app's pinned MyGo version and read its source (`go list -m -f '{{.Dir}}' github.com/egoist/mygo`): `docs/ui/winforms.md` (the catalogue), `docs/ui/widget.md` (names, threads, dialogs) and `docs/ui/widget/tw.md` (classes). Confirm every API against that source. Build the new screen with the `mygo-widget-app` skill and test it with `mygo-widget-testing`.

## Steps

1. List the screen's controls, its layout, the data each shows, the events it handles and the threads it uses. Do not translate code line by line: translate what the screen does.
2. Name each control in `docs/ui/winforms.md` (each is done, or dropped with the reason) and, for gota, in the names table of `docs/ui/widget.md`. A control marked dropped has no widget: use what the catalogue names instead (`TabControl` and separate forms for MDI, a form built from `widget.Fields` for a property grid), or tell the user.
3. Lay out the form (below), then the data, then the events, then the threads.
4. Write a `widget.Tester` test for each thing the screen does before you rely on it, and compare behavior, not pixels.
5. Report what you could not port and why.

## Layout

- `TableLayoutPanel` of labels and controls: `widget.Fields()` with one `widget.FieldsWidget.Row` per row. Other `TableLayoutPanel`s: `widget.Grid` and `widget.GridWidget.Attach`.
- `FlowLayoutPanel`: `widget.FlowPanel`. `Panel` with `AutoScroll`: `widget.ScrollPanel`. `GroupBox`: `widget.GroupBox`. `SplitContainer`, `TabControl`, `StatusStrip`, `ToolStrip`, `MenuStrip`: the widgets of those names.
- `Dock` and `Anchor` do not exist. Put controls in `widget.VBox` and `widget.HBox`: a control that fills the room left grows (`HExpand` in an `HBox`, `VExpand` in a `VBox`); alignment across the box is `Align` in a `VBox` and `VAlign` in an `HBox`. gota's `halign-*`, `valign-*`, `hexpand` and `vexpand` work the same way, in the box that holds the control.
- Fixed pixel positions become gaps and padding. Padding, margin and radius are in theme units (4 DIPs by default); widths, heights and borders in DIPs.

## Events and data

- WinForms events become `On…` methods that take a function: `widget.ButtonWidget.OnClick`, `widget.TextBoxWidget.OnChange`, `widget.ComboBoxWidget.OnChange`, `widget.CheckBoxWidget.OnChange`, `widget.Form.SetAcceptButton` and `widget.Form.SetCancelButton` for `AcceptButton` and `CancelButton`. Look an event up in the control's page of `docs/ui/widget.md` or `docs/ui/widget/`; do not invent a name.
- Handlers run one at a time on the event goroutine, never on the main thread.
- `BindingSource` and `DataBindings`: `widget.Bind(control, get, set)` for a value, `widget.NewRowSource` and `widget.DataGridView` for rows.
- Validation: `widget.ErrorProvider` marks a control, the control's `OnValidating` says whether it is valid, and `widget.Form.ValidateChildren` runs them all (`docs/ui/widget/errorprovider.md`).

```go
package app

import (
	"sync"

	"github.com/egoist/mygo/ui/widget"
)

type Profile struct {
	mu         sync.Mutex
	Name       string
	Newsletter bool
}

// Edit shows the profile form as an OK and Cancel dialog does in WinForms:
// the controls are bound to a copy, OK commits it, Cancel drops it, and both
// close the form. Enter is OK and Escape is Cancel.
func Edit(p *Profile) error {
	p.mu.Lock()
	draft := Profile{Name: p.Name, Newsletter: p.Newsletter} // only the form touches it
	p.mu.Unlock()

	return widget.Run("Profile", func(w *widget.Form) {
		ok := widget.Button("OK").Suggested().OnClick(func() {
			p.mu.Lock()
			p.Name, p.Newsletter = draft.Name, draft.Newsletter
			p.mu.Unlock()
			w.Close()
		})
		cancel := widget.Button("Cancel").OnClick(func() { w.Close() })
		w.SetAcceptButton(ok).SetCancelButton(cancel)
		w.SetContent(widget.VBox(12).Add(
			widget.Fields().
				Row("Name", widget.Bind(widget.TextBox(),
					func() string { return draft.Name },
					func(v string) { draft.Name = v })).
				Row("", widget.Bind(widget.CheckBox("Send me the newsletter"),
					func() bool { return draft.Newsletter },
					func(v bool) { draft.Newsletter = v })),
			widget.ButtonBox().Add(cancel, ok),
		).Padding(4))
	})
}
```

## Threads

- `BackgroundWorker`: a goroutine. Report with setters (safe from any goroutine) or `widget.BeginInvoke`.
- `Control.Invoke`: `widget.Invoke(fn)`. Unlike WinForms it is safe to call in a handler, where it runs `fn` at once. It panics on the main thread and returns without running `fn` before the app runs and after it quit.
- `Control.BeginInvoke`: `widget.BeginInvoke(fn)`. `Timer`: `widget.Timer` (`docs/ui/widget/timer.md`).
- `MessageBox.Show`, `ShowDialog`: `widget.Alert`, `widget.Confirm`, `widget.Dialog.Run`. They block the handler that calls them and keep the other handlers running meanwhile.

## From gota

- Controls: the names table of `docs/ui/widget.md` (`Window` is `Form`, `Entry` is `TextBox`, `Notebook` is `TabControl`, `Frame` is `GroupBox`, `Table` is `DataGridView`, and so on).
- `IdleAddSync` is `widget.Invoke`, which does not deadlock in a handler.
- gota's classes become `tw.S("…")`; its stylesheet is Tailwind v2's, so its palette names differ from Tailwind v3's: v2 `green`, `yellow` and `purple` are `emerald`, `amber` and `violet` here, and classes with other color names keep their names and shades. `text-green` with no shade and `text-italic` have no effect in gota: leave them out. Read `docs/ui/widget/tw.md` (From gota) before translating a class.
- A GTK `VBox` of a label and a control per row is one `widget.Fields` row each.

## Not to port

Docking, printing and a reflection-driven property grid are not in `ui/widget`. Say so and ask the user for the replacement rather than approximating it.
