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

Not in the ribbon: key tips, the quick access toolbar, the application menu
(the File tab), contextual tabs, galleries, a group's dialog launcher, and
groups that shrink to fewer or smaller items in a narrow window.
