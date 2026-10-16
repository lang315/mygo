package ui

import "testing"

// The same view is built in two testers, two engines: first in a, then in
// b only, as an app moving a panel to another window, which a keeps alive
// meanwhile.
func TestGlobalKeyStateMovesToAnotherWindow(t *testing.T) {
	for _, global := range []bool{true, false} {
		inA := true
		text := "hello"
		var n, m *int
		view := func(show bool) func(c *Context) {
			return func(c *Context) {
				if inA != show {
					if global {
						// The first window keeps it until the other builds it.
						c.KeepAlive("panel")
					}
					return
				}
				key := c.Key("panel")
				if global {
					key = c.GlobalKey("panel")
				}
				e := Column(key)
				n = Local(e, "n", func() int { return 0 })
				e.Children(func() {
					r := Row(c)
					m = Local(r, "m", func() int { return 0 })
					TextInput(c, &text).Label("Name")
				})
			}
		}
		a := NewTester(view(true), 300, 200)
		b := NewTester(view(false), 300, 200)
		*n, *m = 5, 7
		if err := a.Click("Name"); err != nil {
			t.Fatal(err)
		}
		a.Key(0, KeyHome)
		a.Key(Shift, KeyRight) // "h" selected: caret 1, anchor 0
		a.Frame()
		inA = false
		a.Frame()
		b.Frame()
		b.Frame() // a second frame copies nothing again
		ed := editorOf(b)
		got := [2]int{*n, *m}
		if global {
			if got != [2]int{5, 7} {
				t.Errorf("locals %v in the other window, want [5 7]", got)
			}
			if ed == nil {
				t.Error("no editor in the other window")
			} else if ed.caret != 1 || ed.anchor != 0 {
				t.Errorf("caret %d anchor %d in the other window, want 1 and 0", ed.caret, ed.anchor)
			}
		} else if got != [2]int{0, 0} {
			t.Errorf("Key, not GlobalKey: locals %v moved, want fresh [0 0]", got)
		}
	}
}

// editorOf returns the editor of the text input tt's last frame built.
func editorOf(tt *Tester) *editor {
	for _, s := range tt.rt.states {
		if s.editor != nil && s.seen == tt.rt.frame {
			return s.editor
		}
	}
	return nil
}

// movePanel returns the views of two windows showing panel under
// GlobalKey("panel"): the first while *inA, then the other, the first
// keeping it alive meanwhile.
func movePanel(inA *bool, panel func(c *Context)) (a, b func(c *Context)) {
	view := func(show bool) func(c *Context) {
		return func(c *Context) {
			if *inA != show {
				c.KeepAlive("panel")
				return
			}
			Column(c.GlobalKey("panel")).Padding(10).Children(func() { panel(c) })
		}
	}
	return view(true), view(false)
}

// A select open in the first window comes to the other closed.
func TestMovedPopupComesBackClosed(t *testing.T) {
	inA, size := true, "S"
	va, vb := movePanel(&inA, func(c *Context) {
		Select(c, &size, []string{"S", "M", "L"}).Label("Size")
	})
	a, b := NewTester(va, 300, 300), NewTester(vb, 300, 300)
	if err := a.Click("Size"); err != nil {
		t.Fatal(err)
	}
	if !a.HasText("M") {
		t.Fatalf("no option M after a click on the select: %q", a.Texts())
	}
	inA = false
	a.Frame()
	b.Frame()
	if b.HasText("M") {
		t.Fatal("the select's popup came to the other window open")
	}
}

// The focused input of a panel has the focus in the window it moves to;
// the first window's focus leaves it.
func TestMovedFocusFollowsThePanel(t *testing.T) {
	inA, text := true, "ab"
	va, vb := movePanel(&inA, func(c *Context) { TextInput(c, &text).Label("Name") })
	a, b := NewTester(va, 300, 200), NewTester(vb, 300, 200)
	if err := a.Click("Name"); err != nil {
		t.Fatal(err)
	}
	inA = false
	a.Frame()
	b.Frame()
	b.Type("X")
	b.Frame()
	if text != "abX" {
		t.Fatalf("typed into the other window: %q, want abX", text)
	}
	if a.rt.focused != 0 || a.rt.lostFocus != 0 {
		t.Error("the first window kept the focus of the panel that left")
	}
}

// A composition in the first window is committed as the panel moves.
func TestMoveCommitsComposition(t *testing.T) {
	inA, text := true, "ab"
	va, vb := movePanel(&inA, func(c *Context) { TextInput(c, &text).Label("Name") })
	a, b := NewTester(va, 300, 200), NewTester(vb, 300, 200)
	if err := a.Click("Name"); err != nil {
		t.Fatal(err)
	}
	a.Compose("世", 1)
	a.Frame()
	inA = false
	a.Frame()
	b.Frame()
	b.Frame()
	if text != "ab世" {
		t.Fatalf("text %q after the move, want ab世", text)
	}
}

// A panel the first window let go of, unkept, comes fresh to the other,
// and the first window is no longer recorded as its owner.
func TestMoveFromPrunedOwnerStartsFresh(t *testing.T) {
	inA := true
	var n *int
	view := func(show bool) func(c *Context) {
		return func(c *Context) {
			if inA == show {
				n = Local(Column(c.GlobalKey("pruned panel")), "n", func() int { return 0 })
			}
		}
	}
	a := NewTester(view(true), 300, 200)
	*n = 4
	inA = false
	a.Frame()
	globalOwners.Lock()
	_, owned := globalOwners.m[globalID("pruned panel")]
	globalOwners.Unlock()
	if owned {
		t.Error("the first window is still the owner of a panel it pruned")
	}
	NewTester(view(false), 300, 200)
	if *n != 0 {
		t.Fatalf("n %d from a pruned state, want 0", *n)
	}
}
