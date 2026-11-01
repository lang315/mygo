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
| `Alert`, `Confirm` | MessageBox | `Alert`, `Confirm` |
| `Custom` | UserControl | — |
| `MenuStrip` (`Form.MenuStrip`) | MenuStrip | `MenuBar` |
| `ToolStrip` | ToolStrip | `Toolbar` |
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
than the screen's work area less 80 DIPs, where the content scrolls instead.
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
says no or `SetDeletable(false)`.

```go
dt := widget.TestDialog(t, d.Dialog)
go func() { result <- d.Run() }()
dt.Type("Ada")
dt.Click("OK")
got := <-result
```

Ready-made dialogs, each with an icon for its `Kind` (`Info`, `Warning`,
`Question`, `Error`), 420 DIPs wide, their OK and Cancel in the app's
language:

- `Alert(parent, kind, title, text)` waits for OK.
- `ColorDialog()`: a color picker with OK and Cancel; see [ColorDialog](widget/colordialog.md).
- `FontDialog()`: families, styles, a size and a sample, with OK and Cancel; see [FontDialog](widget/fontdialog.md).
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
  not change.
- **`StatusStrip`** holds `StatusLabel(text)` (`SetText`, `SetIcon`),
  `Spring()` (takes the free width), `StatusProgress()` (a small
  `ProgressBar`) and `Separator()`. `Form.StatusStrip` docks it below the
  content.
- **`ToolStrip`** is a `ui.Toolbar`: its buttons take no face until hovered,
  it is one stop of Tab with the arrow keys moving among its items, and
  what does not fit goes from the last item into a menu at its end (a
  `ComboBox` there under its chosen text, a `DropDownButton` under its own);
  there is no overflow of the package's own. `ToolButton(text)` has `SetIcon`,
  `SetTooltip` (from every control), `OnClick`, and `SetToggle` with
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
as in WinForms or GTK:

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
- **Variants** apply on top while their state holds, in this order, a later
  one winning: `Dark`, `Hover`, `Focused`, `Pressed`, `Disabled`. A
  state's variant can have a `Dark` of its own, as Tailwind's `dark:hover:`.
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
| Text | `text-xs … text-9xl`, `font-thin … font-black`, `italic`, `not-italic`, `font-mono` |
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
| `TextBox` | `string` | `Bind` |
| `DomainUpDown` | the selected item's text, a `string` | `Bind` |
| `CheckBox` | `bool` | `Bind` |
| `ComboBox` | the `Value` of its option, a `string` (the text, if `Editable` first) | `Bind` |
| `NumericUpDown` | `float64` | `Bind`; `BindInt` for an `int` |
| `TrackBar` | `float64` | `Bind` (clamps to its range, as `NumericUpDown`) |
| `ComboBox` of a yes and a no | `bool` | `BindBool(c, yes, no, get, set)`: gives the combo the two options and makes it a plain choice, not `Editable`; `set` is called for yes and no only |

- The control takes `get()`'s value at once. After each edit of the user,
  `set(v)` runs with the new value on the event goroutine, after the
  control's own `OnChange` (`Bind` does not replace it), and off the
  package's lock, so `set` may call the control's getters.
- **A program setter does not call `set`.** `SetText`, `SetChecked` and
  `SetValue` change what the control shows, not the source: the program
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
(by `Text` otherwise), `Width`, `MinWidth`, `Align`, `Sortable`,
`Filterable`, `Filter` (case-insensitive containment otherwise), `Spans` (draws
`[]ui.Span` in the cell, as `Label.Rich` does, for a badge or a colored word:
`Text` still sorts, searches, copies and names the row, while assistive
technology reads the cell as its spans' text; a selected row's span text takes
the selection's color and keeps the spans' backgrounds, which copies the spans
of each cell drawn, so keep them with the rows if you can), `Color`
(not on a selected row, whose text takes the selection's color) and
`Image`. The column functions and the key run under the package's
lock: they read the row and call no widget method.

- Rows: `SetRows`, `AddRow` and `AddRows` (after the others, or in place
  of the row of the same key), `UpdateRow(k, fn)` (fn changes the row in
  place, not its key), `DeleteRows(ks...)`, `Clear`, `Row(k)`, `Len`.
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
  getters, but should be quick.

  ```go
  table.CopyFunc(func(ids []string) string { return strings.Join(ids, "\n") })
  ```

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
- `GroupBox(title)`: a border and a title around one widget.
- `Card()`: a titled box whose body `ShowBody(false)` hides.
- `FlowPanel(spacing)`: a row that wraps.
- `ScrollPanel(w)`: scrolls a widget larger than its room.
- `SplitContainer(first, second)`: two panes with a divider; see [SplitContainer](widget/split-container.md).
- `TrackBar(min, max)`: a slider; see [TrackBar](widget/trackbar.md).
- `DomainUpDown(items...)`: arrows over a list of texts; see [DomainUpDown](widget/domainupdown.md).
- `DragSource`, `OnDrop[T]`, `OnDragOver[T]`, `DragData`, `OnDropData`, `OnDropFiles`: drag and drop; see [Drag and drop](widget/drag-and-drop.md).
- `Chart(kind)`: a line, bar or pie chart; see [Chart](widget/chart.md).
- `richtextbox.New()`: styled text, in package `ui/widget/richtextbox`; see [RichTextBox](widget/richtextbox.md).
- `WebBrowser()`: a web page; see [WebBrowser](widget/webbrowser.md).
- `MaskedTextBox(mask)`: an input that takes what a mask allows; see [MaskedTextBox](widget/maskedtextbox.md).
- `ErrorProvider()`, `OnValidating`, `Form.ValidateChildren`: an error mark on a control and an explicit validation; see [ErrorProvider](widget/errorprovider.md).
- `&` mnemonics in a `Button` and the labels of `Fields` (Alt+key; off on macOS); see [Mnemonics](widget/mnemonics.md).
- `ImageList(size)`: images by key, shared by `ListBox`, `TreeView` and menus (`ImagesBy`, `PNG`); see [ImageList](widget/imagelist.md).
- `Separator()`: a line.
- `TreeView(key, text)`: nodes in a tree, with lazy children and check boxes; see [TreeView](widget/treeview.md).
- `ListBox(key, text)`, `CheckedListBox(key, text)`: items in a list, chosen by key, with check boxes on the second; see [ListBox](widget/listbox.md).
- `TabControl()`: pages under tabs; `AddPage`, `Page`, `SetPage`, `OnSwitch`.
  `SetPageLabel(i, s)` renames a tab ("Items (3)"), `InsertPage(i, label, w)`
  and `RemovePage(i)` add and take out pages. The page shown stays the page
  shown, whatever its index becomes; removing it shows the one before it (the
  first, if it was the first). The tab with the keyboard focus keeps it; if its
  page goes, the tab now shown takes it. As
  `SetPage` does, none of them runs `OnSwitch`: the app made the change.
- `Sidebar()`: pages chosen in a list at the side; `Add(name, title, w)`,
  `Remove`, `Selected`, `Select`, `SetMenuWidth`.

Input and display:

- `DateTimePicker()`, `MonthCalendar()`: a `time.Time` in a date and time field, or a calendar; see [DateTimePicker](widget/date-time-picker.md).
- `Label(text)`: a line of text; `Rich(spans...)` shows styled `ui.Span`s
  instead, where gota used Pango markup. `ParseMarkup(s)` makes the spans of a
  Pango subset: `<span foreground background weight>`, `<b>`, `<i>`, `<u>` and
  the entities; anything else, malformed markup too, stays as text and never
  fails: `Label("").Rich(widget.ParseMarkup(status)...)`.
- `Icon(svg, size)`: an SVG icon in the color of the text (`TextColor`
  changes it). There are no named icons: embed the SVGs you need.
- `PictureBox(data, w, h)`: a PNG or GIF, or a JPEG, WebP or BMP once the
  program imports its decoder (see "Formats" in [Images](image.md)),
  `SetImage` to change it. The same bytes of an image up to 64 KB, as an icon, decode once (the
  last 256 are kept); bytes that are no image leave an empty room.
- `TextBox()`, `MultilineTextBox()`: one line or several of text; `Text`,
  `SetText`, `SetPlaceholder`, `SetEditable`, `OnChange`; `TextBox` also has
  `Password`, `OnEnter`, `MaxLength(n)` (runes, as in WinForms: typing or
  pasting past it is cut at it, and so is `SetText`; 0 for no limit) and
  `Icon(bitmap)` (16 DIPs, inside the box before the text; nil removes it).
- `Bind(control, get, set)`, `BindInt`, `Reload(control)`: tie a `TextBox`,
  `CheckBox`, `ComboBox` or `NumericUpDown` to a value kept elsewhere; see
  [Binding](#binding).
- `Button(text)`: `OnClick`, `SetText`, `SetIcon`; `Suggested()` and
  `Destructive()` are the button's own style (tw's `suggested-action` and
  `destructive-action`). `LinkLabel(text, url)`; `OnClick(fn)` runs `fn` on the event goroutine
  in place of opening `url` (click, Enter or Space, assistive technology; nothing
  when disabled), as WinForms' `LinkClicked`.
- `CheckBox(text)`, `Toggle(text)`, `Switch()`: on or off, with `OnChange`.
- `CheckBoxGroup(row, options...)`, `TagSelect(options...)`: several of
  the options; `Values`, `SetValues`.
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
  `SectionLen`, `Selected`, `Unselect`; sections, icons and texts change
  while the combo shows, and it redraws. `Editable()` lets the user type a
  text that is no option, which `Text` and `SetText` give and set; `Value`
  is then the `Value` of the option with that text, or ""; `OnChange` runs
  on typing and on choosing. `Bind` takes the text of a combo made
  `Editable` before it is bound, and the `Value` of one bound before.
- `MenuButton(text)`: see [Menus](#menus).
- `NumericUpDown(min, max, step)`: a number; `Value`, `Int`, `SetValue`;
  `SetAccessibleName(s)` names it and its buttons for assistive technology.
  `RangeSpin` is two, with text between and after; they are "minimum" and
  "maximum" to assistive technology unless `SetAccessibleNames(low, high)`.
- `ProgressBar()`: `SetFraction` (negative for unknown length), `SetText`.
- `DataGridView(key, columns...)`: rows in columns; see [DataGridView](#datagridview).
  `Spinner()`: `Start`, `Stop`.
- `ListView(key, columns...)`: rows in columns, as icons, or as a list, over the same choice and checks; see [ListView](widget/listview.md).

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
