# MenuBar

`MenuBar(menus...)` is a row of menus that mygo draws in the window's
content, as WinForms' `MenuStrip` docked in a form. Put it where it goes:
first in the content puts it right under the header bar of
`Form.UseHeaderBar`, which the system's menu bar cannot do.

```go
f := widget.NewForm("Notes", 800, 600)
f.UseHeaderBar("") // the title bar the app draws
bar := widget.MenuBar(
	widget.MenuItem{Text: "&File", Children: []widget.MenuItem{
		{Text: "&Open…", OnClick: openFile, Shortcut: widget.Shortcut{Mods: ui.Ctrl, Key: ui.KeyO}},
		{Text: "Open &recent", Children: []widget.MenuItem{{Text: "notes.txt", OnClick: openRecent}}},
		{},
		{Text: "E&xit", OnClick: f.Close},
	}},
	widget.MenuItem{Text: "&View", Children: []widget.MenuItem{
		{Text: "&Word wrap", Checked: wrap, OnClick: toggleWrap},
		{Text: "&Delete all notes", Destructive: true, OnClick: deleteAll},
	}},
)
f.SetContent(widget.VBox(0).Add(bar, editor))
f.Shortcut(ui.Ctrl, ui.KeyO, openFile) // the shortcut an item shows answers nothing itself
```

- Each `MenuItem` is a title, and its `Children` are the items of its menu.
  They have what a `MenuItem` has: submenus, separators (`MenuItem{}`),
  `Checked`, `Disabled`, `Icon` or `IconPNG`, `Destructive`, and `Shortcut`,
  which the item shows at its end. A shortcut shown answers nothing: give
  the form the same `Form.Shortcut`. A title `Disabled`, or with no item
  but separators, opens nothing and looks disabled. Only a `Checked` item
  is a check box item: a toggle not checked is a plain item, and a menu
  with no item checked has no column for the mark, as in WinForms.
- `SetMenus(menus...)` sets them again, as when an item is checked or
  disabled: menus take their items, not a function building them (see
  [Menus](../widget.md#menus)).
- A click on a title opens its menu, and a second click closes it. While a
  menu shows, the pointer on another title opens that one instead. Left
  and Right go to the menu before and after, past the titles that open
  nothing, round the bar; Right on an item that opens a submenu opens it,
  and in a submenu Left closes it. A menu opened by the keys starts on its
  first item. Up, Down, Home, End, Enter, Space and typing the first letters
  of an item are those of every drawn menu ([`ui.MenuBase`](../menu-button.md#without-a-look)).
- Escape closes the menu, and the keyboard focus goes back to where it was
  before the first menu opened. A click on a title does not take the focus,
  and Tab passes the titles by, as the system's bar does.
- **Access keys.** Alt and the letter after `&` open a menu on its first
  item (Alt+Shift too), and the letters are underlined while Alt is held.
  They come before the access keys of the form's controls: a button
  `&File` does not answer Alt+F. A `Form.Shortcut` of the same key wins.
  The `&` of an item is taken out of its text; the letter does not choose
  the item, but typing the first letters of an item goes to it. Access keys
  answer only in a form, as every [mnemonic](mnemonics.md).
- **The bar's keyboard mode.** F10, and on Windows Alt pressed and let go
  alone, take the keyboard to the bar, as in the system's: the first title
  that opens a menu takes the focus, and the letters are underlined. Left
  and Right go along the bar (the other way round right to left); a
  title's letter, without Alt, opens its menu, as do Down, Up, Enter and
  Space. Escape, F10 or Alt alone again leave the mode, and the focus goes
  back to the control that had it, or nowhere when none had it; so does a
  menu opened in the mode as it closes, chosen or not (Windows goes back
  to the bar on Escape), and F10 or Alt alone while a menu shows close it.
  The mode ends too when the window goes inactive, the bar is hidden or
  its menus are set again (the focus goes back), and when a click or Tab
  takes the focus elsewhere (it stays there). F10 held does not go in and
  out. The first `MenuBar` of a form answers; only a form with one asks for
  these keys, so without one Windows has F10 and Alt alone as before (the
  window menu). A form that also has a bar of the system's (`SetMenuBar`,
  `MenuStrip`) leaves them to that bar. Not in `widget.View`, nor on macOS.
  Alt alone reaches the form as `ui.KeyMenuBar`, which only a shortcut
  gets: no control sees a key, as a terminal would send F10 to its program.
- Assistive technology sees a menu bar of menu items, each expanded while
  its menu shows, and the menus as menus of menu items and check box items.

## Under a HeaderBar

A form with a `HeaderBar` has no title bar of the system's, so the
system's menu bar of `Form.SetMenuBar` or `MenuStrip` does not sit under
the header:

| | `MenuBar` | `SetMenuBar`, `MenuStrip` |
|---|---|---|
| Windows | Where the content puts it. Alt+letter opens its menu, and Alt alone or F10 take the keyboard to it; the system does not beep or enter its own menu mode. | No bar: a window without a caption has no room for Win32's. Alt and F10 open its menus in a popup below the title bar. |
| Linux | Where the content puts it. | GTK's bar, at the top of the window, above the HeaderBar. |
| macOS | Where the content puts it, without access keys: Option types characters there. | The app's menu bar at the top of the screen (`MenuStrip`; `SetMenuBar` is not used there). |

On macOS the menu bar at the top of the screen stays the norm: keep a
`MenuStrip` or `mygo.App.SetMenu` there, with the roles of the app menu
and the Edit menu, and show a `MenuBar` only where it is not macOS
(`runtime.GOOS != "darwin"`) unless the app wants its menus in the window
too.

## Not yet

- An item's letter choosing it while its menu shows.
- `MenuStripItem`'s live setters: set the menus again instead.
- Drawn by `ui`, the menus do not leave the window: a menu longer than the
  window scrolls.
