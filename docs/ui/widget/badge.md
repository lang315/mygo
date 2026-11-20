# Badge

`Badge(text)` is a short text in a pill colored by its status, as "Live" or
"Sai mật khẩu" beside an account. It never grows: in a box that stretches it,
the pill keeps the width of its text.

```go
live := widget.Badge("Live").SetStatus(widget.StatusSuccess)
sync := widget.Badge("Đang chạy").SetStatus(widget.StatusRunning).SetDetail("Bước 2/5")
```

- `SetStatus(s)` and `Status()`: one of `StatusNone` (the default),
  `StatusInfo`, `StatusRunning`, `StatusSuccess`, `StatusWarning` and
  `StatusDanger`. `Status.String()` names it ("none", "info", …).
- `SetText(s)` and `Text()`.
- `SetDetail(s)` and `Detail()`: the current step of a running badge, drawn
  after the pill in muted text at normal weight. It shows only for
  `StatusRunning`, which also draws a 6 DIP dot before the text.

The pill is 20 DIPs high with 12 DIP semibold text, padded 2 units on each
side, and grows with the text scale. Its colors:

- **With a skin:** the status family's `Subtle` fill and `Text` (Info's for
  `StatusInfo` and `StatusRunning`), the dot in its `Base`; `StatusNone` is
  `TextMuted` on `SurfaceHover`.
- **Without:** the theme's `Success`, `Warning`, `Danger` or `Accent` at 14 %
  alpha, under the theme's `Text`; `StatusNone` is `ui.Badge`'s, `Text` at 10 %.

A skin's [`Looks.Badge`](../skin.md#looks) styles the pill, not the detail, and
may name the status's colors with `bg-status`, `text-status` and
`border-status`, so one look serves every status.

Assistive technology reads the badge as one text: its text, then the detail
when it shows ("Đang chạy, Bước 2/5"). In a test,
`tt.Find(widget.ByRole(ui.RoleText).Label("Live"))` finds it (see
[`widget.Tester`](tester.md)).
