package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func find(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(s)
	if !ok {
		t.Fatalf("no %q in %q", s, tt.Texts())
	}
	return r
}

func TestVBoxStacksDown(t *testing.T) {
	tt := tester(t, VBox(4).Add(Label("A"), Label("B")))
	if a, b := find(t, tt, "A"), find(t, tt, "B"); b.Y <= a.Y {
		t.Fatalf("B at %v not below A at %v", b, a)
	}
}

func TestHBoxLaysOutRight(t *testing.T) {
	tt := tester(t, HBox(4).Add(Label("A"), Label("B")))
	if a, b := find(t, tt, "A"), find(t, tt, "B"); b.X <= a.X {
		t.Fatalf("B at %v not right of A at %v", b, a)
	}
}

func TestPackEndGoesToTheEnd(t *testing.T) {
	tt := tester(t, HBox(0).Add(Label("A")).PackEnd(Label("B"), false, false))
	if b := find(t, tt, "B"); b.X < 700 {
		t.Fatalf("B at %v, want at the right end of 800", b)
	}
}

func TestPackExpandWithoutFillCentersInItsRoom(t *testing.T) {
	tt := tester(t, HBox(0).Pack(Label("A"), true, false).Add(Label("B")))
	a, b := find(t, tt, "A"), find(t, tt, "B")
	if a.X < 300 || a.X > 450 || b.X < 700 {
		t.Fatalf("A at %v (want mid-row), B at %v (want at the end)", a, b)
	}
}

func TestRemoveAndClear(t *testing.T) {
	a, b := Label("A"), Label("B")
	box := VBox(0).Add(a, b)
	tt := tester(t, box)
	box.Remove(a)
	tt.Frame()
	if tt.HasText("A") || !tt.HasText("B") {
		t.Fatalf("after Remove: %q", tt.Texts())
	}
	box.Clear()
	tt.Frame()
	if tt.HasText("B") {
		t.Fatalf("after Clear: %q", tt.Texts())
	}
}

// Review Focus 2: the same widget added twice shows once, without a
// duplicate-key panic.
func TestAddingAWidgetTwiceKeepsOne(t *testing.T) {
	a := Label("A")
	box := VBox(0).Add(a, a).Add(a)
	tt := tester(t, box)
	n := 0
	for _, s := range tt.Texts() {
		if s == "A" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("A shown %d times: %q", n, tt.Texts())
	}
}

func TestButtonBoxPutsButtonsAtTheEnd(t *testing.T) {
	tt := tester(t, ButtonBox().Add(Label("OK")))
	if r := find(t, tt, "OK"); r.X < 700 {
		t.Fatalf("OK at %v, want at the end", r)
	}
}

// count returns how many texts of the last frame are s.
func count(tt *ui.Tester, s string) int {
	n := 0
	for _, have := range tt.Texts() {
		if have == s {
			n++
		}
	}
	return n
}

func TestHiddenPackedWidgetLeavesNoRoom(t *testing.T) {
	tt := tester(t, HBox(0).Pack(Label("gone").SetVisible(false), true, false).Add(Label("B")))
	if b := find(t, tt, "B"); b.X > 1 {
		t.Fatalf("B at %v: the hidden widget's room still grows", b)
	}
}

func TestButtonBoxButtonsShareAWidth(t *testing.T) {
	tt := tester(t, ButtonBox().Add(Button("OK"), Button("Cancel the whole thing")))
	// Find gives the texts' rects; each is centered in its button. Equal
	// buttons ending at 800, 8 apart: Cancel's button is w = 2*(800-c2)
	// wide, so OK's center is c2 - w - 8.
	ok, cancel := find(t, tt, "OK"), find(t, tt, "Cancel the whole thing")
	c1, c2 := ok.X+ok.W/2, cancel.X+cancel.W/2
	w := 2 * (800 - c2)
	if want := c2 - w - 8; c1 < want-1 || c1 > want+1 {
		t.Fatalf("OK centered at %v, want %v for buttons of one width %v (Cancel %v)", c1, want, w, cancel)
	}
}

func TestWidgetInTwoBoxesShowsInBoth(t *testing.T) {
	l := Label("twice")
	tt := tester(t, VBox(0).Add(VBox(0).Add(l), VBox(0).Add(l)))
	if n := count(tt, "twice"); n != 2 {
		t.Fatalf("shown %d times: %q", n, tt.Texts())
	}
}
