# TreeView

`TreeView(key, text)` shows nodes in a tree, as WinForms' `TreeView` does:
roots at the top, and below each node open its children, indented, with an
arrow that opens and closes it, and optionally a check box before each node.
It is [`ui.Outline`](../outline.md) underneath, so it builds only the rows in
view. A node is told apart by a key you choose, of any comparable type, so
its place, its choice, its checks and whether it is open follow it as nodes
open and close above it.

The children can load when the user opens a node. This tree of folders reads
a directory the first time one is opened:

```go
type entry struct {
	path, name string
	dir        bool
}

tree := widget.TreeView(
	func(e *entry) string { return e.path },
	func(e *entry) string { return e.name },
).CheckBoxes().
	SetRoots([]entry{{"/home", "Home", true}, {"/tmp", "Temp", true}}).
	SetHasChildren("/home", true).
	SetHasChildren("/tmp", true)

tree.OnExpand(func(path string) {
	go func() {
		kids := listDir(path) // the entries of a directory, read from the disk
		tree.SetChildren(path, kids)
		for _, k := range kids {
			if k.dir {
				tree.SetHasChildren(k.path, true)
			}
		}
	}()
}).OnSelect(func(path string, ok bool) {
	if ok {
		status.SetText(path)
	}
})
```

`key` and `text` run under the package's lock, as the callbacks of
[`DataGridView`](../widget.md) do, and so does the function of `UpdateNode`
and `SetImage`'s: they read the node and call no widget method.

## Nodes

- `SetRoots(roots)` shows roots instead of those shown. A root whose key is
  shown stays, with its children, and whether it is open, checked or chosen;
  its data is replaced. A root that goes takes its subtree with it.
- `SetChildren(parent, kids)` sets the children of a node, and marks them
  loaded. The same rules hold: a child whose key stays keeps what it holds,
  and one that goes takes its subtree, and its place among the open, checked
  and chosen nodes. A node left without children is a leaf.
- `SetHasChildren(k, true)` gives a node an arrow before its children are
  loaded (see below).
- `UpdateNode(k, fn)` changes a node's data in place; `RemoveNodes(ks...)`
  removes nodes with their subtrees.
- `Node(k)`, `Parent(k)`, `Children(k)` and `IsLoaded(k)` read the tree.
  `SetImage(fn)` shows a 16 DIPs image before each node's text.

Keys are unique in a tree. A key given twice, in `SetRoots` or `SetChildren`,
or one that is already the key of a node elsewhere, is a programmer's error:
the call panics with a message naming the key, and changes nothing.

A call for a key that is not a node (`SetChildren`, `SetHasChildren`,
`LoadFailed`, `UpdateNode`, `Expand`, `Collapse`, `SetChecked`, `SetSelected`,
`RemoveNodes`) does nothing.

`RemoveNodes` takes the nodes out of `Expanded`, `Checked` and the selection,
and fires no event, so a key added again starts clean: not open, not checked,
not chosen.

## Lazy children

A node with `SetHasChildren(k, true)` shows an arrow although its children
are not set. When the user opens it, `OnExpand` runs with its key, and the
node shows one row, "Loading…" (`SetLoadingText` changes the text, for another
language), until `SetChildren` sets them. That row can be neither chosen nor
checked, and no query returns it.

- `OnExpand` runs for every node the user opens, loaded or not. For a node
  whose children are not loaded it runs once, however often the user closes
  and opens it before `SetChildren`: the load is pending.
- `SetHasChildren(k, true)` on a node that is open, not loaded and not pending
  is a request to load, and the one setter that fires an event: `OnExpand` runs
  once with the key, and the load is pending. While it is pending nothing asks
  again, so an app that refreshes the parent with `SetChildren` and then says
  the node has children again cannot start a second load.
- If the load failed, `LoadFailed(k)` ends the pending load without an event and
  leaves the node open on its Loading row. Then `SetHasChildren(k, true)` asks
  again, or the user closes and opens the node.
- `SetHasChildren(k, false)` on a node whose children are not loaded makes it a
  leaf and closes it, firing nothing, so no Loading row stays.
- `OnCollapse` runs when the user closes a node.
- `Expand(ks...)`, `Collapse(ks...)`, `ExpandAll()` and `CollapseAll()` are
  the program's, and fire neither event. An `Expand` of a node whose children
  are not loaded shows its Loading row, but loading is up to you.
- `IsExpanded(k)` and `Expanded()`, the keys of the nodes open in the order of
  the tree, read the state.
- Option-click (Alt-click) on an arrow opens all inside the node. For nodes
  whose children are not loaded that fires `OnExpand` for each, so the tree
  loads one level at a time.

Setting children on an open node keeps the nodes that stay open, and the
selection if its node stays.

## Selection

One node is chosen, by its key: a click, the arrow keys, Home and End, or
typing the first letters of a node's text choose it. Modifier-clicks that
would choose several (Cmd or Ctrl, Shift, Cmd+A) leave the one clicked last.

- `Selected()` returns the key of the node chosen, and false when none is.
- `SetSelected(k)` chooses a node and opens its ancestors so that it shows;
  `ClearSelection()` chooses none. Neither fires `OnSelect`.
- `OnSelect(fn)` runs `fn(k, true)` when the user chooses a node, and
  `fn(zero, false)` when the choice goes. Closing a node whose child was chosen
  chooses the node, and that is a choice of the user's.
- `OnActivate(fn)` runs with the key of a node double-clicked, or chosen with
  Enter.

## Check boxes

`CheckBoxes()` shows a check box before each node. Checks are the node's
own: checking a node checks no other. An app that wants a parent to check its
children does it in `OnCheck`, with `Children` and `SetChecked`:

```go
tree.OnCheck(func(k string, on bool) {
	tree.SetChecked(tree.Children(k), on)
})
```

- `Checked()` returns the keys of the nodes checked in the order of the tree,
  `IsChecked(k)` asks about one.
- `SetChecked(ks, on)` checks or unchecks nodes, firing nothing.
- `OnCheck(fn)` runs `fn(k, on)` for each node the user checked or unchecked.

## Sorting, filtering and renaming

A tree sorts its nodes, shows only those a search finds, and lets the user
rename a node in place, as WinForms' `TreeView` does with `TreeViewNodeSorter`
and `LabelEdit`:

```go
tree.SetSort(func(a, b *entry) int {
	return strings.Compare(a.name, b.name)
}).SetLabelEdit(true).OnAfterLabelEdit(func(path, name string) bool {
	if err := rename(path, name); err != nil {
		return false // the old name shows again
	}
	tree.UpdateNode(path, func(e *entry) { e.name = name })
	return true
})

search := widget.SearchBox()
search.OnChange(func() {
	q := strings.ToLower(search.Text())
	if q == "" {
		tree.SetFilter(nil)
		return
	}
	tree.SetFilter(func(e *entry) bool { return strings.Contains(strings.ToLower(e.name), q) })
})
```

- `SetSort(cmp)` shows the children of every node, and the roots, in `cmp`'s
  order (negative when `a` comes first, as `cmp.Compare`). Nodes `cmp` finds
  equal keep the order given. The order holds through `SetRoots`,
  `SetChildren` and `UpdateNode`. It is the order shown only: `Children`,
  `Expanded` and `Checked` keep the order given. `SetSort(nil)` shows the
  order given again. One sort holds for all siblings: the tree does not sort
  by a column.
- `SetFilter(keep)` shows a loaded node when it, or a loaded descendant,
  passes `keep`. The ancestors of a node that passes open while the filter is
  set, so it shows; a node that passes itself shows without its children that
  do not. Nodes set while the filter is set are filtered too. `SetFilter(nil)`
  shows every node again, open and closed as the user had them before the
  first filter; a filter changed for another starts from that expansion too.
- The filter never loads children: a node whose children are not loaded
  (`SetHasChildren`) shows when it passes itself, and `OnExpand` does not
  fire. Neither setter fires an event.
- The choice and the checks belong to keys, so sorting and filtering keep
  them, a node hidden by the filter too: `Selected` and `Checked` still
  return it.
- `SetLabelEdit(true)` lets the user rename the node chosen: F2 (and Enter on
  macOS, as in Finder, where Enter then renames rather than activates), or a
  click on its text once a second click would no longer make a double click,
  shows an input of its text, all selected. Enter commits, as moving the
  focus away does, and Escape cancels. It is ui's
  [`EditableText`](../editable-text.md) in the row.
- `OnAfterLabelEdit(fn)` runs `fn(k, text)` on the event goroutine with the
  new text. Change the node with `UpdateNode` and return true to keep the
  edit; return false to refuse it, and the node's text shows again. Until
  `fn` returns, the node shows the new text, so the old one does not flash
  before `UpdateNode`. An edit that leaves the text as it was calls nothing.
  Without `OnAfterLabelEdit`, edits are refused: the input does not open.
- `cmp` and `keep` run under the package's lock, as `key` and `text` do.

A tree given none of the three builds as one that never was, and links none
of their code.

## Columns

`SetColumns(treeTitle, cols...)` shows the tree in columns under a header,
as DevExpress's TreeList does: a file tree with sizes, an account tree with
balances. It is [`ui.OutlineTable`](../outline.md) underneath, on the same
nodes, so all of the above holds: lazy children, the selection, check boxes,
images, the keys and the events.

```go
type file struct{ path, name, size, kind string }

tree := widget.TreeView(
	func(f *file) string { return f.path },
	func(f *file) string { return f.name },
).SetColumns("Name",
	widget.Column[file]{Title: "Size", Text: func(f *file) string { return f.size }, Width: 80, Align: widget.End},
	widget.Column[file]{Title: "Kind", Text: func(f *file) string { return f.kind }, Width: 100},
).SetRoots([]file{{"/src", "src", "", "Folder"}, {"/README", "README", "3 KB", "Text"}})
```

- The first column, headed `treeTitle`, holds what a row holds without
  columns: the arrow, indented as the node is deep, the check box, the image
  and the text. It shares the room the others leave, and is at least 120 DIPs
  wide.
- Then comes one column for each of `cols`, showing its `Text` (which runs
  under the package's lock). Of a `Column` the tree reads `Title`, `Text`,
  `Width`, `MinWidth`, `Align` and `Hidden`. The grid's other fields are not
  read: the tree does not sort by a column (`SetSort` sorts all siblings by
  one order).
- The Loading row shows its text in the first column, and nothing in the
  others.
- Titles tell the columns apart, `treeTitle` among them, so they must
  differ.
- The user resizes and moves the columns, as a table's. The header takes a
  skin's `GridHeader` look, as `DataGridView`'s does.
- `UpdateNode` shows the new cells on the next frame. `SetColumns` again
  replaces the columns. `SetColumns("")`, with no columns, shows the tree
  without columns again.

## Keyboard

While the tree has the focus: Up and Down move the choice, Home and End go to
the first and the last row, Right opens the node chosen or goes to its first
child, and Left closes it or goes to its parent. With Option on macOS and Shift
elsewhere, Right and Left open and close all inside. Enter activates the node.
Typing the first letters of a node's text chooses the node. With
`SetLabelEdit(true)`, F2 renames the node chosen, and so does Enter on macOS.

The Loading row is skipped by all of them: Down and Up move past it, End goes to
the last row that can be chosen, Right on an open node whose children are
loading stays on the node, and typing finds no text in it (it is
`ListState.Skip` of the list underneath).

## Size

Opening, closing and choosing keep the sets of open, chosen and checked nodes
by key, and change only the keys that changed, so a click costs what the rows
shown cost, not the size of the tree. The test of `TreeView` clicks in a tree
of a million nodes and checks that a click allocates what it does in one of
two thousand. Each frame walks the nodes that are open, as `ui.Outline` does.
It allocates for the rows in view, not for the tree: an idle frame of the
test's window (800 by 600) over 300 roots allocated 91 times, and 199 times
with check boxes, which each hold a closure and a value for their row.

Not yet there: a context menu for a node, and `CopyFunc`; the menu of
`ContextMenu` is the tree's.

## Accessibility

Assistive technology sees a tree whose rows say their level, whether they
open, and whether they are open, and it opens and closes them as the arrows
do. Check boxes are named by their node's text.
