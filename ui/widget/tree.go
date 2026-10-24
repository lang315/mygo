package widget

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// treeItem is what ui's outline holds for a node: its key, or the Loading
// row below the node of that key (wait).
type treeItem[K comparable] struct {
	k    K
	wait bool
}

// treeNode is a node of a TreeView: its data, where it hangs, and what is
// known of its children.
type treeNode[T any, K comparable] struct {
	data   T
	parent K
	child  bool // not a root: parent is its parent's key
	kids   []treeItem[K]
	// loaded is set by SetChildren; has by SetHasChildren, for a node whose
	// children are not loaded yet; pending once OnExpand asked for them.
	loaded, has, pending bool
	// wait is the one row below the node while its children load.
	wait []treeItem[K]
}

// TreeViewWidget shows nodes in a tree, as WinForms' TreeView: roots, and
// below each node open its children, indented, with an arrow that opens
// and closes it; optionally a check box before each node. A node is told
// apart by a key of type K, so that its place, its choice, its checks and
// whether it is open follow it as nodes open and close above it. The
// children of a node can be set when the user opens it (OnExpand,
// SetChildren), and a tree of a million nodes costs no more to click than
// the rows it shows.
//
// The functions the tree calls back (key, text, SetImage's, and UpdateNode's)
// run under the package's lock: they call no widget method.
type TreeViewWidget[T any, K comparable] struct {
	common[TreeViewWidget[T, K]]
	key   func(*T) K
	text  func(*T) string
	image func(*T) *ui.Bitmap

	nodes   map[K]*treeNode[T, K]
	roots   []treeItem[K]
	loading string
	checks  bool

	// open, chosen and checked hold the items open, chosen (at most one)
	// and checked: value is the tree's, and ui's copy of each is committed
	// from at the start of the next build, which reports what the user
	// changed.
	open, chosen, checked keys[treeItem[K]]

	onExpand, onCollapse func(K)
	onSelect             func(K, bool)
	onCheck              func(K, bool)
	onActivate           func(K)

	// ui's, on the main thread:
	state  ui.OutlineState[treeItem[K]]
	cursor int
}

// TreeView makes a tree whose nodes, of type T, key tells apart by a key of
// type K, and text shows. key and text run under the package's lock.
func TreeView[T any, K comparable](key func(*T) K, text func(*T) string) *TreeViewWidget[T, K] {
	t := &TreeViewWidget[T, K]{key: key, text: text, nodes: map[K]*treeNode[T, K]{}, loading: "Loading…", cursor: -1}
	t.init(t)
	return t
}

// SetImage shows an image before the text of each node, 16 DIPs square;
// nil, or a nil image, for none. fn runs under the package's lock.
func (t *TreeViewWidget[T, K]) SetImage(fn func(*T) *ui.Bitmap) *TreeViewWidget[T, K] {
	t.change(func() { t.image = fn })
	return t
}

// SetLoadingText sets the text of the row that shows below a node the user
// opened while its children load, "Loading…" by default. That row can be
// neither chosen nor checked.
func (t *TreeViewWidget[T, K]) SetLoadingText(s string) *TreeViewWidget[T, K] {
	t.change(func() { t.loading = s })
	return t
}

// CheckBoxes shows a check box before each node. Checks are the node's own:
// checking a node checks no other (an app that wants that does it in OnCheck
// with SetChecked).
func (t *TreeViewWidget[T, K]) CheckBoxes() *TreeViewWidget[T, K] {
	t.change(func() { t.checks = true })
	return t
}

// panics names a key a tree was given twice.
func dupKey[K comparable](k K) string {
	return fmt.Sprintf("widget: TreeView: duplicate key %v: a key stands for one node, in one place", k)
}

// claim panics unless keys can be the children of parent, or the roots:
// none repeats, and none is the key of a node elsewhere. Under mu, before
// anything changes.
func (t *TreeViewWidget[T, K]) claim(keys []K, parent K, root bool) {
	seen := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		if _, ok := seen[k]; ok {
			panic(dupKey(k))
		}
		seen[k] = struct{}{}
		if n := t.nodes[k]; n != nil && (n.child == root || !root && n.parent != parent) {
			panic(dupKey(k))
		}
	}
}

// put makes the node of key k hold data, keeping what it has if it is
// there. Under mu.
func (t *TreeViewWidget[T, K]) put(k K, data T, parent K, child bool) {
	if n := t.nodes[k]; n != nil {
		n.data = data
		return
	}
	t.nodes[k] = &treeNode[T, K]{data: data, parent: parent, child: child}
}

// items returns the items of keys, and drops those of old that are not
// among them. Under mu.
func (t *TreeViewWidget[T, K]) replace(old []treeItem[K], keys []K) []treeItem[K] {
	keep := make(map[K]struct{}, len(keys))
	items := make([]treeItem[K], len(keys))
	for i, k := range keys {
		keep[k] = struct{}{}
		items[i] = treeItem[K]{k: k}
	}
	for _, it := range old {
		if _, ok := keep[it.k]; !ok {
			t.drop(it)
		}
	}
	return items
}

// SetRoots shows roots at the top of the tree, instead of those shown: a
// root of a key shown stays, with what it holds (its children, whether it is
// open, checked or chosen), and its data replaced; one that goes takes its
// subtree with it. It panics, naming the key, if a key repeats or is that of
// a node below another, and changes nothing then.
func (t *TreeViewWidget[T, K]) SetRoots(roots []T) *TreeViewWidget[T, K] {
	t.change(func() {
		keys := make([]K, len(roots))
		for i := range roots {
			keys[i] = t.key(&roots[i])
		}
		var zero K
		t.claim(keys, zero, true)
		for i := range roots {
			t.put(keys[i], roots[i], zero, false)
		}
		t.roots = t.replace(t.roots, keys)
	})
	return t
}

// SetChildren sets the children of the node of key parent, and marks them
// loaded: the Loading row goes, and a node left without children is a
// leaf. A child of a key that is among the children stays, with what it
// holds, and its data replaced; one that goes takes its subtree with it, and
// its place in the open, checked and chosen nodes. It does nothing for a key
// that is not a node. It panics, naming the key, if a key repeats or is
// that of a node elsewhere, and changes nothing then.
func (t *TreeViewWidget[T, K]) SetChildren(parent K, kids []T) *TreeViewWidget[T, K] {
	t.change(func() {
		p := t.nodes[parent]
		if p == nil {
			return
		}
		keys := make([]K, len(kids))
		for i := range kids {
			keys[i] = t.key(&kids[i])
		}
		t.claim(keys, parent, false)
		for i := range kids {
			t.put(keys[i], kids[i], parent, true)
		}
		p.kids = t.replace(p.kids, keys)
		p.loaded = true
	})
	return t
}

// SetHasChildren says whether the node of key k has children that are not
// loaded, so that its arrow shows: opening it by the user calls OnExpand, and
// the Loading row shows until SetChildren. With true it also lets OnExpand
// fire again for a load that failed: it fires once while one is pending. It
// does nothing for a key that is not a node.
func (t *TreeViewWidget[T, K]) SetHasChildren(k K, has bool) *TreeViewWidget[T, K] {
	t.change(func() {
		if n := t.nodes[k]; n != nil {
			n.has = has
			if has {
				n.pending = false
			}
		}
	})
	return t
}

// UpdateNode runs fn on the node of key k, if there is one, to change its
// data in place. fn runs under the package's lock: it changes the data but
// not the key, and calls no widget method.
func (t *TreeViewWidget[T, K]) UpdateNode(k K, fn func(*T)) *TreeViewWidget[T, K] {
	t.change(func() {
		if n := t.nodes[k]; n != nil {
			fn(&n.data)
		}
	})
	return t
}

// RemoveNodes removes the nodes of keys ks, each with its subtree. The
// subtree leaves the open, checked and chosen nodes too, and no event fires,
// so that a key added again starts clean. A key that is not a node is
// ignored.
func (t *TreeViewWidget[T, K]) RemoveNodes(ks ...K) *TreeViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			n := t.nodes[k]
			if n == nil {
				continue
			}
			gone := func(it treeItem[K]) bool { return it.k == k }
			if n.child {
				p := t.nodes[n.parent]
				p.kids = slices.DeleteFunc(p.kids, gone)
			} else {
				t.roots = slices.DeleteFunc(t.roots, gone)
			}
			t.drop(treeItem[K]{k: k})
		}
	})
	return t
}

// drop removes the node of it and its subtree, and forgets them in the
// sets. Under mu.
func (t *TreeViewWidget[T, K]) drop(it treeItem[K]) {
	n := t.nodes[it.k]
	if n == nil {
		return
	}
	for _, kid := range n.kids {
		t.drop(kid)
	}
	delete(t.nodes, it.k)
	for _, s := range []*keys[treeItem[K]]{&t.open, &t.chosen, &t.checked} {
		if s.value.Has(it) {
			s.set(it, false)
		}
	}
}

// Node returns the data of the node of key k.
func (t *TreeViewWidget[T, K]) Node(k K) (T, bool) {
	mu.Lock()
	defer mu.Unlock()
	if n := t.nodes[k]; n != nil {
		return n.data, true
	}
	var zero T
	return zero, false
}

// Parent returns the key of the parent of the node of key k; false for a
// root, and for a key that is not a node.
func (t *TreeViewWidget[T, K]) Parent(k K) (K, bool) {
	mu.Lock()
	defer mu.Unlock()
	if n := t.nodes[k]; n != nil && n.child {
		return n.parent, true
	}
	var zero K
	return zero, false
}

// Children returns the keys of the children of the node of key k, in order:
// none for a leaf, for a node whose children are not loaded, and for a key
// that is not a node.
func (t *TreeViewWidget[T, K]) Children(k K) []K {
	mu.Lock()
	defer mu.Unlock()
	n := t.nodes[k]
	if n == nil {
		return nil
	}
	ks := make([]K, len(n.kids))
	for i, kid := range n.kids {
		ks[i] = kid.k
	}
	return ks
}

// IsLoaded reports whether the children of the node of key k were set with
// SetChildren.
func (t *TreeViewWidget[T, K]) IsLoaded(k K) bool {
	mu.Lock()
	defer mu.Unlock()
	n := t.nodes[k]
	return n != nil && n.loaded
}

// branch reports whether a node has an arrow. Under mu.
func (n *treeNode[T, K]) branch() bool { return len(n.kids) > 0 || !n.loaded && n.has }

// Expand opens the nodes of keys ks that have children. It fires neither
// OnExpand nor OnCollapse: an app opening a node whose children are not
// loaded loads them itself. A key that is not a node is ignored.
func (t *TreeViewWidget[T, K]) Expand(ks ...K) *TreeViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			if n := t.nodes[k]; n != nil && n.branch() {
				t.open.set(treeItem[K]{k: k}, true)
			}
		}
	})
	return t
}

// Collapse closes the nodes of keys ks, firing no event. A key that is not
// a node is ignored.
func (t *TreeViewWidget[T, K]) Collapse(ks ...K) *TreeViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			if it := (treeItem[K]{k: k}); t.nodes[k] != nil && t.open.value.Has(it) {
				t.open.set(it, false)
			}
		}
	})
	return t
}

// ExpandAll opens every node that has children, firing no event.
func (t *TreeViewWidget[T, K]) ExpandAll() *TreeViewWidget[T, K] {
	t.change(func() {
		for k, n := range t.nodes {
			if n.branch() {
				t.open.set(treeItem[K]{k: k}, true)
			}
		}
	})
	return t
}

// CollapseAll closes every node, firing no event.
func (t *TreeViewWidget[T, K]) CollapseAll() *TreeViewWidget[T, K] {
	t.change(func() {
		for _, it := range slices.Collect(t.open.value.All()) {
			t.open.set(it, false)
		}
	})
	return t
}

// IsExpanded reports whether the node of key k is open.
func (t *TreeViewWidget[T, K]) IsExpanded(k K) bool {
	mu.Lock()
	defer mu.Unlock()
	return t.open.value.Has(treeItem[K]{k: k})
}

// Expanded returns the keys of the nodes open, in the order of the tree.
func (t *TreeViewWidget[T, K]) Expanded() []K {
	mu.Lock()
	defer mu.Unlock()
	return t.inOrder(&t.open.value)
}

// inOrder returns the keys of the nodes of in, depth first. Under mu.
func (t *TreeViewWidget[T, K]) inOrder(in *ui.Selection[treeItem[K]]) []K {
	if in.Len() == 0 {
		return nil
	}
	var ks []K
	var walk func(items []treeItem[K])
	walk = func(items []treeItem[K]) {
		for _, it := range items {
			if in.Has(it) {
				ks = append(ks, it.k)
			}
			if n := t.nodes[it.k]; n != nil {
				walk(n.kids)
			}
		}
	}
	walk(t.roots)
	return ks
}

// Selected returns the key of the node chosen.
func (t *TreeViewWidget[T, K]) Selected() (K, bool) {
	mu.Lock()
	defer mu.Unlock()
	if it, ok := t.selection(); ok {
		return it.k, true
	}
	var zero K
	return zero, false
}

// selection returns the item chosen. Under mu.
func (t *TreeViewWidget[T, K]) selection() (treeItem[K], bool) {
	for it := range t.chosen.value.All() {
		return it, true
	}
	return treeItem[K]{}, false
}

// SetSelected chooses the node of key k, and opens its ancestors so that it
// shows, firing no event. A key that is not a node is ignored.
func (t *TreeViewWidget[T, K]) SetSelected(k K) *TreeViewWidget[T, K] {
	t.change(func() {
		n := t.nodes[k]
		if n == nil {
			return
		}
		for p := n; p.child; p = t.nodes[p.parent] {
			t.open.set(treeItem[K]{k: p.parent}, true)
		}
		t.clearChosen()
		t.chosen.set(treeItem[K]{k: k}, true)
	})
	return t
}

// ClearSelection chooses no node, firing no event.
func (t *TreeViewWidget[T, K]) ClearSelection() *TreeViewWidget[T, K] {
	t.change(t.clearChosen)
	return t
}

// clearChosen takes every node out of the choice. Under mu.
func (t *TreeViewWidget[T, K]) clearChosen() {
	for _, it := range slices.Collect(t.chosen.value.All()) {
		t.chosen.set(it, false)
	}
}

// Checked returns the keys of the nodes checked, in the order of the tree.
func (t *TreeViewWidget[T, K]) Checked() []K {
	mu.Lock()
	defer mu.Unlock()
	return t.inOrder(&t.checked.value)
}

// IsChecked reports whether the node of key k is checked.
func (t *TreeViewWidget[T, K]) IsChecked(k K) bool {
	mu.Lock()
	defer mu.Unlock()
	return t.checked.value.Has(treeItem[K]{k: k})
}

// SetChecked checks, or unchecks, the nodes of keys ks, firing no event. A
// key that is not a node is ignored.
func (t *TreeViewWidget[T, K]) SetChecked(ks []K, on bool) *TreeViewWidget[T, K] {
	t.change(func() {
		for _, k := range ks {
			if t.nodes[k] != nil {
				t.checked.set(treeItem[K]{k: k}, on)
			}
		}
	})
	return t
}

// OnExpand runs fn on the event goroutine with the key of a node the user
// opened (with the arrow, the keys or assistive technology). For a node whose
// children are not loaded (SetHasChildren), it is the call to load them,
// off the event goroutine if that takes time, then SetChildren: it fires
// once until then, however often the user closes and opens the node. A
// program's Expand fires nothing.
func (t *TreeViewWidget[T, K]) OnExpand(fn func(k K)) *TreeViewWidget[T, K] {
	t.change(func() { t.onExpand = fn })
	return t
}

// OnCollapse runs fn on the event goroutine with the key of a node the user
// closed.
func (t *TreeViewWidget[T, K]) OnCollapse(fn func(k K)) *TreeViewWidget[T, K] {
	t.change(func() { t.onCollapse = fn })
	return t
}

// OnSelect runs fn on the event goroutine when the user changes the node
// chosen: its key and true, or false when no node is chosen.
func (t *TreeViewWidget[T, K]) OnSelect(fn func(k K, ok bool)) *TreeViewWidget[T, K] {
	t.change(func() { t.onSelect = fn })
	return t
}

// OnCheck runs fn on the event goroutine with the key of a node the user
// checked or unchecked, and whether it is checked.
func (t *TreeViewWidget[T, K]) OnCheck(fn func(k K, on bool)) *TreeViewWidget[T, K] {
	t.change(func() { t.onCheck = fn })
	return t
}

// OnActivate runs fn on the event goroutine with the key of a node the user
// double-clicked, or chose with Enter.
func (t *TreeViewWidget[T, K]) OnActivate(fn func(k K)) *TreeViewWidget[T, K] {
	t.change(func() { t.onActivate = fn })
	return t
}

// foldKeys brings s up to date with the changes the user made to shown, the
// set ui held, since the last build, and returns the keys added and removed.
// A key that valid rejects is taken back out of shown. Under mu.
func foldKeys[I comparable](s *keys[I], shown *ui.Selection[I], valid func(I) bool) (on, off []I) {
	ks, all := shown.TakeChanges()
	if all {
		// The changes of setters not yet handed to ui win over it.
		var pending []bool
		if !s.dirty {
			for _, k := range s.changed {
				pending = append(pending, s.value.Has(k))
			}
		}
		for k := range shown.All() {
			if !s.value.Has(k) {
				on = append(on, k)
			}
		}
		for k := range s.value.All() {
			if !shown.Has(k) {
				off = append(off, k)
			}
		}
		s.value.CopyFrom(shown)
		for i, k := range s.changed[:len(pending)] {
			if pending[i] {
				s.value.Add(k)
			} else {
				s.value.Remove(k)
			}
		}
		if s.dirty {
			s.changed, s.dirty = s.changed[:0], false
		}
		on = slices.DeleteFunc(on, func(k I) bool { return !s.value.Has(k) })
		off = slices.DeleteFunc(off, func(k I) bool { return s.value.Has(k) })
	} else {
		for _, k := range ks {
			switch {
			case shown.Has(k) && !s.value.Has(k):
				s.value.Add(k)
				on = append(on, k)
			case !shown.Has(k) && s.value.Has(k):
				s.value.Remove(k)
				off = append(off, k)
			}
		}
	}
	reject := func(k I) bool {
		if valid(k) {
			return false
		}
		shown.Remove(k)
		s.value.Remove(k)
		return true
	}
	return slices.DeleteFunc(on, reject), slices.DeleteFunc(off, reject)
}

// takeUser folds what the user did since the last build into the tree's
// sets and posts the events. Under mu, at the start of a build.
func (t *TreeViewWidget[T, K]) takeUser() {
	valid := func(it treeItem[K]) bool { return !it.wait && t.nodes[it.k] != nil }
	on, off := foldKeys(&t.open, &t.state.Open, valid)
	for _, it := range on {
		n := t.nodes[it.k]
		fire := n.loaded || !n.pending
		if !n.loaded && n.has {
			n.pending = true
		}
		if h := t.onExpand; h != nil && fire {
			k := it.k
			Post(func() { h(k) })
		}
	}
	if h := t.onCollapse; h != nil {
		for _, it := range off {
			k := it.k
			Post(func() { h(k) })
		}
	}

	before, had := t.selection()
	on, off = foldKeys(&t.chosen, &t.chosen.shown, valid)
	if len(on)+len(off) > 0 {
		// One node is chosen: the newest of those the user added.
		if t.chosen.value.Len() > 1 && len(on) > 0 {
			keep := on[len(on)-1]
			for _, it := range slices.Collect(t.chosen.value.All()) {
				if it != keep {
					t.chosen.value.Remove(it)
					t.chosen.shown.Remove(it)
				}
			}
		}
		now, has := t.selection()
		if h := t.onSelect; h != nil && (has != had || now != before) {
			k := now.k
			Post(func() { h(k, has) })
		}
	}

	on, off = foldKeys(&t.checked, &t.checked.shown, valid)
	if h := t.onCheck; h != nil {
		for _, it := range on {
			k := it.k
			Post(func() { h(k, true) })
		}
		for _, it := range off {
			k := it.k
			Post(func() { h(k, false) })
		}
	}
}

// kidsOf is ui's children: nil for a leaf, the children of a node that has
// them, and for a node whose children are not loaded the one Loading row.
func (t *TreeViewWidget[T, K]) kidsOf(it treeItem[K]) []treeItem[K] {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	n := t.nodes[it.k]
	switch {
	case it.wait || n == nil:
		return nil
	case n.loaded:
		if len(n.kids) == 0 {
			return nil
		}
		return n.kids
	case n.has:
		if n.wait == nil {
			n.wait = []treeItem[K]{{k: it.k, wait: true}}
		}
		return n.wait
	}
	return nil
}

// labelAt is ui's label of the row shown at row: typing finds it.
func (t *TreeViewWidget[T, K]) labelAt(row int) string {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	it := t.state.Item(row)
	if it.wait {
		return t.loading
	}
	if n := t.nodes[it.k]; n != nil {
		return t.text(&n.data)
	}
	return ""
}

// cursorRow returns the row of the last frame that shows the node chosen,
// -1 for none. Under mu.
func (t *TreeViewWidget[T, K]) cursorRow() int {
	it, ok := t.selection()
	if !ok {
		return -1
	}
	s := &t.state
	if c := t.cursor; c >= 0 && c < s.Rows() && s.Item(c) == it {
		return c
	}
	for i := range s.Rows() {
		if s.Item(i) == it {
			return i
		}
	}
	return -1
}

func (t *TreeViewWidget[T, K]) build(c *ui.Context) ui.Element {
	t.takeUser()
	s := &t.state
	s.List.Selection = t.chosen.bind()
	t.open.bindInto(&s.Open)
	t.cursor = t.cursorRow()
	s.List.Selected = &t.cursor
	s.List.Label = t.labelAt
	e := ui.Outline(c, s, t.roots, t.kidsOf, func(it treeItem[K]) { t.row(c, it) }).Grow(1)
	if fn := t.onActivate; fn != nil {
		e.OnActivate(func(row int) {
			if it := s.Item(row); !it.wait {
				k := it.k
				Post(func() { fn(k) })
			}
		})
	}
	return e
}

// row builds the content of the row of it, after its arrow: while the tree
// builds, or when ui lays out rows it did not build then.
func (t *TreeViewWidget[T, K]) row(c *ui.Context, it treeItem[K]) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	if it.wait {
		ui.Text(c, t.loading).SingleLine().TextColor(c.Theme().TextMuted)
		return
	}
	n := t.nodes[it.k]
	if n == nil {
		return
	}
	text := t.text(&n.data)
	if t.checks {
		on := t.checked.shown.Has(it)
		ui.Checkbox(c, &on, "").Label(text).OnChange(func() {
			if on {
				t.checked.shown.Add(it)
			} else {
				t.checked.shown.Remove(it)
			}
		})
	}
	if t.image != nil {
		if bmp := t.image(&n.data); bmp != nil {
			ui.Image(c, bmp).Size(16, 16)
		}
	}
	ui.Text(c, text).SingleLine()
}
