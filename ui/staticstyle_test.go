package ui

import "testing"

// The rows of a Table style themselves with static functions, which a later
// frame can tell are the same (setStyle): a closure cannot be compared.
func TestRowsStyleThemselvesStatically(t *testing.T) {
	var s ListState
	var table *node
	tt := coreNewTester(func(c *context) {
		table = coreTable(c, &s, []TableColumn{{Title: "A"}}, 3, func(row, col int) { coreText(c, "x") }).Grow(1)
	}, 200, 200)
	tt.Frame()
	styled := 0
	var walk func(e *node)
	walk = func(e *node) {
		if e.styleFn != nil {
			styled++
			if e.styleID == staticNone {
				t.Errorf("an element of kind %v styles itself with a closure, not a static style", e.kind)
			}
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(table)
	if styled == 0 {
		t.Fatal("no styled element in a table")
	}
}
