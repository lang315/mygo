package widget

import (
	"sync"
	"testing"

	"github.com/egoist/mygo"
)

// recordInvalidations has a change record the windows it redraws, instead
// of redrawing them, until the test ends.
func recordInvalidations(t *testing.T) (redrawn func() map[*mygo.Window]int) {
	t.Helper()
	var rec sync.Mutex
	got := map[*mygo.Window]int{}
	prev := invalidate
	invalidate = func(w *mygo.Window) { rec.Lock(); got[w]++; rec.Unlock() }
	t.Cleanup(func() { invalidate = prev })
	return func() map[*mygo.Window]int {
		rec.Lock()
		defer rec.Unlock()
		out := map[*mygo.Window]int{}
		for w, n := range got {
			out[w] = n
		}
		clear(got)
		return out
	}
}

// showFrame shows content in a window and draws a frame, so the window's
// keeper has seen what it holds.
func showFrame(t *testing.T, content Control) (*Form, *mygo.Window) {
	t.Helper()
	f := NewForm("Invalidate", 300, 200)
	f.SetContent(content)
	f.Show()
	t.Cleanup(f.Close)
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
	return f, f.Native()
}

func wantRedrawn(t *testing.T, got map[*mygo.Window]int, want ...*mygo.Window) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("redrew %d windows, want %d", len(got), len(want))
	}
	for _, w := range want {
		if got[w] == 0 {
			t.Errorf("did not redraw a window that shows the widget")
		}
	}
}

func TestChangeRedrawsOnlyTheWindowHoldingTheWidget(t *testing.T) {
	a, b := Label("a"), Label("b")
	_, wa := showFrame(t, a)
	_, wb := showFrame(t, b)
	redrawn := recordInvalidations(t)
	a.SetText("a2")
	wantRedrawn(t, redrawn(), wa)
	b.SetText("b2")
	wantRedrawn(t, redrawn(), wb)
}

func TestChangeRedrawsEveryWindowHoldingTheWidget(t *testing.T) {
	l := Label("both")
	_, wa := showFrame(t, VBox(0).Add(l))
	_, wb := showFrame(t, VBox(0).Add(l))
	redrawn := recordInvalidations(t)
	l.SetText("changed")
	wantRedrawn(t, redrawn(), wa, wb)
}

func TestGlobalChangeRedrawsEveryShownWindow(t *testing.T) {
	_, wa := showFrame(t, Label("a"))
	_, wb := showFrame(t, Label("b"))
	redrawn := recordInvalidations(t)
	Change(func() {})
	wantRedrawn(t, redrawn(), wa, wb)
}

func TestChangeOfAWidgetNoWindowBuiltRedrawsNothing(t *testing.T) {
	showFrame(t, Label("shown"))
	lost := Label("never added")
	redrawn := recordInvalidations(t)
	lost.SetText("still nowhere")
	wantRedrawn(t, redrawn())
}

func TestChangeOfAHiddenWidgetRedrawsItsWindow(t *testing.T) {
	l := Label("hidden soon")
	f, w := showFrame(t, VBox(0).Add(l))
	l.SetVisible(false)
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
	_ = f
	redrawn := recordInvalidations(t)
	l.SetText("while hidden")
	wantRedrawn(t, redrawn(), w)
}

func TestFormSetterRedrawsOnlyItsOwnWindow(t *testing.T) {
	fa, wa := showFrame(t, Label("a"))
	showFrame(t, Label("b"))
	redrawn := recordInvalidations(t)
	fa.SetTitle("renamed")
	wantRedrawn(t, redrawn(), wa)
}

func TestAddToABoxRedrawsTheBoxsWindows(t *testing.T) {
	box := VBox(0)
	_, wa := showFrame(t, box)
	showFrame(t, Label("elsewhere"))
	redrawn := recordInvalidations(t)
	box.Add(Label("new"))
	wantRedrawn(t, redrawn(), wa)
}

// With 20 windows, one change redraws one, where it redrew 20.
func TestOneChangeAmongTwentyWindowsRedrawsOne(t *testing.T) {
	labels := make([]*LabelWidget, 20)
	wins := make([]*mygo.Window, 20)
	for i := range labels {
		labels[i] = Label("l")
		_, wins[i] = showFrame(t, labels[i])
	}
	redrawn := recordInvalidations(t)
	labels[7].SetText("changed")
	wantRedrawn(t, redrawn(), wins[7])
}
