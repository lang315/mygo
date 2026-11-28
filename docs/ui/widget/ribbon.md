# Ribbon

`Ribbon()` is a command bar of tabs ("Home", "Insert"), each holding groups
("Clipboard", "Font") of large buttons, an icon above the text, and small
controls stacked three high. It is the command bar of Office and of
DevExpress's WinForms apps. Put it first in a form's content:

```go
paste := widget.ToolButton("Paste").SetIcon(pasteIcon).OnClick(doPaste)
bold := widget.ToolButton("Bold").SetIcon(boldIcon).SetToggle(true)
ribbon := widget.Ribbon().
	AddTab("Home").
	AddGroup("Clipboard").AddLarge(paste).AddSmall(cut, copy, painter).
	AddGroup("Font").AddSmall(bold, italic, underline, size).
	AddTab("Insert").
	AddGroup("Tables").AddLarge(table)
form.SetContent(widget.VBox(0).Add(ribbon, editor.VExpand()))
```

- `AddTab(title)` adds a tab after the others. `AddGroup(title)` adds a
  group to the last tab; before any tab it adds a tab with no title first.
- `AddLarge(buttons...)` adds `ToolButton`s to the last group, each drawn
  large: its icon 32 DIPs high (times the text size the user chose) above its
  text, centred, as high as the group. A long text wraps to two lines at most,
  no wider than twice the icon. A button with no icon shows its text alone.
- `AddSmall(controls...)` adds controls to the last group: tool buttons,
  drop-down buttons (`DropDownButton`), combo boxes, check boxes. Consecutive
  small controls fill columns of three, top to bottom; a large button is a
  column of its own. Items keep the order added.
- Before any group, `AddLarge` and `AddSmall` add a group with no title
  first, which has no title line. A control the ribbon holds already, or nil,
  is not added again.
- The ribbon's tool buttons are what they are in a `ToolStrip`: `OnClick`,
  `SetToggle` with `OnChange`, `SetTooltip`, `SetDisabled`. They take no face
  until hovered, pressed or checked.
- `Tab()` returns the index of the tab shown and `SetTab(i)` shows tab `i`,
  as `TabControl`'s `Page` and `SetPage`: an index out of range is kept until
  the next frame, which clamps it. A click on a tab, or the arrows while a tab
  has the focus, shows it.
- `SetCollapsed(true)` shows the tabs alone, and the content below moves up;
  `Collapsed()` reports it. A double click on a tab, and Ctrl+F1 anywhere in
  the window, toggle it. While collapsed, a click on a tab shows it and
  expands the ribbon, in one click; a double click on a tab of a collapsed
  ribbon leaves it expanded.
- Keys and assistive technology: the tabs take the arrows, as a
  `TabControl`'s; Tab goes from them into the shown tab's controls. `Focus()`
  focuses the tab shown. Each group is a group named by its title.
- The look: under a skin, the tabs take its `TabHeader` look and the groups
  row its `Chrome`; without one, the tabs are drawn as document tabs, the
  shown one filled with the theme's `Background` and joined to the groups
  row, filled with it too. A line of `Border` runs under the groups row and
  between groups. A group's title is under its items, in `TextMuted`, smaller
  than the text.

When the window is narrower than the groups, the groups row clips at its end:
the groups do not shrink. The ribbon is opt-in: an app that makes no ribbon
does not link it.

## Keyboard

Key tips reach every command from the keyboard, as in Office on Windows.
The ribbon gives them their letters; there is no API to set them.

- F10, and on Windows Alt pressed and let go alone, show a letter on each
  tab. The shown tab takes the keyboard focus from the control that has it,
  which gets it back as the tips end.
- Typing a tab's letter shows that tab, expanding a collapsed ribbon, with a
  letter on each of its commands. Alt and a tab's letter go straight there,
  as an access key does.
- Typing a command's letter acts on it and ends the tips: a tool button
  clicks, a toggle checks or unchecks and runs `OnChange`, and any other
  control takes the focus. A drop-down button takes the focus too: Down or
  Enter then opens its menu. A disabled command's letter does nothing but
  end the tips.
- Escape on the commands goes back to the tabs' letters; on the tabs it ends
  the tips. F10 or Alt alone again, any other letter or digit, a press in the
  window and the window going inactive end them too.
- Letters: a tab takes the first letter of its title that no tab before it
  took ("Home", then "Help" gets E), else the first free letter, A to Z, then
  1 to 9. A command takes its letter the same way among the commands of its
  tab, in the order of its groups and items, from a tool button's text (its
  tooltip with none), else the control's accessible name or text. Accents
  are dropped: "Định dạng" gets D. A tab or a command past the 35th has no
  letter.
- Each letter shows in a badge centred on its element's bottom edge: a
  rounded box of the theme's `Text`, the letter in its `Background`, at
  0.75 rem, so it grows with the text size the user chose. The badges move
  nothing. While a badge shows, assistive technology reads its letter as the
  element's key shortcut.
- The tips are off where access keys are: on macOS, whose menu bar has the
  keys. In a form with a `MenuBar`, or the system's menu bar (`SetMenuBar`,
  `MenuStrip`), F10 and Alt alone stay the bar's; Alt and a tab's letter
  still work, unless a menu's access key has that letter. A ribbon outside a
  form (`widget.View`) has no tips.

## Quick Access Toolbar

The Quick Access Toolbar is a row of small icon buttons, Office's Save and
Undo, that stays in view whatever tab shows:

```go
save := widget.ToolButton("Save").SetIcon(saveIcon).OnClick(doSave)
undo := widget.ToolButton("Undo").SetIcon(undoIcon).OnClick(doUndo)
ribbon.AddQuickAccess(save, undo).OnQuickAccessChange(func() {
	settings.Set("qat", ribbon.QuickAccess()) // save what the user chose
})
if saved, ok := settings.Strings("qat"); ok { // and restore it, if the user changed it
	ribbon.SetQuickAccess(saved...)
}
```

- `AddQuickAccess(buttons...)` adds `ToolButton`s after those the toolbar
  holds: the ribbon's own commands, or buttons only in the toolbar. nil,
  and a button it holds, are skipped. The buttons are the same widgets
  wherever they show: a click runs `OnClick` once, and a toggle checks in
  both places.
- Where it shows: in a row above the tabs; in the title bar's start area,
  after `PackStart`'s items, when the form has a `HeaderBar`, where a press
  on the empty bar still drags the window; or below the ribbon after
  `SetQuickAccessBelow(true)`. Each button shows its icon alone (its text
  with none), named by its text, or its tooltip with none.
- The user changes it: a right-click on a large or small tool button of the
  ribbon offers "Add to Quick Access Toolbar", disabled while the toolbar
  holds it; a right-click on a toolbar button offers "Remove from Quick
  Access Toolbar" and "Show below the ribbon", checked while it is. The
  menus come with the first call of a toolbar method (`AddQuickAccess()`
  with no buttons too): a ribbon that calls none offers none. A command
  with a `ContextMenu` of its own keeps it; controls other than tool
  buttons offer no menu.
- `QuickAccess()` returns the buttons' names in order (their text, or their
  tooltip with none), and `SetQuickAccess(names...)` sets the toolbar from
  them, finding the ribbon's commands and the buttons `AddQuickAccess`
  gave. A name it has no button for is skipped, and so is a name twice.
  `SetQuickAccess()` with no names empties the toolbar: restore only what
  was saved. A button only in the toolbar (Save, Undo) that the user
  removed has no command to add it back from: only the app brings it back.
- `OnQuickAccessChange(fn)` runs `fn` on the event goroutine after the user
  adds or removes a button; not after the setters, nor when the toolbar
  moves below the ribbon (there is no getter for where it is).
- Key tips: F10 or Alt alone shows 1 to 9 on the toolbar's first nine
  buttons, beside the tabs' letters, which never take those digits; typing
  one acts on its button as a command's letter does. A tenth button and
  after have no tip.

## Contextual tabs

A contextual tab shows only while what it acts on is selected, as Office's
Picture Tools Format tab:

```go
ribbon.AddContextualTab("Picture Tools", "Format").
	AddGroup("Arrange").AddSmall(rotate, crop)
// as the app's selection changes:
ribbon.SetCategoryVisible("Picture Tools", selection.IsPicture())
```

- `AddContextualTab(category, title)` adds a tab to `category`, after the
  others; the `AddGroup` calls that follow fill it, as `AddTab`'s. Its
  index counts for `Tab` and `SetTab` whether it shows or not; `SetTab` on
  a tab whose category is hidden is ignored, the tab shown stays.
- `SetCategoryVisible(category, visible)` shows or hides the category's
  tabs. They start hidden. A hidden tab takes no room, no key tip, no
  access key, and the arrows pass it. Hiding the category of the tab shown
  shows the first tab.
- The tabs of a category that follow one another sit under one band of the
  category's name, while the other tabs keep to the band's bottom line. The
  band takes the skin's series of colors (its `Chart`, as charts draw; the
  charts' own colors without a skin) by the order the categories came in,
  its text the theme's `Text` or `Background`, whichever reads better on it.
- A visible contextual tab has a key tip and an access key as any tab.

## Galleries

A gallery is a row of items in a group, Office's Styles, with arrows that
scroll it a row at a time and a button that opens all its items in a grid:

```go
styles := ribbon.AddGroup("Styles").AddGallery([]widget.GalleryItem{
	{Text: "Normal"}, {Text: "Heading 1", Image: heading1}, {Text: "Title"},
}, 4)
styles.SetAccessibleName("Styles") // its name, and its key tip's letter
styles.OnHover(func(i int) { preview(i) }) // -1: back to what was
styles.OnSelect(func(i int) { apply(i) })
```

- `AddGallery(items, columns)` adds a gallery to the last group (a new one
  with no title before any), `columns` items wide (1 for less) and as high
  as the group. It returns the `RibbonGalleryWidget`. An item shows its
  `Image`, 32 DIPs high times the text size the user chose, above its
  `Text`; an item with no image shows its text alone.
- At the gallery's end, a column of three buttons: Previous and Next scroll
  the row of items up and down a row, each disabled at its end, and More
  opens the drop-down under the gallery: all the items, `columns` to a row.
  A click on More while it is open closes it.
- `Selected()` returns the item selected, -1 for none (at first).
  `SetSelected(i)` selects item `i`, none out of range, and scrolls the row
  to it. `SetItems(items)` replaces the items: the selection stays where its
  item still is, else it is none, and the row shown is clamped.
- `OnSelect(fn)` runs `fn` with the item's index on the event goroutine
  after the user picks an item: a click in the row or the drop-down, or
  Enter in the drop-down, which then closes, the row scrolled to the pick.
  Picking the item selected runs it too. The setters never run it.
- `OnHover(fn)` is for a live preview: it runs `fn` on the event goroutine
  as the pointer moves onto an item, in the row or the drop-down, or the
  drop-down's arrow keys move to one, and with -1 when the pointer leaves
  the items, or the drop-down closes.
- Keyboard: the drop-down takes the focus as it opens. The arrows move in
  its grid, the item they are on outlined in the theme's `Accent`, stopping
  at its edges; with nothing selected the first arrow goes to the first
  item. Enter picks, and Escape, or a press outside it, closes it with no
  pick. More takes the focus from Tab and opens on Enter or Space.
- Key tips: the gallery takes its letter as any command, from its
  accessible name (`SetAccessibleName`); typing it opens the drop-down. A
  disabled gallery's letter ends the tips. Without a name, it takes the
  first free letter.
- In RTL the items run from the right, the buttons at the left end, and
  Right and Left mirror in the grid.
- The look: the gallery is lined with the theme's `Border`; the item
  selected is filled as a selected row (the theme's `RowSelected`, or its
  `Accent`), and an item under the pointer with `SurfaceHover`. The
  drop-down is a combo box's.
- The gallery is a control of the ribbon as any: `SetDisabled` disables it
  and closes its drop-down, which closes too when its tab is no longer
  shown. It is opt-in: an app that adds no gallery does not link it.

Not in the ribbon: the application menu (the File tab), a gallery's
groups of items, its filter and resizing, a group's dialog launcher, a
dialog to customize the toolbar, and groups that shrink to fewer or
smaller items in a narrow window.
