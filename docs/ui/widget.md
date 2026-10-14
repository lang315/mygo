# Widgets that live on

Package `ui/widget` builds an interface from widgets you make once, keep,
and change from any goroutine, as in GTK: a label is a `*LabelWidget`, and
`SetText` changes what every window showing it draws. It is `ui` underneath:
each widget builds `ui` elements every frame, and the windows that show it
redraw when it changes.

Use it to port code written for a retained toolkit (GTK, gota, Qt), or when
the state lives in long-lived objects that background work updates. For
new code, a `ui` view over your own state is shorter and does less work.

```go
package main

import (
	"log"

	"github.com/egoist/mygo/ui/widget"
)

func main() {
	err := widget.Run("Greeter", func(w *widget.Window) {
		var name *widget.EntryWidget
		greeting := widget.Label("Type a name")
		w.SetContent(widget.VBox(8).Add(
			widget.Entry().SetPlaceholder("Name").AssignTo(&name),
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
from `NewWindow(title, width, height)`, set up the same way, then `Show()`.
`Native()` returns the `*mygo.Window` once shown. To put widgets in a
window of your own, `widget.View(w)` is a `ui` view showing `w`:
`mygo.WindowOptions{Content: ui.View(widget.View(w))}`. The
[widget tour](../../examples/widget-tour) shows every widget.

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
- **Blocking calls** (`Update`, and the dialogs to come) panic on the main
  thread, which must never wait. In a handler, they keep running the other
  queued handlers while they wait, as GTK runs its main loop under a dialog.

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
`OnFocusOut`, and `AssignTo(&v)`, which keeps a widget made inside a chain.

## What a widget keeps

Each widget builds under a key of its own, so the caret, selection, scroll
position and open popup of its `ui` elements follow it when its box
reorders. A widget that is not built (hidden, on a tab not shown, or taken
out of its box) keeps its values but loses that state: it shows again with
the caret at the start and scrolled to the top.

## Widgets

Layout:

- `VBox(spacing)`, `HBox(spacing)`: a column or a row. `Add` at the
  natural size, `Pack(w, expand, fill)`, `PackEnd`, `Remove`, `Clear`. A
  widget already in the box is not added twice.
- `ButtonBox()`: a row whose widgets sit at its end, as a dialog's buttons.
- `Grid(colSpacing, rowSpacing)`: `Attach(w, col, row, width, height)`.
- `Frame(title)`: a border and a title around one widget.
- `Card()`: a titled box whose body `ShowBody(false)` hides.
- `Flow(spacing)`: a row that wraps.
- `Scrolled(w)`: scrolls a widget larger than its room.
- `Separator()`: a line.
- `Notebook()`: pages under tabs; `AddPage`, `Page`, `SetPage`, `OnSwitch`.
- `Sidebar()`: pages chosen in a list at the side; `Add(name, title, w)`,
  `Remove`, `Selected`, `Select`, `SetMenuWidth`.

Input and display:

- `Label(text)`: a line of text.
- `Entry()`, `TextView()`: one line or several of text; `Text`, `SetText`,
  `SetPlaceholder`, `SetEditable`, `OnChange`; `Entry` also has
  `Password` and `OnEnter`.
- `Button(text)`: `OnClick`, `SetText`, `SetIcon`. `Link(text, url)`.
- `Checkbox(text)`, `Toggle(text)`, `Switch()`: on or off, with `OnChange`.
- `CheckboxGroup(row, options...)`, `TagSelect(options...)`: several of
  the options; `Values`, `SetValues`.
- `RadioGroup(row, options...)`, `Combo(options...)`: one option; `Value`,
  `SetValue`. An `Option` has the `Value` the program reads and the `Text`
  the user sees; a combo's texts must differ.
- `Spin(min, max, step)`: a number; `Value`, `Int`, `SetValue`.
  `RangeSpin` is two, with text between and after.
- `Progress()`: `SetFraction` (negative for unknown length), `SetText`.
  `Spinner()`: `Start`, `Stop`.

Windows: `SetTitle`, `SetSize`, `SetIcon`, `SetResizable`, `Maximize`,
`Unmaximize`, `SetContent`, `OnClose(func() bool)` (false keeps the
window open; quitting the app asks it too, and a yes quits again), `Close`.
`UseHeaderBar(subtitle)`, before `Show`, draws the title bar with `ui` and
returns a `*HeaderBarWidget` with `PackStart` and `PackEnd`.
