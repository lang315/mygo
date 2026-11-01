package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// tabNames are the names of the tabs shown, in order, and the index of the
// one chosen (-1 for none).
func tabNames(tt *ui.Tester) (names []string, chosen int) {
	chosen = -1
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleTab {
			if n.Checked {
				chosen = len(names)
			}
			names = append(names, n.Label)
		}
	}
	return
}

// abc is a tab control of pages A, B and C, each a label "page X", whose
// OnSwitch counts.
func abc(t *testing.T) (*TabControlWidget, *ui.Tester, *int) {
	t.Helper()
	switches := new(int)
	nb := TabControl().AddPage("A", Label("page A")).AddPage("B", Label("page B")).AddPage("C", Label("page C")).
		OnSwitch(func() { *switches++ })
	return nb, tester(t, nb), switches
}

func settle(tt *ui.Tester) {
	tt.Frame()
	tt.Frame()
}

func TestTabControlSetPageLabelRenamesTheTab(t *testing.T) {
	nb, tt, _ := abc(t)
	nb.SetPageLabel(1, "Items (3)")
	settle(tt)
	if names, _ := tabNames(tt); !slices.Equal(names, []string{"A", "Items (3)", "C"}) {
		t.Fatalf("tabs %q", names)
	}
	nb.SetPageLabel(-1, "x").SetPageLabel(3, "y")
	settle(tt)
	if names, _ := tabNames(tt); !slices.Equal(names, []string{"A", "Items (3)", "C"}) {
		t.Fatalf("an index out of range changed the tabs: %q", names)
	}
	if err := tt.Click("Items (3)"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !tt.HasText("page B") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestTabControlRemovePageKeepsTheShownPage(t *testing.T) {
	for _, c := range []struct {
		name       string
		shown      int // the page shown first
		remove     int
		names      []string
		chosen     int
		text       string
		wantSwitch bool
	}{
		{"before the shown one", 1, 0, []string{"B", "C"}, 0, "page B", false},
		{"after the shown one", 1, 2, []string{"A", "B"}, 1, "page B", false},
		{"the shown one", 1, 1, []string{"A", "C"}, 0, "page A", false},
		{"the shown first one", 0, 0, []string{"B", "C"}, 0, "page B", false},
		{"the shown last one", 2, 2, []string{"A", "B"}, 1, "page B", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			nb, tt, switches := abc(t)
			nb.SetPage(c.shown)
			settle(tt)
			nb.RemovePage(c.remove)
			settle(tt)
			names, chosen := tabNames(tt)
			if !slices.Equal(names, c.names) || chosen != c.chosen || nb.Page() != c.chosen {
				t.Fatalf("tabs %q chosen %d, Page %d; want %q chosen %d", names, chosen, nb.Page(), c.names, c.chosen)
			}
			if !tt.HasText(c.text) {
				t.Fatalf("texts %q, want %q", tt.Texts(), c.text)
			}
			flushEvents(t)
			if *switches != 0 {
				t.Fatalf("OnSwitch ran %d times for a removal made by the app", *switches)
			}
		})
	}
}

func TestTabControlRemoveEveryPage(t *testing.T) {
	nb, tt, _ := abc(t)
	nb.RemovePage(0).RemovePage(0).RemovePage(0).RemovePage(0)
	settle(tt)
	if names, _ := tabNames(tt); len(names) != 0 || nb.Page() != 0 || tt.HasText("page A") {
		t.Fatalf("tabs %q, Page %d, texts %q", names, nb.Page(), tt.Texts())
	}
	nb.AddPage("Z", Label("page Z"))
	settle(tt)
	if !tt.HasText("page Z") {
		t.Fatalf("a page added to an empty control is not shown: %q", tt.Texts())
	}
}

func TestTabControlRemovePageAfterTheUserChoseAnother(t *testing.T) {
	nb, tt, _ := abc(t)
	click(t, tt, "C")
	settle(tt)
	nb.RemovePage(0)
	settle(tt)
	if names, chosen := tabNames(tt); !slices.Equal(names, []string{"B", "C"}) || chosen != 1 || !tt.HasText("page C") {
		t.Fatalf("tabs %q chosen %d, texts %q", names, chosen, tt.Texts())
	}
}

func TestTabControlInsertPage(t *testing.T) {
	for _, c := range []struct {
		name   string
		at     int
		names  []string
		chosen int
	}{
		{"before the shown one", 0, []string{"N", "A", "B", "C"}, 2},
		{"at the shown one", 1, []string{"A", "N", "B", "C"}, 2},
		{"after the shown one", 2, []string{"A", "B", "N", "C"}, 1},
		{"past the end", 99, []string{"A", "B", "C", "N"}, 1},
		{"before the start", -4, []string{"N", "A", "B", "C"}, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			nb, tt, switches := abc(t)
			nb.SetPage(1)
			settle(tt)
			nb.InsertPage(c.at, "N", Label("page N"))
			settle(tt)
			names, chosen := tabNames(tt)
			if !slices.Equal(names, c.names) || chosen != c.chosen || !tt.HasText("page B") {
				t.Fatalf("tabs %q chosen %d, texts %q; want %q chosen %d showing B", names, chosen, tt.Texts(), c.names, c.chosen)
			}
			flushEvents(t)
			if *switches != 0 {
				t.Fatalf("OnSwitch ran %d times for an insertion made by the app", *switches)
			}
		})
	}
	t.Run("into an empty control", func(t *testing.T) {
		nb := TabControl()
		tt := tester(t, nb)
		nb.InsertPage(0, "N", Label("page N"))
		settle(tt)
		if !tt.HasText("page N") {
			t.Fatalf("texts %q", tt.Texts())
		}
	})
}

// The page keeps its state (the scroll of its list) while its index moves.
func TestTabControlPageKeepsItsStateWhenItsIndexMoves(t *testing.T) {
	nb := TabControl().AddPage("One", list100()).AddPage("Two", Label("two"))
	tt := tester(t, nb)
	before := scrollDown(t, tt)
	nb.InsertPage(0, "Zero", Label("zero"))
	settle(tt)
	if names, chosen := tabNames(tt); !slices.Equal(names, []string{"Zero", "One", "Two"}) || chosen != 1 {
		t.Fatalf("tabs %q chosen %d", names, chosen)
	}
	if y := rowY(t, tt); y != before {
		t.Fatalf("row 30 at %v after an insertion before the page, was %v", y, before)
	}
	nb.RemovePage(0)
	settle(tt)
	if y := rowY(t, tt); y != before {
		t.Fatalf("row 30 at %v after a removal before the page, was %v", y, before)
	}
}

// The focus stays on the tab of the page it was on.
func TestTabControlFocusFollowsItsTab(t *testing.T) {
	focused := func(tt *ui.Tester) string {
		for _, n := range tt.Nodes() {
			if n.Role == ui.RoleTab && n.Focused {
				return n.Label
			}
		}
		return ""
	}
	for _, c := range []struct {
		name string
		do   func(nb *TabControlWidget)
	}{
		{"a removal before it", func(nb *TabControlWidget) { nb.RemovePage(0) }},
		{"an insertion before it", func(nb *TabControlWidget) { nb.InsertPage(0, "N", Label("n")) }},
		{"a rename", func(nb *TabControlWidget) { nb.SetPageLabel(2, "C2") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			nb, tt, _ := abc(t)
			click(t, tt, "C")
			settle(tt)
			if focused(tt) != "C" {
				t.Fatalf("a click did not focus the tab: %q", focused(tt))
			}
			c.do(nb)
			settle(tt)
			if got := focused(tt); got != "C" && got != "C2" {
				t.Fatalf("focus on %q, want the tab of page C", got)
			}
		})
	}
	t.Run("the removal of the focused tab", func(t *testing.T) {
		nb, tt, _ := abc(t)
		click(t, tt, "B")
		settle(tt)
		nb.RemovePage(1)
		settle(tt)
		if got := focused(tt); got == "C" {
			t.Fatalf("focus moved to the tab after the removed one")
		}
		tt.Key(0, ui.KeyRight)
		settle(tt)
		if names, _ := tabNames(tt); len(names) != 2 {
			t.Fatalf("tabs %q", names)
		}
	})
}
