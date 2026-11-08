//go:build !mygo_noinspector

package automation

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
	"github.com/egoist/mygo/ui"
)

var fb *fake.Backend

// TestMain runs the app on the fake backend, as package mygo's tests do, so
// that windows of native UI attach to the plugin.
func TestMain(m *testing.M) {
	fb = fake.New()
	platform.TestBackend = fb
	mygo.App.SetName("AutomationTest")
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		go func() { os.Exit(m.Run()) }()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// TestAttachedDrawsMenus: a window of native UI that attaches to the plugin
// draws its context menu in the window, so that a right click through the
// plugin's input shows the menu's items in its tree, and a click on one
// runs its action (#444).
func TestAttachedDrawsMenus(t *testing.T) {
	a := &app{ready: make(chan struct{}), targets: map[*mygo.Window]target{}}
	close(a.ready)
	surface.Automate = a.attached
	t.Cleanup(func() { surface.Automate = nil })

	chosen := false
	w := mygo.NewWindow(mygo.WindowOptions{Width: 300, Height: 200, Content: ui.View(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			row := ui.Row(c).Padding(8).Children(func() { ui.Text(c, "Report.pdf") })
			row.ContextMenu(func(m *ui.Menu) {
				if m.Item("Open").Chosen() {
					chosen = true
				}
				m.Item("Delete")
			})
		})
	})})
	t.Cleanup(w.Destroy)
	wins := fb.Windows()
	s := wins[len(wins)-1].FakeSurface()
	frame := func() { mygo.RunOnMain(func() { s.Frame() }) }
	frame()

	// center finds the element in the tree and returns its center.
	center := func(role platform.AccessRole, label string) (x, y float64) {
		t.Helper()
		_, tree, _, err := a.tree(w.ID())
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range tree.Nodes {
			if n.Role == role && n.Label == label {
				return n.Bounds.X + n.Bounds.W/2, n.Bounds.Y + n.Bounds.H/2
			}
		}
		t.Fatalf("the tree has no %v %q: %+v", role, label, tree.Nodes)
		return
	}
	click := func(x, y float64, button string) {
		t.Helper()
		m := do(t, a, fmt.Sprintf(`{"id": 1, "method": "input", "params": {"window": %d, "events": [{"type": "click", "x": %v, "y": %v, "button": %q}]}}`, w.ID(), x, y, button))
		if m["error"] != nil {
			t.Fatal(m["error"])
		}
		frame()
	}

	x, y := center(platform.RoleText, "Report.pdf")
	click(x, y, "right")
	for _, label := range []string{"Open", "Delete"} {
		center(platform.RoleMenuItem, label)
	}
	x, y = center(platform.RoleMenuItem, "Open")
	click(x, y, "left")
	if !chosen {
		t.Error("clicking Open in the snapshot's menu did not run its action")
	}
}
