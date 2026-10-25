---
name: mygo-widget-app
description: Build or change the interface of a Go app with MyGo's ui/widget package (Form, VBox/HBox/Grid/Fields, TextBox, ComboBox, DataGridView, dialogs, Bind, Style and Tailwind classes). Use for forms, settings dialogs, tables and tool windows in a native MyGo app; use mygo-maintenance for the lifetime rules of raw ui views.
---

# MyGo widget apps

`ui/widget` is the layer for an app's forms, dialogs and tables. Controls are made once, kept, and changed from any goroutine, as in WinForms; each builds `ui` elements every frame. Check the app's pinned MyGo version first and read the source it pins: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, then `docs/ui/widget.md` there. Confirm every API you use against that source.

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

- `VBox(gap)` and `HBox(gap)` stack controls; the gap is in DIPs. `Grid(colGap, rowGap).Attach(w, col, row, width, height)` places by cell. `FlowPanel(gap)` wraps. `ScrollPanel(w)` scrolls what is larger than its room.
- Use `Fields()` for a form: `Row("Label", control)` per row, the labels in one column. `Stacked()` puts labels above; `Gap(row, col)` is in DIPs.
- Alignment and growth work in one box only. In a `VBox`: `Align` places a control horizontally and `VExpand` gives it the free height. In an `HBox`: `VAlign` places it vertically and `HExpand` gives it the free width. The other axis does nothing. `box.Pack(w, expand, fill)` is the explicit form.
- A table, list or text area needs room: `Expand` the boxes and the `TabControl` that hold it, or it gets none.
- `SetVisible(false)` hides a control and frees its room (and its `Fields` label).

## Binding

Bind a control to a value kept elsewhere instead of pairing `SetText` with `OnChange`:

```go
package main

import "github.com/egoist/mygo/ui/widget"

type Config struct{ name, email string }

func settings(cfg *Config) widget.Control {
	return widget.Fields().
		Row("Name", widget.Bind(widget.TextBox(),
			func() string { return cfg.name },
			func(v string) { cfg.name = v })).
		Row("Email", widget.Bind(widget.TextBox(),
			func() string { return cfg.email },
			func(v string) { cfg.email = v })).
		Gap(8, 12)
}
```

- `set` runs on the event goroutine after the user's edit. A program setter (`SetText`, `SetChecked`, `SetValue`) does not call it. Call `widget.Reload(control)` when the source changed elsewhere.
- `get` and `set` can run at the same time: guard shared state with a mutex.

## Threads

- Handlers (`OnClick`, `OnChange`, `Bind`'s `set`) run one at a time on the event goroutine, never on the main thread. A handler may block; the window keeps drawing.
- Setters and getters are safe from any goroutine. Run work that must happen in step with handlers through `widget.BeginInvoke(fn)` (does not wait) or `widget.Invoke(fn)` (waits; safe inside a handler, panics on the main thread).
- Keep slow I/O off handlers that must stay quick: start a goroutine, report with setters or `BeginInvoke`.
- Call no widget method inside a callback that runs under the package's lock: a `Column`'s `Text`/`Compare` and key functions, `UpdateRow`'s function, a `Custom` control's build function. They read their arguments only.

## Tables

```go
package main

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
		for _, a := range rows {
			time.Sleep(time.Second)
			table.UpdateRow(a.ID, func(a *Account) { a.Status = "done" })
		}
	}()
	return table
}
```

- Every row has a key; `AddRow(s)` replaces the row of an existing key; change a row in place with `UpdateRow(key, fn)`, never by rebuilding the table. It scales to a million rows.
- `Selected()`, `Checked()` and `CountSelected()` read the user's choice; `OnActivate(fn)` is a double click or Enter; `ContextMenuFunc` builds a menu on the click.
- Save column widths with `SetColumnsLayout` and `OnColumnsChange`, debounced: it fires many times during one drag.

## Dialogs

- `NewDialog(parent, title, subtitle, width, height, content)` makes one; `Run()` blocks until it closes, so call it from a handler or a goroutine, never the main thread. A `height` of 0 fits the content.
- `Alert`, `Confirm`, `InputString`, `InputInt`, `InputSelect` and the file dialogs `OpenFile`, `SaveFile`, `SelectFolder` are ready made.
- For Enter and Escape use `AcceptButton` and `CancelButton` on the form.

## Style

- `widget.Style` is typed: `Padding`, `Margin` and `Radius` are in theme units (4 DIPs by default); `Width`, `Height`, `BorderWidth` and font sizes are in DIPs. It has `Hover`, `Pressed`, `Focused`, `Disabled` and `Dark` variants and `Transition`.
- `ui/widget/tw` compiles Tailwind classes to a Style: `control.Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))`. An unknown class is logged once and ignored; `go tool mygo vet .` reports it in a literal. Read `docs/ui/widget/tw.md` for the classes and for what Tailwind has that `tw` does not (`flex`, `gap`, `w-full`, arbitrary values).
- Follow the theme's colors (`bg-accent`, `text-muted`) rather than fixed palette colors so the dark theme works.

## Controls of your own

Use `widget.Custom` only for what no widget does; it draws with raw `ui` (see the `mygo-maintenance` skill for the rules of `ui` views). Its build function runs every frame and under the package's lock.

## Verify

- Test without a window with `widget.NewTester` and `widget.TestForm` (`docs/ui/widget/tester.md`): find controls by label or role, click, type, read state.
- Run `go tool mygo vet .` and `CGO_ENABLED=0 go test ./...`; add `-race` when handlers share state.
