package ui

// buildingRow is the row whose callback is being built, for the queries of an
// element that is not there: Element{} has no context to find the row by, and a
// query that returns at once (Dismissed of a Modal that is closed, Highlighted
// of the item of a select whose popup is closed) tells the row nothing it can
// depend on, so a row linked later would never ask again when the element
// appears. It is set around the callback of every row (listFrame.build), nil
// outside one. It is a package variable because there is no other way to the row
// from a zero Element: the UI of a process builds on one thread, and no test of
// ui runs windows in parallel.
var buildingRow *rowBuild

// noteMissingElement is called by the queries of Element (api_gen.go) for an
// element that is not there: the row being built, and the rows around it, are
// built in every frame, as their answer is not tied to anything they can compare.
func noteMissingElement() {
	for rb := buildingRow; rb != nil; rb = rb.outer {
		rb.outside = true
	}
}
