package widget

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// The cases of P2e's plan (an empty state in ListBox and ListView), one test
// each, and the review focus's.

const listEmptyTitle = "Chưa có mục nào"

var emptyRed = ui.RGB(255, 0, 0)

// redState is an empty state filled red, whose box redBox finds in the image.
func redState() *EmptyStateWidget {
	return EmptyState(listEmptyTitle, "Thêm một mục để bắt đầu.").Apply(NewStyle().Background(Fixed(emptyRed)))
}

// redBox is the box, in DIPs at scale 1, of the red pixels of the last frame.
func redBox(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	img := tt.Image()
	r := img.Bounds()
	x0, y0, x1, y1 := r.Max.X, r.Max.Y, -1, -1
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.RGBAAt(x, y) == (color.RGBA{255, 0, 0, 255}) {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		t.Fatalf("no red pixel: the empty state is not drawn (%q)", tt.Texts())
	}
	return ui.Rect{X: float32(x0), Y: float32(y0), W: float32(x1 - x0 + 1), H: float32(y1 - y0 + 1)}
}

// sameBox reports whether a and b have the same edges within 1 DIP.
func sameBox(a, b ui.Rect) bool {
	return approx(a.X, b.X) && approx(a.Y, b.Y) && approx(a.X+a.W, b.X+b.W) && approx(a.Y+a.H, b.Y+b.H)
}

// titles counts the texts of the empty state's title in the last frame.
func titles(tt *ui.Tester) int {
	n := 0
	for _, s := range tt.Texts() {
		if s == listEmptyTitle {
			n++
		}
	}
	return n
}

func emptyView() *ListViewWidget[vitem, string] { return newView().SetRows(nil) }

// listNode is the node of the list control, of role RoleList.
func listNode(t *testing.T, tt *ui.Tester) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleList {
			return n
		}
	}
	t.Fatal("no list node")
	return ui.TestNode{}
}

// focusInList reports whether the keyboard focus is on the list or one of its
// items.
func focusInList(tt *ui.Tester) bool {
	for _, n := range tt.Nodes() {
		if n.Focused && (n.Role == ui.RoleList || n.Role == ui.RoleListItem) {
			return true
		}
	}
	return false
}

// Case 1: a ListBox with no items draws its empty state over its whole box.
func TestListBoxEmptyStateFillsTheList(t *testing.T) {
	calmOff()
	lb := newListBox().SetSize(300, 200).SetEmptyState(redState())
	tt := tester(t, VBox(0).Add(lb).Padding(20))
	if !tt.HasText(listEmptyTitle) {
		t.Fatalf("an empty list box shows %q, want its empty state", tt.Texts())
	}
	if red, list := redBox(t, tt), listNode(t, tt).Bounds; !sameBox(red, list) {
		t.Errorf("the empty state's box is %v, the list's %v: want the same within 1 DIP", red, list)
	}
}

// Case 2: items take the state away; removing them all brings it back at the
// next frame.
func TestListBoxEmptyStateGoesWithItems(t *testing.T) {
	calmOff()
	lb := newListBox().SetItems([]litem{{"a", "Apple"}}).SetEmptyState(redState()).Expand()
	tt := tester(t, lb)
	if tt.HasText(listEmptyTitle) {
		t.Fatal("the empty state shows over an item")
	}
	lb.AddItem(litem{"b", "Banana"})
	tt.Frame()
	if tt.HasText(listEmptyTitle) || !tt.HasText("Banana") {
		t.Fatalf("a list of two shows %q", tt.Texts())
	}
	lb.RemoveItems("a", "b")
	tt.Frame()
	if !tt.HasText(listEmptyTitle) {
		t.Fatalf("a list whose items were all removed shows %q, want its empty state", tt.Texts())
	}
}

// Case 3 (as changed by the orchestrator: SetItems and Clear drop the choice
// of the keys gone, by contract): the list's element stays where it is while
// the state comes and goes, so the keyboard focus stays in the list across
// items, none and items again, and Down then chooses the first; and a list
// with items keeps its scroll and its choice across SetEmptyState(e) and
// SetEmptyState(nil).
func TestListBoxEmptyStateKeepsTheListsPlace(t *testing.T) {
	calmOff()
	items := followItems(1, 200)
	lb := newListBox().SetItems(items[:3]).SetEmptyState(redState()).Expand()
	tt := tester(t, lb)
	click(t, tt, "r0002")
	if !focusInList(tt) {
		t.Fatal("clicking an item does not focus the list")
	}
	lb.Clear()
	tt.Frame()
	if !tt.HasText(listEmptyTitle) {
		t.Fatalf("the emptied list shows %q, want its empty state", tt.Texts())
	}
	if !focusInList(tt) {
		t.Fatal("the list lost the focus as its empty state came")
	}
	lb.SetItems(items[:3])
	tt.Frame()
	if !focusInList(tt) {
		t.Fatal("the list lost the focus as its empty state went")
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Frame()
	if got := lb.Selected(); len(got) != 1 || got[0] != "k1" {
		t.Fatalf("Down after the empty state chose %v, want [k1]", got)
	}

	// Scroll and choice across SetEmptyState, with items.
	lb2 := newListBox().SetItems(items).Expand()
	tt2 := tester(t, lb2)
	lb2.SetSelected("k150").EnsureVisible("k150")
	tt2.Frame()
	tt2.Frame()
	if !inView(tt2, "r0150") || inView(tt2, "r0001") {
		t.Fatalf("EnsureVisible(k150) shows %q", tt2.Texts())
	}
	for _, e := range []*EmptyStateWidget{redState(), nil} {
		lb2.SetEmptyState(e)
		tt2.Frame()
		tt2.Frame()
		if !inView(tt2, "r0150") || inView(tt2, "r0001") {
			t.Errorf("SetEmptyState(%v) moved the list: %q", e != nil, tt2.Texts())
		}
		if got := lb2.Selected(); len(got) != 1 || got[0] != "k150" {
			t.Errorf("SetEmptyState(%v) changed the choice to %v", e != nil, got)
		}
	}
}

// Case 4: SetEmptyState(nil) removes the state; the list's tree is that of a
// list that never had one (the box it built in stays, and is no node).
func TestListBoxEmptyStateNil(t *testing.T) {
	calmOff()
	lb := newListBox().SetEmptyState(redState()).SetSize(300, 200)
	tt := tester(t, lb)
	if !tt.HasText(listEmptyTitle) {
		t.Fatal("no empty state to remove")
	}
	lb.SetEmptyState(nil)
	tt.Frame()
	if tt.HasText(listEmptyTitle) {
		t.Fatal("SetEmptyState(nil) still draws it")
	}
	plain := tester(t, newListBox().SetSize(300, 200))
	got, want := tt.Nodes(), plain.Nodes()
	if len(got) != len(want) {
		t.Fatalf("%d nodes after SetEmptyState(nil), %d in a list that never had one", len(got), len(want))
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.Role != w.Role || g.Label != w.Label || !sameBox(g.Bounds, w.Bounds) {
			t.Errorf("node %d is %v %q %v, want %v %q %v", i, g.Role, g.Label, g.Bounds, w.Role, w.Label, w.Bounds)
		}
	}
}

// Case 5: in Details the state sits under the header row, as the grid's: its
// top is the header's bottom + 1, and the headers stay.
func TestListViewDetailsEmptyStateUnderTheHeaders(t *testing.T) {
	calmOff()
	lv := emptyView().SetEmptyState(redState()).SetSize(400, 300)
	tt := tester(t, VBox(0).Add(lv).Padding(20))
	if !tt.HasText(listEmptyTitle) || !tt.HasText("Name") || !tt.HasText("Kind") {
		t.Fatalf("an empty Details view shows %q, want the headers and the empty state", tt.Texts())
	}
	head := nodeOf(t, tt, ui.RoleColumnHeader, "Name").Bounds
	if red := redBox(t, tt); !approx(red.Y, head.Y+head.H+1) {
		t.Errorf("the empty state's top is %v, the header's bottom %v: want it 1 under", red.Y, head.Y+head.H)
	}
}

// Case 6: in an icon view the state covers the whole item area, from the
// list's top.
func TestListViewIconEmptyStateCoversTheGrid(t *testing.T) {
	calmOff()
	for _, m := range allModes[1:] {
		t.Run(m.name, func(t *testing.T) {
			lv := emptyView().SetView(m.mode).SetEmptyState(redState()).SetSize(400, 300)
			tt := tester(t, VBox(0).Add(lv).Padding(20))
			if red, grid := redBox(t, tt), listBox(t, tt); !sameBox(red, grid) {
				t.Errorf("the empty state's box is %v, the grid's %v: want the same within 1 DIP", red, grid)
			}
		})
	}
}

// Case 7: switched from Details to each icon view and back with no rows, the
// state shows in each, once.
func TestListViewEmptyStateAcrossViews(t *testing.T) {
	calmOff()
	lv := emptyView().SetEmptyState(redState()).Expand()
	tt := tester(t, lv)
	for _, m := range append(allModes, allModes[0]) {
		lv.SetView(m.mode)
		tt.Frame()
		if n := titles(tt); n != 1 {
			t.Errorf("%s: the empty state's title shows %d times, want once (%q)", m.name, n, tt.Texts())
		}
	}
}

// Case 8: rows take the state away in every view; deleting them all brings it
// back.
func TestListViewEmptyStateGoesWithRows(t *testing.T) {
	calmOff()
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode).SetEmptyState(redState()).Expand()
			tt := tester(t, lv)
			if tt.HasText(listEmptyTitle) {
				t.Fatal("the empty state shows over rows")
			}
			lv.DeleteRows("a", "b", "c", "d")
			tt.Frame()
			if !tt.HasText(listEmptyTitle) {
				t.Fatalf("a list view whose rows were all deleted shows %q, want its empty state", tt.Texts())
			}
		})
	}
}

// Case 9: the state's action runs when clicked, in a ListBox and in each view
// of a ListView, and Tab reaches it.
func TestListEmptyStateAction(t *testing.T) {
	calmOff()
	ran := make(chan string, 8)
	state := func(name string) *EmptyStateWidget {
		return redState().SetAction("Thêm", func() { ran <- name })
	}
	lists := map[string]Control{"ListBox": newListBox().SetEmptyState(state("ListBox")).Expand()}
	for _, m := range allModes {
		lists[m.name] = emptyView().SetView(m.mode).SetEmptyState(state(m.name)).Expand()
	}
	for name, l := range lists {
		t.Run(name, func(t *testing.T) {
			tt := tester(t, l)
			click(t, tt, "Thêm")
			if got := received(t, ran); got != name {
				t.Fatalf("clicking the action ran %q's", got)
			}
			for range 6 {
				if nodeOf(t, tt, ui.RoleButton, "Thêm").Focused {
					return
				}
				tt.Key(0, ui.KeyTab)
				tt.Frame()
			}
			t.Fatal("Tab does not reach the action")
		})
	}
}

// Case 10: a hidden empty state draws nothing, and shows again.
func TestListEmptyStateHidden(t *testing.T) {
	calmOff()
	lists := map[string]func(*EmptyStateWidget) Control{
		"ListBox": func(e *EmptyStateWidget) Control { return newListBox().SetEmptyState(e).Expand() },
		"Details": func(e *EmptyStateWidget) Control { return emptyView().SetEmptyState(e).Expand() },
		"LargeIcon": func(e *EmptyStateWidget) Control {
			return emptyView().SetView(ViewLargeIcon).SetEmptyState(e).Expand()
		},
	}
	for name, mk := range lists {
		t.Run(name, func(t *testing.T) {
			e := redState()
			tt := tester(t, mk(e))
			e.SetVisible(false)
			tt.Frame()
			if tt.HasText(listEmptyTitle) {
				t.Fatal("a hidden empty state shows")
			}
			e.SetVisible(true)
			tt.Frame()
			if !tt.HasText(listEmptyTitle) {
				t.Fatalf("the empty state shown again shows %q", tt.Texts())
			}
		})
	}
}

// Case 11: a list without SetEmptyState builds as on main: no hook is set, and
// a ListBox's element is the list itself. (The studio's shots check the rest.)
func TestListsWithoutEmptyStateBuildAsBefore(t *testing.T) {
	calmOff()
	lb, lv := newListBox().SetSize(300, 200), newView().SetView(ViewLargeIcon)
	if lb.overlay != nil || lv.overlay != nil || lv.t.withEmpty != nil {
		t.Fatal("a list without SetEmptyState has a hook")
	}
	tt := tester(t, VBox(0).Add(lb).Padding(20))
	if got := listNode(t, tt).Bounds; !approx(got.W, 300) || !approx(got.H, 200) {
		t.Errorf("the list's box is %v, want 300 by 200", got)
	}
}

// Review Focus 1: at text scale 1.5, in a list box 100 DIPs high, the state is
// clipped by the list's box: nothing is drawn above it or under it.
func TestListBoxEmptyStateClippedAtTextScale(t *testing.T) {
	calmOff()
	e := EmptyState(listEmptyTitle, "Thêm một mục để bắt đầu, rồi chọn nó trong danh sách.").SetAction("Thêm", func() {})
	lb := newListBox().SetSize(300, 100).SetEmptyState(e)
	tt := tester(t, VBox(0).Add(lb).Padding(100))
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	box, img, bg := listNode(t, tt).Bounds, tt.Image(), page(tt)
	for _, y := range []int{int(box.Y) - 40, int(box.Y) - 2, int(box.Y+box.H) + 2, int(box.Y+box.H) + 40} {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if img.RGBAAt(x, y) != bg {
				t.Fatalf("at (%d, %d), outside the list %v, a pixel %v is drawn: the state is not clipped", x, y, box, img.RGBAAt(x, y))
			}
		}
	}
}

// Review Focus 2: SetEmptyState and items changed from other goroutines while
// frames build (under -race).
func TestListEmptyStateFromGoroutines(t *testing.T) {
	calmOff()
	lb := newListBox().Expand()
	lv := emptyView().SetView(ViewLargeIcon)
	tt := tester(t, VBox(0).Add(lb, lv))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 40 {
			var e *EmptyStateWidget
			if i%2 == 0 {
				e = redState()
			}
			lb.SetEmptyState(e)
			lv.SetEmptyState(e)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 40 {
			k := fmt.Sprint(i)
			lb.AddItem(litem{k, k})
			lv.AddRow(vitem{id: k, name: k})
			lb.RemoveItems(k)
			lv.DeleteRows(k)
		}
	}()
	for range 40 {
		tt.Frame()
	}
	wg.Wait()
	lb.SetEmptyState(redState())
	tt.Frame()
	if !tt.HasText(listEmptyTitle) {
		t.Fatalf("after the goroutines, the empty list box shows %q", tt.Texts())
	}
}

// Review Focus 3: a ListView with groups and no rows draws no group header
// under the state, in every view.
func TestListViewGroupsEmptyState(t *testing.T) {
	calmOff()
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetGroupBy(func(v *vitem) string { return v.kind }).SetView(m.mode).SetEmptyState(redState()).Expand()
			tt := tester(t, lv)
			lv.Clear()
			tt.Frame()
			if !tt.HasText(listEmptyTitle) {
				t.Fatalf("an empty grouped view shows %q, want its empty state", tt.Texts())
			}
			for _, s := range tt.Texts() {
				if strings.Contains(s, "(") {
					t.Errorf("a group header %q shows with no rows", s)
				}
			}
		})
	}
}

// Review Focus 4: frames at rest of an empty list box with a state, and of an
// empty icon view with one, allocate no more than those of the same lists
// without (none, once built).
func TestListEmptyStateFrameAllocs(t *testing.T) {
	calmOff()
	mk := func(c Control) *ui.Tester {
		tt := ui.NewTester(content(c), 800, 600)
		for range 10 {
			tt.Frame()
		}
		return tt
	}
	for name, pair := range map[string][2]Control{
		"ListBox":   {newListBox().Expand(), newListBox().SetEmptyState(redState()).Expand()},
		"LargeIcon": {emptyView().SetView(ViewLargeIcon), emptyView().SetView(ViewLargeIcon).SetEmptyState(redState())},
	} {
		plain, with := mk(pair[0]), mk(pair[1])
		p, w := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
		t.Logf("%s: %v allocations a frame without the state, %v with", name, p, w)
		if w > p {
			t.Errorf("%s: a frame at rest with the empty state allocates %v, without %v", name, w, p)
		}
	}
}

// Review Focus 5: a ListBox given a state, then items before its first frame,
// never builds the state.
func TestListBoxEmptyStateNeverBuiltWithItems(t *testing.T) {
	calmOff()
	e := redState()
	lb := newListBox().SetEmptyState(e).AddItems([]litem{{"a", "Apple"}}).Expand()
	tt := tester(t, lb)
	tt.Frame()
	if tt.HasText(listEmptyTitle) {
		t.Fatal("the state shows over an item")
	}
	mu.Lock()
	built := e.base().el != (ui.Element{})
	mu.Unlock()
	if built {
		t.Fatal("the state was built though the list never was empty")
	}
}
