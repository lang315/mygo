package widget

import "github.com/egoist/mygo/ui"

// keeper tracks the widgets a view builds. Under mu.
type keeper struct {
	built map[int]bool
}

// keeping is the keeper of the view being built. Only views touch it,
// under mu.
var keeping *keeper

// begin starts a build of the view.
func (k *keeper) begin() {
	if k.built == nil {
		k.built = map[int]bool{}
	}
	clear(k.built)
	keeping = k
}

// key keys the next element for b: globally the first time the frame
// builds b, and under its parent for a widget shown in two places.
func (k *keeper) key(c *ui.Context, b *core) {
	if k.built[b.id] {
		c.Key(b.id)
		return
	}
	k.built[b.id] = true
	c.GlobalKey(b.id)
}

// end ends the build of the view.
func (k *keeper) end() { keeping = nil }
