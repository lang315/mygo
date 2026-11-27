package ui

import (
	"strconv"
	"testing"
)

// focusOf returns the role and name of the element assistive technology
// has the focus on.
func focusOf(t *testing.T, tt *Tester) (Role, string) {
	t.Helper()
	for _, a := range tt.AccessAll() {
		if a.Focused {
			return a.Role, a.Name
		}
	}
	t.Fatal("assistive technology has the focus on nothing")
	return 0, ""
}

// An active descendant assistive technology does not see gives the focus to
// its nearest ancestor it sees, below the focused element or the popup's
// panel; one it sees keeps it, as before.
func TestAccessActiveDescendantResolvesToExposedAncestor(t *testing.T) {
	const (
		rowContent = iota // the Row a List's row holds: its list item
		rowText           // the text in it, seen: the text itself
		inFocused         // a box in the focused element: the focused element
		inPopup           // a box of the popup's panel: the focused element
	)
	for _, tc := range []struct {
		name    string
		target  int
		role    Role
		focusTo string
	}{
		{"row content", rowContent, RoleListItem, "Row 1"},
		{"exposed text", rowText, RoleText, "Row 1"},
		{"in the focused element", inFocused, RoleGroup, "Holder"},
		{"in the popup", inPopup, RoleTextField, "Search"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var list ListState
			var q string
			open := true
			tt := NewTester(func(c *Context) {
				if tc.target == inFocused {
					holder := Box(c).Label("Holder").Role(RoleGroup).Focusable().AutoFocus().Width(50).Height(20)
					holder.Children(func() {
						inner := Box(c).Width(10).Height(10)
						holder.ActiveDescendant(inner)
					})
					return
				}
				in := TextInput(c, &q).Label("Search").AutoFocus().Width(200)
				if tc.target == inPopup {
					PopoverBase(c, in, &open, func(panel Element) {
						inner := Box(c).Width(10).Height(10)
						in.ActiveDescendant(inner)
					})
					return
				}
				List(c, &list, 3, func(i int) {
					row := Row(c)
					row.Children(func() {
						text := Text(c, "Row "+strconv.Itoa(i))
						if i == 1 && tc.target == rowText {
							in.ActiveDescendant(text)
						}
					})
					if i == 1 && tc.target == rowContent {
						in.ActiveDescendant(row)
					}
				}).Height(100)
			}, 300, 300)
			tt.Frame()
			tt.Frame()
			if role, name := focusOf(t, tt); role != tc.role || name != tc.focusTo {
				t.Fatalf("the focus is on %v %q, want %v %q", role, name, tc.role, tc.focusTo)
			}
		})
	}
}
