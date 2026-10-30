package ui

// readsHandle is called first by the queries of a Handle (Focused, FocusWithin,
// Bounds): a handle names any element of the window, which may not be built in
// this pass while its state is, so a row cannot compare what it read against the
// states (as retain_reads.go does for the elements under the row). The row that
// asks is built, like one that asks of an element outside it.
func (rt *engine) readsHandle() {
	if rt != nil {
		rt.c.neverKept()
	}
}
