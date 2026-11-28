# SuperTip

`SetSuperTip` gives any control a rich tooltip, as DevExpress's
`SuperToolTip`: a bold title, a body with an image at its start, a
separator, then a footer.

```go
widget.Button("Paste").SetSuperTip(widget.SuperTip{
	Title:  "Paste (Ctrl+V)",
	Body:   "Puts the text of the clipboard where the caret is.",
	Image:  clipboardIcon, // a *ui.Bitmap, decoded once
	Footer: "Press F1 for more help.",
})
```

- Every part may be left empty, and an empty part takes no room. The
  separator shows only with a footer under something.
- The tip grows to 320 DIPs wide, and its text wraps there. The image shows
  at its own size, at the body's start (its end in a right-to-left layout),
  aligned with the body's first line.
- It takes the place of `SetTooltip`'s text while it is set.
  `SetSuperTip(widget.SuperTip{})` removes it, and the text tooltip comes
  back.
- An `ErrorProvider`'s error shows as the tip instead, as it does in place of
  a text tooltip.
- Call it from any goroutine. The next frame shows the new tip.

## Timing and look

It shows, moves and hides as a text tooltip does (see
[Tooltip](../tooltip.md)). It shows after the pointer rests on the control
for the theme's `TooltipDelay`, or at once when the keyboard focus comes to
the control from the keyboard. It goes above the control, centered, or below
it when there is no room above. A click or Escape hides it until the pointer
leaves and comes back.

It has a text tooltip's look: the skin's tooltip colors, the theme's
`Inverse` and `InverseText`, or the theme turned over when the theme sets
neither. The text is one size below the theme's. The separator is the text
color faded into the fill.

## Accessibility

Assistive technology reads the title, the body and the footer as the
control's description. Each part is followed by ". ", or by a space when it
already ends a sentence: "Paste (Ctrl+V). Puts the text of the clipboard
where the caret is. Press F1 for more help." In a `Fields` row with
`Describe` text, that text comes first and the SuperTip's follows, as a
text tooltip's does. The tip itself is a tooltip, which assistive technology
does not read.

## Cost

A control without a SuperTip draws and allocates as before. `SetSuperTip`
stores the tip and installs the code that builds it, so an app that never
calls it does not link that code. A control whose tip is not showing
allocates nothing in a frame.
