package widget

import (
	"weak"

	"github.com/egoist/mygo/ui"
)

// keeper keeps the ui state of the widgets a view built while the app
// holds them, through frames that do not build them: hidden, on a page not
// shown, or out of the view for a while. Under mu.
type keeper struct {
	widgets map[int]weak.Pointer[core]
	built   map[int]bool
}

// keeping is the keeper of the view being built. Only views touch it,
// under mu.
var keeping *keeper

// lastKeeper is the keeper of the view built last, for tests.
var lastKeeper *keeper

// begin starts a build of the view.
func (k *keeper) begin() {
	if k.widgets == nil {
		k.widgets, k.built = map[int]weak.Pointer[core]{}, map[int]bool{}
	}
	clear(k.built)
	keeping, lastKeeper = k, k
}

// hold registers b in the view being built without building it, so a
// change to a widget hidden in this frame still finds the window, and end
// keeps its state.
func (k *keeper) hold(b *core) {
	b.builtIn = building
	if _, ok := k.widgets[b.id]; !ok {
		k.widgets[b.id] = weak.Make(b)
	}
}

// key keys the next element for b: globally the first time the frame
// builds b, and under its parent for a widget shown in two places.
func (k *keeper) key(c *ui.Context, b *core) {
	if k.built[b.id] {
		c.Key(b.id)
		return
	}
	k.built[b.id] = true
	b.builtIn = building
	if _, ok := k.widgets[b.id]; !ok {
		// Weak, so the app dropping the widget frees it and its state.
		k.widgets[b.id] = weak.Make(b)
	}
	c.GlobalKey(b.id)
}

// end keeps alive the state of the widgets not built that the app still
// holds, and forgets the others, whose state ui then frees: those dropped,
// and those another view built since.
func (k *keeper) end(c *ui.Context) {
	for id, p := range k.widgets {
		if k.built[id] {
			continue
		}
		if b := p.Value(); b == nil || b.builtIn != building {
			delete(k.widgets, id)
		} else {
			c.KeepAlive(id)
		}
	}
	keeping = nil
}
