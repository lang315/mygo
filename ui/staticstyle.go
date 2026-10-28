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

// staticStyles are the static styleFns, by their styleID.
var staticStyles = [...]func(e *node){
	staticNone:         nil,
	staticHoverSurface: hoverSurface,
	staticHoverRow:     hoverRow,
	staticPressSurface: pressSurface,
	staticTableRow:     tableRowStyle,
	staticCheckbox:     checkboxStyle,
	staticInput:        inputStyle,
	staticButton:       buttonPaint,
}

// setStyle styles e with a static styleFn.
func (e *node) setStyle(id styleID) { e.styleFn, e.styleID = staticStyles[id], id }
