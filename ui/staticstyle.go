package ui

// styleID names an element's static styleFn, so that a later frame can tell
// the element styles itself as before: closures cannot be compared. An element
// with a styleFn and staticNone has a closure.
type styleID uint8

const (
	staticNone styleID = iota
	staticHoverSurface
	staticHoverRow
	staticPressSurface
	staticTableRow
	staticCheckbox
	staticInput
	staticButton
)

// setStyle styles e with a static styleFn and its identity. Callers pass the
// function, so that a program links only the styles it uses.
func setStyle(e *node, fn func(e *node), id styleID) { e.styleFn, e.styleID = fn, id }
