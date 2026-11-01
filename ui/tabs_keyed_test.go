package ui

import (
	"slices"
	"testing"
)

// TabsKeyed tells its tabs apart by key, so that the tab with the keyboard
// focus keeps it when tabs come before it; Tabs, by position, does not.
func TestTabsKeyedFocusFollowsTheKey(t *testing.T) {
	for _, keyed := range []bool{true, false} {
		name := "positional"
		if keyed {
			name = "keyed"
		}
		t.Run(name, func(t *testing.T) {
			labels := []string{"A", "B", "C"}
			var sel int
			tt := NewTester(func(c *Context) {
				if keyed {
					keys := make([]any, len(labels))
					for i, l := range labels {
						keys[i] = l
					}
					TabsKeyed(c, &sel, keys, labels)
					return
				}
				Tabs(c, &sel, labels...)
			}, 400, 100)
			tt.Frame()
			if err := tt.Click("C"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			labels = slices.Insert(labels, 0, "N")
			tt.Frame()
			tt.Frame()
			var focused string
			for _, n := range tt.Nodes() {
				if n.Role == RoleTab && n.Focused {
					focused = n.Label
				}
			}
			if keyed && focused != "C" {
				t.Errorf("keyed: focus on %q after a tab came first, want C", focused)
			}
			if !keyed && focused == "C" {
				t.Errorf("positional: the focus followed the label; the test shows nothing")
			}
		})
	}
}

func TestTabsKeyedKeysAreOptional(t *testing.T) {
	var sel int
	tt := NewTester(func(c *Context) { TabsKeyed(c, &sel, nil, []string{"A", "B"}) }, 300, 100)
	tt.Frame()
	if err := tt.Click("B"); err != nil || sel != 1 {
		t.Fatalf("click: %v, selected %d", err, sel)
	}
}
