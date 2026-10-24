# Drag and drop

Any control can be the source of a drag or take drops. It is [`ui`'s drag and
drop](../drag-and-drop.md) underneath, declared on the control instead of
called when a drag starts.

```go
card := widget.Label("Task 1").Padding(8).DragSource(task)
done := widget.Label("Done").Padding(12)
widget.OnDragOver(done, func(over bool, t *Task) { done.SetText(hint(over)) })
widget.OnDrop(done, func(t *Task) { t.Done = true })

zone := widget.Label("Drop files here").Padding(24)
zone.OnDropFiles(func(paths []string) { open(paths) })
```

## Within the window

- `c.DragSource(value)`: once the pointer pressing `c` moves a few DIPs, a copy
  of it follows the pointer. A press that does not move still clicks, so a
  `Button` can be a source; a drag does not click it. Escape cancels a drag.
- `widget.OnDrop[T](c, fn)`: `c` takes values of type `T` and `fn` gets the
  one dropped, on the event goroutine. A control takes several types with
  several `OnDrop` calls. A value of another type does not drop on it, and the
  innermost control under the pointer that takes the type gets it. A nil `fn`
  removes the target. There is no `AllowDrop`: a control takes drops while it
  has a handler. `T` is matched as a Go type assertion: for an interface `T`, a
  value whose pointer type has the methods does not match a non-pointer value,
  and `OnDrop[any]` takes everything.
- `widget.OnDragOver[T](c, fn)`: `fn(true, v)` when a value of type `T` comes
  over `c` or anything inside it, `fn(false, zero)` when it leaves, is
  dropped, the drag is cancelled, `c` is hidden, or the handler is replaced or
  removed, so the hover feedback never sticks. The end of the hover always
  comes before `OnDrop`'s handler. It does **not** make `c` take the value: a
  drop on a highlighted child falls to the nearest control around it with an
  `OnDrop` of the type, as in a drop zone with highlighted children.
- A source hidden in the middle of a drag cancels the drag silently.

## Between applications and from the OS

- `c.DragData(fn, opts...)`: a system drag of text, files or URLs
  ([package transfer](../drag-and-drop.md#between-windows-and-applications)).
  `fn` returns the data when the gesture starts. At most one `DragOptions`
  (more panic at the call). It runs on the main thread
  with no lock of the package held, so it may read widgets, and it is quick:
  it blocks the UI while the pointer waits.
- `c.OnDropData(formats, fn)`: takes drops in one of `formats` from other apps
  or `DragData`; `fn` gets data that can be read later.
- `c.OnDropFiles(fn)`: the paths of files dropped from the system's file
  manager.

## What is not here

- **DoDragDrop(data, effects).** WinForms starts a drag by calling it in a
  mouse handler. ui starts one from the press that moves, so the source is
  declared, not called; there is no way to start a drag from code.
- Effects (copy, move, link) and custom drag images: `transfer.DragOptions` and
  `transfer.DropOptions` carry the operations if you need them.

In a test, drag with the pointer: `tt.Press(x, y)`, `tt.Move(...)` and
`tt.Release(x, y)` of the embedded `ui.Tester` at the bounds of
`tt.Find(...)` nodes, and `tt.DropFiles(x, y, paths...)` for files (see
[`widget.Tester`](tester.md)).
