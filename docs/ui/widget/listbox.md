# ListBox and CheckedListBox

`ListBox(key, text)` shows items in a scrolling list, as WinForms' `ListBox`
does; `CheckedListBox(key, text)` is the same list with a check box before each
item. An item, of any type, is told apart by a key you choose, of any comparable
type, so its choice and its check follow it as items are added, removed and
replaced above it. It is a [`ui.List`](../list.md) underneath, so it builds only
the rows in view.

```go
type fruit struct{ id, name string }

list := widget.CheckedListBox(
	func(f *fruit) string { return f.id },
	func(f *fruit) string { return f.name },
).SetItems([]fruit{{"a", "Apple"}, {"b", "Banana"}, {"c", "Cherry"}}).
	Multi()

list.OnSelect(func(selected []string) {
	status.SetText(fmt.Sprint(len(selected), " chosen"))
}).OnCheck(func(id string, on bool) {
	if on {
		basket.Add(id)
	} else {
		basket.Remove(id)
	}
}).OnActivate(func(id string) {
	open(id)
})

picked := list.Checked() // the keys of the items checked, in the order of the list
```

`key` and `text` run under the package's lock, as the callbacks of
[`DataGridView`](../widget.md) do, and so do the functions of `UpdateItem` and
`SetImage`: they read the item and call no widget method.

## Items

- `SetItems(items)` shows a copy of items instead of the items shown. Of items
  with the same key, the last stays, where the first was. An item whose key is
  among the new ones keeps its choice and its check; the others lose them, and a
  key added again starts clean.
- `AddItem(item)` and `AddItems(items)` show items after the others, each instead
  of the item of its key if one shows.
- `UpdateItem(k, fn)` changes the item of key `k` in place; `RemoveItems(ks...)`
  removes items; `Clear()` removes all of them.
- `Item(k)` and `Len()` read the list. `SetImage(fn)` shows a 16 DIPs image
  before each item's text.

A call for a key that is not an item (`UpdateItem`, `RemoveItems`, `SetSelected`,
`SetChecked`) does nothing.

## Choosing

One item is chosen, the one clicked last, unless the list is `Multi()`: then a
click chooses one, Cmd-click (Ctrl-click on Linux and Windows) adds an item or
takes it out, and Shift-click chooses the items from the one last chosen to the
one clicked. The arrow keys and Home and End move the choice (Page Up and Down
too, on Linux and Windows), and typing the first letters of an item's text
chooses it.

- `Selected()` returns the keys of the items chosen, in the order of the list,
  and `CountSelected()` how many.
- `SetSelected(ks...)` chooses those items instead of the ones chosen, the last
  alone unless the list is `Multi()`. `ClearSelection()` chooses none.
- `OnSelect(fn)` runs `fn` with the keys chosen, in the order of the list, on the
  event goroutine, after the user changes the choice, as WinForms'
  `SelectedIndexChanged`. Setters do not run it, nor do items added or removed.
- `OnActivate(fn)` runs with the key of an item double-clicked, or chosen with
  Enter.

## Check boxes

On a `CheckedListBox` (or after `CheckBoxes()` on a `ListBox`), the box checks
the item it is in front of, apart from the item chosen. Space checks the items
chosen, or unchecks them when all are checked.

- `Checked()` returns the keys of the items checked (WinForms' `CheckedItems`),
  in the order of the list, and `IsChecked(k)` asks about one.
- `SetChecked(ks, on)` checks or unchecks items.
- `OnCheck(fn)` runs `fn(k, on)` on the event goroutine for each item the user
  checked or unchecked, as WinForms' `ItemCheck`. Setters do not run it.

Removing an item, or replacing the items without it, takes it out of the choice
and the checks without an event. A list is not [`Bindable`](../widget.md): it
holds several values, and `Selected` and `Checked` are its values to read.

## Size

The choice and the checks are kept by key, as a table's are, and a click changes
only the keys it changed, so it costs what the rows shown cost, not the number of
items. The test of `ListBox` clicks in a list of a million items and checks that
a click allocates what it does in one of a thousand.

## Accessibility

Assistive technology sees a list of items, each named by its text and, with check
boxes, checked or not.
