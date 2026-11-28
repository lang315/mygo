package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// treeOps is the state of the sort, the filter and the label edit of a
// TreeView, kept in the tree as an any so that the tree's own type does not
// name it: a program that calls none of their setters links none of it.
// Under mu.
type treeOps[T any, K comparable] struct {
	cmp  func(a, b *T) int
	keep func(*T) bool
	// views are the children shown by parent, and roots the roots shown
	// (rootsOK once made): made when ui first asks after a change, and kept
	// until the next, so that a frame at rest makes none.
	views   map[K][]treeItem[K]
	roots   []treeItem[K]
	rootsOK bool
	// pass holds, under a filter, the nodes that show: those that pass and
	// their ancestors. stale says the nodes changed since it was made; expand
	// asks the next build to open the ancestors of the nodes that pass.
	pass          map[K]bool
	stale, expand bool
	// saved is the expansion the user had before the filter.
	saved []K

	label bool
	after func(K, string) bool
	// val is the text of the node of valKey that the EditableText of the
	// chosen row holds, given the text it was given: they differ once the
	// user committed an edit, which takeLabel then hands to after.
	valKey     K
	valOK      bool
	val, given string
	// pend is a text committed, shown in place of the node's until after
	// returns, so that the old text does not flash before the app's
	// UpdateNode.
	pend struct {
		k    K
		text string
		on   bool
		gen  uint64
	}
}

// ops returns the tree's sort, filter and label edit, made at the first call
// with the hooks the tree's builds call. Under mu.
func (t *TreeViewWidget[T, K]) ops() *treeOps[T, K] {
	if o, ok := t.opsAny.(*treeOps[T, K]); ok {
		return o
	}
	o := &treeOps[T, K]{views: map[K][]treeItem[K]{}, pass: map[K]bool{}}
	t.opsAny = o
	t.shown, t.reshape, t.prepare = t.opsShown, t.opsReshape, t.opsPrepare
	return o
}

// SetSort shows the children of each node, and the roots, in cmp's order,
// which compares two nodes as cmp.Compare does; nil shows them in the order
// given. The order stays through SetRoots, SetChildren and UpdateNode, and
// it is the order shown only: Children keeps the order given. Nodes cmp
// finds equal keep the order given. cmp runs under the package's lock.
func (t *TreeViewWidget[T, K]) SetSort(cmp func(a, b *T) int) *TreeViewWidget[T, K] {
	t.change(func() {
		o := t.ops()
		o.cmp = cmp
		o.dropViews()
	})
	return t
}

// SetFilter shows only the loaded nodes that keep reports true for, and
// their ancestors, which open while the filter is set; nil shows every node
// again, opened and closed as the user had them before the filter. It never
// loads children (SetHasChildren): a node whose children are not loaded
// shows when it passes itself. The choice and the checks stay with their
// nodes, shown or not. keep runs under the package's lock.
func (t *TreeViewWidget[T, K]) SetFilter(keep func(*T) bool) *TreeViewWidget[T, K] {
	t.change(func() {
		o := t.ops()
		switch {
		case keep != nil && o.keep == nil:
			o.saved = slices.Collect(func(yield func(K) bool) {
				for it := range t.open.value.All() {
					if !yield(it.k) {
						return
					}
				}
			})
		case keep != nil: // another filter: from the user's expansion again
			t.restoreOpen(o.saved)
		case o.keep != nil:
			t.restoreOpen(o.saved)
			o.saved = nil
		}
		o.keep = keep
		o.dropViews()
		clear(o.pass)
		o.stale, o.expand = false, false
		if keep != nil {
			t.filterPass(o)
			t.openPassing(o)
		}
	})
	return t
}

// SetLabelEdit lets the user edit the text of the node chosen in place: F2,
// Enter on macOS, or a click on its text once a second click would no longer
// make a double click, show an input of it; Enter commits, as moving the
// focus away does, and Escape cancels. OnAfterLabelEdit decides each edit:
// without it, edits are refused.
func (t *TreeViewWidget[T, K]) SetLabelEdit(on bool) *TreeViewWidget[T, K] {
	t.change(func() { t.ops().label = on; t.wireLabel() })
	return t
}

// OnAfterLabelEdit runs fn on the event goroutine with the key of a node the
// user renamed and its new text: fn changes the node (UpdateNode) and returns
// true to keep the edit. Until fn returns the node shows the new text; then
// it shows its own, which is the old one unless fn changed it. An edit that
// leaves the text as it was calls nothing.
func (t *TreeViewWidget[T, K]) OnAfterLabelEdit(fn func(k K, text string) bool) *TreeViewWidget[T, K] {
	t.change(func() { t.ops().after = fn; t.wireLabel() })
	return t
}

// wireLabel builds the rows with an editable text, or without. Under mu.
func (t *TreeViewWidget[T, K]) wireLabel() {
	o := t.ops()
	if o.label && o.after != nil {
		if t.labelRow == nil {
			t.labelRow = t.opsLabelRow
		}
	} else {
		t.labelRow, o.valOK = nil, false
	}
}

// dropViews forgets the children shown, for ui's next ask. Under mu.
func (o *treeOps[T, K]) dropViews() {
	clear(o.views)
	o.roots, o.rootsOK = nil, false
}

// opsShown is the tree's shown: kids as they show, sorted and filtered.
// Under mu.
func (t *TreeViewWidget[T, K]) opsShown(parent K, root bool, kids []treeItem[K]) []treeItem[K] {
	o := t.opsAny.(*treeOps[T, K])
	if o.cmp == nil && o.keep == nil {
		return kids
	}
	if root && o.rootsOK {
		return o.roots
	}
	if v, ok := o.views[parent]; ok && !root {
		return v
	}
	if o.keep != nil && o.stale {
		t.filterPass(o)
	}
	var out []treeItem[K]
	if o.keep != nil {
		out = make([]treeItem[K], 0, len(kids))
		for _, it := range kids {
			if o.pass[it.k] {
				out = append(out, it)
			}
		}
	} else {
		out = slices.Clone(kids)
	}
	if cmp := o.cmp; cmp != nil {
		slices.SortStableFunc(out, func(a, b treeItem[K]) int { return cmp(&t.nodes[a.k].data, &t.nodes[b.k].data) })
	}
	if root {
		o.roots, o.rootsOK = out, true
	} else {
		o.views[parent] = out
	}
	return out
}

// opsReshape is the tree's reshape: the nodes changed. Under mu.
func (t *TreeViewWidget[T, K]) opsReshape() {
	o := t.opsAny.(*treeOps[T, K])
	o.dropViews()
	if o.keep != nil {
		o.stale, o.expand = true, true
	}
}

// opsPrepare is the tree's prepare: under a filter, it opens the ancestors
// of nodes that came to pass; and it hands an edit the user committed to
// OnAfterLabelEdit. Under mu, at the start of a build.
func (t *TreeViewWidget[T, K]) opsPrepare() {
	o := t.opsAny.(*treeOps[T, K])
	if o.keep != nil && o.expand {
		if o.stale {
			t.filterPass(o)
		}
		t.openPassing(o)
		o.expand = false
	}
	t.takeLabel(o)
}

// filterPass finds the nodes that show under the filter: those that pass,
// and those with a loaded descendant that does. Under mu.
func (t *TreeViewWidget[T, K]) filterPass(o *treeOps[T, K]) {
	clear(o.pass)
	var walk func(items []treeItem[K]) bool
	walk = func(items []treeItem[K]) bool {
		found := false
		for _, it := range items {
			n := t.nodes[it.k]
			if n == nil {
				continue
			}
			below := walk(n.kids) // loaded children only: the filter loads none
			if below || o.keep(&n.data) {
				o.pass[it.k] = true
				found = true
			}
		}
		return found
	}
	walk(t.roots)
	o.stale = false
}

// openPassing opens the nodes that show under the filter with a child that
// does: the ancestors of the nodes that pass. Under mu.
func (t *TreeViewWidget[T, K]) openPassing(o *treeOps[T, K]) {
	for k := range o.pass {
		n := t.nodes[k]
		if n == nil || !slices.ContainsFunc(n.kids, func(kid treeItem[K]) bool { return o.pass[kid.k] }) {
			continue
		}
		if it := (treeItem[K]{k: k}); !t.open.value.Has(it) {
			t.open.set(it, true)
		}
	}
}

// restoreOpen opens the nodes of keys, those that are still nodes with
// children, and closes the others, firing nothing. Under mu.
func (t *TreeViewWidget[T, K]) restoreOpen(keys []K) {
	want := make(map[K]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	for _, it := range slices.Collect(t.open.value.All()) {
		if !want[it.k] {
			t.open.set(it, false)
		}
	}
	for _, k := range keys {
		if n := t.nodes[k]; n != nil && n.branch() && !t.open.value.Has(treeItem[K]{k: k}) {
			t.open.set(treeItem[K]{k: k}, true)
		}
	}
}

// opsLabelRow is the tree's labelRow: the text of the node chosen, which the
// user edits in place (ui's EditableText), and of a node whose edit
// OnAfterLabelEdit is deciding. Under mu.
func (t *TreeViewWidget[T, K]) opsLabelRow(c *ui.Context, it treeItem[K], text string) bool {
	o := t.opsAny.(*treeOps[T, K])
	t.takeLabel(o) // a commit ui made since the last build, before val is given again
	pending := o.pend.on && o.pend.k == it.k
	if pending {
		text = o.pend.text
	}
	if !t.chosen.shown.Has(it) {
		if pending {
			ui.Text(c, text).SingleLine()
		}
		return pending
	}
	o.valKey, o.valOK, o.val, o.given = it.k, true, text, text
	ui.EditableText(c, &o.val)
	return true
}

// takeLabel hands an edit the user committed to OnAfterLabelEdit, posted,
// and shows its text until that returns. Under mu.
func (t *TreeViewWidget[T, K]) takeLabel(o *treeOps[T, K]) {
	if !o.valOK || o.val == o.given {
		return
	}
	k, text, fn := o.valKey, o.val, o.after
	o.given = o.val
	if fn == nil {
		return
	}
	o.pend.gen++
	o.pend.k, o.pend.text, o.pend.on = k, text, true
	gen := o.pend.gen
	t.gens.bump(k) // the row shows the text committed
	Post(func() {
		fn(k, text)
		t.change(func() {
			if o.pend.on && o.pend.gen == gen {
				o.pend.on = false
				t.gens.bump(k) // the row shows the node's text
			}
		})
	})
}
