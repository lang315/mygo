# Widgets that live on

Package `ui/widget` builds an app's interface from controls you make once,
keep, and change from any goroutine, as in WinForms: a text box is a
`*TextBoxWidget`, and `SetText` changes what every window showing it
draws. It is `ui` underneath: each control builds `ui` elements every
frame, and the windows that show it redraw when it changes.

It is the layer to build an app's forms, dialogs and tables with. Reach
for [`ui`](README.md) directly to draw a control of your own (see
[Controls of your own](#controls-of-your-own)) or a view whose state
lives in the view itself.

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

`Run` starts the app with one window, 1000 by 700, titled with the app's
name; the function sets the window up before it shows. Other windows come
from `NewForm(title, width, height)`, set up the same way, then `Show()`.
`Native()` returns the `*mygo.Window` once shown. To put widgets in a
window of your own, `widget.View(w)` is a `ui` view showing `w`:
`mygo.WindowOptions{Content: ui.View(widget.View(w))}`. The
[widget tour](../../examples/widget-tour) shows every widget.

## Names

The controls take the names WinForms gives them, where WinForms has one;
gota's names are listed for code ported from it.

| ui/widget | WinForms | gota |
|---|---|---|
| `Form` (`NewForm`) | Form | `Window` |
| `TextBox` | TextBox | `Entry` |
| `MultilineTextBox` | TextBox with Multiline | `TextView` |
| `ComboBox` | ComboBox | `ComboBox`, `ComboBoxText` |
| `TabControl` | TabControl | `Notebook` |
| `NumericUpDown` | NumericUpDown | `SpinButton` |
| `GroupBox` | GroupBox | `Frame` |
| `DataGridView` | DataGridView | `Table` |
| `LinkLabel` | LinkLabel | `LinkButton` |
| `CheckBox`, `CheckBoxGroup` | CheckBox, CheckedListBox | `Checkbox`, `H/VCheckboxGroup` |
| `ProgressBar` | ProgressBar | `ProgressBar` |
| `PictureBox` | PictureBox | `Image` |
| `ScrollPanel` | Panel with AutoScroll | `ScrolledArea` |
| `FlowPanel` | FlowLayoutPanel | `FlowBox` |
| `Grid` | TableLayoutPanel | `Grid` |
| `VBox`, `HBox` | — | `VBox`, `HBox` |
| `Label`, `Button`, `RadioGroup` | Label, Button, RadioButton | same |
| `Alert`, `Confirm` | MessageBox | `Alert`, `Confirm` |
| `Custom` | UserControl | — |

## Threads

- **Setters and getters** are safe from any goroutine. A setter returns at
  once; the windows redraw on the main thread.
- **Handlers** (`OnClick`, `OnChange`, `OnClose`, ...) run one at a time on
  a goroutine of the package, the event goroutine, never on the main
  thread. A handler may block: the window keeps drawing.
- **`Post(fn)`** runs `fn` on the event goroutine after what was posted
  before it, without waiting. It works from anywhere.
- **`Update(fn)`** runs `fn` on the event goroutine and waits for it, for a
  worker that must change widgets in step with the handlers. In a handler it
  runs `fn` at once.
- **Blocking calls** (`Update`, `Dialog.Run`, `Alert`, `Confirm`, the
  `Input` functions and the file dialogs) panic on the main thread, which
  must never wait. In a handler, they keep running the other queued
  handlers while they wait, as GTK runs its main loop under a dialog; on
  another goroutine they just wait.

## Dialogs

A dialog is a window over another, its parent, which it blocks while it is
open: a sheet on the parent on macOS, a modal window on Linux (where it
blocks every window of the app) and Windows.

```go
var d *widget.Dialog
d = widget.NewDialog(w, "New account", "", 420, 260, widget.VBox(8).Add(
	name,
	widget.ButtonBox().Add(widget.Button("Done").Suggested().OnClick(func() { d.Close() })),
))
d.Run() // returns once the dialog closes
```

`NewDialog(parent, title, subtitle, width, height, content)` takes a nil
parent for a dialog of its own. Width and height are the window's, title
bar included, so a macOS sheet, which has none, shows that much shorter;
the title and subtitle are drawn in the dialog for the same reason.
`Run` shows it and returns when it closes: by `Close`, by the user (its
close button, or Escape once no drop-down in it is open), or because its
parent closed. For a dialog that returns a value, embed `*widget.Dialog`
in a type of your own whose `Run` returns it. A dialog opened from a
dialog takes the outer one's `Form` as its parent; closing a window
closes the dialogs over it, the innermost first. `SetDeletable(false)` keeps the user from closing it (its
`Close` and its parent still can), and `OnBeforeRun` runs before it shows.

Ready-made dialogs, each with an icon for its `Kind` (`Info`, `Warning`,
`Question`, `Error`), 420 DIPs wide, their OK and Cancel in the app's
language:

- `Alert(parent, kind, title, text)` waits for OK.
- `Confirm(parent, kind, title, text) bool`: OK is true; Cancel and closing
  the dialog are false.
- `InputString(parent, kind, title, label, value) (string, bool)`,
  `InputInt(parent, kind, title, label, min, max, step, value) (int, bool)`
  and `InputSelect(parent, kind, title, label, options...) (string, bool)`
  ask for a value. Enter is OK in `InputString` and `InputInt`; in
  `InputSelect`, whose combo takes Enter to open its list, OK stays
  disabled until a choice.

The file dialogs are the system's: `OpenFile(parent, title, filters...)`,
`OpenFiles`, `SaveFile(parent, title, name, filters...)` and
`SelectFolder(parent, title, dir)` return the path, or `""` (`nil`) when the
user cancels. A `Filter` has a `Name` and `Extensions` without the dot.

## Menus

A `MenuItem` has `Text` (a separator when empty), `OnClick`, which runs on
the event goroutine and so may open a dialog, `Children` for a submenu, and
`Checked` and `Disabled`, and `Icon` (an `*ui.SVG`) or `IconPNG` (a PNG,
32×32 pixels) before the text: `MenuItem{Text: "Copy", Icon: copyIcon,
OnClick: …}`. A menu bar keeps the colors of its SVG icons from when it was
set: on Linux and Windows they do not follow a later change of the theme;
on macOS they do.

- `MenuButton(text).SetMenu(items...)`: a button opening a menu.
- `ContextMenu(items...)` on any widget: the menu of a right click.
- `Form.SetMenuBar(items...)`: the window's menu bar, on Linux and
  Windows. On macOS the menu bar is the app's: set it with
  `mygo.App.SetMenu`.

Menus take their items, not a function building them, because `ui` builds
menus while it draws, under the package's lock: set the items again when
they change.

## Controls of your own

`Custom` makes a control that draws itself with [`ui`](README.md), every
frame it is shown. It has the methods every control has (`SetVisible`,
`OnClick`, `ContextMenu`, the style methods):

```go
badge := widget.Custom(func(c *ui.Context) ui.Element {
	return ui.Text(c, "NEW").Bold().Padding(2, 6).Radius(999).
		Background(c.Theme().Accent).TextColor(c.Theme().AccentText)
})
```

To make a control type of your own, as a WinForms UserControl, embed a
`*widget.CustomWidget` and change the state its build reads within
`widget.Change`, which is safe from any goroutine:

```go
type Gauge struct {
	*widget.CustomWidget
	value float64
}

func NewGauge() *Gauge {
	g := &Gauge{}
	g.CustomWidget = widget.Custom(func(c *ui.Context) ui.Element {
		return ui.Text(c, fmt.Sprintf("%.0f%%", g.value*100))
	})
	return g
}

func (g *Gauge) SetValue(v float64) *Gauge {
	widget.Change(func() { g.value = v })
	return g
}
```

A build runs while the package holds its lock: it must not call a setter
or `Change`, which would wait forever. A control made of other controls
needs no `Custom`: return a `VBox` of them from a function of your own.

## DataGridView

```go
t := widget.DataGridView(func(a *Account) uint { return a.ID },
	widget.Column[Account]{Title: "UID", Text: func(a *Account) string { return a.UID },
		Width: 150, Sortable: true, Filterable: true},
	widget.Column[Account]{Title: "Friends", Text: friends,
		Compare: func(a, b *Account) int { return cmp.Compare(a.Friends, b.Friends) },
		Align: widget.End, Sortable: true},
).CheckboxColumn().IndexColumn().BottomPanel()
```

`DataGridView(key, columns...)` shows rows of any type, each told apart by its
key, for as many as a million rows on `ui.TableView`. The table keeps a
copy of the rows; to share rows between tables, make them pointers.
A `Column` has a `Title`, the `Text` of its cells, and optionally `Compare`
(by `Text` otherwise), `Width`, `MinWidth`, `Align`, `Sortable`,
`Filterable`, `Filter` (case-insensitive containment otherwise), `Color`
(not on a selected row, whose text takes the selection's color) and
`Image`. The column functions and the key run under the package's
lock: they read the row and call no widget method.

- Rows: `SetRows`, `AddRow` and `AddRows` (after the others, or in place
  of the row of the same key), `UpdateRow(k, fn)` (fn changes the row in
  place, not its key), `DeleteRows(ks...)`, `Clear`, `Row(k)`, `Len`.
- `CheckboxColumn()`: `Checked`, `SetChecked(ks, on)`, `IsChecked`,
  `AutoCheck` (unchecks the selected rows when half or more are checked,
  checks them otherwise). The header's box checks the rows a search
  shows. `Selected` and `CountSelected` read the selection. `Checked` and
  `Selected` list the rows shown, in the order shown; deleted rows are
  neither.
- `IndexColumn()`: each row's position as shown, from 1. A header click
  sorts by its column; `ClearSort` shows the rows in the order added.
- `BottomPanel()`: the counts of rows, selected and checked, and a search
  over the `Filterable` columns. `PanelText` sets its words (English by
  default), `SetSearchPlaceholder` its keyword's placeholder, and
  `OnBeforeSearch(fn)` runs before a search from it. `Search(column,
  keyword)` searches from code. The filter runs once per search.
- `OnActivate(fn)` (double click, Enter), `OnKey(key, mods, fn)`,
  `ContextMenu(items...)` (on rows only; read them with `Selected`),
  `OnColumnsChange(fn)` with `SetColumnsLayout(l)` to restore it.
  Cmd/Ctrl+C copies the selected rows, a line each, cells apart by tabs.

A table shows in one window at a time. It fills the room its parent
gives it and scrolls; in a box or a notebook page, `Expand` the boxes and
the notebook holding it, or it gets no room.

## Clipboard, theme and screen

`Clipboard.SetText(s)`, `Clipboard.SetImage(png)` and `Clipboard.Text()`
use the system clipboard. `SetTheme(ThemeSystem/ThemeLight/ThemeDark)` sets
the app's appearance, which every window follows, and `IsDark` reports it.
`ScreenSize()` returns the main display's size in DIPs, once the app runs.

## Styling

Style methods take theme units of 4 DIPs, with half steps: `Padding(2)` is
8 DIPs. Box spacing (`VBox(8)`) and `SetSize` are in DIPs. Colors come from
the theme (`Muted`) or `ui.RGB`.

| Method | gota's classes |
|---|---|
| `Padding(n)`, `PaddingX`, `PaddingY`, `PaddingTop/Right/Bottom/Left` | `p-2`, `px-3`, `py-2`, `pt-3`, ... |
| `Margin(n)`, `MarginX`, `MarginY`, `MarginTop/Right/Bottom/Left` | `mt-3`, `mb-3`, `ml-5`, `m-10`, ... |
| `Align(Start/Center/End/Fill)` across a VBox, `VAlign(...)` across an HBox | `halign-*`, `valign-*` |
| `HExpand()`, `VExpand()`, `Expand()` | `hexpand`, `vexpand` |
| `TextSize(XS/SM/Base/LG/XL/XXL/XXXL)` | `text-xs` ... `text-3xl` |
| `Bold()`, `Italic()`, `Mono()` | `font-bold`, `italic`, `font-mono` |
| `Muted()`, `TextColor(c)` | `text-gray-500`, `text-muted` |
| `Border(width, color)`, `Radius(n)`, `Background(c)` | `border-*`, `rounded-*`, `bg-*` |
| `Suggested()`, `Destructive()` on buttons | `suggested-action`, `destructive-action` |

Every widget also has `SetVisible`, `Visible`, `SetDisabled`, `SetTooltip`,
`OnFocusOut`, `OnClick` (gota's `EventBox`), `ContextMenu`, `Focus`, which
moves the keyboard focus to it, and `AssignTo(&v)`, which keeps a widget
made inside a chain.

## What a widget keeps

Each widget builds under a key of its own, so the caret, selection, scroll
position and open popups of its `ui` elements follow it when its box
reorders or it moves to another box. A widget that is not built (hidden, on
a tab not shown, or out of the window's content for a while) keeps all of
it while your program holds the widget; it shows again as it was, with the
popup of a select, combobox, date or color picker closed. A widget that
had the keyboard focus gets it back, unless another took it meanwhile. Moving a widget to another window starts its state anew,
and so does the second place of a widget shown in two.

## Widgets

Layout:

- `VBox(spacing)`, `HBox(spacing)`: a column or a row. `Add` at the
  natural size, `Pack(w, expand, fill)`, `PackEnd`, `Remove`, `Clear`. A
  widget already in the box is not added twice.
- `ButtonBox()`: a row whose widgets sit at its end, one width each, as a
  dialog's buttons.
- `Grid(colSpacing, rowSpacing)`: `Attach(w, col, row, width, height)`; a
  widget attached to a taken cell replaces the one there, and cells must
  not overlap.
- `GroupBox(title)`: a border and a title around one widget.
- `Card()`: a titled box whose body `ShowBody(false)` hides.
- `FlowPanel(spacing)`: a row that wraps.
- `ScrollPanel(w)`: scrolls a widget larger than its room.
- `Separator()`: a line.
- `TabControl()`: pages under tabs; `AddPage`, `Page`, `SetPage`, `OnSwitch`.
- `Sidebar()`: pages chosen in a list at the side; `Add(name, title, w)`,
  `Remove`, `Selected`, `Select`, `SetMenuWidth`.

Input and display:

- `Label(text)`: a line of text; `Rich(spans...)` shows styled `ui.Span`s
  instead, where gota used Pango markup.
- `Icon(svg, size)`: an SVG icon in the color of the text (`TextColor`
  changes it). There are no named icons: embed the SVGs you need.
- `PictureBox(data, w, h)`: a PNG, JPEG, GIF, WebP or BMP, `SetImage` to change
  it. The same bytes of an image up to 64 KB, as an icon, decode once (the
  last 256 are kept); bytes that are no image leave an empty room.
- `TextBox()`, `MultilineTextBox()`: one line or several of text; `Text`,
  `SetText`, `SetPlaceholder`, `SetEditable`, `OnChange`; `TextBox` also has
  `Password` and `OnEnter`.
- `Button(text)`: `OnClick`, `SetText`, `SetIcon`. `LinkLabel(text, url)`.
- `CheckBox(text)`, `Toggle(text)`, `Switch()`: on or off, with `OnChange`.
- `CheckBoxGroup(row, options...)`, `TagSelect(options...)`: several of
  the options; `Values`, `SetValues`.
- `RadioGroup(row, options...)`, `ComboBox(options...)`: one option; `Value`,
  `SetValue`. An `Option` has the `Value` the program reads and the `Text`
  the user sees. A combo's values must differ (its texts may repeat), and
  it groups options in titled sections with icons: `AddSection(title, png,
  options...)`, `RemoveSection`, `RemoveOption`, `SetOptionText`,
  `SectionLen`, `Selected`, `Unselect`.
- `MenuButton(text)`: see [Menus](#menus).
- `NumericUpDown(min, max, step)`: a number; `Value`, `Int`, `SetValue`.
  `RangeSpin` is two, with text between and after.
- `ProgressBar()`: `SetFraction` (negative for unknown length), `SetText`.
- `DataGridView(key, columns...)`: rows in columns; see [DataGridView](#datagridview).
  `Spinner()`: `Start`, `Stop`.

Windows: `SetTitle`, `SetSize`, `SetIcon`, `SetResizable`, `Maximize`
(before `Show` too), `Unmaximize`, `SetContent`, `SetMenuBar`,
`OnClose(func() bool)` (false keeps the window open; quitting the app asks
it too, and a yes quits again; it asks once at a time), `Close`. A window
that closed shows again with `Show`, and `Native()` is nil while it is not
shown.
`UseHeaderBar(subtitle)`, before `Show`, draws the title bar with `ui` and
returns a `*HeaderBarWidget` with `PackStart` and `PackEnd`. Under a Wayland
compositor that decorates windows (Sway, KWin), the compositor's title bar
shows above the header bar; `ClientDecorations(true)` on the header bar,
before `Show`, makes the header bar the only one, with the desktop's window
buttons at its ends (Sway: floating windows only; a tiled one keeps
Sway's title bar unless its config says `default_border pixel` or `none`).
Hyprland draws no title bar: there the option only adds the window
buttons.
