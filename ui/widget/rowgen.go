package widget

// rowGens versions the rows a control shows, by key, for ui.ListState.RowVersion:
// every change of an item gives its key a generation from one counter, so that the
// list builds that row again and keeps the others. Under mu.
type rowGens[K comparable] struct {
	n uint64
	m map[K]uint64
}

// bump gives the row of k a generation no row had.
func (g *rowGens[K]) bump(k K) {
	g.n++
	if g.m == nil {
		g.m = map[K]uint64{}
	}
	g.m[k] = g.n
}

// drop forgets the row of k, which went.
func (g *rowGens[K]) drop(k K) { delete(g.m, k) }

// reset forgets every row: all that come are new.
func (g *rowGens[K]) reset() { g.n++; g.m = nil }

// of returns the generation of the row of k, 0 for a key it holds none of.
func (g *rowGens[K]) of(k K) uint64 { return g.m[k] }

// versionOf mixes what a row's build reads: the generation of its row, one of
// the control's own for how rows look (a column, an image), and the place the
// row has among the rows shown, which a numbered column prints.
// foldLook folds the parts of a row's look, each in a step of its own, so that
// two parts that move together (a setter and an ImageList's change) never cancel.
func foldLook(parts ...uint64) uint64 {
	h := uint64(0xcbf29ce484222325)
	for _, p := range parts {
		h = (h ^ p) * 0x100000001b3
	}
	return h
}

func versionOf(row, look uint64, place int) uint64 {
	return row*0x9e3779b97f4a7c15 ^ look*0xc2b2ae3d27d4eb4f ^ uint64(place)*0x165667b19e3779f9
}
