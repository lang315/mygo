# Segmented

`Segmented(labels...)` is one choice among a row of joined buttons, as a switch
between views. DevExpress builds it from `CheckButton`s of one `GroupIndex`;
this is the familiar segmented control. A click chooses a segment, as do the
arrows while it has the keyboard focus. It is
[`ui.Segmented`](../segmented.md) underneath.

```go
var view *widget.SegmentedWidget
view = widget.Segmented("List", "Grid").OnChange(func() { fmt.Println("view", view.Selected()) })
period := widget.Segmented("Day", "Week", "Month").SetSelected(1).SetFillWidth(true)
```

- It starts at the first segment. `Selected()` is the index chosen. A
  Segmented always has one choice, as `ui.Segmented` does: there is no "none".
- `SetSelected(i)` chooses segment `i` and does not run `OnChange`. An index
  out of range, -1 included, is ignored.
- `SetFillWidth(true)` makes it as wide as the box that holds it, its segments
  sharing the width equally (`ui`'s `FillEqually`).
- `OnChange(fn)` runs on the event goroutine after the user chose a segment.

Assistive technology sees a radio group, each segment a radio button; in a
test, `tt.Find(widget.ByLabel("Grid")).Click()` chooses one.
