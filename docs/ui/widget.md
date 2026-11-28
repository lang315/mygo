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
gota's names are listed for code ported from it. What of WinForms' toolbox is
done, pending or dropped is in the [WinForms catalogue](winforms.md).

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
| `Fields` | TableLayoutPanel of labels and controls | `VBox` of a label and a control per row |
| `VBox`, `HBox` | — | `VBox`, `HBox` |
| `Label`, `Button`, `RadioGroup` | Label, Button, RadioButton | same |
| `Alert`, `Confirm` | MessageBox (OK; Yes and No) | `Alert`, `Confirm` |
| `Custom` | UserControl | — |
| `MenuStrip` (`Form.MenuStrip`) | MenuStrip | `MenuBar` |
| `MenuBar` (in the content) | MenuStrip docked in the form | — |
| `ToolStrip` | ToolStrip | `Toolbar` |
| `Ribbon` | DevExpress RibbonControl | — |
| `StatusStrip` | StatusStrip | — |
| `NotifyIcon` | NotifyIcon | — |
| `Invoke` | Control.Invoke | `IdleAddSync` |
| `BeginInvoke` (`Post`) | Control.BeginInvoke | `IdleAdd` |

## Threads

- **Setters and getters** are safe from any goroutine, but not from a
  `Custom` build (see [Controls of your own](#controls-of-your-own)). A
  setter returns at once; the windows redraw on the main thread.
- **Handlers** (`OnClick`, `OnChange`, `OnClose`, ...) run one at a time on
  a goroutine of the package, the event goroutine, never on the main
  thread. A handler may block: the window keeps drawing.
- **`OnChange` is for the user.** A setter (`SetText`, `SetChecked`, `SetOn`,
  `SetActive`, `SetValue`, `SetValues`, `SetSelectedIndex`, a `ComboBox`'s
  `Unselect`) does not run the `OnChange` of the control it sets,
  nor does `Reload` or a binding; only an edit by the user, or by a test
  driving the control as the user does, does. This holds for `TextBox`,
  `MultilineTextBox`, `MaskedTextBox`, `CheckBox`, `Switch`, `Toggle`, `RadioButton`,
  `RadioGroup`, `CheckBoxGroup`, `ComboBox`, `NumericUpDown`, `RangeSpin`,
  `TrackBar`, `DomainUpDown`, `DateTimePicker`, `MonthCalendar`, `TagSelect`,
  `RangeTrackBar`, `Rating`, `CheckButton`, `Segmented` (each pinned by its
  own `...SetterDoesNotFireOnChange` test)
  and a toggle `ToolButton` (`TestSettersDoNotRunOnChange` pins it), for
  `TokenEdit`'s `SetTokens` (its own tests pin it), for `SearchBox`'s
  `SetText`, `Accordion`'s `SetOpen` and `ColorEdit`'s `SetColor` (theirs
  do), and
  for `TabControl`'s `OnSwitch` after `SetPage`, `InsertPage` and `RemovePage`
  (but a page index the control had to clamp is reported). WinForms and GTK
  differ, as their `TextChanged` and `changed` also run for code: a handler
  that reacts to a setting's change must also run where the code changes it.
  From gota: a combo's `changed` ran for a choice the code made, as GTK's
  does, and `SelectedIndexChanged` does in WinForms; here `SetValue` and
  `Unselect` leave `OnChange` alone, as every control's setter does, so
  call the handler after them where it must react.
- **`BeginInvoke(fn)`**, or `Post(fn)`, runs `fn` on the event goroutine
  after what was posted before it, without waiting. It works from anywhere,
  the main thread too.
- **`Invoke(fn)`**, or `Update(fn)`, runs `fn` on the event goroutine and
  waits for it, as WinForms' `Control.Invoke`, for a worker that must change
  widgets in step with the handlers. **In a handler it runs `fn` at once**:
  it never waits for itself, so calling it from a handler or a goroutine is
  safe (gota's `IdleAddSync` did deadlock there). Not from a callback that
  runs under the package's lock, such as `UpdateRow`'s function or a
  `RowSource`'s key function: they must call no widget method, and `Invoke`
  would wait for the lock they hold. Before the app runs, or after it quit
  (mygo's `OnQuit`), it returns without running `fn`.
- **A panic in a handler** is recovered on the event goroutine, so the app
  keeps running: in anything run through `Post`, `BeginInvoke` or `Invoke`,
  and in a widget's event (`OnClick`, `OnChange`, a binding's `set`, a menu
  item's click...). It is logged with its stack, or given to
  `widget.OnPanic(func(v any, stack []byte))`, once. `Invoke` also panics
  again, with the same value, in the goroutine that called it, as WinForms
  hands its caller the exception.
- **Blocking calls** (`Invoke`, `Update`, `Dialog.Run`, `Alert`, `Confirm`, the
  `Input` functions and the file dialogs) panic on the main thread, which
  must never wait. In a handler, they keep running the other queued
  handlers while they wait, as GTK runs its main loop under a dialog; on
  another goroutine they just wait.
- **`Timer(interval)`** calls a function at an interval on the event
  goroutine; see [Timer](widget/timer.md).

## Inspector

Every control leaves a note for the [inspector](inspector.md) while it is open:
its Widget tab shows the control's type, whether it is disabled, and the
Styles that style it, layer by layer (those of `Apply`, then those of its own
methods), with each property set and the variants (`hover:`, `focus:`,
`pressed:`, `disabled:`, `dark:`) and their units (padding, margin and radius in
theme units, the rest in DIPs). Its buttons force the hover, pressed and focus
states of the control to see the variants without the pointer. Above the layers
it shows the Style in effect in the state of the frame: each property's winning
value and the layer and variant that set it, and the values it won over, struck
through. The Widgets tab lists the controls as the tree you wrote. The notes are
made only when the inspector is open and not at all in a build with
`-tags mygo_noinspector`. A test reads them with `Tester.Inspect(label)`
(`docs/ui/testing.md`). A Style made by `tw.S` notes its classes (not
public API: `tw` tells `widget` through an internal package), which the tab shows
under the style they made; if a method (`Bold`, `Hover`, …) changed it after, the
line says "then changed by methods". `reflect.DeepEqual` of a `tw.S` Style with a
Style of the typed methods differs by the note: `tw`'s own tests compare without it.

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
A `height` of 0 fits the dialog to its content, as WinForms' `AutoSize`
(`GrowAndShrink`) does at show: `Run` measures the content laid out at `width`
(one headless layout) and the window's page area gets that height, no higher
than the screen's work area less 80 DIPs, where the content scrolls instead,
vertically only: it keeps the dialog's width, so its labels wrap there. A control
wider than the dialog is cut at its right edge, as in a dialog that fits: wrap it
in `ScrollPanel` to reach it.
A `width` of 0 is the window default, 800. The size is taken when the dialog
shows: later changes of its content do not resize it, `SetSize` after
`NewDialog` wins, and a list in the content counts eight rows, a multi-line
text eight lines and a `WebBrowser` 300 DIPs (give the dialog a height to show
more or less). The height is measured at the scale of the display the dialog
opens on (the owner window's, else the primary one); a width narrower than the
content's least width gives the height at that width. The measure is a headless
build before the window exists: it leaves a focus, a scroll or a selection you
asked for before `Run` to the window, and a `Custom` control's build function
should start nothing and post nothing while `c.Measuring()` is true. `NewForm`
keeps 800 by 600 for 0.
`Run` shows it and returns when it closes: by `Close`, by the user (its
close button, or Escape once no drop-down in it is open), or because its
parent closed. For a dialog that returns a value, embed `*widget.Dialog`
in a type of your own whose `Run` returns it. A dialog opened from a
dialog takes the outer one's `Form` as its parent; closing a window
closes the dialogs over it, the innermost first. `SetDeletable(false)` keeps the user from closing it (its
`Close` and its parent still can), and `OnBeforeRun` runs before it shows.
`Parent()` returns the form the dialog was made over (nil for none), also
after it closes, so the code that ran it can tell the outcome there:

```go
d.Run()
if p := d.Parent(); p != nil {
	p.ShowToast(widget.Toast{Status: widget.StatusSuccess, Title: "Account added"})
}
```

A form takes shortcuts of its own, and `AcceptButton`/`CancelButton` for Enter
and Escape; see [Form shortcuts](widget/form-keys.md).

To test a dialog without a window, `TestDialog(t, d)` returns a
[`widget.Tester`](widget/tester.md)
over its form's view (a `DialogTester`: header, tool strip, content and
status strip), and `d.Run` then runs `OnBeforeRun` and blocks until the
dialog closes, as it does with a window, without showing one: call `Run` on
another goroutine, click its buttons through the tester, and `Closed` tells
whether a button, `Close` or `Escape` closed it. A `Run` after a close is a
new show. The test's end closes the dialog, so no `Run` outlives it.
`Escape` presses the key as a window does: an open drop-down or a shortcut
takes it, else it is a request to close, which asks `OnClose` (on the event
goroutine: wait for it before `Closed`) and closes the dialog unless that
says no or `SetDeletable(false)`. A `Close` before the dialog first shows,
or in `OnBeforeRun`, makes that `Run` return at once, with a window as with
a tester. A `TestForm` tester's `RequestClose` asks to close the form as its
close button does: it settles, so `OnClose` has answered, and reports whether
the form closed. An `OnClose` that runs a dialog is still asking then.

```go
dt := widget.TestDialog(t, d.Dialog)
go func() { result <- d.Run() }()
dt.Type("Ada")
dt.Click("OK")
got := <-result
```

Ready-made dialogs, as a WinForms `MessageBox` looks: the window's caption
shows the title (the dialog shows it itself only where there is no caption:
a dialog with a parent on macOS, which is a sheet, and in browsers), the
window cannot be minimized, maximized or resized where the platform
honours that (a macOS window without a parent still draws its yellow and
green buttons, disabled; Linux may still show minimize), and the height
fits the text at 420 DIPs wide; `InputSelect`'s fits too, but is no lower
than 240 for its open list. Each has a filled icon for its `Kind` (`Info`,
`Warning`, `Question`, `Error`), drawn the same on every platform and
theme. Their buttons are in the app's language:

- `Alert(parent, kind, title, text)` waits for OK.
- `ColorDialog()`: a color picker with OK and Cancel; see [ColorDialog](widget/colordialog.md).
- `FontDialog()`: families, styles, a size and a sample, with OK and Cancel; see [FontDialog](widget/fontdialog.md).
- `Confirm(parent, kind, title, text) bool`: Yes is true; No, which is the
  default (it has the focus, and Enter answers it), Escape and closing the
  dialog are false.
- `InputString(parent, kind, title, label, value) (string, bool)`,
  `InputInt(parent, kind, title, label, min, max, step, value) (int, bool)`
  and `InputSelect(parent, kind, title, label, options...) (string, bool)`
  ask for a value. Enter is OK in `InputString` and `InputInt`; in
  `InputSelect`, whose combo takes Enter to open its list, OK stays
  disabled until a choice.

A message box waits for the user. To tell only an outcome ("12 accounts
exported"), show a toast instead: `Form.ShowToast(widget.Toast{...})` shows it
near the bottom of the window for a few seconds, with a status, and an action
such as Undo; see [Toasts](widget/toast.md). Keep message boxes for a question
or an error the user must read.

The file dialogs are the system's: `OpenFile(parent, title, filters...)`,
`OpenFiles`, `SaveFile(parent, title, name, filters...)` and
`SelectFolder(parent, title, dir)` return the path, or `""` (`nil`) when the
user cancels. A `Filter` has a `Name` and `Extensions` without the dot.

## Menus

A `MenuItem` has `Text` (a separator when empty), `OnClick`, which runs on
the event goroutine and so may open a dialog, `Children` for a submenu, and
`Checked` and `Disabled`, and `Icon` (an `*ui.SVG`) or `IconPNG` (a PNG,
32×32 pixels) before the text: `MenuItem{Text: "Copy", Icon: copyIcon,
OnClick: …}`. `Shortcut` (`widget.Shortcut{Mods: ui.Ctrl, Key: ui.KeyC}`)
shows a key at the item's end and answers nothing: give the form the same
`Form.Shortcut`. The menus of `MenuBar`, `MenuButton` and `ContextMenu`
show it; the native bar of `Form.SetMenuBar` does not, as a key there is
the system's to answer (use `MenuStrip`'s `SetShortcut`). Only an item
`Checked` is a check box item; the others are plain items, as in the
system's menus and WinForms: a toggle not checked is a plain item, and a
menu with no item checked has no column for the mark. A menu bar keeps the colors of its SVG icons from when it was
set: on Linux and Windows they do not follow a later change of the theme;
on macOS they do.

- `MenuButton(text).SetMenu(items...)`: a button opening a menu.
- `MenuBar(menus...)`: a menu bar that mygo draws in the content, where the
  content puts it, as under a `HeaderBar`; see [MenuBar](widget/menubar.md).
- `ContextMenu(items...)` on any widget: the menu of a right click.
- `Form.SetMenuBar(items...)`: the window's menu bar, on Linux and
  Windows. On macOS the menu bar is the app's: set it with
  `mygo.App.SetMenu`.

**Destructive items.** `MenuItem{Text: "Delete 3 accounts", Destructive: true,
OnClick: …}` marks an item that destroys. The native menus of the system have
no role for it (AppKit's, GTK's and Win32's have none, as UIKit's does), so
only where mygo draws the menu itself does the item look different:

| Platform | What the item shows |
|---|---|
| Windows 11 (popup menus, and the menus under a menu bar's items) | The danger color, `#C42B1C` (`#FF99A4` in the dark theme), and semibold (weight 600). It keeps the danger color when highlighted, and is grey, still semibold, when disabled. In high contrast, which has the system's colors only, it is semibold in the text color. |
| Windows 10, and the menu bar itself on Windows | A normal item: these are Win32's own menus. |
| macOS | A normal item: `NSMenu` draws it. |
| Linux | A normal item: GTK draws it. |
| Web | The danger color (`#c42b1c`, `#ff99a4` in a dark page) and semibold; white on a red highlight; the text color, faded, when disabled. |

`mygo.MenuItem` has the same `Destructive` field, and `SetDestructive` changes
it while the menu shows; so does a `MenuStripItem`'s `SetDestructive`.

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

The methods every control has (`SetVisible`, `AssignTo`, ...) return the
`*widget.CustomWidget`, not your type: chain your own setters apart from
them.

A build runs while the package holds its lock: it must call no method of
any control (getters lock too) and no `Change`, which would wait forever.
A control made of other controls needs no `Custom`: return a `VBox` of
them from a function of your own.

## App shell

A window's shell is its main menu, a tool bar above the content, a status
bar below it, and an icon in the notification area. Each is set once and
changed while the app runs, from any goroutine.

```go
status := widget.StatusLabel("Ready")
open := widget.StripItem("Open…").SetShortcut("CmdOrCtrl+O").OnClick(openFile)

w.MenuStrip(widget.NewMenuStrip(
	widget.StripRole(mygo.RoleAppMenu), // macOS: the app menu
	widget.StripItem("File", open, widget.StripSeparator(), widget.StripRole(mygo.RoleQuit)),
	widget.StripRole(mygo.RoleEditMenu),
))
w.ToolStrip(widget.ToolStrip(
	widget.ToolButton("").SetIcon(saveIcon).SetTooltip("Save").OnClick(save),
	widget.ToolButton("Bold").SetToggle(true).OnChange(bold),
	widget.Separator(),
	widget.ComboBox(options...), // any ComboBox, sections and icons included
	widget.Spring(),
	widget.DropDownButton("More", widget.MenuItem{Text: "Export", OnClick: export}),
))
w.StatusStrip(widget.StatusStrip(status, widget.Spring(), widget.StatusProgress().SetFraction(0.4)))

open.SetEnabled(false)  // the native menu follows
status.SetText("Saved") // and the window redraws
```

- **`MenuStrip`** is the window's main menu on mygo's native menus: on macOS
  the app's menu bar, which is **the focused form's strip** (a form's strip
  takes it when its window gets the focus, or when the form is shown and
  none is taken; a form not shown takes nothing, and `MenuStrip(nil)` on a
  form that does not hold it changes nothing; when the holder closes the
  next form with a strip takes it, else the app's own menu comes back; put
  the `Role` items you want, as the app and Edit menus, in a strip); on
  Windows and Linux the window's own, above the content, the last of
  `SetMenuBar` and `MenuStrip` (`MenuStrip(nil)` removes a strip and leaves
  a bar `SetMenuBar` set). A `MenuStripItem` is
  `StripItem(text, children...)`, `StripCheck(text, checked)` (it toggles
  before its `OnClick` runs), `StripSeparator()` or `StripRole(role)`. Its
  `SetText`, `SetShortcut`, `SetEnabled`, `SetVisible` and `SetChecked` update
  the native menu; `OnClick` runs on the event goroutine. `SetMenuBar`, of
  plain `MenuItem` values you set again to change, stays for a menu that does
  not change. A form with a `HeaderBar` shows no such bar on Windows (Alt
  and F10 open its menus in a popup) and has GTK's above the header on
  Linux: for menus under the header, draw a [`MenuBar`](widget/menubar.md)
  in the content.
- **`StatusStrip`** holds `StatusLabel(text)` (`SetText`, `SetIcon`),
  `Spring()` (takes the free width), `StatusProgress()` (a small
  `ProgressBar`) and `Separator()`. `Form.StatusStrip` docks it below the
  content.
- **`ToolStrip`** is a `ui.Toolbar`: its buttons take no face until hovered,
  it is one stop of Tab with the arrow keys moving among its items, and
  what does not fit goes from the last item into a menu at its end (a
  `ComboBox` there under its chosen text, a `DropDownButton` under its own);
  there is no overflow of the package's own. `ToolButton(text)` has `SetIcon`
  and `SetIconPNG` (as `Button`'s), `SetTooltip` (from every control, which
  names an icon-only button), `OnClick`, and `SetToggle` with
  `Checked`, `SetChecked` and `OnChange`, and `Suggested()` and
  `Destructive()`, which look as a `Button`'s do (the accent, or the danger
  color, with the same hover and pressed looks, in both themes). In the
  overflow menu a `Destructive()` button is a destructive item (see
  *Destructive items* above); a `Suggested()` one is a normal item, since
  menus have no suggested role. `Form.ToolStrip` docks it above the
  content, below the header bar and the window's menu bar.
- **`NotifyIcon`** is an icon in the menu bar or the notification area:
  `NewNotifyIcon()` makes one, `SetIcon(png, template)`, `SetText` (the
  tooltip), `SetContextMenu(items...)`, `OnClick`, `OnRightClick`,
  `SetVisible` (a hidden icon comes back with all it had; it returns the
  error if the system cannot make the icon, as of an image it cannot decode
  that was set meanwhile) and `Dispose`. A click queued before the icon was
  hidden or disposed runs nothing.
  Gaps are the platforms': `OnClick` runs only when the icon has no context
  menu, since a click opens the menu; there is no `OnDoubleClick`, as no
  platform's tray tells mygo of one; on Linux the panel places the icon.
  Create it once the app is ready.

## Fields

`Fields()` lays out a form: a row is a label and the control it names. By
default the labels make a column as wide as the widest of them, each at
the middle of its control's height, and the controls fill a second column,
as in WinForms or GTK. Assistive technology reads each control by its
row's label (a control with a name of its own, or a text that names it, as a
check box's, keeps it; a group of controls is a group named by the label; a
hidden label names nothing):

```go
widget.Fields().
	Row("Name", widget.TextBox().AssignTo(&name)).
	Row("Email address", widget.TextBox().AssignTo(&email)).
	Row("Country", widget.ComboBox(options...)).
	Gap(8, 12)
```

- `Stacked()` puts each label above its control, over the form's width.
- `Gap(row, col)` is in DIPs: between rows, and between the two columns (or,
  stacked, between a label and its control). The theme's default is 2 units
  between rows and 3 between the columns, 1 between a stacked label and its
  control.
- The labels are real `*LabelWidget`s, styled like any: `Label(i)` returns
  row i's, or nil if there is no row i. A label's own `Align` and `VAlign`
  place it in its cell, and a style that gives it a background draws it as
  wide as its text.
- Hiding a row's control (`SetVisible(false)`) hides its label and the
  row's gap too, and showing it brings them back. A hidden label alone
  keeps its cell empty, so the control stays in its column.
- A row without a control, or with one the form already has, is not added.
- `Section(title)` adds a bold heading across both columns before the rows
  that follow, up to the next `Section`. All the rows of a form share one
  label column, whatever their section: use it instead of one `Fields` per
  group, whose label columns each fit their own labels. A section after the
  first has more room above it than a row has, the first none. A section whose
  rows are all hidden hides its heading and that room. The sections share the
  control column too (in `SettingsRows` each card has its own), and the
  labels stay left-aligned, as before. The heading
  is a heading to screen readers and takes no focus. In `SettingsRows` each
  section is a card, its heading above it. There is no divider option.
  `Section("")` is a group break: the room a section has above it, with no
  heading and no node for screen readers. It adds none first, nor when its
  rows are all hidden, and in `SettingsRows` it starts a card with no heading.
- `Describe(text)` gives the last row added a description: a line in
  `TextMuted` under its label (stacked, between the label and the control),
  which wraps. Screen readers read it as the control's description, before
  its tooltip. `Describe("")` takes it away; before any row it does nothing.
- `Wide()` makes the last row added a wide one: its label (and description)
  across the form, and its control under them, as wide as the form, or as
  the inside of its card in `SettingsRows`, unless its `Align` says
  otherwise. Use it for a list box, a multiline text box, a table, or a link
  or a button among ordinary rows. The gap under the label is a stacked
  row's (`Gap`'s column value when set). A wide label does not widen the
  label column of the other rows, and a wide row with an empty label has no
  label at all. Stacked rows are all wide, so there it changes nothing;
  before any row it does nothing.
- `Add(c)` adds such a row, wide with no label, as `Row("", c).Wide()` does:
  `c` across the form (in `SettingsRows`, across the card's inside), for a
  note, a check box or a button among the rows. `Describe` and `Action` then
  apply to it as to any row, and it counts in `Label(i)` (its label is
  empty). As `Row`, it adds nothing for a nil control or one already in the
  form, and then leaves the last row as it was.
- `Action(c)` gives the last row added one small action control, such as a
  `LinkLabel` or a flat button. In a wide or stacked row it sits at the right
  end of the label's line, in its own width, centred on the label, which
  takes the rest and wraps before it reaches the action; in an ordinary row
  (and a settings row) it follows the control, `Gap`'s column value after it.
  The label still names the control alone, its access key focuses the control
  (or a container's first control), never the action, and the action comes
  in Tab order where it is drawn: before a wide row's control, after an
  ordinary row's. Hiding the row's control hides the action too; disabling
  the form disables it. A second call replaces it, `Action(nil)` removes it,
  a control already in the form (but this row's own action) is not taken,
  and before any row it does nothing.

```go
widget.Fields().
	Row("&Content", widget.MultilineTextBox()).Wide().
	Action(widget.LinkLabel("How to write it", "").OnClick(showHelp)).
	Row("Kind", kindBox).Action(widget.Button("+"))
```

```go
widget.Fields().
	Section("Account").
	Row("Name", nameBox).
	Row("Email address", emailBox).
	Section("Privacy").
	Row("Country", countryBox)
```

`Fields` is a control like the others: `Apply` a style, `Align` it in a
box, and put it in a `GroupBox`, a `Card` or a dialog.

From gota, a form of `VBox`es and `BoxItem`s, a label with `halign-start`
and its control in each:

```go
// before
widget.VBox(4).Add(widget.Label("Name").Align(widget.Start), nameBox)
// after: one row, and the form lines every row up
widget.Fields().Stacked().Row("Name", nameBox)
```

`SettingsRows()` lays the form out as a page of settings, as Windows 11's
Settings does: each section is a card (the skin's `SettingsCard` look, else
its `Card` look: `Surface`, a line of `Border`, the card radius and shadow),
its heading above it, outside the card. Rows before the first `Section` make a
card with no heading, and so does each `Section("")`. In a card the label
takes the width the control leaves and wraps, the control keeps its own size
at the right edge (its `Align` overrides), and a divider separates each row
from the next. Dividers are always on. A hidden control takes its label and
its divider with it, so there is never a double or a trailing one, and a
section whose rows are all hidden takes its card and heading away. The row
gap applies on each side of a divider. Each card measures its own columns, so
a control that asks for `Start` lines up with the widest control of its card.
Of `Stacked` and `SettingsRows`, the last call wins. `Describe` adds a line
under a label, and `Wide` puts a row's control under its label, across the
card, with dividers as for any row. With a skin each row is at
least the skin's `SettingsRowHeight`; the page behind the cards reads best in
the skin's `RoleChrome`, which is the app's choice.

**Changed:** `SettingsRows` drew one list of rows with headings between them;
it now draws a card per section, inside the form's width, so an existing
settings page looks different (its controls move in by the card's padding).
An app that draws its own card around `SettingsRows` now draws a card inside
a card: drop it, and use `Section` for each card, with `Wide` rows for the
lists, multiline boxes, links and buttons that made it draw its own.

A control keeps its own size, centered on a wrapped label. A `ProgressBar` and
an empty `ListBox` get no width in the fit column, and a `ListBox` with items or
a table a narrow one: give them `SetSize`, or make their row `Wide` for
multi-line boxes and tables. The label column is no narrower than its longest word, so in a
narrow window the controls are pushed toward or past the right edge, by a long
word and by ordinary ones alike (a `ComboBox` beside "Notifications" needs about
229 DIPs): give the window a `SetMinimumSize` wide enough. The cards keep
their height: free height stays below the last. In a `ScrollPanel` use `.Vertical()`:
the both-ways panel gives the form no width to wrap at.

```go
settings := widget.Fields().SettingsRows().
	Section("Account").
	Row("Name shown to others", widget.TextBox()).
	Describe("Others see it beside your messages.").
	Row("Blocked words", widget.MultilineTextBox()).Wide().
	Describe("One a line.").
	Section("Notifications").
	Row("Sounds", widget.Switch())
page := widget.VBox(0).Add(settings).Apply(widget.NewStyle().Background(widget.RoleChrome).Padding(4))
```

## Styles

A `widget.Style` is a set of properties, made once and shared by as many
controls as you like, the way a CSS class is. `Apply` gives a control its
styles; a later one wins, and the control's own methods (`Padding`,
`Bold`, …) win over them all:

```go
card := widget.NewStyle().Padding(4).Radius(3).
	Background(widget.RoleSurface).Shadow(0, 4, 12, 0, widget.Fixed(ui.RGBA(0, 0, 0, 0.12))).
	Hover(widget.NewStyle().Background(widget.RoleSurfaceHover)).
	Pressed(widget.NewStyle().Background(widget.RoleSurfacePressed)).
	Dark(widget.NewStyle().Shadow(0, 4, 12, 0, widget.Fixed(ui.RGBA(0, 0, 0, 0.4)))).
	Transition(150 * time.Millisecond)

widget.VBox(8).Add(
	widget.Label("Inbox").Apply(card),
	widget.Label("Drafts").Apply(card),
	widget.Label("Sent").Apply(card).Bold(),
)
```

- **Units.** Padding, margin and radius are in theme units, the theme's
  `Spacing` (4 DIPs): Tailwind's scale, where `p-4` is `Padding(4)`.
  Border widths, sizes, shadows and blur are in DIPs.
- **Colors** are `widget.Fixed(c)`, `widget.Hex("#2563eb")`, or a role of
  the theme, `widget.RoleSurface`, `RoleText`, `RoleAccent`…, which
  follows it, dark included.
- **Skin roles** name the colors a skin decides (package `skin`, see
  [Skins](skin.md)), so an
  app's own controls draw in the skin's colors: `RoleChrome` (the frame's
  ground: strips, the header bar), `RoleBorderStrong` (headers' bottoms, the
  window's edge), `RoleControlBorder` (inputs' outline), `RoleButtonBorder`
  (buttons' outline), `RoleRowHover`, `RoleRowSelected` and `RoleInfo` (the
  running and info state). Without a skin each is the plain theme's nearest
  field: `Surface`, `Border`, `ControlBorder` (else `Border`),
  `ButtonBorder` (else as `RoleControlBorder`), `RowHover` (else
  `SurfaceHover`), `RowSelected` (else `Accent`), and `Accent`.
- **Variants** apply on top while their state holds, in this order, a later
  one winning: `Dark`, `Hover`, `Selected`, `Focused`, `Pressed`,
  `Disabled`. A state's variant can have a `Dark` of its own, as Tailwind's
  `dark:hover:`. `Selected` holds while the control is checked or selected:
  a `CheckBox` or `RadioButton` checked, a `Toggle` on, a toggle
  `ToolButton` checked; other controls are never selected. A selected
  control keeps its `Selected` look while hovered.
- **What a style does not set keeps the control's own look**, its own hover
  colors included. A style that sets a background turns that hover off:
  give it a `Hover` too.
- **Transition** fades the background, border and text colors as states
  change. The text color fades only if the style sets a `TextColor` of its
  own besides the variant's, as a style cannot know the color the control's
  text rests in (a button's is the theme's `RoleAccentText`); otherwise the
  text jumps.
- **A Style is a value.** Every method returns a new one and none changes
  an earlier one, so one Style can be shared by controls and goroutines.
  To change a control's look, `Apply` another.
- **One side or corner** (`PaddingTop`, `RadiusTopLeft`, `BorderLeft`, …)
  leaves the control's other sides as its builder set them.
- **Placing and weight.** `Align`, `VAlign`, `HExpand` and `VExpand` place
  the control in its box, and count in the base style only: a variant's are
  ignored. `FontWeight(100…900)` sets the weight, and `Bold` is
  `FontWeight(700)`: the later of the two wins.
- **A skin's looks** (`skin.Looks`, `Skin.WithLooks`, see [Skins](skin.md#looks))
  style each kind of
  control, and a `TabControl`'s tabs and a `DataGridView`'s column headers,
  as Tailwind classes. What styles a control, strongest first: its own
  methods, its `Apply`'d styles, a `TabControl`'s `StripStyle` (its strip),
  the skin's look, the rest of the skin (its strips' `Chrome` and the like),
  and the plain theme. A look's placing is ignored, as a variant's is.

## Tailwind classes

Package `ui/widget/tw` is an opt-in shorthand for Styles, with Tailwind v3's
classes and values. An app that does not import it pays nothing.

```go
import "github.com/egoist/mygo/ui/widget/tw"

save := widget.Button("Save").Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))
name := widget.Label("Name").Apply(tw.S("halign-start font-semibold"))
```

`tw.S` parses a string once and caches it. A class it does not know is
logged once and ignored, and `mygo vet` reports it in a literal string.
`tw.Parse` returns the error instead.

The [cheat sheet](widget/tw.md) lists every class, made from the parser's own
tables, and what to use for the Tailwind classes it does not have.

| Group | Classes |
|---|---|
| Spacing | `p- px- py- pt- pr- pb- pl- m- mx- …` with `0 px 0.5 1 … 96`; `-mt-2` |
| Sizes | `w- h- min-w- max-w- min-h- max-h-` (same scale, 4 DIPs a unit); `max-w-none` lifts the cap a `w-` has at its container's room |
| Radius | `rounded`, `rounded-{none,sm,md,lg,xl,2xl,3xl,full}`, sides `t r b l`, corners `tl tr br bl` |
| Border | `border`, `border-{0,2,4,8}`, `border-{x,y,t,r,b,l}[-N]`, `border-solid`, `border-dashed` |
| Colors | `bg- text- border-` with the v3 palette (`blue-600`), `black white transparent` |
| Theme colors | `default muted surface surface-hover surface-pressed accent accent-hover accent-pressed on-accent danger warning success selection focus inverse on-inverse` |
| Text | `text-xs … text-9xl`, `font-thin … font-black`, `italic`, `not-italic`, `font-mono`, `truncate` (a `Label` on one line, ending in an ellipsis; it shrinks below its text, whose whole is still its name) |
| Effects | `shadow-sm shadow shadow-md lg xl 2xl inner none`, `opacity-0 … 100`, `cursor-pointer …` |
| Layout (gota) | `halign-/valign-{start,center,end,fill}`, `hexpand`, `vexpand` |
| Buttons (gota) | `suggested-action`, `destructive-action` |
| Motion | `transition`, `duration-{75,…,1000}` |

Variants: `hover:`, `active:`, `focus:`, `disabled:`, `dark:`. `dark:` stacks on
one other (`dark:hover:bg-slate-700`). On the same property the later class
wins; Tailwind decides by its CSS order instead. Palette colors are fixed in
both themes; theme colors follow light and dark. Layout classes (`halign-*`,
`hexpand`, …), `transition` and `duration-*` are errors in a variant, and so is
`not-italic`: it clears an `italic` earlier in the same string, but a
variant cannot clear what the base sets, so `dark:not-italic` would do
nothing.

Not supported: `m-auto`, `bg-blue-600/50`, `gap`, `grid`, `flex`, gradients,
blur, arbitrary values (`p-[13px]`), `ring`, `current`.

From gota: `combobox-rounded-r-none` is `rounded-r-none`; gota's palette is Tailwind v2's, so its `green`, `yellow` and
`purple` are `emerald`, `amber` and `violet` here; `text-green` with no
shade and `text-italic` have no effect in gota (it defines `.italic` only); `gota-card`, `gota-table` and `gota-tag-select*` are
widget controls, not classes ([the cheat sheet](widget/tw.md#from-gota)).

## Binding

`widget.Bind` ties a control to a value kept elsewhere, such as a config,
so a form needs no pair of `SetValue` and `OnChange` for each setting:

```go
// before
name := widget.TextBox().SetText(config.Get("name")).
	OnChange(func() { config.Set("name", name.Text()) })

// after
name := widget.Bind(widget.TextBox(),
	func() string { return config.Get("name") },
	func(v string) { config.Set("name", v) })
```

| Control | Value | |
|---|---|---|
| `TextBox`, `MultilineTextBox` | `string` | `Bind` |
| `SearchBox` | `string` | `Bind` |
| `ButtonEdit` | `string` | `Bind` (a click on its Clear button is an edit: `set("")`) |
| `FontEdit` | the family, a `string` | `Bind` (a pick of the user is an edit) |
| `DomainUpDown` | the selected item's text, a `string` | `Bind` |
| `CheckBox`, `CheckButton` | `bool` | `Bind` |
| `Rating` | the stars filled, an `int` | `Bind` (clamps to 0 through its stars) |
| `ComboBox` | the `Value` of its option, a `string` (the text, if `Editable` first) | `Bind` |
| `NumericUpDown` | `float64` | `Bind`; `BindInt` for an `int` |
| `CalcEdit` | `float64` | `Bind` (a typed number and the calculator's Enter or = are edits) |
| `TrackBar` | `float64` | `Bind` (clamps to its range, as `NumericUpDown`) |
| `TimeSpanEdit` | `time.Duration` | `Bind` (clamps to its range, as `NumericUpDown`) |
| `RangeSpin` | its two numbers, `float64` | `BindRange(c, get, set)`, with `get func() (lo, hi float64)` and `set func(lo, hi float64)`: `set` gets both numbers after an edit of either; each is clamped to its range, as `NumericUpDown` |
| `CheckBoxGroup` | the `Value`s of the checked options, a `[]string` | `BindValues(c, get, set)`: `set` gets them in option order; a value of `get()` that is no option's is ignored, and absent from the next `set` |
| `TokenEdit` | its tokens, a `[]string` | `BindTokens(c, get, set)`: takes `get()`'s tokens as `SetTokens` does (trimmed, blanks and repeats dropped); `set` gets the tokens in order after each change by the user, in a slice of its own; with `SetSuggestionsOnly`, a refused entry calls no `set` |
| `ComboBox` of a yes and a no | `bool` | `BindBool(c, yes, no, get, set)`: gives the combo the two options and makes it a plain choice, not `Editable`; `set` is called for yes and no only |

- The control takes `get()`'s value at once. After each edit of the user,
  `set(v)` runs with the new value on the event goroutine, after the
  control's own `OnChange` (`Bind` does not replace it), and off the
  package's lock, so `set` may call the control's getters.
- **A program setter does not call `set`.** `SetText`, `SetChecked`,
  `SetValue`, `SetValues` and `SetTokens` change what the control shows, not the source: the program
  already knows the value it gave. This keeps a binding from looping.
- `widget.Reload(c)` takes the source's value again, for a source that
  changed elsewhere; it does not call `set`, and does nothing for a control
  that is not bound. Call it where the source changes, or when the window
  shows again.
- A `ComboBox` whose options have no value `get()` returned stays empty, as
  `SetValue` leaves it; set its options before binding, or `Reload` after.
  A `NumericUpDown` clamps `get()`'s value to its range. Neither is written
  back to the source.
- `set` gets the value the user entered, taken as the edit is committed: an
  `OnChange` that normalizes the text with `SetText`, or a worker that sets
  the control before `set` runs, does not change what `set` receives.
- `get` runs on the goroutine that calls `Bind` or `Reload`, and `set` on the
  event goroutine, so both must be safe to call at the same time (a mutex
  around the config, as a config store has).
- Binding a control again replaces its binding: the last wins, and an edit
  already committed is written through the binding of that moment.
- No reflection: `get` and `set` are plain functions, so a binding to a
  field, a map or a config store is the same few lines.

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
copy of the rows, in a `RowSource` of its own; to show the same rows in
several tables, make one with `NewRowSource(key)` and give it to each
table with `DataGridViewOf(src, columns...)`. Each table has its own sort,
search, selection and checks; a change to the source (`SetRows`, `AddRows`,
`UpdateRow`, `DeleteRows`, or the same on any table over it) reaches them
all and redraws their windows, and a table not on screen reads it when it
next shows. One slice and one index serve the lot; a table not drawn since
`SetRows` or `DeleteRows` keeps the old slice alive (its frame and view
hold it) until it draws again.
A `Column` has a `Title`, the `Text` of its cells, and optionally `Compare`
(by `Text` otherwise; `IntColumn(title, width, get)` makes a number's column,
sorted by value), `Width`, `MinWidth`, `Align`, `Sortable`,
`Filterable`, `Filter` (case-insensitive containment otherwise), `Spans` (draws
`[]ui.Span` in the cell, as `Label.Rich` does, for a badge or a colored word:
`Text` still sorts, searches, copies and names the row, while assistive
technology reads the cell as its spans' text; a selected row's span text takes
the selection's color and keeps the spans' backgrounds, which copies the spans
of each cell drawn, so keep them with the rows if you can), `Color`
(not on a selected row, whose text takes the selection's color) and
`Image`. The column functions and the key run under the package's
lock: they read the row and call no widget method.

Statuses and formats, as an account list shows them:

```go
state := widget.Column[Account]{Title: "Trạng thái", Text: func(a *Account) string { return a.State },
	Badge:       func(a *Account) widget.Status { return a.Status },  // the Text as a badge; StatusNone: plain Text
	BadgeDetail: func(a *Account) string { return a.Step },           // after a StatusRunning pill, in TextMuted
	Format: func(a *Account) widget.CellFormat {                       // this column's cells
		if a.Expired {
			return widget.CellFormat{Status: widget.StatusWarning}
		}
		return widget.CellFormat{}
	}}
t := widget.DataGridView(key, uid, state).
	SetRowFormat(func(a *Account) widget.CellFormat { // every cell of the row
		if a.Banned {
			return widget.CellFormat{Status: widget.StatusDanger, Bold: true}
		}
		return widget.CellFormat{}
	}).
	SetEmptyState(widget.EmptyState("Chưa có tài khoản", "Thêm một tài khoản để bắt đầu."))
```

- `Badge` draws the cell's `Text` as a [badge](widget/badge.md) of the status
  it returns, 20 DIPs high, which fits a Compact row; `BadgeDetail` is a
  running badge's step, called only for `StatusRunning`. The cell is read as
  the text, then the step; it sorts, searches and copies as `Text` alone. On a
  selected row the pill keeps its colors.
- A `CellFormat` styles a cell: `Status` gives its family's `Subtle` fill and
  `Text` color, `Fill` and `Text` override them, and `Bold` makes the text
  bold. The fill covers the cell's whole box, to its end line with
  `SetGridLines`. A column's `Format` wins over `SetRowFormat` field by field,
  and `Format.Text` over `Column.Color`. A selected row shows the selection,
  not the format. Under HighContrast a status draws no fill: its family's
  text, bold. The `#` column of `IndexColumn` takes the row's format; the
  check box column does not.
- The functions run as a row is built: a row kept from frame to frame does
  not call them again until it changes (`UpdateRow`), and a frame at rest
  allocates nothing for them. `SetRowFormat(nil)` removes the row's format.
- `SetEmptyState(e)` draws an [empty state](widget/emptystate.md) over the
  body, under the headers, which still sort, while no row shows (none, or
  none the search keeps); `nil` removes it.
- A grid without them builds what it did, but every app with a grid links
  the code of badge cells and formats (about 20 KB on darwin): the table
  must look at its columns' fields, so no call of the app's own tells it
  apart. `SetEmptyState` costs nothing until it is called.

Groups and a row of totals, as a manager of many profiles shows them:

```go
status := widget.Column[Profile]{Title: "Status", Text: func(p *Profile) string { return p.Status },
	Summary: func(rows iter.Seq[*Profile]) string { // the rows shown, after the search
		running, failed := 0, 0
		for p := range rows {
			switch p.Status {
			case "running":
				running++
			case "failed":
				failed++
			}
		}
		return fmt.Sprintf("%d running · %d failed", running, failed)
	}}
t := widget.DataGridView(key, name, status).
	SetGroupBy(func(p *Profile) string { return p.Status }). // a header "running (312)" over each group
	SetGroupOrder("failed", "running")                       // these first, the others by title
```

- `SetGroupBy(fn)` puts the rows under a header "Title (n)" for each title
  `fn` returns, as `ListView.SetGroupBy`: the groups by title, or first those
  `SetGroupOrder` names, in its order. Within a group the rows keep the sort
  and the search, and a group the search empties shows no header. A click on
  a header selects nothing; `Selected` and `Checked` return the keys in the
  order shown. `SetGroupBy(nil)` shows no groups. `fn` runs once for each
  row when the rows or the groups change, and for each row changed in place.
  A `ListView` groups with its own `SetGroupBy`, not its `Details()`'.
- A column's `Summary` gives it a cell in a row of totals under the table.
  It gets the rows shown (after the search, in the order shown) and returns
  the cell's text; the cells of columns without one, and of the `#` column,
  stay empty. It runs when the rows shown change (a row added, changed in
  place or deleted, a search, a sort), not in a frame at rest, once per
  change for all the frames that follow, under the package's lock: it calls
  no widget method. A `Summary` that panics leaves its cell empty and goes to
  `OnPanic`. Each cell sits under its column's header and is as wide, and
  follows it when the user resizes, moves or scrolls the columns, or a
  column is hidden. The row takes the header's text size and the skin's
  `GridHeader` look.
- A grid without a `Summary` builds what it did, but every app with a
  `DataGridView` links the row's code (about 17 KB of a release file on
  darwin, 11 KB of it code), as it does that of badges and formats: the
  table must look at its columns' fields, so no call of the app's own tells
  it apart.

Totals for each group, in the groups' header lines:

```go
runs := widget.IntColumn("Runs", 80, func(p *Profile) int { return p.Runs })
runs.GroupSummary = func(rows iter.Seq[*Profile]) string { // one group's rows shown
	total := 0
	for p := range rows {
		total += p.Runs
	}
	return strconv.Itoa(total)
}
t := widget.DataGridView(key, name, status, runs).
	SetGroupBy(func(p *Profile) string { return p.Status }) // "running (312)" … "4210" under Runs
```

- A column's `GroupSummary` gives each group's header line a total under the
  column, as `Summary` gives the table one under it. It gets the group's rows
  shown (after the search and the column filters, in the order shown) and
  returns the text; an empty text shows nothing. Each total sits under its
  column's header and is as wide, aligned as the column, and follows it when
  the user resizes, moves or scrolls the columns; a hidden column's totals and
  the `#` column show none. The group's title starts the line and runs on
  under the columns its text reaches, where a total would cover it: give
  `GroupSummary` to columns past it (not the first). A line pinned at the
  top while the rows scroll keeps its totals under their columns too.
- It runs when the rows shown change (a row added, changed in place or
  deleted, a search, a filter, a sort, a new grouping), once for each group,
  never in a frame at rest, under the package's lock: it calls no widget
  method. Without `SetGroupBy` it does not run. A `GroupSummary` that panics
  leaves its texts empty and goes to `OnPanic`.
- A grid without a `GroupSummary` builds what it did, but every app with a
  `DataGridView` links the totals' code (about 0.5 KB of a release file on
  darwin, 1.9 KB unstripped), as it does that of `Summary`.

Dragging rows, to put them in order or onto another control:

```go
queue := widget.DataGridView(key, name, status).SetRowReorder(true) // drag rows to another place
queue.OnRowsReordered(func() { save(queue.Rows()) })                // after the user moved them
done := widget.DataGridView(key, name, status).
	DragRows(func(rows []*Job) any { return rows }) // drag the chosen rows out
bin := widget.Label("Drop jobs here").Padding(24)
widget.OnDrop(bin, func(rows []*Job) { archive(rows) })
```

- `SetRowReorder(true)` lets the user drag rows to another place: the rows
  chosen, when the row dragged is one, else that row, go before the row they
  drop on, keeping their order, in the table's own rows (`Rows()` returns the
  new order, the selection stays). `OnRowsReordered(fn)` runs on the event
  goroutine after each drop that changed the order, never for a setter.
- It does nothing in a table over a `RowSource` (`DataGridViewOf`), whose
  rows are the app's, and a drag does not start while the rows are sorted,
  grouped, searched or have a column filter: the order shown is then not
  theirs. `ClearSort` (or `ClearFilters`) lets them drag again.
- `DragRows(fn)` lets the user drag rows out: the rows chosen, when the row
  dragged is one, else that row, in the order shown, drag the value `fn`
  returns, for an `OnDrop` or `OnDragOver` of its type elsewhere
  ([Drag and drop](widget/drag-and-drop.md)). `fn` gets copies of the rows and
  runs on the main thread, off the package's lock, as the drag starts: keep it
  quick. A nil result means those rows do not drag; `DragRows(nil)` stops
  dragging out. It works over a `RowSource` too, and sorted, grouped or
  searched.
- A table cannot both reorder and drag rows out (ui's rule): of
  `SetRowReorder(true)` and `DragRows(fn)`, the later call wins.
- Cost: with either on, ui keeps no rows from frame to frame, as in any ui list
  that drags rows, so the rows shown are built again in every frame and their
  column functions (`Text`, `Spans`, `Format`, ...) run for them each frame,
  with about one allocation more a row shown. A table with both off builds and
  allocates what it did.

Fitting and choosing columns, and the full text of a cell:

```go
note := widget.Column[Profile]{Title: "Note", Text: func(p *Profile) string { return p.Note }, Width: 160,
	Hint: func(p *Profile) string { return p.LastError }} // the cell's tooltip; "" for none
t := widget.DataGridView(key, name, note).
	SetColumnChooser(true). // a right-click on the header: the columns, Best fit, Clear sorting
	SetCellHints(true)      // a Text cut off shows whole as a tooltip
t.BestFit("Name") // at the next frame; BestFit() fits every column shown
```

- `BestFit(titles...)` fits each column named to its header and the cells
  in view, as a double click on the edge of its header does; with no
  titles, every column shown. It applies at the
  next frame, and `OnColumnsChange` gets the new widths. Titles of no
  column, or of a hidden one, are ignored, as is the `#` column.
- `SetColumnChooser(true)` gives the header a context menu: every column, as
  an item checked while it shows, which choosing hides or shows as
  `SetColumnVisible` does; the only column still shown is checked and
  disabled, so one always shows. "Best fit" fits the column right-clicked
  (disabled past the columns and on `#`), "Best fit (all columns)" every
  one, and "Clear sorting", there while the rows are sorted, is
  `ClearSort`. The choices run as the setters do, on the event goroutine.
- `Column.Hint(row)` is the tooltip of the column's cells, "" for none; it
  runs under the package's lock as the cell is built, and calls no widget
  method. `SetCellHints(true)` shows the full `Text` of a cell its column
  cuts off (the text measured wider than the column, less its padding, an
  image and a grid line) as a tooltip; a `Hint` that returns text wins. A
  column sharing the room left is measured by its header as the last frame
  laid it out. A `Badge` cell shows its `Hint` alone: its pill is not cut
  off as text is.
- The tooltip covers the whole cell, and only the cell under the pointer
  gets it: the rows stay kept from frame to frame, a frame at rest
  allocates nothing more, and a `Hint` runs only for that cell. The row
  under the pointer is built again as the pointer comes onto a cell and
  leaves it. Assistive technology does not read the tips. With all three
  off the grid builds what it did.

Filters on the columns, as a spreadsheet's header filters:

```go
var t *widget.DataGridViewWidget[Doc, int]
t = widget.DataGridView(key, name, kind).
	SetFilterDropDowns(true). // a funnel in each Filterable header
	SetAutoFilterRow(true).   // a field under each Filterable title
	SetFilterPanel(true).     // the filters as text under the table, while any is set
	SetColumnFilter("Kind", widget.ColumnFilter{Values: []string{"Doc", "Sheet"}}).
	OnFilterChange(func() { save(t.ColumnFilter("Kind")) }) // the user's changes only
```

- A `ColumnFilter` keeps the rows whose cell passes it: `Contains` the rows
  whose text holds it, whatever its case (or, for a column with its own
  `Filter`, those `Filter` keeps for it, as the search), and `Values`, when
  not nil, the rows whose `Text` is one of them (an empty, non-nil `Values`
  keeps none). The zero value filters nothing.
- `SetColumnFilter(title, f)` sets a column's filter (a title no column has is
  ignored; the column need not be `Filterable`), `ColumnFilter(title)` returns
  a copy, and `ClearFilters()` takes them all away and leaves the bottom
  panel's search. A row shows when it passes every column's filter and the
  search. Filters apply to rows set, added and changed in place, as the search
  does; sorting and grouping order the rows that pass; `Summary` gets them.
  Selected rows a filter hides stay selected, and `Selected`, `CountSelected`
  and `Checked` leave them out while hidden, as with the search.
- `OnFilterChange(fn)` runs `fn` on the event goroutine after the user changed
  a filter (a drop-down, the auto filter row, the filter panel), once for each
  change; the setters do not run it.
- `SetFilterDropDowns(true)` gives each `Filterable` column's header a funnel,
  filled while the column has a filter (assistive technology reads it as a
  button, "Filter Name", checked then). A click opens a drop-down: a Contains
  field, the column's distinct `Text`s as a check list, sorted, "Select all",
  which checks them all (`Values` nil), and "Clear", which takes the
  column's filter away. Each change applies at once. The list holds the
  values of the rows the grid has, those loaded so far from a `RowSource`
  still loading, at most 1000 of them, and says so when there are more.
- `SetAutoFilterRow(true)` puts a text field under each `Filterable` column's
  title, the header growing a row taller, bound to the column's `Contains`:
  typing filters at once, and `SetColumnFilter` shows in it. The fields
  scroll sideways with their columns, and a `Frozen` column's stays pinned.
- `SetFilterPanel(true)` shows a strip under the table while a column has a
  filter: the filters as text (`Name contains "an" and Kind is Doc, Sheet`)
  on a check box that turns them all off and on again, keeping them, and
  "Clear", which runs `ClearFilters`. A new filter turns them on again. Its
  words, and the drop-down's, are English.
- A grid that never calls them builds and allocates what it did; a frame at
  rest with them on allocates nothing more.

Exporting what the grid shows, as CSV or an Excel workbook:

```go
header, rows := t.ShownCells() // the columns and the rows as shown
err := export.XLSX(f, header, rows) // or export.CSV; package ui/widget/export
```

- `ShownCells()` returns the titles of the columns shown, in the order the
  header shows them (the user's moves, `SetColumnsLayout`), and the rows as
  shown: filtered by the search and the column filters, sorted, and grouped
  (each group's rows, without its header line). A cell is its column's
  `Text`. Hidden columns, the `#` column and the check box column are left
  out; a `RowSource`'s grid gives the rows loaded so far. It reads the rows
  as they are, also those changed since the last frame, from any goroutine.
- `export.CSV(w, header, rows)` and `export.XLSX(w, header, rows)` write
  them; see [Export](widget/export.md). They live in their own package, so
  an app that does not import it links neither writer.

- Rows: `SetRows`, `AddRow` and `AddRows` (after the others, or in place
  of the row of the same key), `UpdateRow(k, fn)` (fn changes the row in
  place, not its key), `DeleteRows(ks...)`, `Clear`, `Row(k)`, `Len`.
  `InsertRows(i, rows...)` puts rows before the row at `i` of the source
  (0 puts them first; a key held is replaced where it is), and `Rows()`
  returns a copy of all the rows, in the source's order. Both are on
  `RowSource`, `DataGridView` and `ListView`. Rows moved by an insert keep
  their `Version`, so the tables keep the rows they built; it still copies
  the slice and renumbers the rows after `i` (about 1 ms at 100,000 rows,
  40 ms at a million), cheaper than `SetRows`, which builds every row again.
- `RowSource.AddRowsInBatches(rows, batch, progress)` adds a large slice in
  a goroutine, `batch` rows at a time, so the tables fill as it goes, and
  calls `progress(done, total)` on the event goroutine after each batch: bind
  it to a `ProgressBar` or the `StatusProgress` of a `StatusStrip` with
  `SetFraction(float64(done) / float64(max(total, 1)))`. The returned cancel
  stops after the batch in hand and waits for the goroutine to end, so call it
  before `SetRows` or `DeleteRows` if the load must not bring rows back (not
  from a key func, `UpdateRow`'s fn or a view's build, which hold the lock the
  loader needs). A panic in the key func ends the load and goes to `OnPanic`.
  Two loads at once interleave their batches. The source grows once, for all the
  rows, so a table merges each frame's new rows into its view instead of
  sorting them all; it sorts anew when more than an eighth of its rows arrive
  between two frames, so keep batches in the thousands for a big source.
- `CheckboxColumn()`: `Checked`, `SetChecked(ks, on)`, `IsChecked`,
  `AutoCheck` (unchecks the selected rows when half or more are checked,
  checks them otherwise). The header's box checks the rows a search
  shows. `Selected` and `CountSelected` read the selection. `Checked` and
  `Selected` list the rows shown, in the order shown; deleted rows are
  neither. `SetSelected(ks...)` chooses rows from code, firing no
  `OnSelectionChanged` (as `SetChecked`): unknown keys are ignored, any
  goroutine may call it, and `Selected` returns the new set at once, also
  inside a `ContextMenuFunc` fn, where the right-clicked row must stay in
  `ks`, or the menu's build selects it again. It does not move the cursor.
- `IndexColumn()`: each row's position as shown, from 1.
  `IndexColumnBySource()` numbers each row by its place in the source
  instead, the order rows show in unsorted: sorting and searching keep the
  numbers, and a click on its header, `#`, sorts by it. A header click
  sorts by its column; `Sort(column, descending)` does it from code (a
  default order other than the source's), with the header's arrow, and
  ignores a title of no `Sortable` column; `ClearSort` shows the rows in the
  order added.
- Several columns: `SortBy(orders...)` sorts by `ui.SortOrder`s, the first
  first: rows equal by one compare by the next (its `Compare`, or else its
  `Text`), and rows equal by all keep the source's order. It drops an order
  whose title is no `Sortable` column's, or a column's already named, and
  with no orders it is `ClearSort`; `Sort(column, descending)` is `SortBy`
  with one order. `SortOrders()` returns a copy of the orders shown. With
  groups, the rows sort inside each group; a row `UpdateRow` changes moves
  to its sorted place. `SetMultiSort(true)` lets the user add a column: a
  Shift-click or a Cmd-click (Ctrl-click on Linux and Windows) on a
  `Sortable` header adds it after the others, another makes it descending,
  a third takes it out; a plain click sorts by the column alone. Without
  it, the headers show the first column's direction alone.

  ```go
  grid.SetMultiSort(true).SortBy(ui.SortOrder{Column: "Age"}, ui.SortOrder{Column: "Name", Descending: true})
  ```
- `Column.Frozen` pins a column at the table's start while the others
  scroll sideways, as Excel's frozen panes: only a run of `Frozen` columns
  with a `Width`, from the first shown, pins, and the `#` column pins with
  a `Frozen` first column.
- `BottomPanel()`: the counts of rows, and of the rows shown that are
  selected and checked (what `Selected`, `CountSelected` and `Checked`
  return: a search hides the rows it leaves out, selected or not), and a
  search over the `Filterable` columns. `PanelText` sets its words (English
  by default): a format without `%` shows as it is, an empty one hides its
  count, and a `%` shown is written `%%` (`"%d%% done"`). `SetSearchPlaceholder` sets its keyword's placeholder, and
  `OnBeforeSearch(fn)` runs before a search from it. `Search(column,
  keyword)` searches from code. The filter runs once per search.
  `ApplyPanel(styles...)` styles the panel (its `Background`, and a
  `TextSize` or `FontSize` its counts and search take), `SetSearchWidth(w)`
  sets the keyword's width, and `JoinSearch(true)` draws the column, the
  keyword and the button as one control. Call these after `BottomPanel`.
  `SetRowHeight(h)` makes every row `h` DIPs tall, `SetGridLines(true)` draws
  lines between rows and cells, and `SetColumnsMovable(false)` keeps the user
  from reordering the columns by their headers (resizing them still works).

  ```go
  grid.CheckboxColumn().BottomPanel().
  	PanelText(widget.PanelText{Rows: "%d rows", Selected: "%d selected", Search: "Find"}). // no checked count
  	ApplyPanel(widget.Style{}.Background(widget.Fixed(ui.RGB(240, 244, 250))).TextSize(widget.SM)).
  	SetSearchWidth(320).JoinSearch(true).
  	SetRowHeight(28).SetGridLines(true)
  ```
- **Editing cells.** `EditColumn(title, widget.CellEditor{Set: ...})` makes the
  cells of a column editable in place; a column not given to it is read-only.
  Calling it is also what links the code of editing into the program, for that
  table's types only, so a table that never edits pays nothing for it (a
  `Column` field could not do this: it reaches the table through an interface,
  and the linker then keeps the editing code of every table in the program). F2,
  Enter or a double click edit the current cell; Enter writes, Escape cancels. Tab
  writes and edits the next editable cell of the row, then the first of the next
  row; Shift+Tab goes back (the next row is the one the table shows when Tab is
  pressed, even if the write re-sorts the rows). When no cell is left to go to
  (past the last cell, or the rest refuse `CanEdit`) Tab is not taken: it writes
  and the focus moves on to the next element; when the write is refused the edit
  stays. If only the write takes the last cell away (a search now hides the row,
  `CanEdit` changed), the edit ends and the table has the focus. `Validate(row, text) error` of the
  `CellEditor`, asked before `Set(row, text)`, refuses a value: the editor stays
  open, with a red border and the error in a small popover under the cell until
  the next keystroke. A search that hides the edited row, or hiding its column,
  cancels the edit; scrolling, sorting and resizing columns keep it.
  `CanEdit(row)` makes some cells read-only, `SetReadOnly(true)` the whole grid,
  `EditText(row)` is the text the editor starts with. `BeginEdit(key, column)`,
  `CancelEdit()`, `EndEdit() error` (commits now, for a Save button),
  `IsEditing()`, `EditingCell()`, `CurrentCell()`, `SetCurrentCell(key, column)`
  and `OnCellEndEdit(fn)` drive it from code. `SetEditMode(widget.EditOnF2)` keeps
  Enter and the double click for `OnActivate`; `EditProgrammatically` starts
  edits with `BeginEdit` only. A `ListView` over the same table never edits.

  ```go
  grid := widget.DataGridView(func(a *Account) uint { return a.ID },
  	widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }},
  	widget.Column[Account]{Title: "Qty", Text: func(a *Account) string { return strconv.Itoa(a.Qty) }},
  ).
  	EditColumn("Name", widget.CellEditor[Account]{Set: func(a *Account, s string) { a.Name = s }}).
  	EditColumn("Qty", widget.CellEditor[Account]{
  		Validate: func(a *Account, s string) error {
  			if _, err := strconv.Atoi(s); err != nil {
  				return errors.New("Qty must be a number")
  			}
  			return nil
  		},
  		Set: func(a *Account, s string) { a.Qty, _ = strconv.Atoi(s) }})
  ```

  The `CellEditor`'s functions run under the package's lock like
  `Compare`: they read the row and call no widget method and no `RowSource`
  method (`UpdateRow`, `AddRows`, `Row`, ...), which would wait for the lock
  they run under and hang. `Set` must not leave the row half written: build the
  new value on a copy and assign it. A panic in one is recovered, reported with
  `OnPanic`, and cancels the edit. Leaving the editor (a click elsewhere)
  writes what was typed; when `Validate` refuses it the edit stays open with its
  error and nothing takes the focus back, so a Save handler calls `EndEdit()`
  and reads its error. A draft the user did not change writes nothing; one they
  changed wins over a change another goroutine made to the same cell meanwhile
  (the last writer wins on the cell; the row's other fields are kept). Closing
  the window discards an open draft: `IsEditing` tells an app there is one.
- `OnSelectionChanged(fn)` runs `fn` on the event goroutine once the set of
  rows `Selected` returns changes: the user's clicks, Shift and Ctrl clicks,
  keys and select all, a delete of selected rows, and a search that hides or
  shows selected rows (a new search, or rows changed under one by
  `UpdateRow`, `AddRows`, `InsertRows` or `SetRows`, seen as the table next
  builds). Sorting only orders them and does not run it. Changes made before
  `fn` runs run it once, never once per frame. Read the rows with `Selected`
  in it.
- `OnCheck(fn)` runs `fn(key, on)` on the event goroutine for each row the
  user checks or unchecks: a click on the row's box, Space, and the header's
  box, once for each row it flips. `SetChecked` and the other setters do not
  run it.

  ```go
  grid := widget.DataGridView(func(a *Account) uint { return a.ID },
  	widget.Column[Account]{Title: "Name", Text: func(a *Account) string { return a.Name }, Sortable: true},
  ).IndexColumnBySource().Sort("Name", false)
  grid.OnSelectionChanged(func() { status.SetText(fmt.Sprint(len(grid.Selected()), " selected")) })
  grid.InsertRows(0, Account{ID: 7, Name: "newest"}) // first in the source: number 1
  ```
- `OnActivate(fn)` (double click, Enter), `OnKey(key, mods, fn)`,
  `ContextMenu(items...)` (on rows only; read them with `Selected`),
  `ContextMenuFunc(fn)` (fn gets the keys selected at the right-click, on the
  event goroutine without the lock, and may query a database; the menu opens
  when it returns, unless the user pressed a key or a button, or right-clicked
  again, meanwhile; it wins over `ContextMenu`),
  `OnColumnsChange(fn)` with `SetColumnsLayout(l)` to restore it (see
  below).
  `SetColumnVisible(title, on)` and `Column.Hidden` hide a column, as WinForms'
  `Visible`: it is not drawn, read by assistive technology (nor named in the
  header row) or copied, and a row is named by its first column shown. The
  columns' layout (order, widths) is kept for when it shows again: the order
  `OnColumnsChange` gives holds the hidden columns too, each after the column it
  followed. Titles tell the columns apart, so they must differ. A table
  sorted by a column stays sorted when it hides, and `Search` by a `Filterable`
  hidden column still works, as does the panel's choice of it.
  Cmd/Ctrl+C and the Edit menu's Copy copy the selected rows, a line each,
  cells apart by tabs; `CopyFunc(fn)` replaces that text with what `fn` makes
  of the keys of the selected rows, in the order shown, for both, so they
  agree. `fn` runs on the main thread, off the lock: it may call the table's
  getters, but should be quick. `SetClipboardCopy(false)` turns both off, in
  a `ListView`'s every view too (through `Details()`).

  ```go
  table.CopyFunc(func(ids []string) string { return strings.Join(ids, "\n") })
  ```

  `FollowEnd(true)` keeps the last row of the table in view as rows are added,
  as a log does, and starts it there before its first frame (a table already
  shown stays where it is: call `ScrollToEnd()` too). Scrolling away stops
  following, scrolling back to the end follows again, and rows added while away
  do not move the table. `ScrollToEnd()` scrolls to the last row. `ListBox` has both too; a
  `ListView` takes them through `Details()`, and only its Details view follows:
  a `ScrollToEnd()` made while another view shows waits, and scrolls the Details
  view when it shows again. Both take effect at the next frame.

**Saving the columns' widths.** `OnColumnsChange` runs once a frame while the
user drags a column, so it fires dozens of times for one resize: an app that
writes the layout on each call writes dozens of times. Save when the user
has stopped, by debouncing. Handlers run one at a time on the event
goroutine, so the timer needs no lock:

```go
var saveTimer *time.Timer

table.SetColumnsLayout(loadLayout()). // what the last run saved
	OnColumnsChange(func(l ui.TableLayout) {
		if saveTimer != nil {
			saveTimer.Stop()
		}
		saveTimer = time.AfterFunc(500*time.Millisecond, func() { saveLayout(l) })
	})
```

A table shows in one window at a time. It fills the room its parent
gives it and scrolls; in a box or a `TabControl` page, `Expand` the boxes and
the `TabControl` holding it, or it gets no room.

## Clipboard, theme and screen

`Clipboard.SetText(s)`, `Clipboard.SetImage(png)` and `Clipboard.Text()`
use the system clipboard. `SetTheme(ThemeSystem/ThemeLight/ThemeDark)` sets
the app's appearance, which every window follows, and `IsDark` reports it.
The outline of the controls (`TextBox`, `ComboBox`, `NumericUpDown`, buttons
and the other inputs) is the theme's `ControlBorder`: `#8a8a93` in the light
appearance (3.4:1 against the window, 3.1:1 against a control's face) and
`#71717a` in the dark (3.7:1 and 3.1:1), where WCAG 1.4.11 asks 3:1 of a
control's boundary. The box of a check box and a radio button, and the off
track of a switch, are the same color, and an input's outline is a darker
shade of it while the pointer is over it. Dividers, grid lines and the edges
of containers and popups keep the lighter `Border`. `SetControlBorder(light, dark)` sets the outline in
each appearance, with no control touched; a zero color leaves the theme's, and
every window follows the appearance as before. Call it once at startup:
`widget.SetControlBorder(ui.Hex("#6b7280"), ui.Hex("#9ca3af"))`. An app's own
`ui.Theme` that sets no `ControlBorder` outlines its controls with `Border`
(`Theme.Outline`).
Inputs and buttons do not share a fill: `TextBox`, `MultilineTextBox`,
`MaskedTextBox`, `NumericUpDown`, `DomainUpDown`, `DateTimePicker` and an
editable `ComboBox` are filled with the theme's `Field` (`#ffffff` light,
`#101012` dark), and `Button`, `ToolButton`, a drop-down `ComboBox` and the
toggles with its `Surface` (`#f4f4f5`, `#27272a`), so that a button is not
read as a second text box. The rule: the controls you type a value into,
and the date field (as in WinForms), take the `Field`; the controls that
open a list to choose from (`Select`, a drop-down `ComboBox`) keep the
button look. `widget.RoleField` is the `Field` for a `Style`;
a theme that sets none fills the inputs with `Surface`. A destructive
button is the danger color, darker by 8% hovered and 16% pressed, and
darker still where the white text on it would be under 4.5:1 (the dark
theme's `#ef4444` is 3.76:1), and the focus ring
is 3.6:1 against the light window.
`ScreenSize()` returns the main display's size in DIPs, once the app runs.

## Styling

Style methods take theme units of 4 DIPs, with half steps: `Padding(2)` is
8 DIPs. Box spacing (`VBox(8)`) and `SetSize` are in DIPs. Colors come from
the theme (`Muted`) or `ui.RGB`. To share a look between controls, or to
style hover, focus, pressed, disabled and dark states, see
[Styles](#styles).

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
| `Suggested()`, `Destructive()` on buttons and tool buttons | `suggested-action`, `destructive-action` |

Every widget also has `SetVisible`, `Visible`, `SetDisabled`, `SetTooltip`,
`SetSuperTip` (a rich tooltip: title, body, image and footer; see
[SuperTip](widget/supertip.md)), `SetAccessibleName` (the name of the control itself for assistive technology
and `ByLabel`, over the one its text, tooltip or `Fields` row gives; a
container's, or a group's, names the group), `OnFocusOut` (also when the
form closes with the focus in the widget, before a dialog's `Run`
returns; the app quitting does not wait for it), `OnClick` (gota's `EventBox`), `ContextMenu`, `Focus`, which
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
  widget already in the box is not added twice. `fill` means what it did. Controls that scroll or virtualize their rows
(`ListBox`, `DataGridView`, `TreeView`, `MultilineTextBox`, `SplitContainer`,
`TabControl`, `ScrollPanel`) stretch across the box that holds them whatever
the flags (an `HBox` would otherwise center them at the height of all their
rows), so they scroll in the room the row has and build only the rows in view.
A `VBox`/`HBox` or a `GroupBox` does when something it holds does, however
deep, and passes the height down; one that holds only labels and buttons is
placed as before. The widget's own `Align`/`VAlign` still wins.
- `ButtonBox()`: a row whose widgets sit at its end, one width each, as a
  dialog's buttons.
- `Grid(colSpacing, rowSpacing)`: `Attach(w, col, row, width, height)`; a
  widget attached to a taken cell replaces the one there, and cells must
  not overlap.
- `Fields()`: a form of label and control rows; see [Fields](#fields).
- `NewMenuStrip(items...)`, `StatusStrip(items...)`, `ToolStrip(items...)`,
  `NewNotifyIcon()`: the window's main menu, status bar and tool bar, and
  an icon in the notification area; see [App shell](#app-shell).
- `Ribbon()`: tabs of groups of large and small commands, which collapses to
  its tabs, first in a form's content; `AddTab`, `AddGroup`, `AddLarge`,
  `AddSmall`, `Tab`, `SetTab`, `SetCollapsed`, `Collapsed`; a Quick Access
  Toolbar (`AddQuickAccess`, `SetQuickAccessBelow`, `QuickAccess`,
  `SetQuickAccess`, `OnQuickAccessChange`) and contextual tabs
  (`AddContextualTab`, `SetCategoryVisible`); see
  [Ribbon](widget/ribbon.md), [Quick Access Toolbar](widget/ribbon.md#quick-access-toolbar)
  and [contextual tabs](widget/ribbon.md#contextual-tabs).
- `GroupBox(title)`: a border and a title around one widget.
- `Card()`: a titled box whose body `ShowBody(false)` hides.
- `FlowPanel(spacing)`: a row that wraps. `.Justify()` fills every line,
  the last too, to both edges: lines break at the items' natural widths as
  without it, then each line's items grow equally (gaps unchanged, a
  toggle's text stays centred). An item alone on the last line takes the
  whole width. It overrides an item's own width and grow weight (an item
  with `SetSize(80, 0)` grows too). Do not `HExpand()` the items instead:
  that sets their basis to 0, so they shrink to their minimum widths and
  more of them pack onto each line.
- `ScrollPanel(w)`: scrolls a widget larger than its room, both ways.
  `.Vertical()` scrolls up and down only, as a page of a sidebar does: the
  content keeps the panel's width, so its labels wrap there, and it is laid out
  at least as high as the panel, so a root that `Expand()`s takes the room.
  A dialog of height 0 fits the whole content of a `Vertical` panel. A button
  bar outside the panel stays in view when the panel is bounded: at a fixed
  height with every box around the panel growing (`Expand`), or when the fit
  stays below the screen. A height-0
  dialog taller than the screen scrolls as a whole, the bar included. A
  sidebar's or a tab's page already scrolls: it needs no panel. Lists, tables
  and trees scroll themselves: do not put them in a `ScrollPanel` (a
  `DataGridView` of 20000 rows inside one stops at row 4097).
- `SplitContainer(first, second)`: two panes with a divider; see [SplitContainer](widget/split-container.md).
- `TrackBar(min, max)`: a slider; see [TrackBar](widget/trackbar.md).
- `RangeTrackBar(min, max)`: a slider of two knobs choosing a range; see [RangeTrackBar](widget/rangetrackbar.md).
- `Rating(max)`: a row of stars; see [Rating](widget/rating.md).
- `CheckButton(text)`: a button that stays pressed while checked, with an icon; see [CheckButton](widget/checkbutton.md).
- `Segmented(labels...)`: one choice among joined buttons; see [Segmented](widget/segmented.md).
- `SplitButton(text, items...)`: a button joined to an arrow opening a menu; see [SplitButton](widget/splitbutton.md).
- `DomainUpDown(items...)`: arrows over a list of texts; see [DomainUpDown](widget/domainupdown.md).
- `DragSource`, `OnDrop[T]`, `OnDragOver[T]`, `DragData`, `OnDropData`, `OnDropFiles`: drag and drop; see [Drag and drop](widget/drag-and-drop.md).
- `Chart(kind)`: a line, bar or pie chart; see [Chart](widget/chart.md).
- `richtextbox.New()`: styled text, in package `ui/widget/richtextbox`; see [RichTextBox](widget/richtextbox.md).
- `WebBrowser()`: a web page; see [WebBrowser](widget/webbrowser.md).
- `MaskedTextBox(mask)`: an input that takes what a mask allows; see [MaskedTextBox](widget/maskedtextbox.md).
- `ErrorProvider()`, `OnValidating`, `Form.ValidateChildren`: an error mark on a control and an explicit validation; `Rule` with `Required`, `MatchRegexp` and `InRange`, `Validate` and `SetValidateOnLeave` check values with no handler; see [ErrorProvider](widget/errorprovider.md).
- `&` mnemonics in a `Button` and the labels of `Fields` (Alt+key; off on macOS); see [Mnemonics](widget/mnemonics.md).
- `ImageList(size)`: images by key, shared by `ListBox`, `TreeView` and menus (`ImagesBy`, `PNG`); see [ImageList](widget/imagelist.md).
- `Separator()`: a line.
- `SetSuperTip(SuperTip{Title, Body, Image, Footer})`, on every control: a rich tooltip in place of `SetTooltip`'s text, as DevExpress's SuperToolTip; see [SuperTip](widget/supertip.md).
- `TreeView(key, text)`: nodes in a tree, with lazy children and check boxes, in columns under a header with `SetColumns`, sorted (`SetSort`), filtered (`SetFilter`) and renamed in place (`SetLabelEdit`, `OnAfterLabelEdit`); see [TreeView](widget/treeview.md) and its [sorting, filtering and renaming](widget/treeview.md#sorting-filtering-and-renaming).
- `ListBox(key, text)`, `CheckedListBox(key, text)`: items in a list, chosen by key, with check boxes on the second; see [ListBox](widget/listbox.md).
- `LookUpEdit(key, text, cols...)`: a field that picks one record by key from a drop-down of columns, which typing filters, as DevExpress's LookUpEdit; `SetItems`, `Selected`, `SetSelected`, `Unselect`, `OnSelect`, and `SetSearchPanel(true)` for a search box over the records in the drop-down (DevExpress's SearchLookUpEdit); see [LookUpEdit](widget/lookupedit.md).
- `ButtonEdit(buttons...)`: a text field with buttons at its end, inside its frame (`EditEllipsis`, `EditClear`, `EditSearch`, `EditPlus`, `EditMinus`, `EditDropDown`, `EditGlyph`), as DevExpress's ButtonEdit; `Text`, `SetText`, `SetPlaceholder`, `SetReadOnly`, `OnChange`, `SetButtonVisible`; see [ButtonEdit](widget/buttonedit.md).
- `FontEdit()`: a field that picks a font family from a drop-down of `ui.FontFamilies`, each name in its own face, which a search box filters, as DevExpress's FontEdit; `Family`, `SetFamily`, `SetFixedPitchOnly`, `OnChange`; see [FontEdit](widget/fontedit.md).
- `CalcEdit()`: a number field with a calculator in its drop-down, as DevExpress's CalcEdit; `Value`, `SetValue`, `SetDecimals`, `OnChange`, `ShowPopup`, `ClosePopup`; see [CalcEdit](widget/calcedit.md).
- `PopupEdit(controls...)`: a field showing a text whose drop-down holds any controls, as DevExpress's PopupContainerEdit; `SetText`, `Text`, `ShowPopup`, `ClosePopup`, `IsOpen`, `OnClosed`; see [PopupEdit](widget/popupedit.md).
- `ProgressPanel(caption)`: a spinner with a caption and a description, as DevExpress's ProgressPanel; `Form.ShowWait(caption, description)` and `Form.CloseWait()` show one over a dimmed form that takes no input; see [ProgressPanel and the wait form](widget/progresspanel.md).
- `ShowSplash(Splash{Image, Title, Status, Width, Height})`: a frameless, centered, always-on-top window with no taskbar button while the app starts, holding an image, a title, a status line and a progress bar, as DevExpress's SplashScreenManager; it may be called before `Run`, from any goroutine. `SetStatus`, `Close`, `CloseWhenShown(form)`; see [Splash screen](widget/splash.md).
- `Badge(text)`: a short text in a pill colored by its `Status` (`StatusNone`, `StatusInfo`, `StatusRunning`, `StatusSuccess`, `StatusWarning`, `StatusDanger`); see [Badge](widget/badge.md).
- `InfoBar(status, title, message)`: a message inline in a page, with an optional action and close button; see [InfoBar](widget/infobar.md).
- `Form.ShowToast(Toast{...})`, `Form.CloseToast(id)`: a message near the bottom of the window for a few seconds, the outcome of an action, with a status and an optional action; see [Toasts](widget/toast.md).
- `EmptyState(title, message)`: what a view shows when it has nothing in it, with an optional icon and action; a `DataGridView`, a `ListView` and a `ListBox` draw one over themselves while they show no row (`SetEmptyState`); see [EmptyState](widget/emptystate.md).
- A `Sidebar` or `TabControl` page taller than the room scrolls, as WinForms'
  `AutoScroll` does (a wheel over it, or Tab to a control below the fold), in a
  scroller that lays the page out at least as high as the room: a page that
  `Expand`s, or has a button `PackEnd`ed, still fills a short room. A page that
  is a viewport itself, or a box, group box or card holding one (a table, a list,
  a tree, a multi-line text box, a scroll panel, a web view), or is a `Custom`
  control, whose element is opaque and may be a `ui.List`, is not wrapped: it fills the
  room and scrolls itself, so a page that gains or loses one does not keep its
  scroll position. A dialog of height 0 with a `Sidebar` opens as high as its tallest
  page, no higher than the work area allows, and does not resize when another
  page is chosen.
  Only a `Sidebar` that is the dialog's content is measured so: a `TabControl`
  fits the tab shown, and a sidebar inside another page is measured with that
  page shown.
- `TabControl()`: pages under tabs; `AddPage`, `Page`, `SetPage`, `OnSwitch`.
  `StripStyle(styles...)` styles the strip of tabs as `Apply` styles a
  control, its background across the control past the last tab:
  `StripStyle(tw.S("bg-slate-100 rounded-md"))`; `Apply` styles the whole
  control.
  `SetPageLabel(i, s)` renames a tab ("Items (3)"), `InsertPage(i, label, w)`
  and `RemovePage(i)` add and take out pages. The page shown stays the page
  shown, whatever its index becomes; removing it shows the one before it (the
  first, if it was the first). The tab with the keyboard focus keeps it; if its
  page goes, the tab now shown takes it. As
  `SetPage` does, none of them runs `OnSwitch`: the app made the change.
  `SetClosable(true)` makes the tabs documents' tabs, as DevExpress's tabbed
  document manager: an x on each tab, a middle click on a tab, and Ctrl+F4
  anywhere in the control close a page (Ctrl+F4 the page shown);
  Ctrl+Tab and Ctrl+Shift+Tab show the next and the previous page, wrapping,
  and run `OnSwitch`. The x takes no focus and a tab reads as its label alone,
  so Ctrl+F4 is the way to close for the keyboard and screen readers.
  `OnPageClosing(fn)` runs `fn` on the event goroutine with the page before it
  closes; `false` keeps it, so `fn` may ask with `Confirm`:

  ```go
  tabs := widget.TabControl().SetClosable(true).
  	OnPageClosing(func(page widget.Control) bool {
  		return widget.Confirm(form, widget.Question, "Close", "Discard the changes?")
  	})
  ```

  A page the user closes goes as `RemovePage` takes it out, found by its tab,
  not its index, whatever was added or removed meanwhile; when it is the page
  shown, `OnSwitch` runs. Without a skin's `TabHeader` look the tabs look like
  documents' (the one shown filled as the page, with a line on its top and
  sides, and no accent); with one they take it, with the x. `SetClosable(false)`
  takes the x and the keys away. Not done: reordering tabs by drag, a menu of
  the tabs past the strip's width (the strip clips them), Ctrl+W and Cmd+W
  (the default menu's Close Window), and tabs floated out to windows.
- `Sidebar()`: pages chosen in a list at the side; `Add(name, title, w)`,
  `Remove`, `Selected`, `Select`, `SetMenuWidth`.
- `NavBar()`: a side panel of groups that open and close, each holding items
  of an icon and a text, one of them selected, as DevExpress's NavBarControl;
  `AddGroup`, `AddItem`, `Selected`, `SetSelected`, `OnSelect`,
  `SetGroupOpen`, `GroupOpen`; see [NavBar](widget/navbar.md).

Input and display:

- `DateTimePicker()`, `MonthCalendar()`: a `time.Time` in a date and time field, or a calendar; see [DateTimePicker](widget/date-time-picker.md).
- `TimeSpanEdit()`: a `time.Duration` in a field shown as `[-][d.]hh:mm[:ss]`, whose Up, Down and − + buttons step the part under the caret, as DevExpress's TimeSpanEdit; `Value`, `SetValue`, `SetRange`, `SetShowSeconds`, `SetShowDays`, `OnChange`; see [TimeSpanEdit](widget/timespanedit.md).
- `Label(text)`: text, which wraps at its container's width (a long one in
  a dialog of fixed width takes more lines, none cut); `Rich(spans...)` shows styled `ui.Span`s
  instead, where gota used Pango markup. `ParseMarkup(s)` makes the spans of a
  Pango subset: `<span foreground background weight>`, `<b>`, `<i>`, `<u>` and
  the entities; anything else, malformed markup too, stays as text and never
  fails: `Label("").Rich(widget.ParseMarkup(status)...)`.
- `Icon(svg, size)`: an SVG icon in the color of the text (`TextColor`
  changes it). There are no named icons: embed the SVGs you need.
- `PictureBox(data, w, h)`: a PNG, or a GIF, JPEG, WebP or BMP once the
  program imports its decoder (see "Formats" in [Images](image.md)),
  `SetImage` to change it. The same bytes of an image up to 64 KB, as an icon, decode once (the
  last 256 are kept); bytes that are no image leave an empty room.
- `TextBox()`, `MultilineTextBox()`: one line or several of text; `Text`,
  `SetText`, `SetPlaceholder`, `SetEditable`, `OnChange`; `TextBox` also has
  `Password`, `OnEnter`, `MaxLength(n)` (runes, as in WinForms: typing or
  pasting past it is cut at it, and so is `SetText`; 0 for no limit) and
  `Icon(bitmap)` (16 DIPs, inside the box before the text; nil removes it),
  `SetSuggestions(items...)` and `SetMRU(max)` (suggestions as the user
  types, and the last entries remembered; see
  [TextBox suggestions](widget/textbox-suggestions.md)).
- `SearchBox()`: a field for searching, with a clear button; `OnChange`,
  `OnSubmit`; see [SearchBox](widget/searchbox.md).
- `BreadCrumb(items...)`: a path of links, the last the current page;
  `SetItems`, `OnSelect(func(i int))`; see [BreadCrumb](widget/breadcrumb.md).
- `Accordion()`: sections, one open at a time; `AddSection(title,
  controls...)`, `Open`, `SetOpen`, `OnChange`; see [Accordion](widget/accordion.md).
- `Wizard()`: steps one page at a time, with a step bar and Previous, Next
  (Done) and Cancel; `AddStep(title, content)`, `Step`, `SetStep`,
  `OnValidate(func(step int) bool)`, `OnStepChange`, `OnFinish`, `OnCancel`.
  `StepProgressBar(steps...)`: numbered circles joined by lines; `Step`,
  `SetStep`, `SetSteps`, `SetVertical`. See [Wizard and StepProgressBar](widget/wizard.md).
- `ColorEdit()`: a swatch that opens a color picker inline; `Color`,
  `SetColor`, `OnChange`; see [ColorEdit](widget/coloredit.md).
- `Bind(control, get, set)`, `BindInt`, `BindRange`, `BindValues`,
  `BindTokens`, `Reload(control)`: tie a `TextBox`, `MultilineTextBox`,
  `SearchBox`, `FontEdit`, `DomainUpDown`, `CheckBox`, `ComboBox`, `NumericUpDown`, `CalcEdit`, `TrackBar`,
  `RangeSpin`, `CheckBoxGroup` or `TokenEdit` to a value kept elsewhere;
  see [Binding](#binding).
- `Button(text)`: `OnClick`, `SetText`, `SetIcon` (an SVG in the text's
  color) or `SetIconPNG` (a PNG in its own colors, as high as the
  text, as an SVG icon: give it 32 by 32 pixels to stay sharp on 2×
  displays). `Button("")` is named by its tooltip for assistive technology
  and the Tester, or by `SetAccessibleName`. `Suggested()` and
  `Destructive()` are the button's own style (tw's `suggested-action` and
  `destructive-action`); the last one given wins. Their border is in
  the fill, in each state, not the plain button's gray; under High Contrast
  it stays the strong outline. As their background, it is the own style, so
  an applied border color (`Apply`) loses to it; a `Border` given after
  `Suggested()` or `Destructive()` wins in every state, hover and pressed
  included, and one given before is replaced. `LinkLabel(text, url)`; `OnClick(fn)` runs `fn` on the event goroutine
  in place of opening `url` (click, Enter or Space, assistive technology; nothing
  when disabled), as WinForms' `LinkClicked`.
- `CheckBox(text)`, `Toggle(text)`, `Switch()`: on or off, with `OnChange`.
- `CheckBoxGroup(row, options...)`, `TagSelect(options...)`: several of
  the options; `Values`, `SetValues`. `TagSelect`'s `SetOrdered(true)`
  keeps the order the user picks in: turning an option on puts it last,
  turning it off takes it out, and `Values` returns that order. The chosen
  options show as chips in that order above the toggles, each with its
  `Text` and a remove button ("Remove x", as `TokenEdit`'s, which Tab
  passes by); no row of chips while none is chosen. `SetValues` keeps the
  order it is given, dropping values that are no option's and repeats.
  `OnChange` runs after each change by the user, not after `SetValues`.
  Without it, `Values` is in option order.

  ```go
  fields := widget.TagSelect(
  	widget.Option{Value: "TwoFactorSecret", Text: "2FA"},
  	widget.Option{Value: "Birthday", Text: "Birthday"},
  	widget.Option{Value: "Phone", Text: "Phone"},
  ).SetOrdered(true).SetValues("Phone", "TwoFactorSecret")
  // fields.Values() == []string{"Phone", "TwoFactorSecret"}
  ```
- `TokenEdit(suggestions...)`: free tags shown as chips, as DevExpress'
  TokenEdit, editing a `[]string`. Enter or a comma adds what was typed
  (trimmed; a blank or a repeat adds nothing), a chip's button ("Remove
  x") or Backspace in the empty input takes one out, and the suggestions
  containing the text show below it, those not already tokens; Down in
  the empty input shows all of them, in their order. A paste
  of several lines adds a token a line, as a list of IDs; a paste of one
  line is text. `Tokens` returns a copy, in the order they were added;
  `SetTokens` trims and drops blanks and repeats, without `OnChange`;
  `SetSuggestions`; `SetSuggestionsOnly(true)` takes only what equals a
  suggestion, ignoring case, in the suggestion's spelling (typed, picked or
  pasted; other text adds nothing and runs no `OnChange`; `SetTokens` is
  not filtered), for chips picked in order from a fixed set;
  `SetPlaceholder` shows while there is no token and no text; `OnChange`
  runs after each change by the user. A Fields row's access key focuses
  its input. `BindTokens` ties the tokens to a source, as `Bind` other
  controls, so `Reload` refreshes them; see [Binding](#binding).

  ```go
  te := widget.BindTokens(widget.TokenEdit(allTags...),
  	func() []string { return cfg.Tags },
  	func(v []string) { cfg.Tags = v; cfg.Save() })
  ```
- `RadioButton(text)`: `Checked`, `SetChecked`, `OnChange`. As in WinForms, the
  radio buttons added to one container (a box, a group box, fields, a flow
  panel) are one group: checking one, by the user or `SetChecked`, unchecks
  the others; `OnChange` runs for the user's click, on both buttons, and not
  for `SetChecked`. One outside a container stands alone.
- `RadioGroup(row, options...)`, `ComboBox(options...)`: one option; `Value`,
  `SetValue`. An `Option` has the `Value` the program reads and the `Text`
  the user sees. A combo's values must differ (its texts may repeat), and
  it groups options in titled sections with icons: `AddSection(title, png,
  options...)`, `RemoveSection`, `RemoveOption`, `SetOptionText`,
  `SectionLen`, `Selected`, `Unselect`; `AddOption(options...)` appends to
  the options outside any section, as WinForms' `Items.Add`, keeping the
  choice, and `Options()` returns a copy of those, in order; sections, icons and texts change
  while the combo shows, and it redraws. A title names one section:
  `AddSection` of a title the combo has adds the options to that section
  (and its icon, unless nil). An option of a value the combo has is left
  out of `AddSection` and `AddSubmenu`: the first option of a value stays.
  A section's title chooses nothing until
  `SetSectionValue(title, v)` makes it an option of value `v` ("" undoes
  it). `AddSubmenu(title, png, options...)` adds an option that opens a
  nested list of options, as a menu's submenu, on the pointer, a click,
  Right, Enter or Space; a combo with one is a menu button to assistive
  technology, its options radio items, its menu as wide as the combo and
  opened by Up too, and an `Editable` one shows a submenu's options as a
  section's. `Editable()` lets the user type a
  text that is no option, which `Text` and `SetText` give and set; `Value`
  is then the `Value` of the option with that text, or ""; `OnChange` runs
  on typing and on choosing. `Bind` takes the text of a combo made
  `Editable` before it is bound, and the `Value` of one bound before.
  `Shows(value, controls...)` shows controls while the option of `value`
  is chosen and hides them while none of the values they are shown for
  is, in place of `SetVisible` calls in `OnChange`:

  ```go
  mode := widget.ComboBox(widget.Option{Value: "none", Text: "None"}, widget.Option{Value: "http", Text: "HTTP"}).Shows("http", hostBox, authBox)
  ```

  It applies at once, to the choice the combo has, and on every change of
  it, before `OnChange` runs: the user's pick or typing, `SetValue`,
  `SetText`, `Unselect`, a binding, the chosen option removed. The first
  frame after shows the right controls, those before the combo in the
  form too. A control hides as with `SetVisible(false)`, losing the
  keyboard focus (its `OnFocusOut` runs); a `SetVisible` of the program
  lasts until the choice next changes.
- `MenuButton(text)`: see [Menus](#menus).
- `MenuBar(menus...)`: a menu bar in the content; see [MenuBar](widget/menubar.md).
- `NumericUpDown(min, max, step)`: a number; `Value`, `Int`, `SetValue`;
  `SetAccessibleName(s)` names it and its buttons for assistive technology.
  It is a classic spin box, as WinForms' (`ui.SpinBox`): an up arrow over a
  down one joined at the end of the box, which `Apply`, a `Style` or tw's
  classes style (background, border, radius, padding, size). Padding pads
  the text: the arrows stay at the edge, as high as the box. Up and Down
  step it, as do the arrows, again as one is held; they are no stops of
  Tab, and a press on one leaves the keyboard in the text. A negative step
  steps by its absolute value. `SideButtons()` gives it the look of
  `ui.NumberInput` instead, − and + buttons beside the box, which are
  stops of Tab and take the keyboard when pressed; `Apply` then styles the
  row of the box and the buttons.
  `RangeSpin` is two, with text between and after; they are "minimum" and
  "maximum" to assistive technology unless `SetAccessibleNames(low, high)`.
  Its `SideButtons()` gives both numbers the look `NumericUpDown`'s gives.
- `ProgressBar()`: `SetFraction` (negative for unknown length), `SetText`.
- `DataGridView(key, columns...)`: rows in columns; see [DataGridView](#datagridview).
  `Spinner()`: `Start`, `Stop`.
- `ListView(key, columns...)`: rows in columns, as icons, or as a list, over the same choice and checks; see [ListView](widget/listview.md).

`SetSize(width, 0)` on a form (or dialog) that is shown fits its height, once, to
its content laid out at `width`, on the next frame: call it after `SetContent`
when the content changed (a splash swapped for a login view). It is measured as
a dialog of height 0 is at show, at the scale of the display the window is on,
and no higher than that display's work area less 80 DIPs, where the content
scrolls; the width is the page area's, and a window that would end below the
work area moves up (never above its top) in the same resize. Later changes of
the content do not resize the window again, a later `SetSize` wins, a
maximized or minimized window is left alone, and a dialog fits again at its
next `Show`. A `TestForm` or `TestDialog` tester's viewport follows
on its next frame. Before `Show`, a height of 0 is the window default as ever.

Windows: `SetTitle`, `SetSize`, `SetIcon`, `SetResizable`,
`SetHideDisabledButtons(on)` (drops the maximize button of a form that is
not resizable instead of greying it, for a small fixed-size window such as a
login or key screen; close stays; it applies at the next `Show`, on macOS and
on Windows with `UseHeaderBar`, see `mygo.WindowOptions.HideDisabledButtons`),
`SetMinimumSize(w, h)` (the least the user can resize it to, in DIPs, 0 for
no limit on a side; the window's outer size, the title bar included on macOS
and Windows; before `Show` too, and kept for the next `Show`; a `Dialog` has
it too, and a height-0 dialog fits no lower than its minimum height, which
wins over the fit's clamp to the work area), `Maximize`
(before `Show` too), `Unmaximize`, `SetContent`, `SetMenuBar`,
`OnClose(func() bool)` (false keeps the window open; quitting the app asks
it too, and a yes quits again; it asks once at a time), `Close`. A window
that closed shows again with `Show`, and `Native()` is nil while it is not
shown.
`UseHeaderBar(subtitle)`, before `Show`, draws the title bar with `ui` and
returns a `*HeaderBarWidget` with `PackStart` and `PackEnd`. The bar is 32 DIPs
high (the platform's title bar height, if higher) and grows to hold a subtitle or a
child taller than the bar; its children are centered, and the title (with its
subtitle) is centered on the window's width, as Windows and macOS do. A title that
would run under the items or the window buttons moves aside instead, and is
ellipsized. Under a Wayland
compositor that decorates windows (Sway, KWin), the compositor's title bar
shows above the header bar; `ClientDecorations(true)` on the header bar,
before `Show`, makes the header bar the only one, with the desktop's window
buttons at its ends (Sway: floating windows only; a tiled one keeps
Sway's title bar unless its config says `default_border pixel` or `none`).
Hyprland draws no title bar: there the option only adds the window
buttons.
