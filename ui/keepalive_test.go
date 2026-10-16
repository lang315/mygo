package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

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

// A select open when its page stops being built comes back closed.
func TestKeptPopupComesBackClosed(t *testing.T) {
	show := true
	size := "S"
	tt := NewTester(func(c *Context) {
		Column(c).Padding(10).Children(func() {
			if show {
				Column(c.GlobalKey("page")).Children(func() {
					Select(c, &size, []string{"S", "M", "L"}).Label("Size")
				})
			} else {
				c.KeepAlive("page")
			}
		})
	}, 300, 300)
	if err := tt.Click("Size"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("M") {
		t.Fatalf("no option M after a click on the select: %q", tt.Texts())
	}
	show = false
	tt.Frame()
	show = true
	tt.Frame()
	if tt.HasText("M") {
		t.Fatal("the select's popup came back open")
	}
}

// An app's own local named "open" is not a popup: it stays as it was
// while the element is kept out of sight.
func TestKeptAppLocalOpenStays(t *testing.T) {
	show := true
	var open *bool
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if show {
				open = Local(Column(c.GlobalKey("page")), "open", func() bool { return false })
			} else {
				c.KeepAlive("page")
			}
		})
	}, 300, 200)
	*open = true
	show = false
	tt.Frame()
	show = true
	tt.Frame()
	if !*open {
		t.Fatal("the app's local \"open\" was reset while kept")
	}
}

// Assistive technology pressing a node of the last frame shown does
// nothing once its element is kept but no longer built.
func TestAccessActionSkipsKeptUnbuilt(t *testing.T) {
	show, clicks := true, 0
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if show {
				Column(c.GlobalKey("page")).Children(func() {
					if Button(c, "Go").Clicked() {
						clicks++
					}
				})
			} else {
				c.KeepAlive("page")
			}
		})
	}, 300, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	id := accessNode(t, tt.h.access, platform.RoleButton, "Go").ID
	show = false
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: id, Action: platform.AccessPress})
	show = true
	tt.Frame()
	if clicks != 0 {
		t.Fatalf("a press on the hidden button clicked it %d times", clicks)
	}
}

// A global key no element took does not make the next Key global.
func TestUnusedGlobalKeyStaysOut(t *testing.T) {
	NewTester(func(c *Context) {
		for range 2 {
			Column(c).Children(func() {
				c.GlobalKey("unused")
				Column(c.Key("a"))
			})
		}
	}, 300, 200)
}

// The duplicate of a global key is reported as such, not as siblings'.
func TestGlobalKeyDuplicateMessage(t *testing.T) {
	defer func() {
		if msg, _ := recover().(string); !strings.Contains(msg, "global key") {
			t.Fatalf("panic %q, want it to name the global key", msg)
		}
	}()
	NewTester(func(c *Context) {
		Column(c).Children(func() { Column(c.GlobalKey("x")) })
		Column(c).Children(func() { Column(c.GlobalKey("x")) })
	}, 300, 200)
}

// An element kept with the focus gets it back when built again, unless
// another took the focus meanwhile.
func TestKeptElementGetsItsFocusBack(t *testing.T) {
	for _, other := range []bool{false, true} {
		show := true
		var a, b string
		tt := NewTester(func(c *Context) {
			Column(c).Children(func() {
				TextInput(c, &b).Label("other")
				if show {
					Column(c.GlobalKey("panel")).Children(func() {
						TextInput(c, &a).Label("name")
					})
				} else {
					c.KeepAlive("panel")
				}
			})
		}, 300, 200)
		if err := tt.Click("name"); err != nil {
			t.Fatal(err)
		}
		show = false
		tt.Frame()
		if tt.Focused("name") {
			t.Fatal("the focus stays in an element not built")
		}
		if other {
			if err := tt.Click("other"); err != nil {
				t.Fatal(err)
			}
		}
		show = true
		tt.Frame()
		if got := tt.Focused("name"); got == other {
			t.Errorf("another focused %v: name focused %v as it came back", other, got)
		}
		if other && !tt.Focused("other") {
			t.Error("the element coming back took the focus from another")
		}
	}
}
