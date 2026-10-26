package ui

// The states the inspector can force on an element to see how it looks in
// them, as Chrome's :hov panel does.
const (
	forceHover uint8 = 1 << iota
	forcePressed
	forceFocus
)

// InspectorField is a line of a note an element leaves for the inspector:
// a Name and its Value, or, when Value is empty, a heading. A Struck line is
// drawn struck through and dim: a value that something else took the place
// of, as the browsers' Computed view shows the rules that lost.
type InspectorField struct {
	Name, Value string
	Struck      bool
}

// Inspecting reports whether the window's inspector is open, so that code
// which describes its elements for it does the work only then. For developer
// tools; no effect and no cost when the inspector is off or built with
// mygo_noinspector (it is then false, a constant, and what follows it is
// left out of the program).
func (c *context) Inspecting() bool { return c.rt.insp.inspecting() }

// InspectNote leaves a note for the inspector: its Widget tab shows title
// and the fields fn returns for this element, or for the elements inside it,
// when one of them is chosen, and lets the user force the hover, pressed and
// focus states on the element. fn runs when the note is shown, for the one
// element chosen, on the thread that draws frames after the view built. Call
// it only if Inspecting reports true, and after Key, which gives the element
// another ID. For developer tools; no effect and no
// cost when the inspector is off or built with mygo_noinspector.
func (e *node) InspectNote(title string, fields func() []InspectorField) *node {
	if in := &e.c.rt.insp; in.inspecting() {
		in.note(e.id, title, fields)
	}
	return e
}
