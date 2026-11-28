---
name: mygo-widget-app
description: Build or change the interface of a Go app with MyGo's ui/widget package (Form, VBox/HBox/Grid/Fields, TextBox, ComboBox, DataGridView, dialogs, Bind, Style and Tailwind classes). Use for forms, settings dialogs, tables and tool windows in a native MyGo app; use mygo-maintenance for the lifetime rules of raw ui views.
---

# MyGo widget apps

`ui/widget` is the layer for an app's forms, dialogs and tables. Controls are made once and kept; unlike WinForms, their setters and getters are safe from any goroutine and need no `Invoke`. Each control builds `ui` elements every frame. Check the app's pinned MyGo version first and read the source it pins: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, then `docs/ui/widget.md` there. Confirm every API you use against that source.

## Shape of an app

Make controls once, outside any callback that runs per frame. Keep them in variables or `AssignTo`; change them with setters.

The project's configuration file (mygo.json) says nativeOnly: the app is built without a web view, so `widget.WebBrowser`, `Window.NewWebView` and windows that show a page are errors. Take nativeOnly out of it before adding any of them.

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

- `widget.VBox(gap)` and `widget.HBox(gap)` stack controls; the gap is in DIPs. `widget.Grid(colGap, rowGap)` and `widget.GridWidget.Attach(w, col, row, width, height)` place by cell. `widget.FlowPanel(gap)` wraps; `.Justify()` stretches each line, the last too, to both edges (never `HExpand()` its items). `widget.ScrollPanel(w)` scrolls what is larger than its room; `.Vertical()` scrolls up and down only, so labels wrap at its width.
- Use `widget.Fields()` for a form: `widget.FieldsWidget.Row("Label", control)` per row, the labels in one column. `widget.FieldsWidget.Stacked` puts labels above; `widget.FieldsWidget.SettingsRows` makes a settings page (a card per section with its title above it, wrapping labels, controls at the right edge, dividers; put it on `widget.RoleChrome`); `widget.FieldsWidget.Describe("text")` after a row adds a muted line under its label, read as the control's description; `widget.FieldsWidget.Wide()` after a row puts its control under its label, across the form or card (for a list, multiline box, table, link or button among settings rows: never wrap `SettingsRows` in a card of your own); `widget.FieldsWidget.Add(control)` adds a wide row with no label (what `Row("", control).Wide()` does), for a note, check box or button among the rows; `widget.FieldsWidget.Action(control)` after a row adds one small control (a `widget.LinkLabel`, a flat button) at the right end of a wide row's label line, or after an ordinary row's control: use it for a caption with a link over a box, not an `HBox` of a label, a spacer and a link, so the label still names the box and its access key focuses it; `widget.FieldsWidget.Gap(row, col)` is in DIPs. `widget.FieldsWidget.Section("Title")` adds a bold heading across the form before the rows that follow; one form with sections keeps one label column (not one `Fields` per group), and a section whose rows are all hidden hides its heading. `Section("")` is a group break (the gap, no heading node).
- Alignment and growth work in one box only. In a `VBox`: `Align` places a control horizontally and `VExpand` gives it the free height. In an `HBox`: `VAlign` places it vertically and `HExpand` gives it the free width. The other axis does nothing. `widget.BoxWidget.Pack(w, expand, fill)` is the explicit form.
- A table, list or text area needs room: `Expand` the boxes and the `TabControl` that hold it, or it gets none.
- `SetVisible(false)` hides a control and frees its room (and its `Fields` label).
- Show the parts of a form that belong to one choice of a combo with `widget.ComboBoxWidget.Shows`, not with `SetVisible` calls in its `OnChange`: `mode := widget.ComboBox(widget.Option{Value: "none", Text: "None"}, widget.Option{Value: "http", Text: "HTTP"}).Shows("http", hostBox, authBox)`. A control shows while any value it is named for is chosen, from every change of the choice: the user's, a setter's, a binding's.
- Show a state with a `widget.Status` (`widget.StatusNone`, `widget.StatusInfo`, `widget.StatusRunning`, `widget.StatusSuccess`, `widget.StatusWarning`, `widget.StatusDanger`), not a hand-made pill or card: `widget.Badge(text)` with `widget.BadgeWidget.SetStatus` (a running one's step goes in `widget.BadgeWidget.SetDetail`); `widget.InfoBar(status, title, message)` for a message in the page, with `widget.InfoBarWidget.SetAction`, `widget.InfoBarWidget.SetClosable` and `widget.InfoBarWidget.OnClose`; `widget.EmptyState(title, message)` where a list has no rows, with `widget.EmptyStateWidget.SetAction` (a grid shows it over its body with `widget.DataGridViewWidget.SetEmptyState`, a list with `widget.ListViewWidget.SetEmptyState` and `widget.ListBoxWidget.SetEmptyState`). The skin colors them.
- Pick values with the control made for them, not a hand-made row of buttons: `widget.RangeTrackBar(min, max)` for a range (`widget.RangeTrackBarWidget.Values`, `widget.RangeTrackBarWidget.SetValues`), `widget.Rating(max)` for stars, `widget.CheckButton(text)` for a button that stays pressed (`widget.CheckButtonWidget.SetIcon`), `widget.Segmented(labels...)` for one choice among joined buttons (`widget.SegmentedWidget.Selected`, `widget.SegmentedWidget.SetFillWidth`), and `widget.SplitButton(text, items...)` for an action with related ones in a menu (`widget.SplitButtonWidget.OnClick`, `widget.SplitButtonWidget.Suggested`). Their setters do not run `OnChange`.

## Navigation pane

For the left pane of an app ("Mail / Contacts", "Settings / Reports"), use `widget.NavBar()`: groups that open and close, each holding items of an icon and a text, one selected. Do not build one from buttons in a `VBox`, or from a `TreeView`.

```go
package app

import "github.com/egoist/mygo/ui/widget"

func pane(show func(key string)) widget.Control {
	nav := widget.NavBar().
		AddGroup("Mail").
		AddItem("inbox", "Inbox", nil).
		AddItem("sent", "Sent", nil).
		AddGroup("Contacts").
		AddItem("people", "People", nil).
		SetSelected("inbox").
		OnSelect(show)
	return widget.HBox(0).Add(nav.SetSize(220, 0), widget.Label("Page"))
}
```

- `widget.NavBarWidget.AddItem(key, text, icon)` adds to the last group (`icon` is a `*ui.SVG`, or nil); before any group it goes into a group with no header. A key given twice panics.
- `widget.NavBarWidget.OnSelect(fn)` runs on the event goroutine with the key the user selected, once per change; `widget.NavBarWidget.SetSelected(key)` does not run it. `widget.NavBarWidget.Selected` reads the key, "" for none.
- `widget.NavBarWidget.SetGroupOpen(i, open)` and `widget.NavBarWidget.GroupOpen(i)` take a group's index in the order added. The selected item takes the skin's `RowSelected` colors. Read `docs/ui/widget/navbar.md`.

## Picking a record

For a form field that picks one record by key (a customer, a product, an account) while the user reads several of its fields, use `widget.LookUpEdit(key, text, cols...)`: a drop-down of `widget.Column` columns with a header, filtered as the user types. Not a `ComboBox` of joined texts, nor a `DataGridView` in a dialog of its own.

```go
package app

import "github.com/egoist/mygo/ui/widget"

type Customer struct {
	ID               int
	Code, Name, City string
}

func customerField(all []Customer, pick func(id int, ok bool)) widget.Control {
	field := widget.LookUpEdit(
		func(c *Customer) int { return c.ID },
		func(c *Customer) string { return c.Name },
		widget.Column[Customer]{Title: "Code", Text: func(c *Customer) string { return c.Code }, Width: 60},
		widget.Column[Customer]{Title: "Name", Text: func(c *Customer) string { return c.Name }},
		widget.Column[Customer]{Title: "City", Text: func(c *Customer) string { return c.City }, Width: 80},
	).SetItems(all).OnSelect(pick)
	return widget.Fields().Row("&Customer", field)
}
```

- `widget.LookUpEditWidget.OnSelect(fn)` runs on the event goroutine when the user chooses a record (`ok` true) or empties the field (`ok` false), once per change; `widget.LookUpEditWidget.SetSelected(k)`, `widget.LookUpEditWidget.Unselect` and `widget.LookUpEditWidget.SetItems` do not run it. `widget.LookUpEditWidget.Selected` reads the key and whether one is chosen.
- A typed text that is no record's goes back to the chosen record's text when the focus leaves the field. Of a `widget.Column` it reads `Title`, `Text`, `Width`, `MinWidth`, `Align` and `Hidden`. Read `docs/ui/widget/lookupedit.md`.
- For a long list (hundreds to tens of thousands of records), add `widget.LookUpEditWidget.SetSearchPanel(true)`: the drop-down gets a search box matching any shown column, case ignored, and builds only the rows in view; the field then takes no typing. Up and Down in the search box move the highlight, Enter chooses, Escape closes. Not a separate SearchLookUpEdit type, nor a `DataGridView` in a dialog.
- For a field whose drop-down holds controls of its own (check boxes of columns, a date range, a small form), use `widget.PopupEdit(controls...)`: read the controls in `widget.PopupEditWidget.OnClosed(fn)` and show the outcome with `widget.PopupEditWidget.SetText`. Not a `Button` that opens a `Dialog`. `widget.PopupEditWidget.ShowPopup` and `widget.PopupEditWidget.ClosePopup` do not run `OnClosed`. Read `docs/ui/widget/popupedit.md`.
- For a text field with buttons in its frame (browse with …, clear, search, + and −, an icon of your own), use `widget.ButtonEdit(buttons...)` with a `widget.EditButton` each (`Kind` such as `widget.EditEllipsis` or `widget.EditClear`, `Tooltip`, `OnClick`). Not an `HBox` of a `TextBox` and `Button`s. `widget.EditClear` empties the text, runs `widget.ButtonEditWidget.OnChange` once, then its own `OnClick`; `widget.ButtonEditWidget.SetText` runs neither. `widget.ButtonEditWidget.SetReadOnly(true)` stops typing, not the buttons. Read `docs/ui/widget/buttonedit.md`.
- For a font family setting, use `widget.FontEdit()`: a drop-down of the system's families, each in its own face, with a search box. Not a `ComboBox` filled from `ui.FontFamilies`. `widget.FontEditWidget.OnChange(fn)` runs once per family the user picks; `widget.FontEditWidget.SetFamily` does not run it, and keeps a family the system lacks. `widget.FontEditWidget.SetFixedPitchOnly(true)` lists monospaced families alone. Read `docs/ui/widget/fontedit.md`.
- For an amount the user may want to work out (a price, a total, a quantity), use `widget.CalcEdit()`: a number field with a desk calculator in its drop-down. Not a `NumericUpDown` with a `Dialog` of buttons. `widget.CalcEditWidget.SetDecimals(n)` sets the decimals shown (2 by default); `widget.CalcEditWidget.OnChange` runs after a typed number or the calculator's Enter or =, never after `widget.CalcEditWidget.SetValue`, `widget.CalcEditWidget.ShowPopup` or `widget.CalcEditWidget.ClosePopup`. Read `docs/ui/widget/calcedit.md`.
- For a duration (a timeout, a run time, a break), use `widget.TimeSpanEdit()`, which holds a `time.Duration` shown as `[-][d.]hh:mm[:ss]`. Not a `MaskedTextBox` or a `NumericUpDown` of seconds. Up, Down and its − + buttons step the part under the caret, carrying into the next part. A typed value is read when the focus leaves the field or on Enter; text that does not parse goes back to the last value. `widget.TimeSpanEditWidget.SetRange(min, max)` (0 to 9999:59:59 at first) clamps; `widget.TimeSpanEditWidget.SetShowDays(true)` stops the hours at 23; `widget.TimeSpanEditWidget.SetShowSeconds(false)` hides the seconds. `widget.TimeSpanEditWidget.OnChange` runs once per change by the user, never for `widget.TimeSpanEditWidget.SetValue`. Read `docs/ui/widget/timespanedit.md`.
- For a tooltip that needs more than a line (a command's name, what it does, its shortcut, an icon, a hint to more help), give the control `SetSuperTip(widget.SuperTip{Title, Body, Image, Footer})`, on every control: a bold title, the body with the image at its start, a separator, the footer. Not a `Label` shown on hover, nor a long `SetTooltip` with line breaks. It replaces `SetTooltip`'s text while set; `SetSuperTip(widget.SuperTip{})` brings that back. It shows, places and hides as a text tooltip, and assistive technology reads its parts as the description. Read `docs/ui/widget/supertip.md`.

## Waiting

While the app loads or saves and the user must wait, call `widget.Form.ShowWait(caption, description)` and `widget.Form.CloseWait()` (safe from any goroutine): a `widget.ProgressPanel` centred over the dimmed form, which takes no click or key until it closes. A second `ShowWait` changes its text. Not a modal `Dialog` of your own, nor disabling every control. Show a `widget.ProgressPanel(caption)` inline where only part of a page is busy. Read `docs/ui/widget/progresspanel.md`.

## Search, paths, sections, colors

Use these, not a hand-made equivalent:

- `widget.SearchBox()` filters a list: `widget.SearchBoxWidget.OnChange` on each edit, `widget.SearchBoxWidget.OnSubmit` on Enter, `widget.SearchBoxWidget.Text`. Not a `TextBox` with a search `Icon`. Read `docs/ui/widget/searchbox.md`.
- `widget.BreadCrumb(items...)` shows where the user is in a path; `widget.BreadCrumbWidget.OnSelect(fn)` gets the index clicked, and the app trims its path to `[:i+1]` and calls `widget.BreadCrumbWidget.SetItems`. Read `docs/ui/widget/breadcrumb.md`.
- `widget.Accordion()` with `widget.AccordionWidget.AddSection(title, controls...)` is sections of a form, one open at a time; `widget.AccordionWidget.Open` is -1 for none. For a navigation pane use `widget.NavBar()`. Read `docs/ui/widget/accordion.md`.
- `widget.ColorEdit()` picks a color inline in a form (`widget.ColorEditWidget.Color`, `widget.ColorEditWidget.SetColor`); `widget.ColorDialog` is the picker in a dialog. Read `docs/ui/widget/coloredit.md`.
- `widget.TextBoxWidget.SetSuggestions(items...)` suggests as the user types; `widget.TextBoxWidget.SetMRU(max)` remembers the last entries after Enter. A `Password` box ignores both. Read `docs/ui/widget/textbox-suggestions.md`.

## Tags

For free tags or a list of IDs the user types (labels, recipients, codes), use `widget.TokenEdit(suggestions...)`: chips with a remove button, Enter or a comma adds what was typed, and a paste of several lines adds one token a line. Use `widget.TagSelect` only to pick among fixed options. Where the order of the picks matters, as the fields of an export, `widget.TagSelectWidget.SetOrdered(true)` shows them as chips in pick order above the toggles, each with a remove button, and `widget.TagSelectWidget.Values` returns them in that order, as `Option` values, not their texts: `sel := widget.TagSelect(widget.Option{Value: "TwoFactorSecret", Text: "2FA"}, widget.Option{Value: "Birthday", Text: "Birthday"}).SetOrdered(true).SetValues(cfg.Fields...).OnChange(save)`. `SetValues` drops values that are no option's, as stale config keys, and runs no `OnChange`.

```go
package app

import "github.com/egoist/mygo/ui/widget"

type Account struct{ Tags []string }

func tagsField(acc *Account, allTags []string, save func()) widget.Control {
	te := widget.BindTokens(widget.TokenEdit(allTags...),
		func() []string { return acc.Tags },
		func(v []string) { acc.Tags = v; save() })
	return widget.Fields().Row("&Tags", te)
}
```

- Keep the tokens with `widget.BindTokens`, not `SetTokens` plus an `OnChange` that saves `Tokens()`: `set` gets the tokens, in a slice of its own, after each change by the user (after `OnChange`), and `widget.Reload` refreshes the field from the source. `SetTokens` and `Reload` do not call `set`.
- `widget.TokenEditWidget.Tokens` returns a copy, in order. `widget.TokenEditWidget.SetTokens` trims and drops blanks and repeats, and does not run `widget.TokenEditWidget.OnChange`, which runs on the event goroutine after each change by the user. `widget.TokenEditWidget.SetSuggestions` and `widget.TokenEditWidget.SetPlaceholder` (shown while there is no token and no text). Read `docs/ui/widget.md`.
- For chips picked from a fixed set, kept in pick order, call `widget.TokenEditWidget.SetSuggestionsOnly(true)`: the user adds only suggestions (Down in the empty input lists them), other text adds nothing.

## Document tabs

For documents the user opens and closes in tabs (editors, invoices, customer records), use `widget.TabControl()` with `widget.TabControlWidget.SetClosable(true)`: an x on each tab, a middle click and Ctrl+F4 close a page, and Ctrl+Tab and Ctrl+Shift+Tab switch. Do not build tabs with close buttons from an `HBox` of buttons.

```go
package app

import "github.com/egoist/mygo/ui/widget"

func documents(form *widget.Form) *widget.TabControlWidget {
	tabs := widget.TabControl().SetClosable(true).
		OnPageClosing(func(page widget.Control) bool {
			return widget.Confirm(form, widget.Question, "Close", "Discard the changes?")
		})
	tabs.AddPage("Invoice 1042", widget.MultilineTextBox())
	return tabs
}
```

- `widget.TabControlWidget.OnPageClosing(fn)` runs on the event goroutine with the page before it closes; `false` keeps it. `widget.Confirm` may block there.
- A closed page goes as `widget.TabControlWidget.RemovePage` takes it out; `widget.TabControlWidget.OnSwitch` runs when the page shown closes, and on Ctrl+Tab. Read `docs/ui/widget.md` (TabControl).

## Command ribbon

For an app with many commands in tabs ("Home / Insert"), as Office and DevExpress apps have, use `widget.Ribbon()` first in the form's content: tabs of titled groups, each of large `widget.ToolButton`s (icon above text) and small controls stacked three high. For a single row of tools use `widget.ToolStrip` instead. Do not build a ribbon from a `TabControl` of `HBox`es.

```go
package app

import "github.com/egoist/mygo/ui/widget"

func commands(form *widget.Form, editor widget.Control, paste func()) {
	ribbon := widget.Ribbon().
		AddTab("Home").
		AddGroup("Clipboard").
		AddLarge(widget.ToolButton("Paste").OnClick(paste)).
		AddSmall(widget.ToolButton("Cut"), widget.ToolButton("Copy")).
		AddGroup("Font").
		AddSmall(widget.ToolButton("Bold").SetToggle(true), widget.ComboBox(widget.Option{Value: "11", Text: "11"})).
		AddTab("Insert").
		AddGroup("Tables").
		AddLarge(widget.ToolButton("Table"))
	form.SetContent(widget.VBox(0).Add(ribbon, editor))
}
```

- `widget.RibbonWidget.AddGroup` adds to the last tab, `widget.RibbonWidget.AddLarge` and `widget.RibbonWidget.AddSmall` to the last group; before any, they add one with no title. A control the ribbon holds already is not added again.
- `widget.RibbonWidget.SetCollapsed` shows the tabs alone; a double click on a tab and Ctrl+F1 toggle it, and a click on a tab of a collapsed ribbon expands it. `widget.RibbonWidget.Tab` and `widget.RibbonWidget.SetTab` read and set the tab shown. A narrow window clips the groups; they do not shrink. Key tips come with it, letters it gives itself: F10 (and Alt alone on Windows) shows a letter on each tab, then on each command of the tab typed, in a form with no menu bar, off on macOS. There is no application menu or gallery. Read `docs/ui/widget/ribbon.md`.
- `widget.RibbonWidget.AddQuickAccess` puts tool buttons (Save, Undo) in the Quick Access Toolbar: small icons above the tabs, in the title bar with a `HeaderBar`, or below the ribbon after `widget.RibbonWidget.SetQuickAccessBelow`. The user adds and removes commands from their right-click menus; save `widget.RibbonWidget.QuickAccess` in `widget.RibbonWidget.OnQuickAccessChange` and restore it with `widget.RibbonWidget.SetQuickAccess`. Do not build it from a `ToolStrip` above the ribbon.
- `widget.RibbonWidget.AddContextualTab` adds a tab to a category ("Picture Tools") that shows only while `widget.RibbonWidget.SetCategoryVisible` says so, under a colored band; call it from the selection's handler instead of adding and removing tabs.

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
- `widget.Bind` takes a `TextBox`, `MultilineTextBox`, `SearchBox`, `ButtonEdit`, `FontEdit`, `CheckBox`, `CheckButton`, `ComboBox`, `NumericUpDown`, `CalcEdit`, `TrackBar`, `TimeSpanEdit` (a `time.Duration`), `Rating` or `DomainUpDown`. Bind a `RangeSpin` with `widget.BindRange` (`get` returns both numbers, `set` takes both), a `CheckBoxGroup` with `widget.BindValues` (the checked values, a `[]string`), and a `TokenEdit` with `widget.BindTokens` (its tokens, a `[]string`).
- A program setter (`widget.TextBoxWidget.SetText`, `widget.CheckBoxWidget.SetChecked`) does not call `set`. Call `widget.Reload(control)` when the source changed elsewhere.

## Validation

Check a form's values with rules on a `widget.ErrorProvider`, not an `OnValidating` handler per field that calls `SetError` by hand:

```go
package app

import (
	"regexp"

	"github.com/egoist/mygo/ui/widget"
)

func signUp(form *widget.Form, save func(name, email string)) widget.Control {
	errs := widget.ErrorProvider().SetValidateOnLeave(true)
	name := widget.Rule(errs, widget.TextBox(), widget.Required("Enter your name"))
	email := widget.Rule(errs, widget.TextBox(), widget.Required("Enter an email address"))
	widget.Rule(errs, email, widget.MatchRegexp(regexp.MustCompile(`^\S+@\S+\.\S+$`), "Not an email address"))
	return widget.Fields().Row("&Name", name).Row("&Email", email).Add(
		widget.Button("Sign up").OnClick(func() {
			if !form.ValidateChildren() {
				errs.FocusFirst(form)
				return
			}
			save(name.Text(), email.Text())
		}))
}
```

- `widget.Rule(p, control, check)` takes any control `widget.Bind` takes; `check` gets its value (a `string`, a `float64` for a `NumericUpDown` or a `CalcEdit`) and returns the error text, "" when valid. Several rules on one control: the first failing one's text shows. Ready rules: `widget.Required`, `widget.MatchRegexp`, `widget.InRange`.
- `widget.Form.ValidateChildren` runs the rules of the form's controls after their `OnValidating` handlers; `widget.ErrorProviderWidget.Validate` runs every rule of one provider. Neither moves the focus: `widget.ErrorProviderWidget.FocusFirst` does. `widget.ErrorProviderWidget.SetValidateOnLeave(true)` also checks a field when the focus leaves it. Read `docs/ui/widget/errorprovider.md`.

## Threads

- Handlers (`OnClick`, `OnChange`, `Bind`'s `set`) run one at a time on the event goroutine, never on the main thread. The queue is the app's: a handler that blocks holds back every other handler of every window, though the windows keep drawing.
- Setters and getters are safe from any goroutine. Run work that must happen in step with handlers through `widget.BeginInvoke(fn)` (does not wait) or `widget.Invoke(fn)` (waits; safe inside a handler, panics on the main thread). Before the app runs and after it quit, `widget.Invoke` returns without running `fn`.
- Keep slow I/O off handlers that must stay quick: start a goroutine, report with setters or `BeginInvoke`.
- Call no widget method inside a callback that runs under the package's lock: the key function and every function of a `widget.Column` (`Text`, `Compare`, `Filter`, `Spans`, `Color`, `Image`, `Badge`, `BadgeDetail`, `Format`, `Summary`, `GroupSummary`, `Hint`) and of `widget.DataGridViewWidget.SetRowFormat` and `widget.DataGridViewWidget.SetGroupBy`, the function of `UpdateRow`, of `widget.ListBoxWidget.UpdateItem` and of `widget.TreeViewWidget.UpdateNode`, the functions of a `ListBox`, `TreeView` or `ListView` (its key and columns, `widget.ListViewWidget.SetGroupBy`, `widget.ListViewWidget.SetImage`), a `Custom` control's build function. They read their arguments only.

## Tables

```go
package app

import (
	"iter"
	"strconv"
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
		widget.Column[Account]{Title: "Status", Text: func(a *Account) string { return a.Status }, Width: 120,
			Badge: func(a *Account) widget.Status {
				if a.Status == "done" {
					return widget.StatusSuccess
				}
				return widget.StatusRunning
			},
			Summary: func(rows iter.Seq[*Account]) string { // a cell in the row of totals under the table
				done := 0
				for a := range rows {
					if a.Status == "done" {
						done++
					}
				}
				return strconv.Itoa(done) + " done"
			}},
	).CheckboxColumn().BottomPanel().SetEmptyState(widget.EmptyState("No accounts", "Add one to start.")).
		SetGroupBy(func(a *Account) string { return a.Status }).SetFilterDropDowns(true).Expand()
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

- Show a row's state in the grid with the column's `Badge` (the cell's `Text` as a badge of the `widget.Status` it returns; `BadgeDetail` for a running one's step) and color cells with a `widget.CellFormat` (`Status`, `Fill`, `Text`, `Bold`): the column's `Format`, or `widget.DataGridViewWidget.SetRowFormat` for every cell of the row, the column's winning field by field. Not `Spans` or `Color` for a status: the skin colors these, a selected row shows the selection, and they are built once per row change. `widget.DataGridViewWidget.SetEmptyState(widget.EmptyState(...))` shows while no row shows, the search's included.
- Totals go in a column's `Summary`: it gets the rows shown (after the search) and returns the text of the column's cell in a row under the table; it runs when the rows change, not in every frame, so count there instead of in a label you update. `widget.DataGridViewWidget.SetGroupBy(fn)` puts the rows under a header "Title (n)" per title, by title or in the order `widget.DataGridViewWidget.SetGroupOrder` gives. A total for each group goes in the column's `GroupSummary` (same signature as `Summary`): it gets one group's rows shown and its text shows in that group's header line, under the column; it runs only when the rows change and only while the grid is grouped. Give it to a column after the first, whose place the group's title takes.
- Filter by columns with `widget.DataGridViewWidget.SetColumnFilter(title, widget.ColumnFilter{Contains: ..., Values: ...})` (read back with `widget.DataGridViewWidget.ColumnFilter`, all off with `widget.DataGridViewWidget.ClearFilters`, which keeps the bottom panel's search), not by filtering the rows yourself before `SetRows`: rows changed later move in and out by themselves. Let the user filter from the header with `widget.DataGridViewWidget.SetFilterDropDowns(true)` (a funnel and a check list of values on each `Filterable` column) and `widget.DataGridViewWidget.SetAutoFilterRow(true)` (a field under each title), and show what is filtered with `widget.DataGridViewWidget.SetFilterPanel(true)`. `widget.DataGridViewWidget.OnFilterChange(fn)` runs on the user's changes only, as a place to save them.
- Every row has a key; `widget.DataGridViewWidget.AddRows` replaces the row of an existing key; change a row in place with `widget.DataGridViewWidget.UpdateRow(key, fn)`, never by rebuilding the table. It scales to a million rows.
- An icon your image function returns later (a thumbnail that loads) is not seen by a row already shown: call `UpdateRow(k, func(*T){})` for its row when it arrives, or keep the icons in a `widget.ImageList` (`SetImageList`), whose changes the list view, list box and tree show.
- Each getter is a snapshot of its own: between `widget.DataGridViewWidget.Selected` and `widget.DataGridViewWidget.Row` a row may have been deleted, so check the `ok` of `Row`. `Row` returns a shallow copy of the row: if the row type holds a slice, a map or a pointer, reading it races with `UpdateRow`. Keep rows plain values, or copy deeply.
- `widget.DataGridViewWidget.Selected`, `widget.DataGridViewWidget.Checked` and `widget.DataGridViewWidget.CountSelected` read the user's choice, and `widget.DataGridViewWidget.OnSelectionChanged(fn)` tells when it changes; `widget.DataGridViewWidget.OnCheck(fn)` tells which row the user ticked or unticked; `widget.DataGridViewWidget.SetSelected(ks...)` chooses rows from code and fires no event; `widget.DataGridViewWidget.OnActivate(fn)` is a double click or Enter; `widget.DataGridViewWidget.ContextMenuFunc` builds a menu on the click.
- Sort by several columns with `widget.DataGridViewWidget.SortBy(orders...)` (`ui.SortOrder`s, the first first; `widget.DataGridViewWidget.SortOrders` reads them), and let the user add columns with a Shift-click on a header after `widget.DataGridViewWidget.SetMultiSort(true)`; not by sorting the rows yourself before `SetRows`. Pin the leading columns that name a row (a code, a name) with the column's `Frozen` and a `Width`, so they stay while a wide grid scrolls sideways.
- Save column widths with `widget.DataGridViewWidget.SetColumnsLayout` and `widget.DataGridViewWidget.OnColumnsChange`, debounced: it fires many times during one drag.
- Let the user put rows in order with `widget.DataGridViewWidget.SetRowReorder(true)` and save the new `Rows()` in `widget.DataGridViewWidget.OnRowsReordered`, which only a user's move fires; it does nothing over a shared `RowSource` (`DataGridViewOf`) and does not start while the rows are sorted, grouped, searched or filtered. Drag chosen rows onto another control with `widget.DataGridViewWidget.DragRows(fn)` and a `widget.OnDrop` of the type `fn` returns. A table cannot do both: the later call wins. Either one rebuilds the rows shown in every frame, so keep column functions cheap there.
- Fit columns to what they show with `widget.DataGridViewWidget.BestFit(titles...)` (none: every column shown), not by measuring texts yourself; `widget.DataGridViewWidget.SetColumnChooser(true)` lets the user hide, show and fit columns from a right-click on the header. For text cut off, `widget.DataGridViewWidget.SetCellHints(true)` shows it whole as a tooltip, and a column's `Hint` gives its cells a tooltip of their own (an error, a full path), rather than widening the column.
- Export what the grid shows with `widget.DataGridViewWidget.ShownCells` (the columns shown, in the header's order, and the rows as filtered, sorted and grouped, each cell its column's `Text`) and package `ui/widget/export`: `export.CSV(w, header, rows)` or `export.XLSX(w, header, rows)`, a path from `widget.SaveFile`. Not by walking `Rows()` yourself, which misses the user's filters, sort and column order, and not with an XLSX library: the package writes the workbook with the standard library alone, and an app that does not import it links neither writer. Read `docs/ui/widget/export.md`.
- For rows in a hierarchy with columns (files with sizes, accounts with balances), use `widget.TreeView` with `widget.TreeViewWidget.SetColumns(treeTitle, cols...)`, not a `DataGridView`: the first column holds the node's arrow, check box, image and text, then one column per `widget.Column` showing its `Text` (it reads `Title`, `Text`, `Width`, `MinWidth`, `Align`, `Hidden`; it does not sort). Read `docs/ui/widget/treeview.md`.

## Dialogs

- `widget.NewDialog(parent, title, subtitle, width, height, content)` makes one; `widget.Dialog.Run` blocks until it closes. Call it from a handler or a goroutine, never the main thread (it panics there). A `height` of 0 fits the content. `widget.Dialog.Parent` returns the form it was made over (nil for none), also after it closes: after `Run`, show the outcome there with `ShowToast`.
- In a handler, `Run` keeps running the other handlers while it waits, so state can change between the call and its return: read it again afterwards.
- `widget.Alert`, `widget.Confirm`, `widget.InputString`, `widget.InputInt`, `widget.InputSelect` and the file dialogs `widget.OpenFile`, `widget.SaveFile`, `widget.SelectFolder` block the same way and panic on the main thread too.
- Enter and Escape: `widget.Form.SetAcceptButton(b)` and `widget.Form.SetCancelButton(b)`.
- Tell the outcome of an action ("12 accounts exported") with a toast, not `widget.Alert`: `widget.Form.ShowToast(widget.Toast{Status: widget.StatusSuccess, Title: ..., Message: ...})` shows it near the bottom of the window for 4 seconds without blocking, from any goroutine; an `Action` with `OnAction` adds a button such as Undo (8 seconds then), and `widget.Form.CloseToast(id)` closes one. Keep message boxes for a question or an error the user must read.

## Style

- `widget.Style` is typed: `Padding`, `Margin` and `Radius` are in theme units (4 DIPs by default); `Width`, `Height`, `BorderWidth` and font sizes are in DIPs. It has `Hover`, `Pressed`, `Focused`, `Disabled`, `Selected` (a checked `CheckBox`, `RadioButton`, toggle `ToolButton`, a `Toggle` on) and `Dark` variants and `Transition`.
- `ui/widget/tw` compiles Tailwind classes to a Style: `control.Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))`. An unknown class is logged once and ignored; `go tool mygo vet .` reports it in a literal. Read `docs/ui/widget/tw.md` for the classes and for what Tailwind has that `tw` does not (`flex`, `gap`, `w-full`, arbitrary values). `truncate` keeps a `Label` on one line ending in an ellipsis: the label shrinks below its text beside other controls, and the whole text stays its accessible name.
- Follow the theme's colors (`bg-accent`, `text-muted`) rather than fixed palette colors so the dark theme works.
- For a skin (a brand color, compact density, high contrast, a skin file, a picker for the user), use the `mygo-widget-skin` skill.
- The outline of the controls (inputs, combo boxes, buttons, toggles) is the theme's `ControlBorder`: 3.4:1 against the window in the light appearance and 3.7:1 in the dark, over the 3:1 WCAG 1.4.11 asks. Dividers, grid lines and the edges of containers keep the lighter `Border`. Inputs (`TextBox`, `MaskedTextBox`, `NumericUpDown`, an editable `ComboBox`, `DateTimePicker`) are filled with the theme's `Field`, buttons with `Surface`, so a button is not read as a text box; use `widget.RoleField` for an input's fill in a `Style`. `widget.SetControlBorder(light, dark)` (colors from `ui.Hex`) sets the outline for both appearances at once, without touching a control, once at startup; a zero color keeps the theme's.

## Controls of your own

Use `widget.Custom` only for what no widget does; it draws with raw `ui` (see the `mygo-maintenance` skill for the rules of `ui` views). Its build function runs every frame and under the package's lock. State it reads must change inside `widget.Change(fn)`, from any goroutine but never from a build: otherwise the read races and nothing redraws.

## Verify

- Test without a window with `widget.NewTester` and `widget.TestForm` (`docs/ui/widget/tester.md`): find controls by label or role, click, type, read state.
- Run `go tool mygo vet .` and `CGO_ENABLED=0 go test ./...`; add `-race` when handlers share state.
