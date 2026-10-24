# SplitContainer

`SplitContainer(first, second)` shows two widgets side by side with a divider
between them. The user drags the divider, or moves it with the arrows once it
has the keyboard focus. It is [`ui.Split`](../split.md) underneath, and any
widget, another `SplitContainer` too, can be a pane.

```go
files := widget.VBox(4).Add(widget.Label("files"), widget.Button("Open"))
editor := widget.SplitContainer(widget.Label("editor"), widget.Label("console")).Stacked()

split := widget.SplitContainer(files, editor).
	SetSplitterDistance(240).
	OnSplitterMoved(func() { config.Set("sidebar", split.SplitterDistance()) })
```

- `Stacked()` puts the first pane above the second, the divider across.
- `SetSplitterDistance(d)` makes the first pane `d` DIPs wide (high when
  stacked). The divider keeps 40 DIPs from either edge; when the window leaves
  less room, the first pane gives way and takes its size back when there is
  room again, so `SplitterDistance()` keeps what the user chose.
- `SetSplitterPercent(p)` makes it `p` percent of the room, from 0 to 100, so
  both panes grow and shrink with the window. This is the default, at 50.
  `SplitterPercent()` and `SplitterDistance()` return the value of the mode
  the container is in, and 0 for the other.
- `OnSplitterMoved(fn)` runs `fn` on the event goroutine after the user moved
  the divider. A setter does not run it.

Assistive technology sees the divider as a splitter named "Divider".
In a test, `tt.Find(widget.ByRole(ui.RoleSplitter))` finds it; see
[`widget.Tester`](tester.md).
