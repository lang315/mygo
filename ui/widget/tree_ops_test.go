package widget

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

func byName(a, b *tnode) int { return strings.Compare(a.name, b.name) }

func named(s string) func(*tnode) bool {
	return func(n *tnode) bool { return strings.Contains(n.name, s) }
}

// rowsInOrder fails unless the rows showing names are shown one below the
// other, in that order.
func rowsInOrder(t *testing.T, tt *ui.Tester, names ...string) {
	t.Helper()
	prev := float32(-1)
	for _, s := range names {
		r, ok := tt.Find(s)
		if !ok {
			t.Fatalf("%q does not show: %q", s, tt.Texts())
		}
		if r.Y <= prev {
			t.Fatalf("%q is not below the row before it, in %v: shows %q", s, names, tt.Texts())
		}
		prev = r.Y
	}
}

// treeRows returns the names of the tree's rows, top to bottom.
func treeRows(tt *ui.Tester) []string {
	var names []string
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTreeItem {
			names = append(names, n.Label)
		}
	}
	return names
}

// hidden fails if any of names shows.
func hidden(t *testing.T, tt *ui.Tester, names ...string) {
	t.Helper()
	for _, s := range names {
		if shows(tt, s) {
			t.Fatalf("%q shows: %q", s, tt.Texts())
		}
	}
}

// sortTree is Cherry, Apple (with Zed and Ann, open) and Banana, in that order.
func sortTree() *TreeViewWidget[tnode, string] {
	tv := newTree().SetRoots([]tnode{{"C", "Cherry"}, {"A", "Apple"}, {"B", "Banana"}})
	tv.SetChildren("A", []tnode{{"A2", "Zed"}, {"A1", "Ann"}})
	return tv.Expand("A")
}

func TestTreeSort(t *testing.T) {
	tv := sortTree()
	tt := tester(t, tv)
	rowsInOrder(t, tt, "Cherry", "Apple", "Zed", "Ann", "Banana")
	tv.SetSort(byName)
	tt.Frame()
	rowsInOrder(t, tt, "Apple", "Ann", "Zed", "Banana", "Cherry")
	// After SetRoots, SetChildren and UpdateNode.
	tv.SetRoots([]tnode{{"C", "Cherry"}, {"A", "Apple"}, {"B", "Banana"}, {"V", "Avocado"}})
	tt.Frame()
	rowsInOrder(t, tt, "Apple", "Ann", "Zed", "Avocado", "Banana", "Cherry")
	tv.SetChildren("A", []tnode{{"A2", "Zed"}, {"A1", "Ann"}, {"A3", "Bob"}})
	tt.Frame()
	rowsInOrder(t, tt, "Apple", "Ann", "Bob", "Zed", "Avocado")
	tv.UpdateNode("C", func(n *tnode) { n.name = "Aardvark" })
	tt.Frame()
	rowsInOrder(t, tt, "Aardvark", "Apple", "Ann", "Bob", "Zed", "Avocado", "Banana")
	// The order shown only: Children keeps the order given.
	if got := tv.Children("A"); !slices.Equal(got, []string{"A2", "A1", "A3"}) {
		t.Errorf("Children(A) %v under a sort: want the order given", got)
	}
	tv.SetSort(nil)
	tt.Frame()
	rowsInOrder(t, tt, "Aardvark", "Apple", "Zed", "Ann", "Bob", "Banana", "Avocado")
	// RemoveNodes under a sort: the node goes from the order shown.
	tv.SetSort(byName)
	tt.Frame()
	tv.RemoveNodes("A3", "V")
	tt.Frame()
	if got, want := treeRows(tt), []string{"Aardvark", "Apple", "Ann", "Zed", "Banana"}; !slices.Equal(got, want) {
		t.Fatalf("rows %q after RemoveNodes, want %q", got, want)
	}

	t.Run("columns", func(t *testing.T) {
		tv := sortTree().SetColumns("Name", Column[tnode]{Title: "Id", Text: func(n *tnode) string { return "#" + n.id }})
		tt := tester(t, tv)
		tv.SetSort(byName)
		tt.Frame()
		rowsInOrder(t, tt, "Apple", "Ann", "Zed", "Banana", "Cherry")
	})
}

// filterTree is fileTree with Needle below Alpha one, and Beta open.
func filterTree() *TreeViewWidget[tnode, string] {
	tv := fileTree()
	tv.SetChildren("A1", []tnode{{"N", "Needle"}})
	return tv.Expand("B")
}

func TestTreeFilter(t *testing.T) {
	events := make(chan string, 16)
	tv := filterTree().CheckBoxes().
		OnExpand(func(k string) { events <- "expand " + k }).
		OnCollapse(func(k string) { events <- "collapse " + k }).
		OnSelect(func(k string, ok bool) { events <- "select " + k }).
		OnCheck(func(k string, on bool) { events <- "check " + k })
	tt := tester(t, tv)
	rowsInOrder(t, tt, "Alpha", "Beta", "Beta one")
	tv.SetFilter(named("Needle"))
	tt.Frame()
	rowsInOrder(t, tt, "Alpha", "Alpha one", "Needle")
	hidden(t, tt, "Alpha two", "Beta", "Beta one")
	if !tv.IsExpanded("A") || !tv.IsExpanded("A1") {
		t.Fatalf("expanded %v under the filter: want the ancestors of Needle", tv.Expanded())
	}
	// A filter changed keeps the expansion the user had before the first.
	tv.SetFilter(named("two"))
	tt.Frame()
	rowsInOrder(t, tt, "Alpha", "Alpha two")
	hidden(t, tt, "Alpha one", "Needle", "Beta")
	if got := tv.Expanded(); !slices.Equal(got, []string{"A", "B"}) {
		t.Fatalf("expanded %v under the second filter: want the user's [B] and Alpha", got)
	}
	// A node passing shows without its children that do not.
	tv.SetFilter(func(n *tnode) bool { return n.name == "Beta" })
	tt.Frame()
	rowsInOrder(t, tt, "Beta")
	hidden(t, tt, "Alpha", "Beta one")
	// Nodes set while the filter is: a passing one shows, with its ancestors open.
	tv.SetFilter(named("Needle"))
	tv.SetChildren("A2", []tnode{{"N2", "Needle two"}}) // A2 was a leaf, closed
	tt.Frame()
	rowsInOrder(t, tt, "Alpha", "Alpha one", "Needle", "Alpha two", "Needle two")
	hidden(t, tt, "Beta", "Beta one")
	tv.Collapse("B") // closed while hidden: open again once the filter goes
	tv.SetFilter(nil)
	tt.Frame()
	if got := tv.Expanded(); !slices.Equal(got, []string{"B"}) {
		t.Fatalf("expanded %v once the filter is cleared: want the user's [B]", got)
	}
	rowsInOrder(t, tt, "Alpha", "Beta", "Beta one")
	hidden(t, tt, "Alpha one", "Needle")
	quiet(t, events, "SetFilter")
}

// Filtering never asks for children: OnExpand does not fire, and a node not
// loaded stays so.
func TestTreeFilterNoLoad(t *testing.T) {
	events := make(chan string, 16)
	tv := newTree().SetRoots([]tnode{{"L", "Lazy"}, {"M", "Lazy closed"}, {"A", "Alpha"}}).
		SetHasChildren("L", true).SetHasChildren("M", true).
		OnExpand(func(k string) { events <- "expand " + k })
	tv.SetChildren("A", []tnode{{"A1", "Alpha one"}})
	tv.Expand("L")
	tt := tester(t, tv)
	for _, keep := range []func(*tnode) bool{named("Lazy"), named("one"), named("nothing")} {
		tv.SetFilter(keep)
		tt.Frame()
		tv.SetFilter(nil)
		tt.Frame()
	}
	quiet(t, events, "filtering")
	if tv.IsLoaded("L") || tv.IsLoaded("M") {
		t.Fatal("filtering loaded a lazy node")
	}
	if !tv.IsExpanded("L") || tv.IsExpanded("M") {
		t.Fatalf("expanded %v: want L only, as before", tv.Expanded())
	}
}

func TestTreeSelectionKeptBySortAndFilter(t *testing.T) {
	tv := fileTree().CheckBoxes()
	tt := tester(t, tv)
	tv.SetSelected("A2").SetChecked([]string{"B1", "A1"}, true)
	tt.Frame()
	kept := func(when string) {
		t.Helper()
		if k, ok := tv.Selected(); !ok || k != "A2" {
			t.Fatalf("%s: Selected %q %v, want A2", when, k, ok)
		}
		if got := tv.Checked(); !slices.Equal(got, []string{"A1", "B1"}) {
			t.Fatalf("%s: Checked %v, want [A1 B1]", when, got)
		}
	}
	tv.SetSort(func(a, b *tnode) int { return -byName(a, b) })
	tt.Frame()
	rowsInOrder(t, tt, "Beta", "Alpha", "Alpha two", "Alpha one")
	kept("sorted")
	tv.SetFilter(named("Beta"))
	tt.Frame()
	tt.Frame()
	kept("filtered")
	tv.SetFilter(nil)
	tt.Frame()
	tt.Frame()
	kept("filter cleared")
	tv.SetSort(nil)
	tt.Frame()
	kept("sort cleared")
	// A click chooses the node clicked, wherever the sort put it.
	tv.SetSort(func(a, b *tnode) int { return -byName(a, b) })
	tt.Frame()
	click(t, tt, "Alpha one")
	if k, _ := tv.Selected(); k != "A1" {
		t.Fatalf("a click on Alpha one, sorted, selected %q", k)
	}
}

// nodeTextAt returns a point on the text of the row showing name, after its
// arrow.
func nodeTextAt(t *testing.T, tt *ui.Tester, name string) (x, y float32) {
	t.Helper()
	r := find(t, tt, name)
	x0 := inkStart(tt, r, int(r.X)+30)
	if x0 < 0 {
		t.Fatalf("no ink on the row of %q", name)
	}
	return float32(x0) + 3, r.Y + r.H/2
}

// editTree is fileTree with label edit on: OnAfterLabelEdit sends the key and
// the text, renames the node when *keep, and returns *keep.
func editTree(edits chan string, keep *bool) *TreeViewWidget[tnode, string] {
	var tv *TreeViewWidget[tnode, string]
	var mu sync.Mutex
	tv = fileTree().SetLabelEdit(true).OnAfterLabelEdit(func(k, text string) bool {
		mu.Lock()
		ok := *keep
		mu.Unlock()
		if ok {
			tv.UpdateNode(k, func(n *tnode) { n.name = text })
		}
		edits <- k + " " + text
		return ok
	})
	return tv
}

func quietEdits(t *testing.T, edits chan string, why string) {
	t.Helper()
	<-posted()
	select {
	case e := <-edits:
		t.Fatalf("%s: OnAfterLabelEdit ran with %q", why, e)
	default:
	}
}

func TestTreeLabelEdit(t *testing.T) {
	edits := make(chan string, 8)
	keep := true
	tv := editTree(edits, &keep)
	tt := tester(t, tv)
	click(t, tt, "Beta")
	// F2 opens the editor on the selected node, its text all selected.
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	// The row keeps its name (its label), and the input takes the place of
	// its text.
	if n := textCount(tt, "Beta"); n != 1 {
		t.Fatalf("after F2 Beta shows %d times, in %q: want the row's label only", n, tt.Texts())
	}
	tt.Type("Gamma")
	tt.Frame()
	quietEdits(t, edits, "typing")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, edits); got != "B Gamma" {
		t.Fatalf("Enter: OnAfterLabelEdit got %q, want B Gamma", got)
	}
	tt.Frame()
	if !shows(tt, "Gamma") || shows(tt, "Beta") {
		t.Fatalf("after true and UpdateNode: %q, want Gamma", tt.Texts())
	}
	if n, _ := tv.Node("B"); n.name != "Gamma" {
		t.Fatalf("node B is %q", n.name)
	}
	quietEdits(t, edits, "a second frame")
	if !tt.Focused("Gamma") {
		t.Error("the tree did not take the focus back")
	}
	// False: the text goes back.
	keep = false
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Type("Delta")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, edits); got != "B Delta" {
		t.Fatalf("Enter: OnAfterLabelEdit got %q, want B Delta", got)
	}
	<-posted()
	tt.Frame()
	if !shows(tt, "Gamma") || shows(tt, "Delta") {
		t.Fatalf("after false: %q, want Gamma back", tt.Texts())
	}
	// Escape: the text goes back, and nothing is called.
	keep = true
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Type("Epsilon")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	quietEdits(t, edits, "Escape")
	tt.Frame()
	if !shows(tt, "Gamma") || shows(tt, "Epsilon") {
		t.Fatalf("after Escape: %q, want Gamma", tt.Texts())
	}
	// Enter with the text unchanged calls nothing.
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	quietEdits(t, edits, "Enter on the text unchanged")

	t.Run("refused without the func", func(t *testing.T) {
		tv := fileTree().SetLabelEdit(true)
		tt := tester(t, tv)
		click(t, tt, "Beta")
		tt.Key(0, ui.KeyF2)
		tt.Frame()
		if n := textCount(tt, "Beta"); n != 2 {
			t.Fatalf("without OnAfterLabelEdit F2 left Beta shown %d times: want no input, its label and its text", n)
		}
		tt.Type("Zeta")
		tt.Key(0, ui.KeyEnter)
		tt.Frame()
		<-posted()
		tt.Frame()
		if shows(tt, "Zeta") || !shows(tt, "Beta") {
			t.Fatalf("without OnAfterLabelEdit: %q, want Beta unedited", tt.Texts())
		}
	})
	t.Run("off", func(t *testing.T) {
		edits := make(chan string, 8)
		keep := true
		tv := editTree(edits, &keep).SetLabelEdit(false)
		tt := tester(t, tv)
		click(t, tt, "Beta")
		tt.Key(0, ui.KeyF2)
		tt.Frame()
		tt.Type("Zeta")
		tt.Key(0, ui.KeyEnter)
		tt.Frame()
		quietEdits(t, edits, "label edit off")
		if shows(tt, "Zeta") {
			t.Fatalf("label edit off: %q", tt.Texts())
		}
	})
}

// A click on the text of the node already selected edits it, once a second
// click would no longer make a double click; a first click only selects.
func TestTreeLabelEditClick(t *testing.T) {
	edits := make(chan string, 8)
	keep := true
	tv := editTree(edits, &keep)
	tt := tester(t, tv)
	x, y := nodeTextAt(t, tt, "Beta")
	tt.ClickAt(x, y)
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if k, _ := tv.Selected(); k != "B" {
		t.Fatalf("the first click selected %q", k)
	}
	if n := textCount(tt, "Beta"); n != 2 {
		t.Fatalf("after a first click Beta shows %d times, in %q: want its label and its text", n, tt.Texts())
	}
	// Typing and moving the focus away would commit an edit (Enter would
	// start one on macOS).
	tt.Type("Q")
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	quietEdits(t, edits, "a first click")
	if !shows(tt, "Beta") {
		t.Fatalf("a first click edited: %q", tt.Texts())
	}
	x, y = nodeTextAt(t, tt, "Beta")
	tt.ClickAt(x, y)
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	tt.Type("Gamma")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, edits); got != "B Gamma" {
		t.Fatalf("a click on the selected node's text: OnAfterLabelEdit got %q", got)
	}
}

// With the three set and cleared, a tree builds what one never told of them
// does: the same pixels, the same nodes.
func TestTreeOpsOffIsUnchanged(t *testing.T) {
	plain := fileTree().Expand("A").SetSelected("A1")
	off := fileTree().Expand("A").SetSelected("A1").
		SetSort(byName).SetFilter(named("one")).SetLabelEdit(true).
		OnAfterLabelEdit(func(string, string) bool { return true })
	tp, to := tester(t, plain), tester(t, off)
	off.SetSort(nil).SetFilter(nil).SetLabelEdit(false)
	tp.Frame()
	to.Frame()
	// Beta, opened by the filter, closes in ui's motion.
	tp.Advance(time.Second)
	to.Advance(time.Second)
	tp.Frame()
	to.Frame()
	if !slices.Equal(tp.Image().Pix, to.Image().Pix) {
		t.Fatal("the frames differ with the three cleared")
	}
	na, nb := tp.Nodes(), to.Nodes()
	if len(na) != len(nb) {
		t.Fatalf("%d nodes, %d with the three cleared", len(na), len(nb))
	}
	for i := range na {
		na[i].ID, nb[i].ID = 0, 0
		if na[i] != nb[i] {
			t.Fatalf("node %d: %+v, with the three cleared %+v", i, na[i], nb[i])
		}
	}
}

// The setters, called from another goroutine while frames run, show at the
// next frame, and nothing races (go test -race).
func TestTreeOpsFromGoroutine(t *testing.T) {
	edits := make(chan string, 8)
	keep := true
	tv := fileTree()
	tt := tester(t, tv)
	tick := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 60 {
			<-tick
			switch i {
			case 10:
				tv.SetSort(func(a, b *tnode) int { return -byName(a, b) })
			case 20:
				tv.SetFilter(named("Alpha"))
			case 30:
				tv.UpdateNode("A2", func(n *tnode) { n.name = "Alpha zero" })
			case 40:
				tv.SetLabelEdit(true).OnAfterLabelEdit(func(k, text string) bool {
					tv.UpdateNode(k, func(n *tnode) { n.name = text })
					edits <- k + " " + text
					return keep
				})
			}
		}
	}()
	for range 60 {
		tick <- struct{}{}
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	rowsInOrder(t, tt, "Alpha", "Alpha zero", "Alpha one")
	hidden(t, tt, "Beta")
	click(t, tt, "Alpha one")
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Type("Alpha best")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, edits); got != "A1 Alpha best" {
		t.Fatalf("OnAfterLabelEdit from a goroutine got %q", got)
	}
}

// A frame at rest with the three on allocates no more than one without.
func TestTreeOpsAllocNothingAtRest(t *testing.T) {
	big := func() *TreeViewWidget[tnode, string] {
		tv := newTree()
		roots := make([]tnode, 40)
		for i := range roots {
			roots[i] = tnode{fmt.Sprint("r", i), fmt.Sprint("Root ", 40-i)}
		}
		tv.SetRoots(roots)
		for i := range roots {
			tv.SetChildren(roots[i].id, []tnode{{fmt.Sprint("r", i, "a"), "Leaf b"}, {fmt.Sprint("r", i, "b"), "Leaf a"}})
		}
		return tv.ExpandAll().SetSelected("r25") // Root 15
	}
	plain := tester(t, big())
	notRoot2 := func(n *tnode) bool { return !strings.HasPrefix(n.name, "Root 2") }
	with := tester(t, big().SetSort(byName).SetFilter(notRoot2).SetLabelEdit(true).
		OnAfterLabelEdit(func(string, string) bool { return true }))
	for range 5 {
		plain.Frame()
		with.Frame()
	}
	a, b := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	if b > a {
		t.Errorf("a frame at rest allocates %v times with the three on, %v without", b, a)
	}
}
