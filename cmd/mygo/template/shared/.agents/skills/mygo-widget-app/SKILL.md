---
name: mygo-widget-app
description: Build or change the interface of a Go app with MyGo's ui/widget package (Form, VBox/HBox/Grid/Fields, TextBox, ComboBox, DataGridView, dialogs, Bind, Style and Tailwind classes). Use for forms, settings dialogs, tables and tool windows in a native MyGo app; use mygo-maintenance for the lifetime rules of raw ui views.
---

# MyGo widget apps

`ui/widget` is the layer for an app's forms, dialogs and tables. Controls are made once and kept; unlike WinForms, their setters and getters are safe from any goroutine and need no `Invoke`. Each control builds `ui` elements every frame. Check the app's pinned MyGo version first and read the source it pins: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, then `docs/ui/widget.md` there. Confirm every API you use against that source.

## Shape of an app

Make controls once, outside any callback that runs per frame. Keep them in variables or `AssignTo`; change them with setters.

```go
package main

import (
	"log"

	"github.com/egoist/mygo/ui/widget"
)

func main() {
	err := widget.Run("Greeter", func(w *widget.Form) {
		var name *widget.TextBoxWidget
		greeting := widget.Label("Type a name")
		w.SetContent(widget.VBox(8).Add(
			widget.TextBox().SetPlaceholder("Name").AssignTo(&name),
			widget.Button("Greet").Suggested().OnClick(func() {
				greeting.SetText("Hello, " + name.Text())
			}),
			greeting,
		).Padding(4))
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Layout

- `widget.VBox(gap)` and `widget.HBox(gap)` stack controls; the gap is in DIPs. `widget.Grid(colGap, rowGap)` and `widget.GridWidget.Attach(w, col, row, width, height)` place by cell. `widget.FlowPanel(gap)` wraps. `widget.ScrollPanel(w)` scrolls what is larger than its room.
- Use `widget.Fields()` for a form: `widget.FieldsWidget.Row("Label", control)` per row, the labels in one column. `widget.FieldsWidget.Stacked` puts labels above; `widget.FieldsWidget.Gap(row, col)` is in DIPs.
- Alignment and growth work in one box only. In a `VBox`: `Align` places a control horizontally and `VExpand` gives it the free height. In an `HBox`: `VAlign` places it vertically and `HExpand` gives it the free width. The other axis does nothing. `widget.BoxWidget.Pack(w, expand, fill)` is the explicit form.
- A table, list or text area needs room: `Expand` the boxes and the `TabControl` that hold it, or it gets none.
- `SetVisible(false)` hides a control and frees its room (and its `Fields` label).

## Binding

Bind a control to a value kept elsewhere instead of pairing `SetText` with `OnChange`:

```go
package app

import (
	"sync"

	"github.com/egoist/mygo/ui/widget"
)

type Config struct {
	mu          sync.Mutex
	name, email string
}

func (c *Config) text(field *string) func() string {
	return func() string { c.mu.Lock(); defer c.mu.Unlock(); return *field }
}

func (c *Config) setText(field *string) func(string) {
	return func(v string) { c.mu.Lock(); defer c.mu.Unlock(); *field = v }
}

func settings(cfg *Config) widget.Control {
	return widget.Fields().
		Row("Name", widget.Bind(widget.TextBox(), cfg.text(&cfg.name), cfg.setText(&cfg.name))).
		Row("Email", widget.Bind(widget.TextBox(), cfg.text(&cfg.email), cfg.setText(&cfg.email))).
		Gap(8, 12)
}
```

- `get` runs when you call `widget.Bind` and on each `widget.Reload`, on the goroutine that calls them. `set` runs on the event goroutine after the user's edit. Lock the state `set` writes whenever another goroutine reads it, as the example does.
- A program setter (`widget.TextBoxWidget.SetText`, `widget.CheckBoxWidget.SetChecked`) does not call `set`. Call `widget.Reload(control)` when the source changed elsewhere.

## Threads

- Handlers (`OnClick`, `OnChange`, `Bind`'s `set`) run one at a time on the event goroutine, never on the main thread. The queue is the app's: a handler that blocks holds back every other handler of every window, though the windows keep drawing.
- Setters and getters are safe from any goroutine. Run work that must happen in step with handlers through `widget.BeginInvoke(fn)` (does not wait) or `widget.Invoke(fn)` (waits; safe inside a handler, panics on the main thread). Before the app runs and after it quit, `widget.Invoke` returns without running `fn`.
- Keep slow I/O off handlers that must stay quick: start a goroutine, report with setters or `BeginInvoke`.
- Call no widget method inside a callback that runs under the package's lock: the key function and every function of a `widget.Column` (`Text`, `Compare`, `Filter`, `Spans`, `Color`, `Image`), the function of `UpdateRow` and of `widget.ListBoxWidget.UpdateItem`, the functions of a `ListBox` or `TreeView`, a `Custom` control's build function. They read their arguments only.

## Tables

```go
package app

import (
	"time"

	"github.com/egoist/mygo/ui/widget"
)

type Account struct {
	ID     uint
	Name   string
	Status string
}

func accounts(rows []Account) *widget.DataGridViewWidget[Account, uint] {
	table := widget.DataGridView(func(a *Account) uint { return a.ID },
		widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }, Sortable: true, Filterable: true},
		widget.Column[Account]{Title: "Status", Text: func(a *Account) string { return a.Status }, Width: 120},
	).CheckboxColumn().BottomPanel().Expand()
	table.AddRows(rows)
	go func() {
		for _, row := range rows {
			time.Sleep(time.Second)
			table.UpdateRow(row.ID, func(a *Account) { a.Status = "done" })
		}
	}()
	return table
}
```

- Every row has a key; `widget.DataGridViewWidget.AddRows` replaces the row of an existing key; change a row in place with `widget.DataGridViewWidget.UpdateRow(key, fn)`, never by rebuilding the table. It scales to a million rows.
- `widget.DataGridViewWidget.Selected`, `widget.DataGridViewWidget.Checked` and `widget.DataGridViewWidget.CountSelected` read the user's choice; `widget.DataGridViewWidget.OnActivate(fn)` is a double click or Enter; `widget.DataGridViewWidget.ContextMenuFunc` builds a menu on the click.
- Save column widths with `widget.DataGridViewWidget.SetColumnsLayout` and `widget.DataGridViewWidget.OnColumnsChange`, debounced: it fires many times during one drag.

## Dialogs

- `widget.NewDialog(parent, title, subtitle, width, height, content)` makes one; `widget.Dialog.Run` blocks until it closes. Call it from a handler or a goroutine, never the main thread (it panics there). A `height` of 0 fits the content.
- In a handler, `Run` keeps running the other handlers while it waits, so state can change between the call and its return: read it again afterwards.
- `widget.Alert`, `widget.Confirm`, `widget.InputString`, `widget.InputInt`, `widget.InputSelect` and the file dialogs `widget.OpenFile`, `widget.SaveFile`, `widget.SelectFolder` block the same way and panic on the main thread too.
- Enter and Escape: `widget.Form.SetAcceptButton(b)` and `widget.Form.SetCancelButton(b)`.

## Style

- `widget.Style` is typed: `Padding`, `Margin` and `Radius` are in theme units (4 DIPs by default); `Width`, `Height`, `BorderWidth` and font sizes are in DIPs. It has `Hover`, `Pressed`, `Focused`, `Disabled` and `Dark` variants and `Transition`.
- `ui/widget/tw` compiles Tailwind classes to a Style: `control.Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))`. An unknown class is logged once and ignored; `go tool mygo vet .` reports it in a literal. Read `docs/ui/widget/tw.md` for the classes and for what Tailwind has that `tw` does not (`flex`, `gap`, `w-full`, arbitrary values).
- Follow the theme's colors (`bg-accent`, `text-muted`) rather than fixed palette colors so the dark theme works.

## Controls of your own

Use `widget.Custom` only for what no widget does; it draws with raw `ui` (see the `mygo-maintenance` skill for the rules of `ui` views). Its build function runs every frame and under the package's lock. State it reads must change inside `widget.Change(fn)`, from any goroutine but never from a build: otherwise the read races and nothing redraws.

## Verify

- Test without a window with `widget.NewTester` and `widget.TestForm` (`docs/ui/widget/tester.md`): find controls by label or role, click, type, read state.
- Run `go tool mygo vet .` and `CGO_ENABLED=0 go test ./...`; add `-race` when handlers share state.
