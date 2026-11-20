# InfoBar

`InfoBar(status, title, message)` is a message inline in a page, as WinUI's
InfoBar: an icon of its status, the title in semibold, the message wrapping at
the bar's width, and at the end an action button and a close button, each if
you set it. It spans its box.

```go
bar := widget.InfoBar(widget.StatusWarning, "Cần xác minh", "Mở email đã gửi để xác minh tài khoản.").
	SetAction("Xác minh", verify).
	SetClosable(true).OnClose(func() { log.Print("dismissed") })
```

- `SetStatus(s)`, `SetTitle(s)`, `SetMessage(s)` and their getters. The status
  (see [Badge](badge.md) for the list) colors the bar and picks its icon:
  info, a check for success, a warning sign, an error.
- `SetAction(text, do)` shows one button at the end, which runs `do` on the
  event goroutine; `SetAction("", nil)` removes it.
- `SetClosable(true)` shows a close button. Clicking it hides the bar
  (`Visible()` is false), then runs `OnClose`'s function, once per click.
  `SetVisible(false)` from your code hides it without running `OnClose`, and
  `SetVisible(true)` shows it again.

Its colors are the status family's: the `Subtle` fill in a 1 DIP `Border`,
with `RadiusControl` corners and the icon in `Base`, under a skin; the theme's
status color at 14 % alpha with no border, without one. A skin's
[`Looks.InfoBar`](../skin.md#looks) styles the bar, and may name
`bg-status`, `text-status` and `border-status`.

Assistive technology announces it as it shows: it is an alert
(`ui.RoleAlert`) for `StatusWarning` and `StatusDanger`, a status
(`ui.RoleStatus`) for the rest. The close button is named "Close", in the
user's language.
