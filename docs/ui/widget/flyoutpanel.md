# FlyoutPanel

`FlyoutPanel(owner, content)` draws `owner` and slides a panel of `content`
in over it from one of its edges, as DevExpress's `FlyoutPanel`. It sits
where `owner` would be. The panel stays inside the owner's box, and the rest
of the form stays as it is.

```go
doc := widget.MultilineTextBox().Expand()
wrap := widget.CheckBox("Wrap lines")
fly := widget.FlyoutPanel(doc, widget.VBox(8).Add(widget.Label("View").Bold(), wrap))
fly.SetEdge(widget.FlyoutRight)
fly.OnClosed(func() { _ = wrap.Checked() })
widget.VBox(8).Add(widget.Button("View…").OnClick(func() { fly.ShowPopup() }), fly)
```

- `ShowPopup()` slides the panel in, in ui's motion. While the desktop asks
  for less motion, it shows at once. `HidePopup()` slides it out.
- `SetEdge(e)` sets the edge it comes from: `FlyoutRight` (the default),
  `FlyoutLeft`, `FlyoutTop` or `FlyoutBottom`.
  - `FlyoutRight` is the end and `FlyoutLeft` the start, so right to left
    they swap sides.
  - From the left or the right, the panel is as high as the owner and as
    wide as its content. From the top or the bottom, it is as wide as the
    owner and as high as its content.
- `SetCloseOnOutsideClick(on)`: a click outside the panel closes it, unless
  this is off.
- `IsOpen()` reports whether the panel shows.
- `OnClosed(fn)` runs after the user closed the panel, with Escape or a
  click outside it. It is posted to the event goroutine. `HidePopup` does
  not run it, as a setter runs no handler.

## Behaviour

- Escape closes the panel, wherever the focus is, unless the focused
  control takes it.
- As the panel shows, the focus goes to it, and Tab goes on into its
  content. As it goes, the focus comes back to where it was.
- While the panel is open, the owner still draws but takes no pointer, with
  `SetCloseOnOutsideClick` on or off.
- The panel's background, border and shadow are the theme's and the skin's,
  as a drop-down's.
- The panel is drawn over the window, on the owner's box. In a
  `ScrollPanel` or another box that clips the owner, it is not clipped
  with it: it covers what is past the clip too.
