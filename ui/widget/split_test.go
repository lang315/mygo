package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func splitTester(t *testing.T, s *SplitContainerWidget) *Tester {
	t.Helper()
	return NewTester(t, s, 800, 600)
}

func TestSplitContainerShowsBothPanes(t *testing.T) {
	tt := splitTester(t, SplitContainer(Label("left pane"), Label("right pane")))
	for _, s := range []string{"left pane", "right pane"} {
		if !tt.Has(ByLabel(s)) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	d := tt.Find(ByRole(ui.RoleSplitter)).Bounds()
	l, r := tt.Find(ByLabel("left pane")).Bounds(), tt.Find(ByLabel("right pane")).Bounds()
	if !(l.X < d.X && d.X < r.X) {
		t.Errorf("panes and divider out of order: left %v divider %v right %v", l, d, r)
	}
}

func TestSplitContainerDistanceAndPercent(t *testing.T) {
	s := SplitContainer(Label("a"), Label("b")).SetSplitterDistance(200)
	tt := splitTester(t, s)
	if x := tt.Find(ByRole(ui.RoleSplitter)).Bounds().X; math.Abs(float64(x)-200) > 8 {
		t.Errorf("divider at x=%v, want about 200", x)
	}
	s.SetSplitterPercent(25)
	tt.Settle()
	if x := tt.Find(ByRole(ui.RoleSplitter)).Bounds().X; math.Abs(float64(x)-200) > 8 { // 25% of 800
		t.Errorf("divider at x=%v with 25%%, want about 200", x)
	}
	tt.SetSize(400, 600)
	tt.Settle()
	if x := tt.Find(ByRole(ui.RoleSplitter)).Bounds().X; math.Abs(float64(x)-100) > 8 {
		t.Errorf("divider at x=%v after the window halved, want about 100 (a percentage follows the room)", x)
	}
}

func TestSplitContainerStackedPutsTheFirstAbove(t *testing.T) {
	s := SplitContainer(Label("top"), Label("bottom")).Stacked().SetSplitterDistance(150)
	tt := splitTester(t, s)
	d := tt.Find(ByRole(ui.RoleSplitter)).Bounds()
	if math.Abs(float64(d.Y)-150) > 8 {
		t.Errorf("divider at y=%v, want about 150", d.Y)
	}
	if top, bottom := tt.Find(ByLabel("top")).Bounds(), tt.Find(ByLabel("bottom")).Bounds(); !(top.Y < d.Y && d.Y < bottom.Y) {
		t.Errorf("top %v divider %v bottom %v", top, d, bottom)
	}
}

func TestSplitContainerKeyboardMovesTheDividerAndReportsIt(t *testing.T) {
	moved := make(chan float32, 8)
	var s *SplitContainerWidget
	s = SplitContainer(Label("a"), Label("b")).SetSplitterDistance(300).OnSplitterMoved(func() { moved <- s.SplitterDistance() })
	tt := splitTester(t, s)
	tt.Find(ByRole(ui.RoleSplitter)).Focus().Key(0, ui.KeyRight)
	got := received(t, moved)
	if got <= 300 || s.SplitterDistance() != got {
		t.Fatalf("OnSplitterMoved saw %v, SplitterDistance %v, want more than 300", got, s.SplitterDistance())
	}
	select {
	case <-moved:
		t.Fatal("one key press reported two moves")
	default:
	}
}

func TestSplitContainerDragMovesTheDivider(t *testing.T) {
	s := SplitContainer(Label("a"), Label("b")).SetSplitterDistance(300)
	tt := splitTester(t, s)
	d := tt.Find(ByRole(ui.RoleSplitter)).Bounds()
	x, y := d.X+d.W/2, d.Y+d.H/2
	tt.Press(x, y)
	tt.Move(x+100, y)
	tt.Release(x+100, y)
	tt.Settle()
	if got := s.SplitterDistance(); math.Abs(float64(got)-400) > 8 {
		t.Fatalf("after dragging 100 DIPs right from 300, SplitterDistance is %v", got)
	}
}

func TestSplitContainerKeepsTheChosenDistanceWhenTheWindowShrinks(t *testing.T) {
	s := SplitContainer(Label("a"), Label("b")).SetSplitterDistance(300)
	tt := splitTester(t, s)
	tt.SetSize(200, 600)
	tt.Settle()
	tt.SetSize(800, 600)
	tt.Settle()
	if got := s.SplitterDistance(); got != 300 {
		t.Fatalf("SplitterDistance is %v after a narrow window, want the user's 300", got)
	}
}

func TestSplitContainerPanesAreWidgetsOfTheirOwn(t *testing.T) {
	clicks := 0
	tt := splitTester(t, SplitContainer(
		VBox(4).Add(Label("files"), Button("Open").OnClick(func() { clicks++ })),
		SplitContainer(Label("editor"), Label("console")).Stacked()))
	tt.Find(ByLabel("Open")).Click()
	if clicks != 1 || !tt.Has(ByLabel("editor")) || !tt.Has(ByLabel("console")) {
		t.Fatalf("clicks %d, texts %q", clicks, tt.Texts())
	}
}

func TestSplitContainerSettersDoNotReportAMove(t *testing.T) {
	moved := make(chan struct{}, 2)
	s := SplitContainer(Label("a"), Label("b")).OnSplitterMoved(func() { moved <- struct{}{} })
	tt := splitTester(t, s)
	s.SetSplitterDistance(250)
	tt.Settle()
	s.SetSplitterPercent(30)
	tt.Settle()
	select {
	case <-moved:
		t.Fatal("a setter ran OnSplitterMoved")
	default:
	}
	if s.SplitterPercent() != 30 || s.SplitterDistance() != 0 {
		t.Fatalf("percent %v, distance %v", s.SplitterPercent(), s.SplitterDistance())
	}
}
