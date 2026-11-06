package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

func rows(n int) *BoxWidget {
	b := VBox(0)
	for i := range n {
		b.Add(Label(fmt.Sprintf("row %d", i)))
	}
	return b
}

// In a Vertical panel the content keeps the panel's width: a long label wraps
// there. In a panel that scrolls both ways it keeps its natural width.
func TestVerticalScrollPanelWrapsLabelsAtItsWidth(t *testing.T) {
	one := find(t, tester(t, Label("row")), "row").H
	tt := NewTester(t, ScrollPanel(VBox(0).Add(Label(longLine))).Vertical(), 300, 400)
	if r := find(t, tt.Tester, longLine); r.H < 2*one || r.W > 300 {
		t.Errorf("label %v (a line is %v high): not wrapped at 300", r, one)
	}
	tb := NewTester(t, ScrollPanel(VBox(0).Add(Label(longLine))), 300, 400)
	if r := find(t, tb.Tester, longLine); r.H >= 2*one {
		t.Errorf("both-ways panel wrapped the label: %v", r)
	}
}

// A dialog of height 0 fits the whole content of a Vertical panel, as it fits
// the same content without the panel.
func TestDialogOfHeightZeroFitsTheContentOfAVerticalScrollPanel(t *testing.T) {
	plain := fitOfForm(500, rows(12))
	if got := fitOfForm(500, ScrollPanel(rows(12)).Vertical()); got != plain {
		t.Errorf("fit %d in a Vertical panel, %d without it", got, plain)
	}
}

// A window smaller than the content scrolls it; what is below is reached.
func TestVerticalScrollPanelScrollsWhenShrunk(t *testing.T) {
	tt := NewTester(t, ScrollPanel(rows(40)).Vertical(), 300, 100)
	before := find(t, tt.Tester, "row 30").Y
	if before < 100 {
		t.Fatalf("row 30 at %v: the content fits, make it taller", before)
	}
	tt.Scroll(150, 50, 0, 200)
	tt.Frame()
	if after := find(t, tt.Tester, "row 30").Y; after >= before {
		t.Errorf("row 30 at %v after scrolling, was %v", after, before)
	}
}

// A window taller than the content: the root's background goes to the bottom.
func TestVerticalScrollPanelPaintsTheRootToTheBottom(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	root := VBox(0).Apply(NewStyle().Background(Fixed(red))).Add(Label("short")).Expand()
	tt := NewTester(t, ScrollPanel(root).Vertical(), 300, 400)
	img := tt.Image()
	if got := img.RGBAAt(5, img.Bounds().Dy()-5); got.R != 255 || got.G != 0 || got.B != 0 {
		t.Errorf("pixel at the bottom is %v, want the root's red", got)
	}
}

// A button bar outside the panel stays in view while the panel scrolls.
func TestButtonBarOutsideAVerticalScrollPanelStaysInView(t *testing.T) {
	tt := NewTester(t, VBox(0).Add(ScrollPanel(rows(40)).Vertical(), Button("OK")).Expand(), 300, 200)
	at := func() ui.Rect { return find(t, tt.Tester, "OK") }
	before := at()
	if before.Y < 0 || before.Y+before.H > 200 {
		t.Fatalf("OK %v is out of the 200 high window", before)
	}
	tt.Scroll(150, 50, 0, 300)
	tt.Frame()
	if after := at(); after != before {
		t.Errorf("OK moved from %v to %v", before, after)
	}
}

// A page that is a Vertical panel is still a viewport: a sidebar does not
// wrap it in a scroller of its own.
func TestVerticalScrollPanelIsAViewport(t *testing.T) {
	if !isViewport(ScrollPanel(Label("x")).Vertical()) {
		t.Error("isViewport is false for a Vertical scroll panel")
	}
}

func minimumOf(fw interface{ MinimumSize() platform.Size }) [2]int {
	s := fw.MinimumSize()
	return [2]int{s.Width, s.Height}
}

// The minimum size is in the window's options at Show, goes to a window that is
// shown, and is asked again by the next Show.
func TestFormMinimumSizeReachesTheBackend(t *testing.T) {
	f := NewForm("Min form", 600, 400).SetMinimumSize(300, 200)
	f.SetContent(Label("x"))
	f.Show()
	fw := waitFake(t, "Min form")
	if fw.Opts.MinSize != (platform.Size{Width: 300, Height: 200}) {
		t.Errorf("options %v at Show, want 300x200", fw.Opts.MinSize)
	}
	f.SetMinimumSize(350, 250)
	wait.Until(t, "the live minimum size", func() bool { return minimumOf(fw) == [2]int{350, 250} })
	f.Close()
	wait.Until(t, "the window to close", fw.IsClosed)
	f.Show()
	var again bool
	wait.Until(t, "the second window", func() bool {
		for _, w := range fb.Windows() {
			if w != fw && w.Title() == "Min form" && !w.IsClosed() {
				again = w.Opts.MinSize == platform.Size{Width: 350, Height: 250}
				return true
			}
		}
		return false
	})
	if !again {
		t.Error("a reshow lost the minimum size")
	}
	f.Close()
}

func TestDialogSetMinimumSizeReturnsTheDialog(t *testing.T) {
	d := NewDialog(nil, "Min dialog", "", 400, 300, Label("x"))
	if d.SetMinimumSize(200, 100) != d {
		t.Fatal("SetMinimumSize returned another dialog")
	}
	d.Show()
	t.Cleanup(d.Close)
	if fw := waitFake(t, "Min dialog"); fw.Opts.MinSize != (platform.Size{Width: 200, Height: 100}) {
		t.Errorf("options %v, want 200x100", fw.Opts.MinSize)
	}
}

// reachable fails the test unless the control labeled label can be scrolled
// into the window of dt: it is in view now, or a wheel turn at one of the
// points brings it there.
func reachable(t *testing.T, dt *DialogTester, label string) {
	t.Helper()
	tt := dt.Tester.Tester
	in := func() bool {
		r, ok := tt.Find(label)
		return ok && r.Y >= 0 && r.Y+r.H <= float32(dt.h)
	}
	for i := 0; i < 60 && !in(); i++ {
		tt.Scroll(250, 100, 0, 400)
		dt.Frame()
	}
	if !in() {
		r, _ := tt.Find(label)
		t.Errorf("%q at %v is out of the %d high window, and scrolling does not bring it in", label, r, dt.h)
	}
}

// A height-0 dialog taller than the work area scrolls as a whole: what is
// below the fold, a button bar included, can be reached, whether the panel is
// the content, nested in a box, or beside long content, and whether it
// scrolls both ways or only up and down.
func TestClampedDialogsWithAScrollPanelStayReachable(t *testing.T) {
	cases := map[string]func() Control{
		"panel and bar": func() Control {
			return VBox(0).Add(ScrollPanel(rows(150)).Vertical(), Button("OK")).Expand()
		},
		"panel in a box": func() Control {
			return VBox(0).Add(VBox(0).Add(ScrollPanel(rows(150)).Vertical()), Button("OK")).Expand()
		},
		"long content beside a short panel": func() Control {
			return VBox(0).Add(rows(150), ScrollPanel(rows(3)).Vertical(), Button("OK")).Expand()
		},
		"long content beside a both-ways panel": func() Control {
			return VBox(0).Add(rows(150), ScrollPanel(rows(3)), Button("OK")).Expand()
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			d := NewDialog(nil, "t", "", 500, 300, mk())
			dt := TestDialog(t, d)
			d.SetSize(500, 0)
			dt.Settle()
			reachable(t, dt, "OK")
			reachable(t, dt, "row 149")
		})
	}
}

// At a fixed height the button bar outside the panel stays in view, and the
// panel scrolls.
func TestFixedHeightDialogKeepsTheButtonBarBesideAVerticalPanel(t *testing.T) {
	d := NewDialog(nil, "t", "", 500, 300, VBox(0).Add(ScrollPanel(rows(150)).Vertical(), Button("OK")).Expand())
	dt := TestDialog(t, d)
	dt.Settle()
	tt := dt.Tester.Tester
	ok := find(t, tt, "OK")
	if ok.Y < 0 || ok.Y+ok.H > float32(dt.h) {
		t.Fatalf("OK %v is out of the %d high window", ok, dt.h)
	}
	before := find(t, tt, "row 100").Y
	tt.Scroll(250, 50, 0, 300)
	dt.Frame()
	if after := find(t, tt, "row 100").Y; after >= before {
		t.Errorf("the panel did not scroll: row 100 at %v, was %v", after, before)
	}
	if again := find(t, tt, "OK"); again != ok {
		t.Errorf("OK moved from %v to %v", ok, again)
	}
}

// A dialog that fits to its content is no lower than its minimum height.
func TestFitIsNoLowerThanTheMinimumHeight(t *testing.T) {
	d := NewDialog(nil, "Fit min", "", 500, 0, Label("short")).SetMinimumSize(0, 400)
	d.Show()
	t.Cleanup(d.Close)
	if h := waitFake(t, "Fit min").Opts.Height; h < 400 {
		t.Errorf("fitted to %d, below the minimum 400", h)
	}
}

// The panel builds its content in a vertical axis, whatever the box around it:
// the content's Align is across the panel.
func TestVerticalScrollPanelBuildsInAVerticalAxis(t *testing.T) {
	tt := NewTester(t, HBox(0).Add(ScrollPanel(Label("x").Align(End)).Vertical().Expand()), 300, 200)
	if x := find(t, tt.Tester, "x").X; x < 250 {
		t.Errorf("label at x %v: Align(End) did not put it at the right", x)
	}
}

// A dialog that is not clamped keeps its window as before: at a fixed height
// the panel scrolls and the button bar stays in view; at height 0 with short
// content it fits the content, and nothing scrolls as a whole.
func TestVerticalPanelInADialogThatIsNotClamped(t *testing.T) {
	d := NewDialog(nil, "t", "", 500, 300, VBox(0).Add(ScrollPanel(rows(150)).Vertical(), Button("OK")).Expand())
	dt := TestDialog(t, d)
	dt.Settle()
	if ok := find(t, dt.Tester.Tester, "OK"); ok.Y+ok.H > 300 {
		t.Errorf("fixed height: OK %v is out of the 300 high window", ok)
	}
	short := NewDialog(nil, "t", "", 500, 0, VBox(0).Add(ScrollPanel(rows(3)).Vertical(), Button("OK")).Expand())
	st := TestDialog(t, short)
	short.SetSize(500, 0)
	st.Settle()
	mu.Lock()
	scrolls := short.fitScroll
	mu.Unlock()
	if scrolls || st.h > 300 {
		t.Errorf("short content: fitScroll %v, window %d high", scrolls, st.h)
	}
	if ok := find(t, st.Tester.Tester, "OK"); ok.Y+ok.H > float32(st.h) {
		t.Errorf("short content: OK %v is out of the %d high window", ok, st.h)
	}
}
