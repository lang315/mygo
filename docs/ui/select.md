# Select

<!-- preview select 230 -->

`ui.Select` creates a drop-down choosing one of a list of strings into a
`*string`: a click on it, or Enter, Space, Up or Down while it has the
keyboard focus, opens its options below it, where the arrows, Home and End
move among them and Enter, Space or a click chooses one; Escape closes
them. `Changed` reports a new
choice.

```go
ui.Select(c, &app.plan, []string{"Free", "Pro", "Team"}).Label("Plan")
```

The popup shows above the select where there is no room below it, and gives
the focus back to the select as it closes. Of many options, it shows the
first that fit in 70 of the theme's spaces and scrolls the rest, keeping
the highlighted one in view; the popups of `SelectBase` and `ComboboxBase`
do too, unless their panel sets another `MaxHeight`. In a window too short
for that, the popup shrinks to the room on the roomier side of the select.
A press in the popup that is not on an option, such as on a header row of
a `SelectBase`, leaves the keyboard on the select.

For a long list that typing filters, use a [combobox](combobox.md); for a
few options shown at once, [radio buttons](radio.md).

## Without a look

`ui.SelectBase` is a select without a look, choosing a value of any
comparable type: a `Trigger` opening a `Popup` of `Item`s, which the arrows
highlight (`Highlighted`) and Enter chooses. `Highlight` moves the
highlight to an option, as a select of yours may as the user types its
first letters:

```go
sel := ui.SelectBase(c, &app.size)
sel.Trigger.Gap(6).Padding(6, 10).Radius(8).Border(1, t.Border).Children(func() {
	ui.Text(c, app.size)
	ui.Icon(c, chevron)
})
sel.Popup(func(panel ui.Element) {
	panel.Margin(4, 0, 0, 0).Padding(4).Radius(10).Background(t.Background).Border(1, t.Border)
	for _, size := range sizes {
		item := sel.Item(size).Gap(8).Padding(6, 10).Radius(6)
		if item.Highlighted() {
			item.Background(t.Accent).TextColor(t.AccentText)
		}
		item.Children(func() {
			ui.Icon(c, sizeIcons[size])
			ui.Text(c, size)
		})
	}
})
```

An option holds whatever its children build: here an icon before the
label, which `ui.Select`'s options, strings, do not show.

See [custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a pop-up button whose value is the choice, named
by its `Label`, and, while it is open, its options, following the one the
arrows are on.
