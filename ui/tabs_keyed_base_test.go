package ui

import "testing"

// keyedBaseTabs builds TabsKeyedBase's tabs, one for each label, keyed by
// keys, each its label's text.
func keyedBaseTabs(c *Context, sel *int, keys []any, labels []string) {
	tabs := TabsKeyedBase(c, sel, keys)
	tabs.List.Children(func() {
		for i, l := range labels {
			tabs.Tab(i).Padding(4, 8).Children(func() { Text(c, l) })
		}
	})
}

// focusedTab returns the label of the tab with the keyboard focus, "" for
// none.
func focusedTab(tt *Tester) string {
	for _, n := range tt.Nodes() {
		if n.Role == RoleTab && n.Focused {
			return n.Label
		}
	}
	return ""
}

// The tab with the focus keeps it when the tab before it goes, as
// TabsKeyed's does, and the selection follows the caller's index.
func TestTabsKeyedBaseKeepsFocus(t *testing.T) {
	keys, labels := []any{"a", "b", "c"}, []string{"A", "B", "C"}
	var sel int
	tt := NewTester(func(c *Context) { keyedBaseTabs(c, &sel, keys, labels) }, 400, 100)
	tt.Frame()
	if err := tt.Click("C"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if sel != 2 || focusedTab(tt) != "C" {
		t.Fatalf("after a click on C: selected %d, focus on %q; want 2 and C", sel, focusedTab(tt))
	}
	keys, labels = []any{"b", "c"}, []string{"B", "C"}
	sel = 1 // the caller's index of its page, which moved down
	tt.Frame()
	tt.Frame()
	if f := focusedTab(tt); f != "C" {
		t.Errorf("focus on %q after the first tab went, want C", f)
	}
	if sel != 1 {
		t.Errorf("selected %d, want 1, C's index", sel)
	}
}

// Without keys, the focus stays on the index: the test above shows a
// difference only keys make.
func TestTabsKeyedBasePositionalWithoutKeys(t *testing.T) {
	labels := []string{"A", "B", "C"}
	var sel int
	tt := NewTester(func(c *Context) {
		tabs := TabsBase(c, &sel, len(labels))
		tabs.List.Children(func() {
			for i, l := range labels {
				tabs.Tab(i).Padding(4, 8).Children(func() { Text(c, l) })
			}
		})
	}, 400, 100)
	tt.Frame()
	if err := tt.Click("C"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	labels = []string{"B", "C"}
	sel = 1
	tt.Frame()
	tt.Frame()
	if f := focusedTab(tt); f == "C" {
		t.Errorf("positional tabs kept the focus on C: the keyed test shows nothing")
	}
}

// The tab that had the focus goes: the tab chosen takes it, as TabsKeyed's.
func TestTabsKeyedBaseFocusPassesToTheChosenTab(t *testing.T) {
	keys, labels := []any{"a", "b", "c"}, []string{"A", "B", "C"}
	var sel int
	tt := NewTester(func(c *Context) { keyedBaseTabs(c, &sel, keys, labels) }, 400, 100)
	tt.Frame()
	if err := tt.Click("B"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	keys, labels = []any{"a", "c"}, []string{"A", "C"}
	sel = 0
	tt.Frame()
	tt.Frame()
	if f := focusedTab(tt); f != "A" {
		t.Fatalf("focus on %q, want A", f)
	}
}

// No keys make no tabs, and one key one tab that a click chooses.
func TestTabsKeyedBaseNoneAndOne(t *testing.T) {
	var keys []any
	var labels []string
	sel := 3
	tt := NewTester(func(c *Context) { keyedBaseTabs(c, &sel, keys, labels) }, 300, 100)
	tt.Frame()
	tabs := func() (n int) {
		for _, nd := range tt.Nodes() {
			if nd.Role == RoleTab {
				n++
			}
		}
		return n
	}
	if n := tabs(); n != 0 {
		t.Fatalf("%d tabs of no keys, want 0", n)
	}
	keys, labels = []any{"a"}, []string{"A"}
	tt.Frame()
	if n := tabs(); n != 1 {
		t.Fatalf("%d tabs of one key, want 1", n)
	}
	if sel != 0 {
		t.Errorf("selected %d of one tab, want 0", sel)
	}
	if err := tt.Click("A"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if f := focusedTab(tt); f != "A" {
		t.Errorf("focus on %q after a click, want A", f)
	}
}
