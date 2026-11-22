package widget

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// navIcon is the icon of the cases' items.
var navIcon = lucide(`<circle cx="12" cy="12" r="8"/>`)

// mailBar is the NavBar of the cases: Mail (Inbox with an icon, Sent) and
// Contacts (People).
func mailBar() *NavBarWidget {
	return NavBar().AddGroup("Mail").AddItem("inbox", "Inbox", navIcon).AddItem("sent", "Sent", nil).
		AddGroup("Contacts").AddItem("people", "People", nil)
}

// selections records the keys OnSelect ran with.
type selections struct {
	mu   sync.Mutex
	keys []string
}

func (s *selections) add(k string) { s.mu.Lock(); s.keys = append(s.keys, k); s.mu.Unlock() }

func (s *selections) got(t *testing.T) []string {
	t.Helper()
	flushEvents(t) // what a frame posted has run
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.keys)
}

// navTester shows w in a form of 240 by 400, with a control beside it that
// keeps the frame's theme in the returned Theme.
func navTester(t *testing.T, w Control) (*Tester, *ui.Theme) {
	t.Helper()
	th := new(ui.Theme)
	keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
	tt := NewTester(t, VBox(0).Add(keep, w), 240, 400)
	tt.Settle()
	return tt, th
}

// headerNames are the names of the NavBar's headers, in order.
func headerNames(tt *Tester) []string {
	var names []string
	for _, n := range tt.FindAll(ByRole(ui.RoleDisclosure)) {
		names = append(names, n.Label())
	}
	return names
}

// item finds the item named s.
func item(tt *Tester, s string) *Node { return tt.Find(ByRole(ui.RoleButton).Label(s)) }

// header finds the header named s.
func header(tt *Tester, s string) *Node { return tt.Find(ByRole(ui.RoleDisclosure).Label(s)) }

// currents are the names of the elements that are the current page.
func currents(tt *Tester) []string {
	var names []string
	for _, a := range tt.AccessAll() {
		if a.Current == ui.CurrentPage {
			names = append(names, a.Name)
		}
	}
	return names
}

// closeRuns lets a group's close or open run to its end.
func closeRuns(tt *Tester) {
	tt.Settle() // a frame starts it: a setter's change shows in the next
	tt.Advance(time.Second)
	tt.Settle()
}

// navSelected is the fill and the text of a selected row in th: RowSelected
// and RowSelectedText (Text when zero), or the accent and AccentText where
// the theme sets no RowSelected (ui/theme.go: "Left zero, they take the
// accent and AccentText").
func navSelected(th *ui.Theme) (bg, fg ui.Color) {
	if th.RowSelected.A == 0 {
		return th.Accent, th.AccentText
	}
	fg = th.RowSelectedText
	if fg.A == 0 {
		fg = th.Text
	}
	// Over the page, as a skin's may be translucent (Classic's in dark).
	return th.RowSelected.Over(th.Background), fg
}

// rowFill is the color of the item named s at its end, past its text.
func rowFill(tt *Tester, s string) color.RGBA {
	b := item(tt, s).Bounds()
	return tt.Image().RGBAAt(int(b.X+b.W-4), int(b.Y+b.H/2))
}

// navInk is the pixel of the text s farthest from bg: its ink.
func navInk(t *testing.T, tt *Tester, s string, bg ui.Color) color.RGBA {
	t.Helper()
	r, ok := tt.Tester.Find(s)
	if !ok {
		t.Fatalf("no text %q", s)
	}
	var best color.RGBA
	far := -1
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := tt.Image().RGBAAt(x, y)
			if d := dist(c, bg); d > far {
				best, far = c, d
			}
		}
	}
	return best
}

func dist(c color.RGBA, u ui.Color) int {
	a := func(x, y uint8) int {
		d := int(x) - int(y)
		return d * d
	}
	return a(c.R, u.R) + a(c.G, u.G) + a(c.B, u.B)
}

// Case 1.
func TestNavBarTree(t *testing.T) {
	n := mailBar()
	tt, _ := navTester(t, n)
	if got := headerNames(tt); !slices.Equal(got, []string{"Mail", "Contacts"}) {
		t.Fatalf("headers %q, want Mail and Contacts", got)
	}
	order := []*Node{header(tt, "Mail"), item(tt, "Inbox"), item(tt, "Sent"), header(tt, "Contacts"), item(tt, "People")}
	for i := 1; i < len(order); i++ {
		if a, b := order[i-1].Bounds(), order[i].Bounds(); b.Y < a.Y+a.H-0.5 {
			t.Errorf("%s at y %v is not below %s (%v to %v)", order[i], b.Y, order[i-1], a.Y, a.Y+a.H)
		}
	}
	for _, h := range []string{"Mail", "Contacts"} {
		if a, _ := tt.AccessByRole(ui.RoleDisclosure, h); !a.Expanded {
			t.Errorf("%s is not expanded", h)
		}
	}
	if !n.GroupOpen(0) || !n.GroupOpen(1) {
		t.Errorf("GroupOpen %v %v, want both open", n.GroupOpen(0), n.GroupOpen(1))
	}
	if s := n.Selected(); s != "" {
		t.Errorf("Selected %q at first, want none", s)
	}
	if c := currents(tt); len(c) != 0 {
		t.Errorf("current %q at first, want none", c)
	}
}

// Case 2.
func TestNavBarClickSelectsOnce(t *testing.T) {
	var sel selections
	n := mailBar().OnSelect(sel.add)
	tt, _ := navTester(t, n)
	item(tt, "Sent").Click()
	if s := n.Selected(); s != "sent" {
		t.Fatalf("Selected %q, want sent", s)
	}
	if got := sel.got(t); !slices.Equal(got, []string{"sent"}) {
		t.Fatalf("OnSelect ran with %q, want sent once", got)
	}
	item(tt, "Sent").Click()
	if got := sel.got(t); !slices.Equal(got, []string{"sent"}) {
		t.Fatalf("a click on the selected item ran OnSelect: %q", got)
	}
}

// Case 3.
func TestNavBarOneCurrent(t *testing.T) {
	var sel selections
	n := mailBar().OnSelect(sel.add)
	tt, th := navTester(t, n)
	item(tt, "Inbox").Click()
	if c := currents(tt); !slices.Equal(c, []string{"Inbox"}) {
		t.Fatalf("current %q after the click on Inbox, want Inbox", c)
	}
	// Every frame from the click on, not only the settled one: the item
	// selected before must not stay current in the frame of the click.
	b := item(tt, "People").Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	if c := currents(tt); len(c) > 1 {
		t.Fatalf("current %q in the frame of the click, want one", c)
	}
	tt.Settle()
	if got := sel.got(t); !slices.Equal(got, []string{"inbox", "people"}) {
		t.Fatalf("OnSelect ran with %q, want inbox then people", got)
	}
	if c := currents(tt); !slices.Equal(c, []string{"People"}) {
		t.Fatalf("current %q, want People alone", c)
	}
	bg, _ := navSelected(th)
	if got := rowFill(tt, "People"); !near(got, bg) {
		t.Errorf("People fills with %v, want the selected fill %v", got, bg)
	}
	if got := rowFill(tt, "Inbox"); near(got, bg) {
		t.Errorf("Inbox still fills with the selected fill %v", got)
	}
}

// Case 4.
func TestNavBarSetSelected(t *testing.T) {
	var sel selections
	n := mailBar().OnSelect(sel.add)
	tt, _ := navTester(t, n)
	n.SetSelected("people")
	tt.Settle()
	if s := n.Selected(); s != "people" {
		t.Fatalf("Selected %q, want people", s)
	}
	if c := currents(tt); !slices.Equal(c, []string{"People"}) {
		t.Fatalf("current %q, want People", c)
	}
	if got := sel.got(t); len(got) != 0 {
		t.Fatalf("SetSelected ran OnSelect with %q", got)
	}
	n.SetSelected("nope")
	tt.Settle()
	if s := n.Selected(); s != "people" {
		t.Fatalf("SetSelected(nope) changed the selection to %q", s)
	}
	n.SetSelected("")
	tt.Settle()
	if s := n.Selected(); s != "" {
		t.Fatalf("SetSelected(\"\") left %q", s)
	}
	if c := currents(tt); len(c) != 0 {
		t.Fatalf("current %q after SetSelected(\"\")", c)
	}
	if got := sel.got(t); len(got) != 0 {
		t.Fatalf("SetSelected ran OnSelect with %q", got)
	}
}

// Case 5.
func TestNavBarSelectedAndHoverColors(t *testing.T) {
	for _, skin := range []string{"", "Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			name := skin
			if name == "" {
				name = "none"
			}
			t.Run(fmt.Sprintf("%s/dark=%v", name, dark), func(t *testing.T) {
				if skin != "" {
					useSkinNamed(t, skin, false)
				} else {
					calmOff()
				}
				n := mailBar().SetSelected("sent")
				tt, th := navTester(t, n)
				tt.SetDark(dark)
				tt.Settle()
				if th.Dark != dark {
					t.Fatalf("theme dark %v, want %v", th.Dark, dark)
				}
				if skin != "" && th.RowSelected.A == 0 {
					t.Fatalf("the skin sets no RowSelected")
				}
				// HighContrast inverts the text of a selected row; Classic's
				// RowSelected is a tint of its accent (skin/palette.go), so its
				// text stays Text.
				if skin == "HighContrast" && th.RowSelectedText == th.Text {
					t.Fatalf("HighContrast's RowSelectedText is its Text: the case cannot tell them apart")
				}
				bg, fg := navSelected(th)
				if got := rowFill(tt, "Sent"); !near(got, bg) {
					t.Errorf("the selected item fills with %v, want %v", got, bg)
				}
				alt := th.Text
				if fg == th.Text {
					alt = th.Background
				}
				if got := navInk(t, tt, "Sent", bg); dist(got, fg) >= dist(got, alt) {
					t.Errorf("the selected text inks %v, nearer %v than %v", got, alt, fg)
				}
				hover := th.RowHover
				if hover.A == 0 {
					hover = th.SurfaceHover
				}
				hover = hover.Over(th.Background)
				b := item(tt, "Inbox").Bounds()
				tt.Move(b.X+b.W-4, b.Y+b.H/2)
				tt.Advance(time.Second)
				tt.Settle()
				if got := rowFill(tt, "Inbox"); !near(got, hover) {
					t.Errorf("the hovered item fills with %v, want %v", got, hover)
				}
			})
		}
	}
}

// Case 6.
func TestNavBarGroupsOpenAndClose(t *testing.T) {
	n := mailBar()
	tt, _ := navTester(t, n)
	header(tt, "Mail").Click()
	closeRuns(tt)
	if tt.HasText("Inbox") || tt.HasText("Sent") {
		t.Fatalf("Mail's items stay after it closed: %q", tt.Texts())
	}
	if n.GroupOpen(0) {
		t.Fatal("GroupOpen(0) after the close")
	}
	header(tt, "Mail").Click()
	closeRuns(tt)
	if !tt.HasText("Inbox") || !tt.HasText("Sent") || !n.GroupOpen(0) {
		t.Fatalf("Mail did not open again: open %v, %q", n.GroupOpen(0), tt.Texts())
	}
	n.SetGroupOpen(1, false)
	closeRuns(tt)
	if tt.HasText("People") || n.GroupOpen(1) {
		t.Fatalf("SetGroupOpen(1, false): open %v, %q", n.GroupOpen(1), tt.Texts())
	}
	if n.GroupOpen(5) {
		t.Fatal("GroupOpen(5) is true")
	}
	n.SetGroupOpen(5, true)
	closeRuns(tt)
	if !tt.HasText("Inbox") || tt.HasText("People") {
		t.Fatalf("SetGroupOpen(5, true) changed the groups: %q", tt.Texts())
	}
}

// Case 6's bounds, each side: the last index is a group, the next is not.
func TestNavBarGroupOpenBounds(t *testing.T) {
	n := mailBar()
	tt, _ := navTester(t, n)
	if !n.GroupOpen(1) {
		t.Fatal("GroupOpen(1), the last group, is false")
	}
	if n.GroupOpen(2) || n.GroupOpen(-1) {
		t.Fatalf("GroupOpen(2) %v, GroupOpen(-1) %v: not groups", n.GroupOpen(2), n.GroupOpen(-1))
	}
	n.SetGroupOpen(2, false).SetGroupOpen(-1, false)
	closeRuns(tt)
	if !n.GroupOpen(0) || !n.GroupOpen(1) || !tt.HasText("People") {
		t.Fatal("an index out of range closed a group")
	}
}

// Case 7.
func TestNavBarKeys(t *testing.T) {
	var sel selections
	n := mailBar().OnSelect(sel.add)
	tt, _ := navTester(t, n)
	header(tt, "Mail").Key(0, ui.KeyDown)
	if !header(tt, "Contacts").Focused() {
		t.Fatal("Down on Mail did not focus Contacts")
	}
	header(tt, "Contacts").Key(0, ui.KeyUp)
	if !header(tt, "Mail").Focused() {
		t.Fatal("Up on Contacts did not focus Mail")
	}
	header(tt, "Mail").Key(0, ui.KeyEnter)
	closeRuns(tt)
	if n.GroupOpen(0) {
		t.Fatal("Enter on Mail did not close it")
	}
	header(tt, "Mail").Key(0, ui.KeyEnter)
	closeRuns(tt)
	if !n.GroupOpen(0) {
		t.Fatal("Enter on Mail did not open it again")
	}
	header(tt, "Mail").Key(0, ui.KeyTab)
	if !item(tt, "Inbox").Focused() {
		t.Fatal("Tab from Mail did not go to Inbox")
	}
	item(tt, "Inbox").Key(0, ui.KeyDown)
	if !item(tt, "Sent").Focused() {
		t.Fatal("Down on Inbox did not focus Sent")
	}
	item(tt, "Sent").Key(0, ui.KeyDown)
	if !item(tt, "Sent").Focused() {
		t.Fatal("Down on Sent, the group's last, moved the focus")
	}
	item(tt, "Sent").Key(0, ui.KeyUp)
	if !item(tt, "Inbox").Focused() {
		t.Fatal("Up on Sent did not focus Inbox")
	}
	item(tt, "Inbox").Key(0, ui.KeyUp)
	if !item(tt, "Inbox").Focused() {
		t.Fatal("Up on Inbox, the group's first, moved the focus")
	}
	if got := sel.got(t); len(got) != 0 {
		t.Fatalf("the arrows selected %q", got)
	}
	item(tt, "Sent").Key(0, ui.KeyEnter)
	if s := n.Selected(); s != "sent" {
		t.Fatalf("Enter on Sent: Selected %q", s)
	}
	if got := sel.got(t); !slices.Equal(got, []string{"sent"}) {
		t.Fatalf("Enter on Sent ran OnSelect with %q, want sent once", got)
	}
}

// Case 8.
func TestNavBarItemBeforeAnyGroup(t *testing.T) {
	var sel selections
	n := NavBar().AddItem("home", "Home", nil).AddGroup("More").AddItem("x", "X", nil).OnSelect(sel.add)
	tt, _ := navTester(t, n)
	if got := headerNames(tt); !slices.Equal(got, []string{"More"}) {
		t.Fatalf("headers %q, want More alone", got)
	}
	if !n.GroupOpen(0) || !n.GroupOpen(1) {
		t.Fatalf("GroupOpen %v %v, want both open", n.GroupOpen(0), n.GroupOpen(1))
	}
	n.SetGroupOpen(0, false)
	closeRuns(tt)
	if !n.GroupOpen(0) || !tt.HasText("Home") {
		t.Fatalf("the group with no title closed: open %v, %q", n.GroupOpen(0), tt.Texts())
	}
	if h, x := item(tt, "Home").Bounds(), header(tt, "More").Bounds(); h.Y >= x.Y {
		t.Fatalf("Home at %v is not above More at %v", h.Y, x.Y)
	}
	item(tt, "Home").Click()
	if s := n.Selected(); s != "home" {
		t.Fatalf("Selected %q, want home", s)
	}
	item(tt, "Home").Click()
	if got := sel.got(t); !slices.Equal(got, []string{"home"}) {
		t.Fatalf("OnSelect ran with %q, want home once", got)
	}
}

// Case 9.
func TestNavBarDuplicateKeyPanics(t *testing.T) {
	n := mailBar()
	defer func() {
		v := recover()
		if s, _ := v.(string); !strings.Contains(s, `"sent"`) {
			t.Fatalf("AddItem of a key given before: recovered %v, want a panic naming \"sent\"", v)
		}
		// The lock is free: the NavBar still works.
		n.AddItem("drafts", "Drafts", nil)
	}()
	n.AddItem("sent", "Sent again", nil)
}

// Case 10, the first half.
func TestNavBarFocusGoesToTheFirstHeader(t *testing.T) {
	n := mailBar()
	tt, _ := navTester(t, n)
	n.Focus()
	tt.Settle()
	if !header(tt, "Mail").Focused() {
		t.Fatal("Focus did not focus the Mail header")
	}
}

// Case 10, the second half.
func TestNavBarAccessKeyFocusesTheFirstHeader(t *testing.T) {
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Go", mailBar()).Row("Other", Button("B")))
	tt.Find(ByLabel("B")).Focus()
	altKey(tt, ui.KeyG)
	if !header(tt, "Mail").Focused() {
		t.Fatal("Alt+G from a button did not focus the Mail header")
	}
}

// Review Focus 5.
func TestNavBarAccessKeyFromATextBox(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Go", mailBar()).Row("Name", TextBox()))
	tt.Find(ByRole(ui.RoleTextField)).Focus()
	altKey(tt, ui.KeyG)
	if !header(tt, "Mail").Focused() {
		t.Fatal("Alt+G from a text box did not focus the Mail header")
	}
}

// Case 11.
func TestNavBarDisabledAndHidden(t *testing.T) {
	var sel selections
	n := mailBar().OnSelect(sel.add)
	tt, _ := navTester(t, n)
	b := item(tt, "Sent").Bounds()
	n.SetDisabled(true)
	tt.Settle()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Settle()
	if s := n.Selected(); s != "" {
		t.Fatalf("a click on a disabled NavBar selected %q", s)
	}
	if got := sel.got(t); len(got) != 0 {
		t.Fatalf("a click on a disabled NavBar ran OnSelect with %q", got)
	}
	n.SetDisabled(false)
	tt.Settle()
	item(tt, "Sent").Click()
	if s := n.Selected(); s != "sent" {
		t.Fatalf("enabled again, a click selected %q", s)
	}
	n.SetVisible(false)
	tt.Settle()
	if tt.Has(ByRole(ui.RoleDisclosure)) || tt.HasText("Sent") {
		t.Fatalf("a hidden NavBar shows %q", tt.Texts())
	}
}

// bareAccordion builds what the cases' NavBar is made of with package ui
// alone: an AccordionBase of two groups whose headers hold a title and a
// chevron, and whose panels hold ButtonBase rows of an icon and a text.
func bareAccordion(c *ui.Context, open *[2]bool) {
	titles := [2]string{"Mail", "Contacts"}
	items := [2][]string{{"Inbox", "Sent"}, {"People"}}
	acc := ui.AccordionBase(c)
	acc.Root.Children(func() {
		for g := range 2 {
			it := acc.Item(&open[g])
			it.Trigger.Children(func() {
				ui.Text(c, titles[g]).Grow(1)
				ui.Icon(c, chevronDown).Rotate(180 * it.Progress())
			})
			it.Panel(func() {
				for _, s := range items[g] {
					ui.ButtonBase(c).Children(func() {
						ui.Icon(c, navIcon)
						ui.Text(c, s)
					})
				}
			})
		}
	})
}

// Case 12.
func TestNavBarFrameAllocations(t *testing.T) {
	open := [2]bool{true, true}
	bare := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { bareAccordion(c, &open) })
	}, 240, 400)
	nav := ui.NewTester(content(mailBar().SetSelected("sent")), 240, 400)
	for range 10 {
		bare.Frame()
		nav.Frame()
	}
	want, got := allocrun.Least(30, bare.Frame), allocrun.Least(30, nav.Frame)
	if got > want {
		t.Fatalf("a frame of the NavBar at rest allocates %v, more than a bare accordion's %v", got, want)
	}
	t.Logf("a frame allocates %v, a bare accordion's %v", got, want)
}

// Case 13.
func TestNavBarSettersFromAnotherGoroutine(t *testing.T) {
	n := mailBar()
	tt := NewTester(t, n, 240, 4000)
	tt.Settle()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			n.AddItem(fmt.Sprintf("k%d", i), fmt.Sprintf("Item %d", i), nil)
			n.SetSelected("k3")
			n.SetGroupOpen(1, i%2 == 0)
		}
		n.SetGroupOpen(1, true)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	closeRuns(tt)
	if !tt.HasText("Item 39") || n.Selected() != "k3" || !n.GroupOpen(1) {
		t.Fatalf("Selected %q, open %v, texts %q", n.Selected(), n.GroupOpen(1), tt.Texts())
	}
	if c := currents(tt); !slices.Equal(c, []string{"Item 3"}) {
		t.Fatalf("current %q, want Item 3", c)
	}
}

// Review Focus 1.
func TestNavBarLongTextsAtTextScale(t *testing.T) {
	title := "A group title far too long for the bar"
	text := "An item text far too long for the bar too"
	n := NavBar().AddGroup(title).AddItem("a", text, navIcon).AddItem("b", "Short", navIcon)
	tt := NewTester(t, n, 160, 600)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Settle()
	h := header(tt, title).Bounds()
	tr, ok := tt.Tester.Find(title)
	if !ok {
		t.Fatalf("no title text")
	}
	// The chevron is the theme's 3 units at the end, after a gap.
	if end := h.X + h.W - space(3); tr.X+tr.W > end+0.5 {
		t.Errorf("the title ends at %v, under the chevron, which starts by %v", tr.X+tr.W, end)
	}
	if tr.X+tr.W > 160 {
		t.Errorf("the title ends at %v, past the bar", tr.X+tr.W)
	}
	long, ok := tt.Tester.Find(text)
	if !ok {
		t.Fatalf("no item text")
	}
	short, _ := tt.Tester.Find("Short")
	if long.X+long.W > 160 {
		t.Errorf("the item's text ends at %v, past the bar", long.X+long.W)
	}
	if !sameSize(long.X, short.X) {
		t.Errorf("the long text starts at %v, the short at %v: the icon shrank", long.X, short.X)
	}
	if ib := item(tt, "Short").Bounds(); short.X-ib.X < 16*1.5 {
		t.Errorf("the text starts %v into its row, less than the icon's 24", short.X-ib.X)
	}
}

// Review Focus 2.
func TestNavBarTextsLineUpWithAndWithoutIcons(t *testing.T) {
	tt, _ := navTester(t, mailBar())
	in, _ := tt.Tester.Find("Inbox")
	sent, _ := tt.Tester.Find("Sent")
	if !sameSize(in.X, sent.X) {
		t.Fatalf("Inbox, with an icon, starts at %v, Sent, without, at %v", in.X, sent.X)
	}
	if ib := item(tt, "Inbox").Bounds(); in.X-ib.X < 16 {
		t.Fatalf("Inbox starts %v into its row, less than its icon", in.X-ib.X)
	}
}

// Review Focus 3.
func TestNavBarOnSelectCallsSetters(t *testing.T) {
	var n *NavBarWidget
	n = mailBar().OnSelect(func(string) {
		n.SetSelected("people").AddItem("new", "New", nil)
	})
	tt, _ := navTester(t, n)
	item(tt, "Sent").Click()
	tt.Settle()
	if s := n.Selected(); s != "people" || !tt.HasText("New") {
		t.Fatalf("Selected %q, texts %q", s, tt.Texts())
	}
	if c := currents(tt); !slices.Equal(c, []string{"People"}) {
		t.Fatalf("current %q, want People", c)
	}
}

// Review Focus 4.
func TestNavBarGroupWithNoItems(t *testing.T) {
	n := NavBar().AddGroup("Empty").AddGroup("Mail").AddItem("inbox", "Inbox", nil)
	tt, _ := navTester(t, n)
	header(tt, "Empty").Click()
	closeRuns(tt)
	if n.GroupOpen(0) {
		t.Fatal("the empty group did not close")
	}
	header(tt, "Empty").Click()
	closeRuns(tt)
	if !n.GroupOpen(0) {
		t.Fatal("the empty group did not open again")
	}
	header(tt, "Empty").Key(0, ui.KeyTab)
	if !header(tt, "Mail").Focused() {
		t.Fatal("Tab from the empty group's header did not go to Mail")
	}
}

// No input: a NavBar with no group draws, and its methods answer.
func TestNavBarEmpty(t *testing.T) {
	n := NavBar()
	tt, _ := navTester(t, n)
	n.SetGroupOpen(0, false).SetSelected("x").Focus()
	tt.Settle()
	if n.Selected() != "" || n.GroupOpen(0) || tt.Has(ByRole(ui.RoleDisclosure)) {
		t.Fatalf("Selected %q, GroupOpen(0) %v", n.Selected(), n.GroupOpen(0))
	}
	n.AddGroup("Later").AddItem("l", "Later item", nil)
	tt.Settle()
	if !tt.HasText("Later item") || !n.GroupOpen(0) {
		t.Fatalf("a group added later: open %v, %q", n.GroupOpen(0), tt.Texts())
	}
}
