package widget

import "testing"

func TestFrameShowsTitleAndChild(t *testing.T) {
	tt := tester(t, Frame("Settings").Add(Label("inside")))
	s, i := find(t, tt, "Settings"), find(t, tt, "inside")
	if i.Y <= s.Y {
		t.Fatalf("child %v not under title %v", i, s)
	}
}

func TestGridPlacesCells(t *testing.T) {
	g := Grid(4, 4)
	g.Attach(Label("A"), 0, 0, 1, 1).Attach(Label("B"), 1, 0, 1, 1).Attach(Label("C"), 0, 1, 2, 1)
	tt := tester(t, g)
	a, b, c := find(t, tt, "A"), find(t, tt, "B"), find(t, tt, "C")
	if b.X <= a.X || c.Y <= a.Y || c.X > a.X+1 {
		t.Fatalf("A %v B %v C %v", a, b, c)
	}
}

func TestGridLeavesHolesEmpty(t *testing.T) {
	tt := tester(t, Grid(0, 0).Attach(Label("far"), 2, 1, 1, 1).Attach(Label("near"), 0, 0, 1, 1))
	n, f := find(t, tt, "near"), find(t, tt, "far")
	if f.X <= n.X || f.Y <= n.Y {
		t.Fatalf("near %v far %v", n, f)
	}
}

func TestNotebookSwitchesPages(t *testing.T) {
	switched := make(chan int, 4)
	var nb *NotebookWidget
	Notebook().AssignTo(&nb).AddPage("One", Label("page one")).AddPage("Two", Label("page two")).
		OnSwitch(func() { switched <- nb.Page() })
	tt := tester(t, nb)
	if !tt.HasText("page one") || tt.HasText("page two") {
		t.Fatalf("texts %q", tt.Texts())
	}
	click(t, tt, "Two")
	if p := wait(t, switched); p != 1 || !tt.HasText("page two") {
		t.Fatalf("page %d, texts %q", p, tt.Texts())
	}
	nb.SetPage(0)
	tt.Frame()
	if !tt.HasText("page one") {
		t.Fatalf("SetPage(0): %q", tt.Texts())
	}
}

// U12: Tabs clamps its choice without OnChange; the widget notices.
func TestNotebookClampedPageIsReported(t *testing.T) {
	switched := make(chan int, 4)
	var nb *NotebookWidget
	Notebook().AssignTo(&nb).AddPage("A", Label("a")).AddPage("B", Label("b")).
		OnSwitch(func() { switched <- nb.Page() })
	tt := tester(t, nb)
	nb.SetPage(5)
	tt.Frame()
	tt.Frame()
	if p := wait(t, switched); p != 1 {
		t.Fatalf("clamped page %d, want 1", p)
	}
}

func TestSidebarSwitchesContent(t *testing.T) {
	sb := Sidebar().Add("mail", "Mail", Label("inbox")).Add("cal", "Calendar", Label("today"))
	tt := tester(t, sb)
	if !tt.HasText("inbox") {
		t.Fatalf("first page not shown: %q", tt.Texts())
	}
	click(t, tt, "Calendar")
	if !tt.HasText("today") || sb.Selected() != "cal" {
		t.Fatalf("texts %q, selected %q", tt.Texts(), sb.Selected())
	}
	sb.Remove("cal")
	tt.Frame()
	if tt.HasText("Calendar") {
		t.Fatalf("removed item shown: %q", tt.Texts())
	}
}

func TestCardHidesItsBody(t *testing.T) {
	c := Card().SetTitle(Label("Title")).SetBody(Label("Body"))
	tt := tester(t, c)
	c.ShowBody(false)
	tt.Frame()
	if !tt.HasText("Title") || tt.HasText("Body") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestScrolledSeparatorFlow(t *testing.T) {
	f := Flow(4)
	for _, s := range []string{"one", "two", "three", "four", "five", "six"} {
		f.Add(Label(s))
	}
	tt := tester(t, VBox(0).Add(Scrolled(Label("scrolled")).SetSize(300, 100), Separator(), f.SetSize(120, 0)))
	if !tt.HasText("scrolled") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if one, six := find(t, tt, "one"), find(t, tt, "six"); six.Y <= one.Y {
		t.Fatalf("flow did not wrap: one %v six %v", one, six)
	}
}

// A hidden cell leaves a hole; the cells after it stay put.
func TestGridHiddenCellKeepsTheOthersInPlace(t *testing.T) {
	a := Label("A")
	g := Grid(0, 0).Attach(a, 0, 0, 1, 1).Attach(Label("B"), 1, 0, 1, 1).Attach(Label("C"), 0, 1, 1, 1)
	tt := tester(t, g)
	b0, c0 := find(t, tt, "B"), find(t, tt, "C")
	a.SetVisible(false)
	tt.Frame()
	if b, c := find(t, tt, "B"), find(t, tt, "C"); b.X != b0.X || c.Y != c0.Y || c.X != c0.X {
		t.Fatalf("B %v→%v, C %v→%v: cells moved", b0, b, c0, c)
	}
}
func TestFrameIgnoresItsTitleAsItsChild(t *testing.T) {
	l := Label("both")
	tt := tester(t, Frame("").SetTitleWidget(l).Add(l)) // a duplicate key panics
	if n := count(tt, "both"); n != 1 {
		t.Fatalf("shown %d times", n)
	}
}

func TestCardIgnoresItsTitleAsItsBody(t *testing.T) {
	l := Label("both")
	tt := tester(t, Card().SetTitle(l).SetBody(l))
	if n := count(tt, "both"); n != 1 {
		t.Fatalf("shown %d times", n)
	}
}

func TestGridAttachOnATakenCellReplacesIt(t *testing.T) {
	g := Grid(0, 0).Attach(Label("old"), 0, 0, 1, 1).Attach(Label("new"), 0, 0, 1, 1)
	tt := tester(t, g)
	mu.Lock()
	n := len(g.cells)
	mu.Unlock()
	if n != 1 || tt.HasText("old") || !tt.HasText("new") {
		t.Fatalf("%d cells, texts %q", n, tt.Texts())
	}
}
