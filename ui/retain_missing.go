package ui

import "sync/atomic"

// missingNotes counts the queries asked of an element that is not there.
// Element{} has no context to find the row being built by, so a row cannot be
// told "you asked": instead listFrame.build reads the count before and after the
// callback of a row, and a row whose callback saw it move is built in every frame
// (a query that returns at once, Dismissed of a Modal that is closed or
// Highlighted of the item of a select whose popup is closed, tells the row
// nothing it can depend on, so a row linked later would never ask again when the
// element appears). A row that is built inside another sees the count move too,
// so the rows around it are marked. It is one counter for the process: windows
// that build at the same time (the tests that make Testers on several
// goroutines do) can move each other's count, which only builds a row more, never
// fewer; there is no pointer to a row that two goroutines could write.
var missingNotes atomic.Uint64

// noteMissingElement is called by the queries of Element (api_gen.go) for an
// element that is not there.
func noteMissingElement() { missingNotes.Add(1) }
