package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSegmentedSelect(t *testing.T) {
	changed := make(chan int, 4)
	var s *SegmentedWidget
	s = Segmented("List", "Grid").OnChange(func() { changed <- s.Selected() })
	tt := NewTester(t, s, 400, 300)
	tt.Find(ByLabel("Grid")).Click()
	if got := received(t, changed); got != 1 || s.Selected() != 1 {
		t.Fatalf("OnChange saw %d, Selected %d after a click on Grid", got, s.Selected())
	}
	tt.Settle()
	select {
	case <-changed:
		t.Fatal("OnChange ran twice for one click")
	default:
	}
}

func TestSegmentedSetSelectedOutOfRangeIgnored(t *testing.T) {
	changed := make(chan struct{}, 4)
	s := Segmented("List", "Grid").SetSelected(1).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, s, 400, 300)
	s.SetSelected(5)
	tt.Settle()
	if s.Selected() != 1 {
		t.Fatalf("Selected %d after SetSelected(5), want 1 kept", s.Selected())
	}
	s.SetSelected(-1)
	tt.Settle()
	if s.Selected() != 1 || !tt.Find(ByLabel("Grid")).Checked() {
		t.Fatalf("Selected %d after SetSelected(-1), want 1 kept: a Segmented always has one choice", s.Selected())
	}
	select {
	case <-changed:
		t.Fatal("a setter ran OnChange")
	default:
	}
}

func TestSegmentedArrowsMove(t *testing.T) {
	changed := make(chan int, 4)
	var s *SegmentedWidget
	s = Segmented("List", "Grid", "Table").OnChange(func() { changed <- s.Selected() })
	tt := NewTester(t, s, 400, 300)
	tt.Find(ByLabel("List")).Key(0, ui.KeyRight)
	if got := received(t, changed); got != 1 {
		t.Fatalf("Right from List gave %d", got)
	}
	tt.Find(ByLabel("Grid")).Key(0, ui.KeyRight)
	if got := received(t, changed); got != 2 {
		t.Fatalf("Right from Grid gave %d", got)
	}
}

func TestSegmentedSetterDoesNotFireOnChange(t *testing.T) {
	changed := make(chan struct{}, 4)
	s := Segmented("List", "Grid").OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, s, 400, 300)
	s.SetSelected(1)
	tt.Settle()
	if s.Selected() != 1 || !tt.Find(ByLabel("Grid")).Checked() {
		t.Fatalf("Selected %d after SetSelected(1)", s.Selected())
	}
	select {
	case <-changed:
		t.Fatal("SetSelected ran OnChange")
	default:
	}
}

func TestSegmentedDisabled(t *testing.T) {
	changed := make(chan struct{}, 4)
	s := Segmented("List", "Grid").OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, s.SetDisabled(true), 400, 300)
	tt.ClickAt(middle(tt.Find(ByLabel("Grid")).Bounds()))
	tt.Settle()
	if s.Selected() != 0 {
		t.Fatalf("a disabled Segmented took a click: Selected %d", s.Selected())
	}
	select {
	case <-changed:
		t.Fatal("a disabled Segmented ran OnChange")
	default:
	}
}

// middle returns the middle of r, for ClickAt on a control that a Node's
// Click refuses, as a disabled one.
func middle(r ui.Rect) (x, y float32) { return r.X + r.W/2, r.Y + r.H/2 }
