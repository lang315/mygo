package ui

import "testing"

// panel builds a column keyed globally by "panel" holding a local n and a
// row inside with a local m, and returns pointers to both.
func panel(c *Context) (n, m *int) {
	e := Column(c.GlobalKey("panel"))
	n = Local(e, "n", func() int { return 0 })
	e.Children(func() {
		r := Row(c)
		m = Local(r, "m", func() int { return 0 })
	})
	return n, m
}

// An element keyed globally, not built for a few frames while KeepAlive
// keeps it, comes back with its state and its children's.
func TestKeepAliveKeepsAnUnbuiltSubtree(t *testing.T) {
	for _, keep := range []bool{true, false} {
		show := true
		var n, m *int
		tt := NewTester(func(c *Context) {
			Column(c).Children(func() {
				if show {
					n, m = panel(c)
				} else if keep {
					c.KeepAlive("panel")
				}
			})
		}, 300, 200)
		*n, *m = 5, 7
		show = false
		for range 5 {
			tt.Frame()
		}
		show = true
		tt.Frame()
		want := map[bool][2]int{true: {5, 7}, false: {0, 0}}[keep]
		if [2]int{*n, *m} != want {
			t.Errorf("KeepAlive %v: n %d, m %d after 5 frames unbuilt, want %v", keep, *n, *m, want)
		}
	}
}

// An element keyed globally keeps its state when it moves to another
// parent.
func TestGlobalKeyKeepsStateAcrossParents(t *testing.T) {
	left := true
	var n *int
	tt := NewTester(func(c *Context) {
		Row(c).Children(func() {
			Column(c).Children(func() {
				if left {
					n, _ = panel(c)
				}
			})
			Column(c).Children(func() {
				if !left {
					n, _ = panel(c)
				}
			})
		})
	}, 300, 200)
	*n = 3
	left = false
	tt.Frame()
	if *n != 3 {
		t.Fatalf("n %d after moving to another column, want 3", *n)
	}
}

// Two elements with one global key in a pass are a duplicate key, which
// a Tester panics for.
func TestGlobalKeyDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("no panic for two elements with one global key")
		}
	}()
	NewTester(func(c *Context) {
		Column(c).Children(func() { Column(c.GlobalKey("x")) })
		Column(c).Children(func() { Column(c.GlobalKey("x")) })
	}, 300, 200)
}
