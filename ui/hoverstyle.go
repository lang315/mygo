package ui

// The hover looks of rows, items, triggers and cells: functions that read the
// node and the theme it was built with (builtWith), so that no closure is made
// for each element in each frame. Set one with `e.builtWith(t)` and
// `e.setStyle(staticHoverSurface)`.

// hoverSurface gives an element the theme's hover color while the pointer is
// over it.
func hoverSurface(e *node) {
	if e.Hovered() {
		e.bg = e.builtTheme().SurfaceHover
	}
}

// hoverRow is hoverSurface for a row or an item that is not chosen: a chosen
// one (checked is 2) keeps the color of its choice.
func hoverRow(e *node) {
	if e.checked != 2 && e.Hovered() {
		e.bg = e.builtTheme().SurfaceHover
	}
}

// pressSurface is hoverSurface with the pressed color while the pointer
// presses the element.
func pressSurface(e *node) {
	if e.Pressed() {
		e.bg = e.builtTheme().SurfacePressed
	} else if e.Hovered() {
		e.bg = e.builtTheme().SurfaceHover
	}
}
