package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestGridView(t *testing.T) {
	const n = 10000
	sel, opened, built := -1, -1, 0
	s := GridState{Selected: &sel, Label: func(i int) string { return fmt.Sprintf("Photo %d", i) }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			if coreGridView(c, &s, n, 100, 80, func(i int) {
				built++
				coreTextf(c, "Photo %d", i)
			}).Grow(1).Submitted() {
				opened = sel
			}
		})
	}, 448, 400)
	// (448 + 8) / (100 + 8): 4 columns, each (448 - 3*8) / 4 = 106 wide.
	if s.cols != 4 {
		t.Fatalf("%d columns", s.cols)
	}
	built = 0
	tt.Frame()
	if built == 0 || built > 40 {
		t.Errorf("a frame built %d items of %d", built, n)
	}
	r, _ := tt.Find("Photo 5")
	tt.ClickAt(r.X, r.Y)
	if sel != 5 {
		t.Fatalf("a click chose %d", sel)
	}
	for _, step := range []struct {
		key  Key
		want int
	}{{KeyRight, 6}, {KeyDown, 10}, {KeyLeft, 9}, {KeyUp, 5}, {KeyUp, 1}, {KeyUp, 1}, {KeyEnd, n - 1}, {KeyHome, 0}} {
		tt.Key(0, step.key)
		if sel != step.want {
			t.Fatalf("%v chose %d, not %d", step.key, sel, step.want)
		}
	}
	tt.Key(0, KeyEnter)
	if opened != 0 {
		t.Errorf("Enter opened %d", opened)
	}
	// End shows the last item.
	tt.Key(0, KeyEnd)
	if !tt.HasText(fmt.Sprintf("Photo %d", n-1)) {
		t.Error("End did not show the last item")
	}
	// The cells share the width.
	cell := accessNode(t, func() *platform.AccessTree {
		tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		return tt.h.access
	}(), platform.RoleListItem, fmt.Sprintf("Photo %d", n-1))
	if cell.Bounds.W != 106 || cell.Bounds.H != 80 {
		t.Errorf("a cell is %v", cell.Bounds)
	}
	// Assistive technology: a list of n items, the one chosen with the focus.
	tree := tt.h.access
	list := accessNode(t, tree, platform.RoleList, "")
	if list.SetSize != n || list.States&platform.AccessSelectable == 0 || cell.PosInSet != n || cell.SetSize != n || cell.States&platform.AccessChecked == 0 || tree.Focus != cell.ID {
		t.Errorf("the list %+v, the item %+v, the focus on %d", list, cell, tree.Focus)
	}
}

func TestGridViewChoosesSeveral(t *testing.T) {
	sel := -1
	var chosen Selection[string]
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	s := GridState{Selected: &sel, Selection: &chosen, Key: func(i int) any { return names[i] }}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, &s, len(names), 100, 60, func(i int) {
			coreText(c, names[i])
		}).Grow(1)
	}, 448, 300)
	click := func(mods Modifiers, name string) {
		r, _ := tt.Find(name)
		tt.ClickAtWith(mods, r.X, r.Y)
	}
	click(0, "b")
	click(Cmd, "d")
	if chosen.Len() != 2 || !chosen.Has("b") || !chosen.Has("d") {
		t.Fatalf("Cmd-click: %d chosen", chosen.Len())
	}
	click(Shift, "g")
	if chosen.Len() != 4 || !chosen.Has("e") || chosen.Has("b") {
		t.Fatalf("Shift-click from d to g: %d chosen", chosen.Len())
	}
	tt.Key(Shift, KeyUp)
	if sel != 2 || chosen.Len() != 2 || !chosen.Has("c") || !chosen.Has("d") {
		t.Fatalf("Shift-Up from g: chose %d, %d chosen", sel, chosen.Len())
	}
	tt.Key(Cmd, KeyA)
	if chosen.Len() != len(names) {
		t.Errorf("Cmd+A chose %d", chosen.Len())
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleList, ""); n.States&platform.AccessMultiselectable == 0 {
		t.Errorf("the list %+v", n)
	}
}

func TestGridViewResizes(t *testing.T) {
	sel := 13
	s := GridState{Selected: &sel}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, &s, 100, 100, 80, func(i int) {
			coreTextf(c, "Item %d", i)
		}).Grow(1)
	}, 448, 300)
	if s.cols != 4 {
		t.Fatalf("%d columns", s.cols)
	}
	tt.SetSize(664, 300)
	// (664 + 8) / 108: 6 columns.
	if s.cols != 6 {
		t.Fatalf("after widening, %d columns", s.cols)
	}
	tt.Click("Item 13")
	tt.Key(0, KeyDown)
	if sel != 19 {
		t.Errorf("Down from 13 in 6 columns chose %d", sel)
	}
}

// nodeByID returns the element with id under e.
func nodeByID(e *node, id uint64) *node {
	if e.id == id {
		return e
	}
	for ch := e.first; ch != nil; ch = ch.next {
		if n := nodeByID(ch, id); n != nil {
			return n
		}
	}
	return nil
}

func TestGridViewChecked(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	sel := -1
	var chosen, checked Selection[string]
	s := GridState{Selected: &sel, Selection: &chosen, Checked: &checked,
		Key: func(i int) any { return names[i] }, Label: func(i int) string { return names[i] }}
	events := 0
	tt := coreNewTester(func(c *context) {
		if coreGridView(c, &s, len(names), 100, 60, func(i int) {
			coreText(c, names[i])
		}).Grow(1).CheckedChanged() {
			events++
		}
	}, 448, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	box := func(name string) platform.AccessNode {
		return accessNode(t, tt.h.access, platform.RoleCheckBox, "Check "+name)
	}
	// shown reports whether the checkbox of an item is to be seen.
	shown := func(name string) bool {
		n := nodeByID(tt.rt.c.root, box(name).ID)
		return n != nil && !(n.opacitySet && n.opacity == 0)
	}
	clickBox := func(name string) { clickAccess(tt, box(name)) }
	// Nothing checked, the pointer away: no checkbox shows, but each item
	// has its node for assistive technology.
	for _, name := range names {
		if shown(name) {
			t.Errorf("the checkbox of %s shows with nothing checked and no pointer over it", name)
		}
	}
	// The pointer over an item shows its checkbox only.
	r, _ := tt.Find("b")
	tt.Move(r.X, r.Y)
	tt.Frame()
	if !shown("b") || shown("c") {
		t.Errorf("hovering b: b shown %v, c shown %v", shown("b"), shown("c"))
	}
	// A click on the box checks the item, and chooses nothing.
	clickBox("b")
	if !checked.Has("b") || checked.Len() != 1 || events != 1 || sel != -1 || chosen.Len() != 0 {
		t.Fatalf("checking b: %v, %d events, chose %d (%d)", checked.keys, events, sel, chosen.Len())
	}
	if box("b").States&platform.AccessChecked == 0 || box("a").States&platform.AccessChecked != 0 {
		t.Errorf("states: b %+v, a %+v", box("b"), box("a"))
	}
	// With one item checked, all show, as a mode.
	tt.Move(0, 0)
	tt.Frame()
	for _, name := range names {
		if !shown(name) {
			t.Errorf("the checkbox of %s hides while b is checked", name)
		}
	}
	// A shown box keeps a background, to be seen over any image.
	if n := nodeByID(tt.rt.c.root, box("b").ID); n.bg == (Color{}) {
		t.Error("the checkbox has no background")
	}
	// Several chosen: checking an item not among them leaves the choice.
	tt.Click("a")
	tt.ClickWith(Cmd, "c")
	if sel != 2 || chosen.Len() != 2 {
		t.Fatalf("chose %d (%d items)", sel, chosen.Len())
	}
	clickBox("d")
	if sel != 2 || chosen.Len() != 2 || !chosen.Has("a") || !chosen.Has("c") || !checked.Has("d") || checked.Len() != 2 || events != 2 {
		t.Fatalf("checking d: chose %d (%v), checked %v, %d events", sel, chosen.keys, checked.keys, events)
	}
	// Space checks the items chosen, and unchecks them when all are.
	tt.Key(0, KeySpace)
	if checked.Len() != 4 || !checked.Has("a") || !checked.Has("c") || events != 3 {
		t.Fatalf("Space: %v, %d events", checked.keys, events)
	}
	tt.Key(0, KeySpace)
	if checked.Len() != 2 || checked.Has("a") || checked.Has("c") || events != 4 {
		t.Fatalf("Space again: %v, %d events", checked.keys, events)
	}
	// Checks are by key: items that move keep theirs.
	names[1], names[5] = names[5], names[1]
	tt.Frame()
	if box("b").States&platform.AccessChecked == 0 || box("f").States&platform.AccessChecked != 0 {
		t.Errorf("after b and f swapped: b %+v, f %+v", box("b"), box("f"))
	}
}

func TestGridViewCheckedWithoutSelected(t *testing.T) {
	var checked Selection[int]
	s := GridState{Checked: &checked}
	events := 0
	tt := coreNewTester(func(c *context) {
		if coreGridView(c, &s, 6, 100, 60, func(i int) {
			coreTextf(c, "Item %d", i)
		}).Grow(1).CheckedChanged() {
			events++
		}
	}, 448, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	// No Label: the name is "Check" alone.
	b := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check")
	if b.Label != "Check" {
		t.Fatalf("the checkbox is named %q", b.Label)
	}
	r, _ := tt.Find("Item 2")
	tt.Move(r.X, r.Y)
	tt.Frame()
	// A grid that does not choose still checks by click on the box.
	var boxes []platform.AccessNode
	for _, n := range tt.h.access.Nodes {
		if n.Role == platform.RoleCheckBox {
			boxes = append(boxes, n)
		}
	}
	clickAccess(tt, boxes[2]) // the boxes come in the order of the items
	if checked.Len() != 1 || events != 1 {
		t.Fatalf("checked %v, %d events", checked.keys, events)
	}
}

func TestGridViewCheckedStartCorner(t *testing.T) {
	var checked Selection[int]
	checked.Add(0)
	s := GridState{Checked: &checked, Label: func(i int) string { return fmt.Sprintf("Item %d", i) }}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, &s, 6, 100, 60, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
	}, 448, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	cell, b := accessNode(t, tt.h.access, platform.RoleListItem, "Item 0"), accessNode(t, tt.h.access, platform.RoleCheckBox, "Check Item 0")
	if b.Bounds.X > cell.Bounds.X+cell.Bounds.W/2 {
		t.Errorf("left to right, the checkbox at %v in a cell at %v+%v", b.Bounds.X, cell.Bounds.X, cell.Bounds.W)
	}
	tt.SetDirection(RightToLeft)
	cell, b = accessNode(t, tt.h.access, platform.RoleListItem, "Item 0"), accessNode(t, tt.h.access, platform.RoleCheckBox, "Check Item 0")
	if b.Bounds.X < cell.Bounds.X+cell.Bounds.W/2 {
		t.Errorf("right to left, the checkbox at %v in a cell at %v+%v", b.Bounds.X, cell.Bounds.X, cell.Bounds.W)
	}
}

// BenchmarkGridViewChecked1M scrolls a grid of a million items a frame a
// step, plain, with Checked and none checked, and with Checked and some
// (where every checkbox shows). The frame's time is for the cells built,
// not the items.
func BenchmarkGridViewChecked1M(b *testing.B) {
	for _, tc := range []struct {
		name         string
		checks, some bool
	}{{"plain", false, false}, {"checked-none", true, false}, {"checked-some", true, true}} {
		b.Run(tc.name, func(b *testing.B) {
			var s GridState
			var checked Selection[int]
			if tc.checks {
				s.Checked = &checked
			}
			if tc.some {
				checked.Add(5)
			}
			tt := coreNewTester(func(c *context) {
				coreGridView(c, &s, 1_000_000, 100, 80, func(i int) {
					coreTextf(c, "Item %d", i)
				}).Grow(1)
			}, 980, 720)
			tt.Move(400, 300)
			b.ReportAllocs()
			for b.Loop() {
				tt.Scroll(400, 300, 0, 97)
			}
		})
	}
}

func TestGridViewItemMenu(t *testing.T) {
	names := []string{"alpha", "bravo", "charlie", "delta"}
	var got [][]int
	menu := func(m *Menu, items []int) {
		if m.Item("Open").Chosen() {
			got = append(got, slices.Clone(items))
		}
	}
	view := func(s *GridState) func(c *context) {
		return func(c *context) {
			coreGridView(c, s, len(names), 100, 60, func(i int) { coreText(c, names[i]) }).Grow(1)
		}
	}
	var tt *Tester
	open := func() {
		t.Helper()
		if m := tt.Menu(); !slices.Equal(m, []string{"Open"}) {
			t.Fatalf("the menu: %q", m)
		}
		if err := tt.ChooseMenuItem("Open"); err != nil {
			t.Fatal(err)
		}
	}
	// Nothing chosen: the menu key opens nothing.
	sel := -1
	var chosen Selection[string]
	s := GridState{Selected: &sel, Selection: &chosen, ItemMenu: menu, Key: func(i int) any { return names[i] }}
	tt = coreNewTester(view(&s), 448, 300)
	tt.Key(Shift, KeyF10)
	if m := tt.Menu(); m != nil {
		t.Fatalf("Shift+F10 with nothing chosen opened %q", m)
	}
	// A right click on an item not chosen chooses it alone first.
	tt.RightClick("bravo")
	open()
	// Several chosen: a right click on one of them keeps them all, and the
	// menu key opens on them.
	tt.ClickWith(Cmd, "alpha")
	tt.ClickWith(Cmd, "charlie")
	tt.RightClick("charlie")
	open()
	tt.Key(Shift, KeyF10)
	open()
	// On one not chosen: that one alone.
	tt.RightClick("delta")
	open()
	// A grid that does not choose: the item clicked.
	s = GridState{ItemMenu: menu}
	tt = coreNewTester(view(&s), 448, 300)
	tt.RightClick("bravo")
	open()
	want := [][]int{{1}, {0, 1, 2}, {0, 1, 2}, {3}, {1}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("the items: %v, want %v", got, want)
	}
}

func TestGridViewCopy(t *testing.T) {
	names := []string{"alpha", "bravo", "charlie", "delta"}
	sel := -1
	var chosen Selection[int]
	s := GridState{Selected: &sel, Selection: &chosen}
	empty := false
	s.Copy = func(items []int) string {
		if empty {
			return ""
		}
		var parts []string
		for _, i := range items {
			parts = append(parts, names[i])
		}
		return strings.Join(parts, ",")
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, &s, len(names), 100, 60, func(i int) { coreText(c, names[i]) }).Grow(1)
			coreButton(c, "Elsewhere")
		})
	}, 448, 300)
	// Without the focus nor a choice, nothing is copied.
	tt.SetClipboard("before")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Fatalf("Cmd+C without the focus nor a choice: %q", got)
	}
	tt.Click("alpha")
	tt.ClickWith(Cmd, "charlie")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "alpha,charlie" {
		t.Fatalf("Cmd+C: %q", got)
	}
	// The Edit menu's Copy does the same.
	tt.SetClipboard("before")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "alpha,charlie" {
		t.Errorf("Edit > Copy: %q", got)
	}
	// "" leaves the clipboard as it was.
	empty = true
	tt.SetClipboard("before")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Errorf("Copy returned \"\": the clipboard is %q", got)
	}
	empty = false
	// Without the focus, nothing is copied.
	tt.Click("Elsewhere")
	tt.SetClipboard("before")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Errorf("Cmd+C with the focus elsewhere: %q", got)
	}
	// A grid that does not choose has nothing to copy.
	s2 := GridState{Copy: func([]int) string { return "x" }}
	tt = coreNewTester(func(c *context) {
		coreGridView(c, &s2, len(names), 100, 60, func(i int) { coreText(c, names[i]) }).Grow(1)
	}, 448, 300)
	tt.SetClipboard("before")
	tt.Click("alpha")
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "before" {
		t.Errorf("a grid that does not choose copied %q", got)
	}
}

// TestGridViewHiddenCheckboxPassesClick checks that a click on the corner of
// an item whose checkbox is hidden (a tap, with no pointer move before it)
// chooses the item, as on any other part of it.
func TestGridViewHiddenCheckboxPassesClick(t *testing.T) {
	names := []string{"a", "b", "c", "d"}
	sel := -1
	var chosen, checked Selection[string]
	s := GridState{Selected: &sel, Selection: &chosen, Checked: &checked,
		Key: func(i int) any { return names[i] }, Label: func(i int) string { return names[i] }}
	tt := coreNewTester(func(c *context) {
		coreGridView(c, &s, len(names), 100, 60, func(i int) { coreText(c, names[i]) }).Grow(1)
	}, 448, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	clickAccess(tt, accessNode(t, tt.h.access, platform.RoleCheckBox, "Check c"))
	if sel != 2 || !chosen.Has("c") || checked.Len() != 0 {
		t.Fatalf("a click on the hidden checkbox of c: chose %d (%v), checked %v", sel, chosen.keys, checked.keys)
	}
}
