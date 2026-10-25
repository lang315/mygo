package ui

import "testing"

// A control made Unfocusable is passed by Tab, and still works when clicked.
func TestUnfocusableIsPassedByTab(t *testing.T) {
	var a, b bool
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreCheckbox(c, &a, "First").Unfocusable()
			coreCheckbox(c, &b, "Second")
		})
	}, 300, 200)
	focused := func() string {
		for _, n := range tt.Nodes() {
			if n.Role == RoleCheckBox && n.Focused {
				return n.Label
			}
		}
		return ""
	}
	tt.Key(0, KeyTab)
	if got := focused(); got != "Second" {
		t.Fatalf("Tab focused %q, want Second: First is not focusable", got)
	}
	tt.Key(Shift, KeyTab)
	if got := focused(); got == "First" {
		t.Fatal("Shift+Tab focused First, which is not focusable")
	}
	if err := tt.Click("First"); err != nil {
		t.Fatal(err)
	}
	if !a {
		t.Fatal("the click did not check First")
	}
}
