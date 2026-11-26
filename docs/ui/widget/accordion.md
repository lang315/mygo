# Accordion

`Accordion()` is sections one above the other in a bordered box, each a
header over its controls, as DevExpress's `AccordionControl`. One section is
open at a time: a click on a header, Enter or Space opens it and closes the
others, and a click on the open one closes it. Up and Down move the focus
between the headers. It is [`ui.Accordion`](../accordion.md) underneath.

```go
settings := widget.Accordion().
	AddSection("General", widget.TextBox().SetPlaceholder("Name"), widget.CheckBox("Start at login")).
	AddSection("Privacy", widget.CheckBox("Send usage data")).
	SetOpen(0)
settings.OnChange(func() { log.Println("open:", settings.Open()) })
```

- `AddSection(title, controls...)` adds a section holding the controls one
  above the other, as a `VBox` does: its radio buttons are one group.
- `Open()` is the index of the open section, in the order added, or -1 for
  none, the default.
- `SetOpen(i)` opens section `i`, or closes them all for -1, without running
  `OnChange`. Any other index out of range is ignored.
- `OnChange(fn)` runs on the event goroutine after the user opened or closed
  a section.

A closed section's controls are not built: they keep their values, as a
hidden control does. For a navigation pane of items with icons, use
[`NavBar`](navbar.md).
