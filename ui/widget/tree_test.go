package widget

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// tnode is a node of the test trees.
type tnode struct{ id, name string }

func newTree() *TreeViewWidget[tnode, string] {
	return TreeView(func(n *tnode) string { return n.id }, func(n *tnode) string { return n.name })
}

// fileTree is Alpha (with Alpha one and Alpha two) and Beta (with Beta one).
func fileTree() *TreeViewWidget[tnode, string] {
	t := newTree().SetRoots([]tnode{{"A", "Alpha"}, {"B", "Beta"}})
	t.SetChildren("A", []tnode{{"A1", "Alpha one"}, {"A2", "Alpha two"}})
	t.SetChildren("B", []tnode{{"B1", "Beta one"}})
	return t
}

// lockHeld reports whether the package lock is held, by the caller.
func lockHeld() bool {
	if mu.TryLock() {
		mu.Unlock()
		return false
	}
	return true
}

// arrow returns where the arrow of the top-level row showing text is. Find
// gives the whole row, not its text: the row has 8 DIPs of padding, then the
// arrow, 16 wide.
func arrow(t *testing.T, tt *ui.Tester, text string) (x, y float32) {
	t.Helper()
	r := find(t, tt, text)
	return r.X + 16, r.Y + r.H/2
}

func shows(tt *ui.Tester, text string) bool {
	_, ok := tt.Find(text)
	return ok
}

func TestTreeViewShowsTheRootsAndOpensTheChildrenWithTheArrow(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	if !shows(tt, "Alpha") || !shows(tt, "Beta") || shows(tt, "Alpha one") {
		t.Fatalf("closed tree shows %q: want Alpha and Beta only", tt.Texts())
	}
	x, y := arrow(t, tt, "Alpha")
	tt.ClickAt(x, y)
	tt.Frame()
	if !shows(tt, "Alpha one") || !shows(tt, "Alpha two") || !tv.IsExpanded("A") {
		t.Fatalf("after the arrow click: %q, expanded %v", tt.Texts(), tv.Expanded())
	}
	a, a1, b := find(t, tt, "Alpha"), find(t, tt, "Alpha one"), find(t, tt, "Beta")
	if !(a.Y < a1.Y && a1.Y < b.Y) {
		t.Fatalf("Alpha %v, Alpha one %v, Beta %v: want the children below their parent", a, a1, b)
	}
	// find gives whole rows: the indent shows in where the text's ink starts,
	// right of the arrows.
	if pa, pc := inkStart(tt, a, 26), inkStart(tt, a1, 26); pc < pa+8 {
		t.Fatalf("the text of Alpha starts at x %d, that of its child at %d: want the child indented", pa, pc)
	}
}

// inkStart returns the first column from x0 where the row r has ink: a pixel
// that is not the page's.
func inkStart(tt *ui.Tester, r ui.Rect, x0 int) int {
	img, bg := tt.Image(), page(tt)
	for x := x0; x < img.Bounds().Dx(); x++ {
		for y := int(r.Y) + 2; y < int(r.Y+r.H)-2; y++ {
			if img.RGBAAt(x, y) != bg {
				return x
			}
		}
	}
	return -1
}

func TestTreeViewLazyNodeFiresOnExpandOnceAndShowsLoadingUntilSetChildren(t *testing.T) {
	expanded := make(chan string, 8)
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true).
		OnExpand(func(k string) { expanded <- k })
	tt := tester(t, tv)
	x, y := arrow(t, tt, "Alpha")
	tt.ClickAt(x, y)
	tt.Frame()
	if got := soon(t, expanded); got != "A" {
		t.Fatalf("OnExpand got %q", got)
	}
	if !shows(tt, "Loading…") {
		t.Fatalf("an open node without its children shows %q: want Loading…", tt.Texts())
	}
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}})
	tt.Frame()
	if shows(tt, "Loading…") || !shows(tt, "Alpha one") {
		t.Fatalf("after SetChildren: %q", tt.Texts())
	}
}

func TestTreeViewOnExpandFiresOnceWhileTheLoadIsPending(t *testing.T) {
	expanded, collapsed := make(chan string, 8), make(chan string, 8)
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true).
		OnExpand(func(k string) { expanded <- k }).OnCollapse(func(k string) { collapsed <- k })
	tt := tester(t, tv)
	toggle := func() {
		x, y := arrow(t, tt, "Alpha")
		tt.ClickAt(x, y)
		tt.Frame()
	}
	toggle() // open: the load starts
	soon(t, expanded)
	toggle() // close
	soon(t, collapsed)
	toggle() // open again, the load still pending
	<-posted()
	select {
	case k := <-expanded:
		t.Fatalf("OnExpand fired again for %q while the load was pending", k)
	default:
	}
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}})
	tt.Frame()
	toggle() // close
	soon(t, collapsed)
	toggle() // open: loaded now, so it reports the user opening it
	if got := soon(t, expanded); got != "A" {
		t.Fatalf("OnExpand after the load got %q", got)
	}
}

func TestTreeViewLoadingRowIsNeitherSelectableNorCheckable(t *testing.T) {
	selected := make(chan string, 8)
	checked := make(chan string, 8)
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true).CheckBoxes().
		OnSelect(func(k string, ok bool) { selected <- fmt.Sprint(k, ok) }).
		OnCheck(func(k string, on bool) { checked <- k })
	tv.Expand("A")
	tt := tester(t, tv)
	click(t, tt, "Loading…")
	<-posted()
	if _, ok := tv.Selected(); ok {
		t.Fatal("the Loading row was selected")
	}
	select {
	case v := <-selected:
		t.Fatalf("OnSelect fired for the Loading row: %q", v)
	case k := <-checked:
		t.Fatalf("OnCheck fired for the Loading row: %q", k)
	default:
	}
	if len(tv.Checked()) != 0 {
		t.Fatalf("Checked %v holds the Loading row", tv.Checked())
	}
}

// loadingTree is Alpha, a lazy node that is open and so shows its Loading row,
// and Lima, with events collected.
func loadingTree(events chan string) *TreeViewWidget[tnode, string] {
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}, {"L", "Lima"}}).SetHasChildren("A", true).
		OnSelect(func(k string, ok bool) { events <- fmt.Sprint("select ", k, " ", ok) }).
		OnExpand(func(k string) { events <- "expand " + k })
	tv.Expand("A")
	return tv
}

// The keys and typing never choose the Loading row, which would clear the
// choice: they move past it, or stay.
func TestTreeViewKeysAndTypingNeverChooseTheLoadingRow(t *testing.T) {
	selected := func(tv *TreeViewWidget[tnode, string]) string { k, _ := tv.Selected(); return k }
	drain := func(events chan string) (got []string) {
		<-posted()
		for {
			select {
			case e := <-events:
				got = append(got, e)
			default:
				return
			}
		}
	}
	setup := func(t *testing.T) (*TreeViewWidget[tnode, string], *ui.Tester, chan string) {
		events := make(chan string, 16)
		tv := loadingTree(events)
		tt := tester(t, tv)
		click(t, tt, "Alpha")
		drain(events)
		return tv, tt, events
	}
	t.Run("Down skips it", func(t *testing.T) {
		tv, tt, events := setup(t)
		tt.Key(0, ui.KeyDown)
		tt.Frame()
		if got := selected(tv); got != "L" {
			t.Fatalf("Down from Alpha selected %q, want Lima, past the Loading row", got)
		}
		if got := drain(events); !slices.Equal(got, []string{"select L true"}) {
			t.Fatalf("events %v: want one OnSelect for Lima, none with an empty key", got)
		}
	})
	t.Run("Up skips it", func(t *testing.T) {
		tv, tt, events := setup(t)
		click(t, tt, "Lima")
		drain(events)
		tt.Key(0, ui.KeyUp)
		tt.Frame()
		if got := selected(tv); got != "A" {
			t.Fatalf("Up from Lima selected %q, want Alpha", got)
		}
		if got := drain(events); !slices.Equal(got, []string{"select A true"}) {
			t.Fatalf("events %v", got)
		}
	})
	t.Run("Right stays on the node", func(t *testing.T) {
		tv, tt, events := setup(t)
		tt.Key(0, ui.KeyRight)
		tt.Key(0, ui.KeyRight)
		tt.Frame()
		if got := selected(tv); got != "A" {
			t.Fatalf("Right, Right selected %q, want Alpha", got)
		}
		if got := drain(events); len(got) != 0 {
			t.Fatalf("events %v: Right on an open node whose children load fires none", got)
		}
	})
	t.Run("End goes to the last real row", func(t *testing.T) {
		tv, tt, _ := setup(t)
		tt.Key(0, ui.KeyEnd)
		tt.Frame()
		if got := selected(tv); got != "L" {
			t.Fatalf("End selected %q, want Lima", got)
		}
	})
	t.Run("End when the Loading row is last", func(t *testing.T) {
		tv := newTree().SetRoots([]tnode{{"L", "Lima"}, {"A", "Alpha"}}).SetHasChildren("A", true)
		tv.Expand("A")
		tt := tester(t, tv)
		click(t, tt, "Lima")
		tt.Key(0, ui.KeyEnd)
		tt.Frame()
		if got := selected(tv); got != "A" {
			t.Fatalf("End selected %q, want Alpha, the last row that can be chosen", got)
		}
	})
	t.Run("typing L finds Lima, not Loading…", func(t *testing.T) {
		tv, tt, _ := setup(t)
		tt.Key(0, ui.KeyL)
		tt.Frame()
		if got := selected(tv); got != "L" {
			t.Fatalf("typing L selected %q, want Lima", got)
		}
	})
}

// quiet fails the test if an event is waiting.
func quiet(t *testing.T, events chan string, why string) {
	t.Helper()
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("%s fired %q", why, e)
	default:
	}
}

// SetHasChildren(k, true) on an open node that is not loaded and not pending
// is a request to load: OnExpand runs once. While the load is pending, more
// calls, the user closing and opening the node, and a refresh of the parent
// ask nothing (the load must not run twice). LoadFailed ends the pending
// load, and then SetHasChildren(k, true) asks again.
func TestTreeViewSetHasChildrenOnAnOpenNodeAsksToLoadOnce(t *testing.T) {
	events := make(chan string, 16)
	tv := loadingTree(events)
	tt := tester(t, tv)
	quiet(t, events, "a program's Expand")
	tv.SetHasChildren("A", true) // nothing pending: asks
	if got := soon(t, events); got != "expand A" {
		t.Fatalf("the request fired %q, want expand A", got)
	}
	tv.SetHasChildren("A", true) // pending: the load is on its way
	quiet(t, events, "SetHasChildren while the load was pending")
	// The docs' pattern: the app refreshes the parent, then says the node has
	// children.
	tv.SetRoots([]tnode{{"A", "Alpha"}, {"L", "Lima"}})
	tv.SetHasChildren("A", true)
	quiet(t, events, "a refresh of the roots and SetHasChildren while the load was pending")
	for range 2 { // closes and opens
		x, y := arrow(t, tt, "Alpha")
		tt.ClickAt(x, y)
		tt.Frame()
	}
	quiet(t, events, "closing and opening a node with a load pending")
	tv.LoadFailed("A")
	tt.Frame()
	quiet(t, events, "LoadFailed")
	if !tv.IsExpanded("A") || !shows(tt, "Loading…") {
		t.Fatalf("LoadFailed left Expanded %v, shows %q: want the node open on its Loading row", tv.Expanded(), tt.Texts())
	}
	tv.SetHasChildren("A", true)
	if got := soon(t, events); got != "expand A" {
		t.Fatalf("the retry after LoadFailed fired %q, want expand A", got)
	}
}

// After LoadFailed the user can retry by closing and opening the node.
func TestTreeViewLoadFailedLetsTheUserRetryByReopening(t *testing.T) {
	expanded := make(chan string, 8)
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true).
		OnExpand(func(k string) { expanded <- k })
	tt := tester(t, tv)
	toggle := func() {
		x, y := arrow(t, tt, "Alpha")
		tt.ClickAt(x, y)
		tt.Frame()
	}
	toggle() // open: asks
	soon(t, expanded)
	toggle() // close
	toggle() // open again, the load still pending: asks nothing
	<-posted()
	if len(expanded) != 0 {
		t.Fatal("reopening a node with a load pending asked again")
	}
	tv.LoadFailed("A")
	toggle() // close
	toggle() // open after the failure
	if got := soon(t, expanded); got != "A" {
		t.Fatalf("OnExpand after LoadFailed got %q", got)
	}
}

func TestTreeViewLoadFailedOnAnUnknownOrLoadedNodeDoesNothing(t *testing.T) {
	events := make(chan string, 8)
	tv := loadingTree(events)
	tv.LoadFailed("zz")
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}})
	tv.LoadFailed("A")
	if !tv.IsLoaded("A") {
		t.Fatal("LoadFailed unloaded a loaded node")
	}
	tester(t, tv)
	quiet(t, events, "LoadFailed")
}

// SetHasChildren(k, false) on an open node whose children are not loaded
// makes it a leaf: it is not open, so a later true shows no Loading row
// without an event.
func TestTreeViewSetHasChildrenFalseClosesAnOpenNodeThatIsNotLoaded(t *testing.T) {
	events := make(chan string, 16)
	tv := loadingTree(events)
	tt := tester(t, tv)
	tv.SetHasChildren("A", false)
	tt.Frame()
	if tv.IsExpanded("A") || len(tv.Expanded()) != 0 {
		t.Fatalf("Expanded %v: a leaf is not open", tv.Expanded())
	}
	if shows(tt, "Loading…") {
		t.Fatalf("shows %q: a leaf shows no Loading row", tt.Texts())
	}
	tv.SetHasChildren("A", true)
	tt.Frame()
	if shows(tt, "Loading…") {
		t.Fatalf("shows %q: the node is closed, so no Loading row", tt.Texts())
	}
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("closing and making it lazy again fired %q", e)
	default:
	}
}

func TestTreeViewSetLoadingTextReplacesLoading(t *testing.T) {
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true).SetLoadingText("Đang tải…")
	tv.Expand("A")
	tt := tester(t, tv)
	if !shows(tt, "Đang tải…") || shows(tt, "Loading…") {
		t.Fatalf("shows %q: want the app's loading text", tt.Texts())
	}
}

func TestTreeViewProgrammaticChangesFireNoUserEvents(t *testing.T) {
	events := make(chan string, 16)
	tv := fileTree().CheckBoxes().
		OnExpand(func(k string) { events <- "expand " + k }).
		OnCollapse(func(k string) { events <- "collapse " + k }).
		OnSelect(func(k string, ok bool) { events <- "select " + k }).
		OnCheck(func(k string, on bool) { events <- "check " + k })
	tt := tester(t, tv)
	tv.Expand("A")
	tv.SetSelected("A2")
	tv.SetChecked([]string{"B", "A1"}, true)
	tv.Collapse("A")
	tt.Frame()
	tt.Frame()
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("a program's change fired %q", e)
	default:
	}
}

func TestTreeViewExpandedAndCheckedComeInTreeOrder(t *testing.T) {
	tv := fileTree()
	tv.Expand("B", "A")
	tv.CheckBoxes().SetChecked([]string{"B1", "A2", "A"}, true)
	if got := tv.Expanded(); !slices.Equal(got, []string{"A", "B"}) {
		t.Errorf("Expanded %v, want [A B]", got)
	}
	if got := tv.Checked(); !slices.Equal(got, []string{"A", "A2", "B1"}) {
		t.Errorf("Checked %v, want [A A2 B1]", got)
	}
	if !tv.IsChecked("A2") || tv.IsChecked("A1") {
		t.Errorf("IsChecked A2 %v, A1 %v", tv.IsChecked("A2"), tv.IsChecked("A1"))
	}
}

func TestTreeViewClickSelectsByKeyAndTheChoiceFollowsTheItem(t *testing.T) {
	selected := make(chan string, 8)
	tv := fileTree().OnSelect(func(k string, ok bool) { selected <- fmt.Sprint(k, ok) })
	tt := tester(t, tv)
	click(t, tt, "Beta")
	if got := soon(t, selected); got != "Btrue" {
		t.Fatalf("OnSelect got %q", got)
	}
	if k, ok := tv.Selected(); !ok || k != "B" {
		t.Fatalf("Selected %q, %v", k, ok)
	}
	// Opening Alpha moves Beta down the list; it stays the one selected.
	tv.Expand("A")
	tt.Frame()
	tt.Frame()
	if k, ok := tv.Selected(); !ok || k != "B" {
		t.Fatalf("after opening above it: Selected %q, %v", k, ok)
	}
	<-posted()
	select {
	case e := <-selected:
		t.Fatalf("opening a node above the selection fired OnSelect %q", e)
	default:
	}
}

// One node is chosen: a Cmd-click, Shift-click or Cmd+A that would choose
// several leaves the newest.
func TestTreeViewChoosesOneNodeWhateverTheModifiers(t *testing.T) {
	selected := make(chan string, 8)
	tv := fileTree().OnSelect(func(k string, ok bool) { selected <- fmt.Sprint(k, ok) })
	tt := tester(t, tv)
	click(t, tt, "Alpha")
	if err := tt.ClickWith(ui.Cmd, "Beta"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if k, _ := tv.Selected(); k != "B" {
		t.Fatalf("Cmd-click added Beta to Alpha: Selected %q", k)
	}
	if err := tt.ClickWith(ui.Shift, "Alpha"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if k, _ := tv.Selected(); k != "A" {
		t.Fatalf("Shift-click chose a range: Selected %q, want the row clicked", k)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Frame()
	if _, ok := tv.Selected(); !ok {
		t.Fatal("Cmd+A left no node chosen")
	}
	if n := tv.chosen.value.Len(); n != 1 {
		t.Fatalf("%d nodes chosen, want one", n)
	}
}

func TestTreeViewSetSelectedOpensTheAncestors(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	tv.SetSelected("A2")
	tt.Frame()
	tt.Frame()
	if !shows(tt, "Alpha two") || !tv.IsExpanded("A") {
		t.Fatalf("shows %q, expanded %v: want Alpha open so that Alpha two shows", tt.Texts(), tv.Expanded())
	}
	if k, ok := tv.Selected(); !ok || k != "A2" {
		t.Fatalf("Selected %q, %v", k, ok)
	}
	tv.ClearSelection()
	if _, ok := tv.Selected(); ok {
		t.Fatal("ClearSelection left a selection")
	}
}

func TestTreeViewCheckBoxesReportTheUsersChecks(t *testing.T) {
	checked := make(chan string, 8)
	tv := fileTree().CheckBoxes().OnCheck(func(k string, on bool) { checked <- fmt.Sprint(k, on) })
	tt := tester(t, tv)
	r := find(t, tt, "Beta")
	tt.ClickAt(r.X+38, r.Y+r.H/2) // the box, after the arrow and its gap
	tt.Frame()
	if got := soon(t, checked); got != "Btrue" {
		t.Fatalf("OnCheck got %q (Checked %v)", got, tv.Checked())
	}
	if !tv.IsChecked("B") || !slices.Equal(tv.Checked(), []string{"B"}) {
		t.Fatalf("Checked %v", tv.Checked())
	}
	if k, ok := tv.Selected(); ok {
		t.Errorf("checking selected %q", k)
	}
}

func TestTreeViewDuplicateKeysPanicWithTheKey(t *testing.T) {
	tv := fileTree()
	for name, fn := range map[string]func(){
		"a repeat in SetRoots":       func() { tv.SetRoots([]tnode{{"X", "x"}, {"X", "again"}}) },
		"a root already a child":     func() { tv.SetRoots([]tnode{{"A1", "Alpha one as a root"}}) },
		"a child of two parents":     func() { tv.SetChildren("B", []tnode{{"A1", "Alpha one again"}}) },
		"a repeat in SetChildren":    func() { tv.SetChildren("B", []tnode{{"X", "x"}, {"X", "x"}}) },
		"a child that is its parent": func() { tv.SetChildren("B", []tnode{{"B", "Beta"}}) },
	} {
		func() {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("%s: no panic", name)
					return
				}
				if msg := fmt.Sprint(v); !strings.Contains(msg, "duplicate key") || !strings.Contains(msg, "TreeView") {
					t.Errorf("%s: panic %q: want it to name the TreeView and the duplicate key", name, msg)
				}
			}()
			fn()
		}()
		if !mu.TryLock() {
			t.Fatalf("%s left the package lock held", name)
		}
		mu.Unlock()
	}
	// Nothing changed: a panic checks before it changes.
	if got := tv.Children("B"); !slices.Equal(got, []string{"B1"}) {
		t.Errorf("a panic changed Beta's children: %v", got)
	}
	if _, ok := tv.Node("A"); !ok {
		t.Error("a panic changed the roots")
	}
}

func TestTreeViewSettingTheSameKeysAgainIsNotADuplicate(t *testing.T) {
	tv := fileTree()
	tv.SetRoots([]tnode{{"A", "Alpha!"}, {"B", "Beta"}})
	tv.SetChildren("A", []tnode{{"A2", "Alpha two"}, {"A1", "Alpha one"}})
	if n, _ := tv.Node("A"); n.name != "Alpha!" {
		t.Errorf("SetRoots did not update the data of Alpha: %v", n)
	}
	if got := tv.Children("A"); !slices.Equal(got, []string{"A2", "A1"}) {
		t.Errorf("Children %v, want the new order", got)
	}
}

func TestTreeViewUnknownKeysDoNothing(t *testing.T) {
	events := make(chan string, 4)
	tv := fileTree().CheckBoxes().OnCheck(func(k string, on bool) { events <- k })
	tv.SetChildren("nope", []tnode{{"N", "n"}})
	tv.SetHasChildren("nope", true)
	tv.UpdateNode("nope", func(n *tnode) { n.name = "changed" })
	tv.Expand("nope")
	tv.Collapse("nope")
	tv.SetChecked([]string{"nope"}, true)
	tv.SetSelected("nope")
	tv.RemoveNodes("nope")
	if _, ok := tv.Node("N"); ok {
		t.Error("SetChildren of an unknown key added a node")
	}
	if len(tv.Expanded()) != 0 || len(tv.Checked()) != 0 {
		t.Errorf("Expanded %v, Checked %v", tv.Expanded(), tv.Checked())
	}
	if _, ok := tv.Selected(); ok {
		t.Error("SetSelected of an unknown key selected")
	}
	tester(t, tv)
	<-posted()
	select {
	case k := <-events:
		t.Errorf("OnCheck fired for %q", k)
	default:
	}
}

func TestTreeViewRemoveNodesClearsTheSubtreeAndFiresNothing(t *testing.T) {
	events := make(chan string, 16)
	tv := fileTree().CheckBoxes().
		OnCollapse(func(k string) { events <- "collapse " + k }).
		OnSelect(func(k string, ok bool) { events <- "select " + k }).
		OnCheck(func(k string, on bool) { events <- "check " + k })
	tv.Expand("A")
	tv.SetSelected("A2")
	tv.SetChecked([]string{"A", "A1", "B"}, true)
	tt := tester(t, tv)
	tv.RemoveNodes("A")
	tt.Frame()
	tt.Frame()
	if shows(tt, "Alpha") || shows(tt, "Alpha one") {
		t.Fatalf("shows %q after removing Alpha", tt.Texts())
	}
	if got := tv.Expanded(); len(got) != 0 {
		t.Errorf("Expanded %v holds the removed subtree", got)
	}
	if got := tv.Checked(); !slices.Equal(got, []string{"B"}) {
		t.Errorf("Checked %v, want only B", got)
	}
	if k, ok := tv.Selected(); ok {
		t.Errorf("Selected %q after its node was removed", k)
	}
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("RemoveNodes fired %q", e)
	default:
	}
	// A key added again starts clean.
	tv.SetRoots([]tnode{{"A", "Alpha"}, {"B", "Beta"}})
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}, {"A2", "Alpha two"}})
	tt.Frame()
	tt.Frame()
	if tv.IsExpanded("A") || tv.IsChecked("A") || tv.IsChecked("A1") {
		t.Errorf("re-added A is expanded %v, checked %v / %v", tv.IsExpanded("A"), tv.IsChecked("A"), tv.IsChecked("A1"))
	}
	if _, ok := tv.Selected(); ok {
		t.Error("a re-added key came back selected")
	}
}

func TestTreeViewSetChildrenOnAnOpenNodeKeepsWhatSurvives(t *testing.T) {
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}})
	tv.SetChildren("A", []tnode{{"A1", "One"}, {"A2", "Two"}, {"A3", "Three"}})
	tv.SetChildren("A1", []tnode{{"A11", "One one"}})
	tv.Expand("A", "A1")
	tv.SetSelected("A1")
	tv.CheckBoxes().SetChecked([]string{"A1", "A3"}, true)
	tt := tester(t, tv)
	// A reload: One survives, with its children; Three is gone; Four is new.
	tv.SetChildren("A", []tnode{{"A1", "One!"}, {"A4", "Four"}})
	tt.Frame()
	tt.Frame()
	if !tv.IsExpanded("A") || !tv.IsExpanded("A1") {
		t.Errorf("Expanded %v: want A and A1 kept", tv.Expanded())
	}
	if k, ok := tv.Selected(); !ok || k != "A1" {
		t.Errorf("Selected %q, %v: want A1 kept", k, ok)
	}
	if got := tv.Checked(); !slices.Equal(got, []string{"A1"}) {
		t.Errorf("Checked %v: want A1 kept and the gone A3 dropped", got)
	}
	if !shows(tt, "One!") || !shows(tt, "One one") || !shows(tt, "Four") || shows(tt, "Three") {
		t.Errorf("shows %q", tt.Texts())
	}
	// A reload that drops the selected node clears the selection.
	tv.SetChildren("A", []tnode{{"A4", "Four"}})
	if _, ok := tv.Selected(); ok {
		t.Error("the selection stayed on a node a reload removed")
	}
}

func TestTreeViewCallbacksRunUnderThePackageLock(t *testing.T) {
	var keyLocked, textLocked, updateLocked bool
	tv := TreeView(func(n *tnode) string { keyLocked = keyLocked || lockHeld(); return n.id },
		func(n *tnode) string { textLocked = textLocked || lockHeld(); return n.name })
	tv.SetRoots([]tnode{{"A", "Alpha"}})
	tv.UpdateNode("A", func(n *tnode) { updateLocked = lockHeld(); n.name = "Alpha!" })
	tester(t, tv)
	if !keyLocked || !textLocked || !updateLocked {
		t.Errorf("under the lock: key %v, text %v, UpdateNode %v: want all true, as the docs say", keyLocked, textLocked, updateLocked)
	}
}

func TestTreeViewUpdateNodeShowsTheChange(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	tv.UpdateNode("B", func(n *tnode) { n.name = "Gamma" })
	tt.Frame()
	if !shows(tt, "Gamma") || shows(tt, "Beta") {
		t.Fatalf("shows %q", tt.Texts())
	}
	if n, ok := tv.Node("B"); !ok || n.name != "Gamma" {
		t.Fatalf("Node B %v, %v", n, ok)
	}
	if p, ok := tv.Parent("A1"); !ok || p != "A" {
		t.Fatalf("Parent of A1 %q, %v", p, ok)
	}
	if _, ok := tv.Parent("A"); ok {
		t.Fatal("a root has a parent")
	}
}

func TestTreeViewKeyboard(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	click(t, tt, "Alpha")
	tt.Key(0, ui.KeyRight) // opens Alpha
	tt.Frame()
	if !tv.IsExpanded("A") || !shows(tt, "Alpha one") {
		t.Fatalf("Right did not open Alpha: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyRight) // goes to its first child
	tt.Frame()
	if k, _ := tv.Selected(); k != "A1" {
		t.Fatalf("Right on an open node selected %q, want its first child A1", k)
	}
	tt.Key(0, ui.KeyLeft) // goes to the parent: A1 is closed or a leaf
	tt.Frame()
	if k, _ := tv.Selected(); k != "A" {
		t.Fatalf("Left on a leaf selected %q, want its parent A", k)
	}
	tt.Key(0, ui.KeyLeft) // closes Alpha
	tt.Frame()
	if tv.IsExpanded("A") || shows(tt, "Alpha one") {
		t.Fatalf("Left did not close Alpha: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	if k, _ := tv.Selected(); k != "B" {
		t.Fatalf("End selected %q, want the last row B", k)
	}
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	if k, _ := tv.Selected(); k != "A" {
		t.Fatalf("Home selected %q, want the first row A", k)
	}
}

func TestTreeViewUpAndDownMoveTheChoice(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	click(t, tt, "Alpha")
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if k, _ := tv.Selected(); k != "B" {
		t.Fatalf("Down selected %q, want the next row B", k)
	}
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	if k, _ := tv.Selected(); k != "A" {
		t.Fatalf("Up selected %q, want A", k)
	}
}

// An Alt-click on an arrow opens all inside (ui's), so OnExpand runs for the
// node and for each inside it that has children to load, once.
func TestTreeViewAltClickOpensAllInsideAndAsksForEachLoad(t *testing.T) {
	expanded := make(chan string, 8)
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}})
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}, {"A2", "Alpha two"}})
	tv.SetHasChildren("A1", true)
	tv.OnExpand(func(k string) { expanded <- k })
	tt := tester(t, tv)
	x, y := arrow(t, tt, "Alpha")
	tt.ClickAtWith(ui.Alt, x, y)
	tt.Frame()
	got := []string{soon(t, expanded), soon(t, expanded)}
	slices.Sort(got)
	if !slices.Equal(got, []string{"A", "A1"}) {
		t.Fatalf("OnExpand ran for %v, want A and A1", got)
	}
	if !shows(tt, "Loading…") {
		t.Errorf("shows %q: want the Loading row of A1", tt.Texts())
	}
}

// The cascade of the docs: an app checks the children of a node the user
// checked, in OnCheck.
func TestTreeViewAppCascadesChecksInOnCheck(t *testing.T) {
	done := make(chan struct{}, 4)
	var tv *TreeViewWidget[tnode, string]
	tv = fileTree().CheckBoxes().OnCheck(func(k string, on bool) {
		tv.SetChecked(tv.Children(k), on)
		done <- struct{}{}
	})
	tv.Expand("A")
	tt := tester(t, tv)
	r := find(t, tt, "Alpha")
	tt.ClickAt(r.X+38, r.Y+r.H/2)
	tt.Frame()
	soon(t, done)
	if got := tv.Checked(); !slices.Equal(got, []string{"A", "A1", "A2"}) {
		t.Fatalf("Checked %v, want Alpha and its children", got)
	}
}

func TestTreeViewTypingFindsARowByItsText(t *testing.T) {
	tv := fileTree()
	tt := tester(t, tv)
	click(t, tt, "Alpha")
	tt.Key(0, ui.KeyB)
	tt.Key(0, ui.KeyE)
	tt.Frame()
	if k, _ := tv.Selected(); k != "B" {
		t.Fatalf("typing be selected %q, want B", k)
	}
}

func TestTreeViewEnterAndDoubleClickActivate(t *testing.T) {
	activated := make(chan string, 4)
	tv := fileTree().OnActivate(func(k string) { activated <- k })
	tt := tester(t, tv)
	click(t, tt, "Beta")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, activated); got != "B" {
		t.Fatalf("Enter activated %q", got)
	}
}

// clickCost builds a tree of a thousand roots with kids children each,
// shown with its first root open, and returns the objects and bytes a click
// on a row allocates, as a frame builds the view.
func clickCost(t *testing.T, kids int) (allocs float64, bytes int64) {
	t.Helper()
	const roots = 1000
	tv := newTree()
	top := make([]tnode, roots)
	for i := range top {
		top[i] = tnode{fmt.Sprint("r", i), fmt.Sprint("Root ", i)}
	}
	tv.SetRoots(top)
	for i := range top {
		ch := make([]tnode, kids)
		for j := range ch {
			ch[j] = tnode{fmt.Sprint("r", i, "c", j), fmt.Sprint("Child ", i, ".", j)}
		}
		tv.SetChildren(top[i].id, ch)
	}
	tt := tester(t, tv)
	click(t, tt, "Root 1")
	pick := func(name string) func() {
		return func() {
			if err := tt.Click(name); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
		}
	}
	a, b := pick("Root 2"), pick("Root 3")
	a() // warm up
	b()
	var n int
	allocs = allocrun.Least(20, func() {
		if n++; n%2 == 0 {
			a()
		} else {
			b()
		}
	})
	res := testing.Benchmark(func(bb *testing.B) {
		for i := 0; i < bb.N; i++ {
			if i%2 == 0 {
				a()
			} else {
				b()
			}
		}
	})
	return allocs, res.AllocedBytesPerOp()
}

// A click on a tree of a million nodes (a thousand roots of a thousand
// children) costs what a click on the same roots with a child each does:
// the rows shown are the same, and nothing is cloned per click, whatever the
// size of the tree.
func TestTreeViewClickAllocationsDoNotGrowWithTheTree(t *testing.T) {
	smallAllocs, smallBytes := clickCost(t, 1)
	bigAllocs, bigBytes := clickCost(t, 1000)
	t.Logf("a click on 2000 nodes: %.0f allocs, %d B/op; on 1001000 nodes: %.0f allocs, %d B/op",
		smallAllocs, smallBytes, bigAllocs, bigBytes)
	// A clone of the million would be a million objects and megabytes.
	if bigAllocs > smallAllocs*1.1+10 {
		t.Errorf("a click allocated %.0f objects on a million nodes, %.0f on two thousand: want the same", bigAllocs, smallAllocs)
	}
	if bigBytes > smallBytes*11/10+1<<10 {
		t.Errorf("a click allocated %d bytes on a million nodes, %d on two thousand: want the same", bigBytes, smallBytes)
	}
	if bigBytes > 512<<10 {
		t.Errorf("a click allocated %d bytes: want far less than a clone of the tree (MB)", bigBytes)
	}
}

func TestTreeViewImageShowsBeforeTheText(t *testing.T) {
	calls, locked := 0, true
	tv := fileTree().SetImage(func(n *tnode) *ui.Bitmap { calls++; locked = locked && lockHeld(); return nil })
	tester(t, tv)
	if calls == 0 {
		t.Fatal("SetImage's function was not called for the rows shown")
	}
	if !locked {
		t.Fatal("SetImage's function ran without the package lock, which the docs say it holds")
	}
}
