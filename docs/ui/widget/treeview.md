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
`UpdateNode`, `Expand`, `Collapse`, `SetChecked`, `SetSelected`,
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
  and opens it before `SetChildren`: the load is pending. If the load failed,
  `SetHasChildren(k, true)` again lets the next opening ask again.
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

## Keyboard

While the tree has the focus: Up and Down move the choice, Home and End go to
the first and the last row, Right opens the node chosen or goes to its first
child, and Left closes it or goes to its parent. With Option on macOS and Shift
elsewhere, Right and Left open and close all inside. Enter activates the node.
Typing the first letters of a node's text chooses the node.

## Size

Opening, closing and choosing keep the sets of open, chosen and checked nodes
by key, and change only the keys that changed, so a click costs what the rows
shown cost, not the size of the tree. The test of `TreeView` clicks in a tree
of a million nodes and checks that a click allocates what it does in one of
two thousand. Each frame walks the nodes that are open, as `ui.Outline` does,
without allocating.

Not yet there: a context menu for a node, and `CopyFunc`; the menu of
`ContextMenu` is the tree's.

## Accessibility

Assistive technology sees a tree whose rows say their level, whether they
open, and whether they are open, and it opens and closes them as the arrows
do. Check boxes are named by their node's text.
