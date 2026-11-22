# NavBar

`NavBar()` is a side panel of groups, each a header that opens and closes
its items: an icon and a text, one of them selected. It is the navigation
pane of an app, as DevExpress's NavBarControl in its Explorer Bar style
("Mail / Calendar / Contacts", "Settings / Reports").

```go
nav := widget.NavBar().
	AddGroup("Mail").
	AddItem("inbox", "Inbox", inboxIcon).
	AddItem("sent", "Sent", nil).
	AddGroup("Contacts").
	AddItem("people", "People", peopleIcon).
	SetSelected("inbox").
	OnSelect(func(key string) { pages.Show(key) })
```

- `AddGroup(title)` adds a group after the others, open. `AddItem(key, text,
  icon)` adds an item to the last group added; `icon` is a `*ui.SVG` drawn
  16 DIPs high (times the text size the user chose), or `nil` for none. Every
  item keeps the icon's room, so that the texts of a group line up.
- An item added before any group goes into a group with no title, which has
  no header and stays open: the items at the top of a pane that belong to no
  group.
- Items are named by a `string` key, as an `ImageList`'s images are: a
  NavBar is a fixed menu of the app, not data. A key given twice panics.
- At most one item is selected, none at first. `Selected()` returns its key,
  `""` for none. A click, or Enter or Space on a focused item, selects it and
  runs `OnSelect(fn)` on the event goroutine with its key, once per change: a
  click on the selected item runs nothing. `SetSelected(key)` selects without
  running `OnSelect`, as every setter; `""` selects none, and a key that is
  not an item is ignored.
- `SetGroupOpen(i, open)` opens or closes the group of index `i`, in the order
  added, and `GroupOpen(i)` reports whether it is open. An index out of range
  does nothing, or returns `false`. A click on a header, or Enter or Space on
  it, opens and closes it, its items growing into view and out.
- The keys: Up and Down move the focus between the headers, as Home and End
  to the first and the last (ui's accordion); Tab goes into an open group. On
  an item, Up and Down move the focus to the item before or after it in its
  group, stopping at its ends. `Focus()`, and the access key of a `Fields` row
  holding the NavBar (`Row("&Go", nav)`), focus the first header.
- A selected item is ui's selected row (`Element.Selected`), as a list's: it
  fills with the theme's `RowSelected` and its text and icon take
  `RowSelectedText` (`Text` when zero); a theme that sets no `RowSelected`
  gives the accent and `AccentText`, the accent faded in a window not drawn
  as active. A hovered item fills with `RowHover`, or `SurfaceHover` where
  that is zero. Skins reach the NavBar through these tokens: HighContrast
  inverts the selected text. Screen readers get the selected item as the
  current page (`ui.CurrentPage`) and as checked, a pressed button.
- The headers show their title in semibold, on the theme's `Surface` with a
  line under it, and a chevron at the end that turns as the group opens. A
  long title or text wraps inside the bar.

The NavBar does not scroll: put it in a `ScrollPanel(nav).Vertical()` when
its groups can be taller than the window. It is opt-in: an app that makes no
NavBar does not link it.
