// Package twsource lets tw tell widget where a Style came from, without a
// public method on Style for it.
package twsource

// Note notes on the *widget.Style style the Tailwind classes that made it, for
// the inspector; empty classes (or only blanks) take the note off. widget sets
// it in init; style is an any because this package cannot import widget.
var Note func(style any, classes string)
