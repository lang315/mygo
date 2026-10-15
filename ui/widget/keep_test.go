package widget

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

// An entry moved to another box keeps its caret: typing goes where the
// caret was, not to the end.
func TestReparentedWidgetKeepsItsState(t *testing.T) {
	e := Entry().SetText("abcdef")
	left, right := VBox(0).Add(e), VBox(0)
	tt := tester(t, HBox(0).Add(left, right))
	e.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	left.Remove(e)
	right.Add(e)
	tt.Frame()
	e.Focus()
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if got := e.Text(); got != "abXcdef" {
		t.Fatalf("text %q after moving the entry and typing X, want abXcdef", got)
	}
}

func list100() *ScrolledWidget {
	box := VBox(0)
	for i := range 100 {
		box.Add(Label(fmt.Sprintf("row %d", i)))
	}
	return Scrolled(box).SetSize(300, 200)
}

// rowY is where the label "row 30" is.
func rowY(t *testing.T, tt *ui.Tester) float32 {
	t.Helper()
	r, ok := tt.Find("row 30")
	if !ok {
		t.Fatalf("no row 30: %q", tt.Texts())
	}
	return r.Y
}

// scrollDown scrolls the list under (100, 100) down by 300 DIPs and
// returns where row 30 is then.
func scrollDown(t *testing.T, tt *ui.Tester) float32 {
	t.Helper()
	top := rowY(t, tt)
	tt.Scroll(100, 100, 0, 300)
	tt.Frame()
	y := rowY(t, tt)
	if y >= top {
		t.Fatalf("row 30 at %v after scrolling down, was %v: the list did not scroll", y, top)
	}
	return y
}

func TestNotebookPageKeepsItsState(t *testing.T) {
	nb := Notebook().AddPage("One", list100()).AddPage("Two", Label("two"))
	tt := tester(t, nb)
	before := scrollDown(t, tt)
	nb.SetPage(1)
	tt.Frame()
	nb.SetPage(0)
	tt.Frame()
	if y := rowY(t, tt); y != before {
		t.Fatalf("row 30 at %v after coming back to the page, was %v", y, before)
	}
}

func TestHiddenWidgetKeepsItsState(t *testing.T) {
	s := list100()
	tt := tester(t, VBox(0).Pack(s, true, true))
	before := scrollDown(t, tt)
	s.SetVisible(false)
	tt.Frame()
	s.SetVisible(true)
	tt.Frame()
	if y := rowY(t, tt); y != before {
		t.Fatalf("row 30 at %v after hiding and showing, was %v", y, before)
	}
}

func TestNestedHiddenWidgetsKeepState(t *testing.T) {
	s := list100()
	inner := VBox(0).Pack(s, true, true)
	outer := VBox(0).Pack(inner, true, true)
	tt := tester(t, VBox(0).Pack(outer, true, true))
	before := scrollDown(t, tt)
	inner.SetVisible(false)
	outer.SetVisible(false)
	tt.Frame()
	outer.SetVisible(true)
	tt.Frame()
	inner.SetVisible(true)
	tt.Frame()
	if y := rowY(t, tt); y != before {
		t.Fatalf("row 30 at %v after hiding both boxes, was %v", y, before)
	}
}

// A widget the app no longer holds leaves the keeper after a GC and a
// frame, and ui then frees its state.
func TestDroppedWidgetStateIsFreed(t *testing.T) {
	box := VBox(0)
	tt := tester(t, box)
	box.Add(list100())
	tt.Frame()
	mu.Lock()
	n := len(lastKeeper.widgets)
	mu.Unlock()
	box.Clear()
	tt.Frame()
	runtime.GC()
	runtime.GC()
	tt.Frame()
	mu.Lock()
	left := len(lastKeeper.widgets)
	mu.Unlock()
	if n < 100 || left >= n {
		t.Fatalf("keeper holds %d widgets after the list was dropped, had %d", left, n)
	}
}

// labelY is where the label of the fake window fw reads s, or -1.
func labelY(fw *fake.Window, s string) float64 {
	for _, n := range accessNodes(fw) {
		if n.Label == s {
			return n.Bounds.Y
		}
	}
	return -1
}

// A window's content swapped out and back keeps its state.
func TestSetContentBackKeepsState(t *testing.T) {
	s := list100()
	w := NewWindow("Swap", 400, 300).SetContent(s)
	w.Show()
	defer w.Close()
	fw := waitFake(t, "Swap")
	frame(fw)
	top := labelY(fw, "row 25")
	mygo.RunOnMain(func() {
		fw.FakeSurface().Send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 100, Y: 100, DY: 100})
		fw.FakeSurface().Frame()
	})
	before := labelY(fw, "row 25")
	if before < 0 || before >= top {
		t.Fatalf("row 25 at %v after scrolling down, was %v: the list did not scroll", before, top)
	}
	w.SetContent(Label("other"))
	frame(fw)
	w.SetContent(s)
	frame(fw)
	if after := labelY(fw, "row 25"); after != before {
		t.Fatalf("row 25 at %v after swapping the content back, was %v", after, before)
	}
}
