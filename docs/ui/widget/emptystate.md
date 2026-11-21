# EmptyState

`EmptyState(title, message)` is what a view shows when it has nothing in it:
an optional icon, a title, a message and an optional action, centered both
ways in its box.

```go
empty := widget.EmptyState("Chưa có tài khoản", "Thêm một tài khoản để bắt đầu.").
	SetIcon(inboxIcon).
	SetAction("Thêm tài khoản", addAccount)
```

- `SetTitle(s)` and `SetMessage(s)`. The title takes the skin's `Title` role of
  text (bold and larger without a skin); the message is in muted text and wraps
  at 360 DIPs at most.
- `SetIcon(svg)` shows an icon 48 DIPs high in muted text above the title;
  `nil` removes it and keeps no room for it.
- `SetAction(text, do)` shows a `Suggested` button under the message, which
  runs `do` on the event goroutine; `SetAction("", nil)` removes it.

A `DataGridView` shows it over its body while it shows no row, under its
headers: `grid.SetEmptyState(empty)` (see
[DataGridView](../widget.md#datagridview)). Elsewhere, show it in place of the
list while that has no rows, for instance in a box whose content you swap, or
with `SetVisible`. It fills the width of its box and grows along it, so give
the box the height the view had.

A skin's [`Looks.EmptyState`](../skin.md#looks) styles its box, as `bg-chrome`
does.
