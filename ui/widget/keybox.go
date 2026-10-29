package widget

// keyBox keeps the keys of the rows in view boxed as an any, for ui's
// ListState.Key: it is asked for every row in view in every frame, and a key
// of 256 or more, a string or a struct converted to an any is allocated each
// time. A direct-mapped memo from the value of a key to its boxed form: the
// box of an equal key is the same, whatever the row, so a row that moved (a
// sort, a scroll) takes the one of its key or is boxed once. Under mu.
type keyBox[K comparable] struct {
	e *[keyBoxSize]any
}

// keyBoxSize is the rows whose boxed keys are kept (a power of two).
const keyBoxSize = 256

// of returns k as an any, boxed once while it stays in the memo; slot is the
// row or position it was asked for, which spreads the keys of the rows in view.
func (b *keyBox[K]) of(slot int, k K) any {
	if b.e == nil {
		b.e = new([keyBoxSize]any)
	}
	e := &b.e[slot&(keyBoxSize-1)]
	if c, ok := (*e).(K); ok && c == k {
		return *e
	}
	*e = k
	return *e
}
