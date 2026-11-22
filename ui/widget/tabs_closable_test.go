package widget

import (
	"fmt"
	"image/color"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// docs is the control of the cases: Alpha, Beta and Gamma, each a VBox
// holding a TextBox, in a form 500 × 300, Alpha shown. OnPageClosing
// records the pages it is asked about and answers answer; OnSwitch counts.
type docs struct {
	nb       *TabControlWidget
	tt       *ui.Tester
	pages    [3]*BoxWidget
	boxes    [3]*TextBoxWidget
	switches atomic.Int32

	mu     sync.Mutex
	asked  []Control
	answer bool
}

func newDocs(t *testing.T, closable bool) *docs {
	t.Helper()
	d := &docs{answer: true}
	for i := range d.pages {
		d.boxes[i] = TextBox()
		d.pages[i] = VBox(4).Add(d.boxes[i])
	}
	d.nb = TabControl().AddPage("Alpha", d.pages[0]).AddPage("Beta", d.pages[1]).AddPage("Gamma", d.pages[2]).
		OnSwitch(func() { d.switches.Add(1) }).
		OnPageClosing(func(p Control) bool {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.asked = append(d.asked, p)
			return d.answer
		})
	if closable {
		d.nb.SetClosable(true)
	}
	d.tt = ui.NewTester(NewForm("Docs", 500, 300).SetContent(d.nb).view, 500, 300)
	settle(d.tt)
	return d
}

// flush draws, waits for what was posted to the event goroutine, and
// draws what it changed.
func (d *docs) flush(t *testing.T) {
	t.Helper()
	settle(d.tt)
	flushEvents(t)
	settle(d.tt)
}

func (d *docs) askedPages() []Control {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.asked)
}

// state fails the test unless the tabs are names with chosen shown.
func (d *docs) state(t *testing.T, names []string, chosen int) {
	t.Helper()
	got, ch := tabNames(d.tt)
	if !slices.Equal(got, names) || ch != chosen || d.nb.Page() != chosen {
		t.Fatalf("tabs %q chosen %d, Page %d; want %q chosen %d", got, ch, d.nb.Page(), names, chosen)
	}
}

// tabBounds returns the box of the tab labeled label.
func tabBounds(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTab && n.Label == label {
			return n.Bounds
		}
	}
	t.Fatalf("no tab %q", label)
	return ui.Rect{}
}

// xAt returns the middle of the x of the tab labeled label, at text scale
// scale: the tab's end padding is Space(2), the x a 12 × scale icon padded
// by Space(0.5).
func xAt(t *testing.T, tt *ui.Tester, label string, scale float32) (float32, float32) {
	t.Helper()
	b := tabBounds(t, tt, label)
	return b.X + b.W - space(2) - (12*scale+2*space(0.5))/2, b.Y + b.H/2
}

// clickAt clicks the primary button at (x, y), as a user does.
func clickAt(tt *ui.Tester, x, y float32) {
	tt.Move(x, y)
	tt.Frame()
	tt.Press(x, y)
	tt.Frame()
	tt.Release(x, y)
	tt.Frame()
}

func clickX(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	x, y := xAt(t, tt, label, 1)
	clickAt(tt, x, y)
}

// middleClick holds the middle button on the tab labeled label for three
// frames, then lets it go.
func middleClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	b := tabBounds(t, tt, label)
	x, y := b.X+space(3)+2, b.Y+b.H/2 // on the label, clear of the x
	tt.Move(x, y)
	tt.PressButton(2, x, y)
	tt.Frame()
	tt.Frame()
	tt.Frame()
	tt.ReleaseButton(2, x, y)
	tt.Frame()
}

// xCount is the number of x buttons shown: elements labeled Close.
func xCount(tt *ui.Tester) int {
	n := 0
	for _, s := range tt.Texts() {
		if s == "Close" {
			n++
		}
	}
	return n
}

// focusBox puts the focus in the TextBox of the page shown.
func (d *docs) focusBox(t *testing.T, i int) {
	t.Helper()
	d.boxes[i].Focus()
	settle(d.tt)
	for _, n := range d.tt.Nodes() {
		if n.Role == ui.RoleTextField && n.Focused {
			return
		}
	}
	t.Fatal("the page's TextBox has no focus")
}

// 1. Each tab shows its label and an x; without SetClosable, none.
func TestClosableTabsShowAnXOnEachTab(t *testing.T) {
	d := newDocs(t, true)
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
	if n := xCount(d.tt); n != 3 {
		t.Fatalf("%d x shown, want 3; texts %q", n, d.tt.Texts())
	}
	plain := newDocs(t, false)
	if n := xCount(plain.tt); n != 0 {
		t.Fatalf("%d x shown without SetClosable", n)
	}
}

// 2. The x of a tab not shown closes its page, asking first; the page
// shown stays, and OnSwitch does not run.
func TestClosableTabsXClosesItsPage(t *testing.T) {
	d := newDocs(t, true)
	clickX(t, d.tt, "Beta")
	d.flush(t)
	if a := d.askedPages(); len(a) != 1 || a[0] != Control(d.pages[1]) {
		t.Fatalf("OnPageClosing asked about %v, want Beta's page once", a)
	}
	d.state(t, []string{"Alpha", "Gamma"}, 0)
	if n := d.switches.Load(); n != 0 {
		t.Fatalf("OnSwitch ran %d times", n)
	}
	mu.Lock()
	has := slices.Contains(d.nb.children(), Control(d.pages[1]))
	mu.Unlock()
	if has {
		t.Fatal("the control still holds Beta's page")
	}
}

// 3. The x of the shown page closes it: the page shown is what RemovePage
// gives (the one before, or the first), and OnSwitch runs once.
func TestClosableTabsXOfTheShownPage(t *testing.T) {
	d := newDocs(t, true)
	clickX(t, d.tt, "Alpha")
	d.flush(t)
	d.state(t, []string{"Beta", "Gamma"}, 0)
	if !d.tt.HasText("Beta") || d.switches.Load() != 1 {
		t.Fatalf("OnSwitch ran %d times, want 1", d.switches.Load())
	}
	// The shown last one: the one before it is shown.
	d2 := newDocs(t, true)
	d2.nb.SetPage(2)
	settle(d2.tt)
	clickX(t, d2.tt, "Gamma")
	d2.flush(t)
	d2.state(t, []string{"Alpha", "Beta"}, 1)
	if n := d2.switches.Load(); n != 1 {
		t.Fatalf("OnSwitch ran %d times, want 1", n)
	}
}

// Mine: a close before the page shown moves Page, and the page shown is the
// same: OnSwitch does not run.
func TestClosableTabsCloseBeforeTheShownOne(t *testing.T) {
	d := newDocs(t, true)
	d.nb.SetPage(2)
	settle(d.tt)
	clickX(t, d.tt, "Alpha")
	d.flush(t)
	d.state(t, []string{"Beta", "Gamma"}, 1)
	if n := d.switches.Load(); n != 0 {
		t.Fatalf("OnSwitch ran %d times for a close that kept the page shown", n)
	}
}

// 4. OnPageClosing returning false keeps the page; with none, pages close at
// once.
func TestClosableTabsOnPageClosingKeeps(t *testing.T) {
	d := newDocs(t, true)
	d.answer = false
	clickX(t, d.tt, "Alpha")
	d.flush(t)
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
	if len(d.askedPages()) != 1 || d.switches.Load() != 0 {
		t.Fatalf("asked %d times, OnSwitch ran %d times", len(d.askedPages()), d.switches.Load())
	}
	d.nb.OnPageClosing(nil)
	clickX(t, d.tt, "Beta")
	d.flush(t)
	d.state(t, []string{"Alpha", "Gamma"}, 0)
	if len(d.askedPages()) != 1 {
		t.Fatal("OnPageClosing(nil) did not remove the handler")
	}
}

// 5. A close removes the page it was asked for, by its tab, whatever the
// handler does to the pages.
func TestClosableTabsCloseTheTabAskedFor(t *testing.T) {
	t.Run("the same widget on two pages", func(t *testing.T) {
		d := newDocs(t, true)
		d.nb.AddPage("Again", d.pages[0])
		settle(d.tt)
		clickX(t, d.tt, "Again")
		d.flush(t)
		d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
		if !d.tt.HasText("Alpha") {
			t.Fatal("Alpha's tab went")
		}
	})
	t.Run("a page inserted before it while asked", func(t *testing.T) {
		d := newDocs(t, true)
		d.nb.OnPageClosing(func(p Control) bool {
			d.nb.InsertPage(0, "New", Label("new"))
			return true
		})
		clickX(t, d.tt, "Beta")
		d.flush(t)
		names, _ := tabNames(d.tt)
		if !slices.Equal(names, []string{"New", "Alpha", "Gamma"}) {
			t.Fatalf("tabs %q", names)
		}
	})
	t.Run("the page removed while asked", func(t *testing.T) {
		d := newDocs(t, true)
		d.nb.OnPageClosing(func(p Control) bool {
			d.nb.RemovePage(1)
			return true
		})
		clickX(t, d.tt, "Beta")
		d.flush(t)
		d.state(t, []string{"Alpha", "Gamma"}, 0)
	})
}

// 6. A middle click on a tab closes it once, however many frames the button
// is held; without SetClosable it does nothing.
func TestClosableTabsMiddleClickClosesOnce(t *testing.T) {
	d := newDocs(t, true)
	middleClick(t, d.tt, "Gamma")
	d.flush(t)
	d.state(t, []string{"Alpha", "Beta"}, 0)
	if a := d.askedPages(); len(a) != 1 || a[0] != Control(d.pages[2]) {
		t.Fatalf("asked about %d pages, want Gamma's once", len(a))
	}
	plain := newDocs(t, false)
	middleClick(t, plain.tt, "Gamma")
	plain.flush(t)
	plain.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
}

// Mine: a page kept is asked about once a press, however long the press,
// and again on the next press.
func TestClosableTabsMiddleClickAsksOncePerPress(t *testing.T) {
	d := newDocs(t, true)
	d.answer = false
	middleClick(t, d.tt, "Gamma")
	d.flush(t)
	if n := len(d.askedPages()); n != 1 {
		t.Fatalf("asked %d times for one press held 3 frames", n)
	}
	middleClick(t, d.tt, "Gamma")
	d.flush(t)
	if n := len(d.askedPages()); n != 2 {
		t.Fatalf("asked %d times after a second press, want 2", n)
	}
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
}

// Mine: a click on a closable tab shows its page and runs OnSwitch once.
func TestClosableTabsClickChoosesATab(t *testing.T) {
	d := newDocs(t, true)
	click(t, d.tt, "Beta")
	d.flush(t)
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 1)
	if n := d.switches.Load(); n != 1 || len(d.askedPages()) != 0 {
		t.Fatalf("OnSwitch ran %d times, asked %d", n, len(d.askedPages()))
	}
}

// Mine: the focus stays on its tab when as many pages are added as
// removed before a frame.
func TestClosableTabsFocusFollowsItsTab(t *testing.T) {
	d := newDocs(t, true)
	click(t, d.tt, "Gamma")
	settle(d.tt)
	d.nb.RemovePage(0).AddPage("Delta", Label("d"))
	settle(d.tt)
	for _, n := range d.tt.Nodes() {
		if n.Role == ui.RoleTab && n.Focused {
			if n.Label != "Gamma" {
				t.Fatalf("focus on %q, want Gamma", n.Label)
			}
			return
		}
	}
	t.Fatal("no tab has the focus")
}

// 7. Ctrl+F4 closes the page shown, with the focus on the strip or in the
// page; without SetClosable it does nothing.
func TestClosableTabsCtrlF4ClosesThePageShown(t *testing.T) {
	t.Run("focus on the strip", func(t *testing.T) {
		d := newDocs(t, true)
		click(t, d.tt, "Beta")
		settle(d.tt)
		d.tt.Key(ui.Ctrl, ui.KeyF4)
		d.flush(t)
		d.state(t, []string{"Alpha", "Gamma"}, 0)
	})
	t.Run("focus in the page", func(t *testing.T) {
		d := newDocs(t, true)
		d.focusBox(t, 0)
		d.tt.Key(ui.Ctrl, ui.KeyF4)
		d.flush(t)
		d.state(t, []string{"Beta", "Gamma"}, 0)
	})
	t.Run("not closable", func(t *testing.T) {
		d := newDocs(t, false)
		d.focusBox(t, 0)
		d.tt.Key(ui.Ctrl, ui.KeyF4)
		d.flush(t)
		d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
	})
}

// 8. Ctrl+Tab shows the next page, wrapping, Ctrl+Shift+Tab the previous,
// each running OnSwitch once, in the page too; without SetClosable nothing.
func TestClosableTabsCtrlTabSwitches(t *testing.T) {
	d := newDocs(t, true)
	click(t, d.tt, "Alpha")
	settle(d.tt)
	for i, want := range []int{1, 2, 0} {
		d.tt.Key(ui.Ctrl, ui.KeyTab)
		d.flush(t)
		if d.nb.Page() != want || d.switches.Load() != int32(i+1) {
			t.Fatalf("Ctrl+Tab %d: Page %d, OnSwitch %d; want %d, %d", i+1, d.nb.Page(), d.switches.Load(), want, i+1)
		}
	}
	d.tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	d.flush(t)
	if d.nb.Page() != 2 || d.switches.Load() != 4 {
		t.Fatalf("Ctrl+Shift+Tab: Page %d, OnSwitch %d; want 2, 4", d.nb.Page(), d.switches.Load())
	}
	d.focusBox(t, 2)
	d.tt.Key(ui.Ctrl, ui.KeyTab)
	d.flush(t)
	if d.nb.Page() != 0 || d.switches.Load() != 5 || !d.tt.HasText("Alpha") {
		t.Fatalf("Ctrl+Tab in the page: Page %d, OnSwitch %d", d.nb.Page(), d.switches.Load())
	}

	plain := newDocs(t, false)
	plain.focusBox(t, 0)
	plain.tt.Key(ui.Ctrl, ui.KeyTab)
	plain.flush(t)
	if plain.nb.Page() != 0 || plain.switches.Load() != 0 {
		t.Fatalf("not closable: Ctrl+Tab gave Page %d, OnSwitch %d", plain.nb.Page(), plain.switches.Load())
	}
}

// Mine: with one page, Ctrl+Tab stays on it and runs no OnSwitch.
func TestClosableTabsCtrlTabOnOnePage(t *testing.T) {
	d := newDocs(t, true)
	d.nb.RemovePage(2).RemovePage(1)
	settle(d.tt)
	d.focusBox(t, 0)
	d.tt.Key(ui.Ctrl, ui.KeyTab)
	d.tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	d.flush(t)
	if d.nb.Page() != 0 || d.switches.Load() != 0 {
		t.Fatalf("Page %d, OnSwitch %d", d.nb.Page(), d.switches.Load())
	}
}

// 9. Closing the last page leaves none, without a panic; the keys then do
// nothing.
func TestClosableTabsCloseTheLastPage(t *testing.T) {
	d := newDocs(t, true)
	d.nb.RemovePage(2).RemovePage(1)
	settle(d.tt)
	clickX(t, d.tt, "Alpha")
	d.flush(t)
	if names, _ := tabNames(d.tt); len(names) != 0 || d.nb.Page() != 0 {
		t.Fatalf("tabs %q, Page %d", names, d.nb.Page())
	}
	sw := d.switches.Load()
	d.tt.Key(ui.Ctrl, ui.KeyTab)
	d.tt.Key(ui.Ctrl, ui.KeyF4)
	d.flush(t)
	if len(d.askedPages()) != 1 || d.switches.Load() != sw || d.nb.Page() != 0 {
		t.Fatalf("on no page: asked %d, OnSwitch %d (was %d), Page %d", len(d.askedPages()), d.switches.Load(), sw, d.nb.Page())
	}
}

// accessTree is what tt's nodes say of the tree: roles, labels, values and
// bounds, without the ids.
func accessTree(tt *ui.Tester) []string {
	var out []string
	for _, n := range tt.Nodes() {
		out = append(out, fmt.Sprintf("%v %q %q %v %v %v", n.Role, n.Label, n.Value, n.Bounds, n.Checked, n.Focused))
	}
	return out
}

// 10. SetClosable(false) after true takes the x and the keys away, and the
// control builds as one never made closable.
func TestClosableTabsTurnedOff(t *testing.T) {
	d := newDocs(t, true)
	d.nb.SetClosable(false)
	settle(d.tt)
	if n := xCount(d.tt); n != 0 {
		t.Fatalf("%d x after SetClosable(false)", n)
	}
	middleClick(t, d.tt, "Gamma")
	d.focusBox(t, 0)
	d.tt.Key(ui.Ctrl, ui.KeyF4)
	d.tt.Key(ui.Ctrl, ui.KeyTab)
	d.flush(t)
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
	if len(d.askedPages()) != 0 || d.switches.Load() != 0 {
		t.Fatalf("asked %d, OnSwitch %d", len(d.askedPages()), d.switches.Load())
	}
	plain := newDocs(t, false)
	plain.focusBox(t, 0)
	if a, b := accessTree(d.tt), accessTree(plain.tt); !slices.Equal(a, b) {
		t.Fatalf("turned off:\n%q\nnever closable:\n%q", a, b)
	}
	mu.Lock()
	hook := d.nb.closable
	mu.Unlock()
	if hook != nil {
		t.Fatal("SetClosable(false) kept the closable build")
	}
}

// tabInk returns the pixel of the tab's label farthest from bg (dist):
// between the tab's start padding and its x.
func tabInk(tt *ui.Tester, b ui.Rect, bg color.RGBA) color.RGBA {
	far, best := bg, -1
	for y := int(b.Y + 2); y < int(b.Y+b.H-2); y++ {
		for x := int(b.X + space(3)); x < int(b.X+b.W-space(2)-12-2*space(0.5)-space(1)); x++ {
			c := tt.Image().RGBAAt(x, y)
			if d := dist(c, ui.Color{R: bg.R, G: bg.G, B: bg.B, A: 255}); d > best {
				far, best = c, d
			}
		}
	}
	return far
}

// 11. Without a skin, the shown tab fills with Background and has Border on
// its sides, its label in Text; another is unfilled, its label in TextMuted,
// and SurfaceHover when hovered. Under Classic, the shown tab fills as
// without SetClosable.
func TestClosableTabsLook(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint("dark ", dark), func(t *testing.T) {
			th := ui.LightTheme()
			if dark {
				th = ui.DarkTheme()
			}
			d := newDocs(t, true)
			d.tt.SetDark(dark)
			settle(d.tt)
			img := d.tt.Image()
			a, b := tabBounds(t, d.tt, "Alpha"), tabBounds(t, d.tt, "Beta")
			fill := img.RGBAAt(int(a.X+4), int(a.Y+4))
			if !near(fill, th.Background) {
				t.Errorf("the shown tab's fill %v, want Background %v", fill, th.Background)
			}
			if side := img.RGBAAt(int(a.X), int(a.Y+a.H/2)); !near(side, th.Border) {
				t.Errorf("the shown tab's side %v, want Border %v", side, th.Border)
			}
			if c := tabInk(d.tt, a, fill); dist(c, th.Text) >= dist(c, th.TextMuted) {
				t.Errorf("the shown label's ink %v, nearer TextMuted %v than Text %v", c, th.TextMuted, th.Text)
			}
			strip := img.RGBAAt(int(b.X+b.W+60), int(b.Y+4)) // past the last tab
			other := img.RGBAAt(int(b.X+4), int(b.Y+4))
			if other != strip || near(other, th.Border) && img.RGBAAt(int(b.X), int(b.Y+b.H/2)) == other {
				t.Errorf("another tab is filled %v, the strip %v", other, strip)
			}
			if side := img.RGBAAt(int(b.X), int(b.Y+b.H/2)); side != strip {
				t.Errorf("another tab has a side %v", side)
			}
			if c := tabInk(d.tt, b, other); dist(c, th.TextMuted) >= dist(c, th.Text) {
				t.Errorf("another label's ink %v, nearer Text %v than TextMuted %v", c, th.Text, th.TextMuted)
			}
			d.tt.Move(b.X+4, b.Y+b.H/2)
			settle(d.tt)
			if c := d.tt.Image().RGBAAt(int(b.X+4), int(b.Y+4)); !near(c, th.SurfaceHover) {
				t.Errorf("another tab hovered %v, want SurfaceHover %v", c, th.SurfaceHover)
			}
		})
	}
	t.Run("Classic", func(t *testing.T) {
		useSkinNamed(t, "Classic", false)
		d, plain := newDocs(t, true), newDocs(t, false)
		a, p := tabBounds(t, d.tt, "Alpha"), tabBounds(t, plain.tt, "Alpha")
		got := d.tt.Image().RGBAAt(int(a.X+4), int(a.Y+a.H/2))
		want := plain.tt.Image().RGBAAt(int(p.X+4), int(p.Y+p.H/2))
		if got != want {
			t.Fatalf("the shown tab under Classic %v, without SetClosable %v", got, want)
		}
		if n := xCount(d.tt); n != 3 {
			t.Fatalf("%d x under Classic", n)
		}
	})
}

// 12. A TabControl never made closable has no closable build.
func TestTabControlNeverClosableHasNoHook(t *testing.T) {
	d := newDocs(t, false)
	mu.Lock()
	hook := d.nb.closable
	mu.Unlock()
	if hook != nil {
		t.Fatal("the closable build is set without SetClosable")
	}
}

// 13. A closable strip's frame at rest allocates no more than ui's bare tabs
// with a ButtonBase in each, beyond what a TabControl does without
// SetClosable.
func TestClosableTabsFrameAllocations(t *testing.T) {
	labels := []string{"Alpha", "Beta", "Gamma"}
	keys := []any{new(pageKey), new(pageKey), new(pageKey)}
	bare := func(withX bool) float64 {
		sel := 0
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Children(func() {
				tabs := ui.TabsKeyedBase(c, &sel, keys)
				tabs.List.Children(func() {
					for i, label := range labels {
						tabs.Tab(i).Children(func() {
							ui.Text(c, label).SingleLine()
							if withX {
								ui.ButtonBase(c).Children(func() { ui.Icon(c, closeIcon).FontSize(12) })
							}
						})
					}
				})
				ui.Text(c, "page")
			})
		}, 500, 300)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(30, tt.Frame)
	}
	control := func(closable bool) float64 {
		nb := TabControl().AddPage("Alpha", Label("page")).AddPage("Beta", Label("b")).AddPage("Gamma", Label("g")).SetClosable(closable)
		tt := ui.NewTester(content(nb), 500, 300)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(30, tt.Frame)
	}
	ref, got := bare(true), control(true)
	if got > ref {
		t.Fatalf("a closable TabControl's frame allocates %v times; ui's bare tabs with an x in each, %v", got, ref)
	}
	t.Logf("closable %v, not closable %v; ui's bare tabs with an x %v, without %v", got, control(false), ref, bare(false))
}

// 14. The setters from another goroutine while frames run, shown on a
// following frame.
func TestClosableTabsSettersFromAnotherGoroutine(t *testing.T) {
	d := newDocs(t, false)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 50 {
			d.nb.SetClosable(i%2 == 0).OnPageClosing(func(Control) bool { return true })
			d.nb.AddPage(fmt.Sprint("P", i), Label("p"))
			d.nb.RemovePage(3)
		}
		d.nb.SetClosable(true)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			d.tt.Frame()
		}
	}
	settle(d.tt)
	if n := xCount(d.tt); n != 3 {
		t.Fatalf("%d x after the goroutine's SetClosable(true), want 3", n)
	}
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
}

// Review Focus 1. OnPageClosing showing Confirm: the tab stays while the box
// shows; Yes closes it, No keeps it.
func TestClosableTabsConfirm(t *testing.T) {
	for _, yes := range []bool{true, false} {
		t.Run(fmt.Sprint("yes ", yes), func(t *testing.T) {
			d := newDocs(t, true)
			title := fmt.Sprint("Close Beta? ", yes)
			d.nb.OnPageClosing(func(Control) bool { return Confirm(nil, Question, title, "Its changes are lost.") })
			clickX(t, d.tt, "Beta")
			fw := waitFake(t, title)
			settle(d.tt)
			d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
			answer := "No"
			if yes {
				answer = "Yes"
			}
			press(t, fw, answer)
			d.flush(t)
			if yes {
				d.state(t, []string{"Alpha", "Gamma"}, 0)
			} else {
				d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
			}
		})
	}
}

// Review Focus 2. Ten tabs at text scale 1.5 in 400 DIPs: no panic, and each
// x shown whole closes its own tab.
func TestClosableTabsManyAtLargeText(t *testing.T) {
	nb := TabControl().SetClosable(true)
	for i := range 10 {
		nb.AddPage(fmt.Sprint("Doc ", i), Label(fmt.Sprint("page ", i)))
	}
	tt := ui.NewTester(NewForm("Many", 400, 300).SetContent(nb).view, 400, 300)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	settle(tt)
	w := tabBounds(t, tt, "Doc 0").W
	closed := 0
	for i := range 10 {
		label := fmt.Sprint("Doc ", i)
		b := tabBounds(t, tt, label)
		if b.W < w-0.5 || b.X+b.W > 400 {
			break // clipped by the strip
		}
		x, y := xAt(t, tt, label, 1.5)
		clickAt(tt, x, y)
		settle(tt)
		flushEvents(t)
		settle(tt)
		if names, _ := tabNames(tt); slices.Contains(names, label) || len(names) != 9-closed {
			t.Fatalf("the x of %s: tabs %q", label, names)
		}
		closed++
	}
	if closed == 0 {
		t.Fatal("no tab shown whole")
	}
}

// Review Focus 3. A disabled control: the x, a middle click and the keys do
// nothing.
func TestClosableTabsDisabled(t *testing.T) {
	d := newDocs(t, true)
	d.focusBox(t, 0) // the focus in the page as it is disabled
	d.nb.SetDisabled(true)
	settle(d.tt)
	clickX(t, d.tt, "Beta")
	middleClick(t, d.tt, "Gamma")
	d.tt.Key(ui.Ctrl, ui.KeyF4)
	d.tt.Key(ui.Ctrl, ui.KeyTab)
	d.flush(t)
	d.state(t, []string{"Alpha", "Beta", "Gamma"}, 0)
	if len(d.askedPages()) != 0 || d.switches.Load() != 0 {
		t.Fatalf("asked %d, OnSwitch %d", len(d.askedPages()), d.switches.Load())
	}
}

// Review Focus 4. Two closes of one page before the first answer: it closes
// once, OnPageClosing runs at most twice, and nothing else goes.
func TestClosableTabsTwoClosesOfOnePage(t *testing.T) {
	d := newDocs(t, true)
	gate := make(chan struct{})
	var calls atomic.Int32
	d.nb.OnPageClosing(func(Control) bool {
		if calls.Add(1) == 1 {
			<-gate
		}
		return true
	})
	clickX(t, d.tt, "Alpha")
	d.tt.Key(ui.Ctrl, ui.KeyF4)
	settle(d.tt)
	close(gate)
	d.flush(t)
	d.state(t, []string{"Beta", "Gamma"}, 0)
	if n := calls.Load(); n < 1 || n > 2 {
		t.Fatalf("OnPageClosing ran %d times", n)
	}
}

// Review Focus 5. The tabs read as before to assistive technology.
func TestClosableTabsAccess(t *testing.T) {
	d := newDocs(t, true)
	if _, ok := d.tt.AccessByRole(ui.RoleTab, "Beta"); !ok {
		t.Fatal("no tab Beta for assistive technology")
	}
	for _, n := range d.tt.Nodes() {
		if n.Label == "Close" {
			t.Fatalf("the x is in the access tree: %+v", n)
		}
	}
}

// Mine: OnPageClosing runs on the event goroutine.
func TestClosableTabsAskOnTheEventGoroutine(t *testing.T) {
	d := newDocs(t, true)
	on := make(chan bool, 1)
	d.nb.OnPageClosing(func(Control) bool { on <- onEvents(); return false })
	clickX(t, d.tt, "Beta")
	if !received(t, on) {
		t.Fatal("OnPageClosing ran off the event goroutine")
	}
}
