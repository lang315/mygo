//go:build !mygo_noinspector

package automation

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

type lateRow struct {
	id   int
	name string
}

func lateColumn() widget.Column[lateRow] {
	return widget.Column[lateRow]{Title: "Name", Text: func(r *lateRow) string { return r.name }, Width: 200}
}

// TestAttachedLateMenus: a late menu (ContextMenuFunc of DataGridView and
// ListView, built through ui.Menu.Later) opens for a right click through
// the plugin's input, shows its items in the tree, and a click on one runs
// its OnClick.
func TestAttachedLateMenus(t *testing.T) {
	a := &app{ready: make(chan struct{}), targets: map[*mygo.Window]target{}}
	close(a.ready)
	surface.Automate = a.attached
	t.Cleanup(func() { surface.Automate = nil })

	rows := []lateRow{{1, "ann"}, {2, "bob"}}
	key := func(r *lateRow) int { return r.id }
	for name, mk := range map[string]func(fn func([]int) []widget.MenuItem) widget.Control{
		"DataGridView": func(fn func([]int) []widget.MenuItem) widget.Control {
			return widget.DataGridView(key, lateColumn()).SetRows(rows).Expand().ContextMenuFunc(fn)
		},
		"ListView": func(fn func([]int) []widget.MenuItem) widget.Control {
			return widget.ListView(key, lateColumn()).SetRows(rows).Expand().ContextMenuFunc(fn)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ran := make(chan int, 1)
			ctl := mk(func(sel []int) []widget.MenuItem {
				return []widget.MenuItem{{Text: "Late open", OnClick: func() { ran <- sel[0] }}}
			})
			w := mygo.NewWindow(mygo.WindowOptions{Width: 400, Height: 300, Content: ui.View(widget.View(ctl))})
			t.Cleanup(w.Destroy)
			wins := fb.Windows()
			s := wins[len(wins)-1].FakeSurface()
			frame := func() { mygo.RunOnMain(func() { s.Frame() }) }
			frame()

			center := func(role platform.AccessRole, label string) (x, y float64, ok bool) {
				_, tree, _, err := a.tree(w.ID())
				if err != nil {
					t.Fatal(err)
				}
				for _, n := range tree.Nodes {
					if n.Role == role && n.Label == label {
						return n.Bounds.X + n.Bounds.W/2, n.Bounds.Y + n.Bounds.H/2, true
					}
				}
				return 0, 0, false
			}
			click := func(x, y float64, button string) {
				t.Helper()
				m := do(t, a, fmt.Sprintf(`{"id": 1, "method": "input", "params": {"window": %d, "events": [{"type": "click", "x": %v, "y": %v, "button": %q}]}}`, w.ID(), x, y, button))
				if m["error"] != nil {
					t.Fatal(m["error"])
				}
				frame()
			}

			var x, y float64
			var ok bool
			for _, role := range []platform.AccessRole{platform.RoleText, platform.RoleCell, platform.RoleRow} {
				if x, y, ok = center(role, "bob"); ok {
					break
				}
			}
			if !ok {
				_, tree, _, _ := a.tree(w.ID())
				t.Fatalf("no bob in the tree: %+v", tree.Nodes)
			}
			click(x, y, "right")
			// The late items arrive from the event goroutine: wait for the
			// menu through the widget's own queue, then draw.
			done := make(chan struct{})
			widget.Post(func() { close(done) })
			<-done
			frame()
			mx, my, ok := center(platform.RoleMenuItem, "Late open")
			if !ok {
				_, tree, _, _ := a.tree(w.ID())
				t.Fatalf("the tree has no menuItem %q: %+v", "Late open", tree.Nodes)
			}
			click(mx, my, "left")
			done = make(chan struct{})
			widget.Post(func() { close(done) })
			<-done
			select {
			case got := <-ran:
				if got != 2 {
					t.Errorf("OnClick got key %s, want 2", strconv.Itoa(got))
				}
			default:
				t.Error("clicking the late item did not run its OnClick")
			}
		})
	}
}
