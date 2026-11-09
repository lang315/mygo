//go:build !mygo_noinspector

package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// TestAutomationTreeKeys reads the keys of elements as plugins/automation
// does: Element.Key's, and Context.Key's for widgets that take theirs
// before they are made.
func TestAutomationTreeKeys(t *testing.T) {
	name, agree := "Ada", false
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Button(c, "Add").Key("add")
			c.Key("name")
			TextInput(c, &name)
			c.Key(7)
			Checkbox(c, &agree, "Agree")
			Button(c, "Plain")
		})
	}, 400, 300)
	tt.Frame()
	tree, keys := automated{&windowHost{rt: tt.rt}}.AutomationTree()
	want := map[platform.AccessRole]any{platform.RoleButton: "add", platform.RoleTextField: "name", platform.RoleCheckBox: 7}
	seen := 0
	for _, n := range tree.Nodes {
		k, ok := keys[n.ID]
		switch {
		case n.Role == platform.RoleButton && n.Label == "Plain":
			if ok {
				t.Errorf("a button without a key has %v", k)
			}
		case want[n.Role] != nil:
			seen++
			if k != want[n.Role] {
				t.Errorf("%v %q: key %v, want %v", n.Role, n.Label, k, want[n.Role])
			}
		}
	}
	if seen != len(want) {
		t.Errorf("found %d of the keyed elements in %+v", seen, tree.Nodes)
	}
}

// TestAutomationDrawsMenusInTheWindow: a window the automation attaches to
// draws its context menus, so that its snapshot lists their items and
// choosing one runs its action; without the automation the menu is the
// system's, and an app that sets the presentation itself wins.
func TestAutomationDrawsMenusInTheWindow(t *testing.T) {
	run := func(attach bool, app func(c *Context)) (*Tester, *bool) {
		chosen := new(bool)
		tt := NewTester(func(c *Context) {
			if app != nil {
				app(c)
			}
			Column(c).Fill().Padding(20).Children(func() {
				row := Row(c).Padding(8).Children(func() { Text(c, "Report.pdf") })
				row.ContextMenu(func(m *Menu) {
					if m.Item("Open").Chosen() {
						*chosen = true
					}
					m.Item("Delete")
				})
			})
		}, 400, 300)
		if attach {
			automated{&windowHost{rt: tt.rt}}.DrawMenus()
		}
		tt.Frame()
		if err := tt.RightClick("Report.pdf"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		return tt, chosen
	}

	tt, chosen := run(true, nil)
	tree, _ := automated{&windowHost{rt: tt.rt}}.AutomationTree()
	items := map[string]bool{}
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleMenuItem {
			items[n.Label] = true
		}
	}
	if !items["Open"] || !items["Delete"] {
		t.Fatalf("the snapshot's menu items are %v, want Open and Delete", items)
	}
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !*chosen {
		t.Error("choosing Open from the snapshot's menu did not run its action")
	}
	if got := tt.Menu(); got != nil {
		t.Errorf("the menu stays open: %q", got)
	}

	// Without the automation the system shows the menu: none is in the tree.
	tt, _ = run(false, nil)
	tree, _ = automated{&windowHost{rt: tt.rt}}.AutomationTree()
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleMenuItem {
			t.Errorf("without automation the tree has the menu item %q", n.Label)
		}
	}
	// An app that chooses the presentation after the attach keeps its own.
	tt, _ = run(true, func(c *Context) { c.SetMenuPresentation(SystemMenus) })
	if got := tt.rt.menuPresentation; got != SystemMenus {
		t.Errorf("presentation %v after the app chose SystemMenus", got)
	}
}

// TestAutomationSettlesOutOfSight presses a button of a window out of
// sight whose surface draws no frame, as macOS's of a hidden window or
// behind the locked screen: the view takes the press once Settle builds
// the frame waiting, which then reads as pending no more, and the tree
// shows the outcome. In a window that shows, Settle leaves the frame to
// the surface.
func TestAutomationSettlesOutOfSight(t *testing.T) {
	t.Setenv("MYGO_GPU", "0")
	s := &occludedSurface{}
	h := &windowHost{conn: &surface.Conn{Surface: s, Post: func(f func()) { f() }}}
	count, builds := 0, 0
	h.rt = newRuntime(func(c *context) {
		builds++
		coreColumn(c).Children(func() {
			coreTextf(c, "Count: %d", count)
			if coreButton(c, "Add").Clicked() {
				count++
			}
		})
	}, h)
	a := automated{h}
	frame := func() { h.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame}) }
	find := func(label string) uint64 {
		tree, _ := a.AutomationTree()
		for _, n := range tree.Nodes {
			if n.Label == label {
				return n.ID
			}
		}
		return 0
	}
	frame()
	add := find("Add")
	if add == 0 {
		t.Fatal("no button")
	}

	s.occluded = true
	h.event(platform.SurfaceEvent{Kind: platform.AccessAction, ID: add, Action: platform.AccessPress})
	if !a.FramePending() || count != 0 {
		t.Fatalf("after the press: pending %v, count %d", a.FramePending(), count)
	}
	a.Settle()
	if count != 1 || a.FramePending() || find("Count: 1") == 0 {
		t.Fatalf("settled: count %d, pending %v", count, a.FramePending())
	}
	built := builds
	a.Settle()
	if builds != built {
		t.Errorf("settling again built %d more", builds-built)
	}

	// A window that shows draws its frames as the surface asks for them.
	s.occluded = false
	h.event(platform.SurfaceEvent{Kind: platform.AccessAction, ID: add, Action: platform.AccessPress})
	a.Settle()
	if count != 1 || !a.FramePending() {
		t.Fatalf("a press in a window that shows, settled: count %d, pending %v", count, a.FramePending())
	}
	frame()
	if count != 2 || a.FramePending() {
		t.Errorf("after its frame: count %d, pending %v", count, a.FramePending())
	}
}
