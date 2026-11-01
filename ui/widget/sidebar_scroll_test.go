package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// linesPage is a page of n lines, "line 0" to "line n-1", each a button, so
// that Tab can walk it.
func linesPage(n int) Control {
	b := VBox(4)
	for i := range n {
		b.Add(Button(fmt.Sprintf("line %d", i)))
	}
	return b
}

// yOf is where the element showing text is, -1 when none.
func buttonY(tt *Tester, text string) float32 {
	for _, n := range tt.Nodes() {
		if n.Label == text && n.Role == ui.RoleButton {
			return n.Bounds.Y
		}
	}
	return -1
}

func TestSidebarTallPageScrollsByWheel(t *testing.T) {
	sb := Sidebar().Add("a", "One", linesPage(30))
	tt := NewTester(t, sb, 500, 300)
	tt.Frame()
	if y := buttonY(tt, "line 29"); y >= 0 && y < 300 {
		t.Fatalf("line 29 is in view at %v before any scrolling", y)
	}
	for range 40 {
		tt.Scroll(350, 150, 0, 300)
		tt.Frame()
		if y := buttonY(tt, "line 29"); y >= 0 && y < 300 {
			return
		}
	}
	t.Fatalf("a wheel over the page never brought line 29 into view (at %v)", buttonY(tt, "line 29"))
}

func TestSidebarTabBringsAControlBelowTheFoldIntoView(t *testing.T) {
	sb := Sidebar().Add("a", "One", linesPage(30))
	tt := NewTester(t, sb, 500, 300)
	tt.Frame()
	for range 80 {
		tt.Key(0, ui.KeyTab)
		tt.Frame()
		for _, n := range tt.Nodes() {
			if n.Label == "line 29" && n.Focused {
				if n.Bounds.Y < 0 || n.Bounds.Y+n.Bounds.H > 300 {
					t.Fatalf("line 29 has the focus but lies at %v", n.Bounds)
				}
				return
			}
		}
	}
	t.Fatal("Tab never reached line 29")
}

// A page that fits draws no scroll bar and nothing moves.
func TestSidebarShortPageDoesNotScroll(t *testing.T) {
	sb := Sidebar().Add("a", "One", linesPage(3))
	tt := NewTester(t, sb, 500, 300)
	tt.Frame()
	before := buttonY(tt, "line 2")
	tt.Scroll(350, 100, 0, 300)
	tt.Frame()
	if after := buttonY(tt, "line 2"); after != before {
		t.Fatalf("line 2 moved from %v to %v", before, after)
	}
}

// longPage is a page of n wrapped labels, the height of which depends on the
// width.
func longPage(n int) Control {
	b := VBox(4)
	for i := range n {
		b.Add(Label(fmt.Sprintf("line %d of a page with some words to wrap when the dialog is narrow enough to need it", i)))
	}
	return b
}

// A dialog of height 0 with a sidebar fits its tallest page, whichever is
// shown first.
func TestSidebarDialogFitsItsTallestPage(t *testing.T) {
	tall := fitShow(t, "tall alone", 500, 0, longPage(30)).Opts.Height
	short := fitShow(t, "short alone", 500, 0, longPage(3)).Opts.Height
	if tall <= short {
		t.Fatalf("pages of 30 and 3 lines fit %d and %d", tall, short)
	}
	for _, c := range []struct {
		name  string
		a, b  int
		shown string
	}{{"short first", 3, 30, "a"}, {"tall first", 30, 3, "a"}, {"tall shown", 3, 30, "b"}} {
		t.Run(c.name, func(t *testing.T) {
			sb := Sidebar().Add("a", "One", longPage(c.a)).Add("b", "Two", longPage(c.b))
			sb.Select(c.shown)
			fw := fitShow(t, "sidebar "+c.name, 500, 0, sb)
			// The page area is the dialog less the sidebar's own padding: the
			// window is no shorter than the tallest page alone.
			if fw.Opts.Height < tall {
				t.Fatalf("window %d high; the tallest page alone needs %d", fw.Opts.Height, tall)
			}
			if fw.Opts.Height > tall+60 {
				t.Fatalf("window %d high; the tallest page alone needs %d", fw.Opts.Height, tall)
			}
		})
	}
}

// Showing another page after the window opened does not resize it, and the
// tall page scrolls inside.
func TestSidebarDialogDoesNotResizeWhenAPageIsChosen(t *testing.T) {
	sb := Sidebar().Add("a", "One", longPage(3)).Add("b", "Two", longPage(30))
	fw := fitShow(t, "sidebar once", 500, 0, sb)
	before := fw.Opts.Height
	sb.Select("b")
	frame(fw)
	frame(fw)
	if fw.Opts.Height != before {
		t.Fatalf("height %d became %d", before, fw.Opts.Height)
	}
}

// A narrow dialog wraps more and is taller; its sidebar dialog follows.
func TestSidebarDialogFitFollowsTheWidth(t *testing.T) {
	var prev int
	for _, w := range []int{900, 500, 360} {
		sb := Sidebar().Add("a", "One", longPage(3)).Add("b", "Two", longPage(30))
		h := fitShow(t, fmt.Sprintf("sidebar width %d", w), w, 0, sb).Opts.Height
		alone := fitShow(t, fmt.Sprintf("tall width %d", w), w, 0, longPage(30)).Opts.Height
		if h < alone {
			t.Errorf("width %d: window %d, the tallest page alone %d", w, h, alone)
		}
		if h < prev {
			t.Errorf("width %d: window %d is shorter than the wider one's %d", w, h, prev)
		}
		prev = h
	}
}

// More page than the screen has opens as high as the work area allows, and
// the page scrolls inside the sidebar, which stays where it is.
func TestSidebarDialogTallerThanTheWorkAreaIsClampedAndItsPageScrolls(t *testing.T) {
	sb := Sidebar().Add("a", "One", longPage(3)).Add("b", "Two", linesPage(150))
	d := NewDialog(nil, "sidebar clamp", "", 500, 0, sb)
	d.Show()
	t.Cleanup(d.Close)
	fw := waitFake(t, "sidebar clamp")
	frame(fw)
	frame(fw)
	if h := fw.Opts.Height; h > 875 || h < 700 {
		t.Errorf("height %d: want the work area's 875 less a margin", h)
	}
	mu.Lock()
	outer := d.fitScroll
	mu.Unlock()
	if outer {
		t.Error("the whole dialog scrolls; the page should, inside the sidebar")
	}
}

// A TabControl page taller than the room scrolls too.
func TestTabControlTallPageScrollsByWheelAndTab(t *testing.T) {
	mk := func() *Tester {
		nb := TabControl().AddPage("One", linesPage(30)).AddPage("Two", Label("two"))
		tt := NewTester(t, nb.Expand(), 500, 300) // as a table is given the room (Expand)
		tt.Frame()
		return tt
	}
	tt := mk()
	if y := buttonY(tt, "line 29"); y >= 0 && y < 300 {
		t.Fatalf("line 29 is in view at %v before any scrolling", y)
	}
	seen := false
	for range 40 {
		tt.Scroll(250, 150, 0, 300)
		tt.Frame()
		if y := buttonY(tt, "line 29"); y >= 0 && y < 300 {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("a wheel never brought line 29 into view (at %v)", buttonY(tt, "line 29"))
	}
	tt = mk()
	for range 80 {
		tt.Key(0, ui.KeyTab)
		tt.Frame()
		for _, n := range tt.Nodes() {
			if n.Label == "line 29" && n.Focused {
				if n.Bounds.Y < 0 || n.Bounds.Y+n.Bounds.H > 300 {
					t.Fatalf("line 29 has the focus but lies at %v", n.Bounds)
				}
				return
			}
		}
	}
	t.Fatal("Tab never reached line 29")
}

// A page that holds a viewport (a table) fills the room and scrolls itself, as
// it did before: no scroller around it that would give it all the height.
func TestSidebarAndTabControlPagesWithATableKeepFillingTheRoom(t *testing.T) {
	rows := make([]acct, 500)
	for i := range rows {
		rows[i] = acct{i + 1, fmt.Sprintf("name%03d", i+1), i}
	}
	mk := func() *DataGridViewWidget[acct, int] { return testTable(rows...).BottomPanel() }
	for name, c := range map[string]Control{
		"Sidebar":    Sidebar().Add("a", "One", mk()),
		"TabControl": TabControl().AddPage("One", mk()).Expand(),
	} {
		t.Run(name, func(t *testing.T) {
			tt := NewTester(t, c, 800, 400)
			tt.Frame()
			r, ok := tt.Tester.Find("Search")
			if !ok || r.Y+r.H > 400 {
				t.Fatalf("the table's panel at %v (found %v), not in a 400 high window", r, ok)
			}
		})
	}
}
