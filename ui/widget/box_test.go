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
