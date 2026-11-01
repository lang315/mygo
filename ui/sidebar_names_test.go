package ui

import "testing"

// A sidebar entry is named by its text; the text is not a second node, so a
// screen reader does not read "Mail" twice.
func TestSidebarEntryIsOneNodeInTheAccessibilityTree(t *testing.T) {
	sel := "mail"
	tt := NewTester(func(c *Context) {
		Sidebar(c, &sel, func() {
			SidebarItem(c, "mail", nil, "Mail")
			SidebarItem(c, "cal", nil, "Calendar")
		})
	}, 300, 200)
	tt.Frame()
	named := map[string]int{}
	for _, n := range tt.Nodes() {
		if n.Label != "" {
			named[n.Label]++
		}
	}
	for _, name := range []string{"Mail", "Calendar"} {
		if named[name] != 1 {
			t.Errorf("%d nodes named %q, want 1 (tree: %v)", named[name], name, named)
		}
	}
	// The entry is still found and chosen by its text.
	if err := tt.Click("Calendar"); err != nil || sel != "cal" {
		t.Errorf("click on Calendar: %v, selected %q", err, sel)
	}
}

func TestSidebarEntryKeepsItsTextDrawn(t *testing.T) {
	sel := "mail"
	tt := NewTester(func(c *Context) {
		Sidebar(c, &sel, func() {
			SidebarItem(c, "mail", nil, "Mail")
		})
	}, 300, 200)
	tt.Frame()
	for _, n := range tt.Nodes() {
		if n.Role == RoleTreeItem && n.Label != "Mail" {
			t.Errorf("the entry is named %q", n.Label)
		}
	}
	if !tt.HasText("Mail") {
		t.Errorf("texts %q: the text is no longer drawn", tt.Texts())
	}
}
