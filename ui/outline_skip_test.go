package ui

import "testing"

// skipTree is an outline of roots whose open "Alpha" has one child, "Wait",
// that cannot be chosen (ListState.Skip), as a tree's Loading row.
func skipTree(roots ...string) (s *OutlineState[string], sel *int, tt *Tester) {
	s, sel = &OutlineState[string]{}, new(int)
	*sel = -1
	s.List.Selected = sel
	s.List.Skip = func(row int) bool { return s.Item(row) == "Wait" }
	s.List.Label = func(row int) string { return s.Item(row) }
	s.Open.Add("Alpha")
	kids := func(item string) []string {
		if item == "Alpha" {
			return []string{"Wait"}
		}
		return nil
	}
	tt = coreNewTester(func(c *context) {
		coreOutline(c, s, roots, kids, func(item string) { coreText(c, item) }).Grow(1)
	}, 400, 400)
	return s, sel, tt
}

func TestOutlineKeysMovePastRowsThatCannotBeChosen(t *testing.T) {
	s, sel, tt := skipTree("Alpha", "Lima")
	chosen := func() string {
		if *sel < 0 {
			return ""
		}
		return s.Item(*sel)
	}
	tt.Click("Alpha")
	if chosen() != "Alpha" {
		t.Fatalf("a click chose %q", chosen())
	}
	tt.Key(0, KeyDown)
	if chosen() != "Lima" {
		t.Fatalf("Down from Alpha chose %q, want Lima, past Wait", chosen())
	}
	tt.Key(0, KeyUp)
	if chosen() != "Alpha" {
		t.Fatalf("Up from Lima chose %q, want Alpha, past Wait", chosen())
	}
	tt.Key(0, KeyRight)
	if chosen() != "Alpha" {
		t.Fatalf("Right on an open item whose only child cannot be chosen chose %q, want Alpha", chosen())
	}
	tt.Key(0, KeyEnd)
	if chosen() != "Lima" {
		t.Fatalf("End chose %q, want Lima", chosen())
	}
	tt.Key(0, KeyHome)
	if chosen() != "Alpha" {
		t.Fatalf("Home chose %q, want Alpha", chosen())
	}
}

func TestOutlineEndStopsAtTheLastRowThatCanBeChosen(t *testing.T) {
	s, sel, tt := skipTree("Lima", "Alpha") // Wait is the last row
	tt.Click("Lima")
	tt.Key(0, KeyEnd)
	if got := s.Item(*sel); got != "Alpha" {
		t.Fatalf("End chose %q, want Alpha: Wait cannot be chosen", got)
	}
}

func TestOutlineClickAndTypingLeaveRowsThatCannotBeChosen(t *testing.T) {
	s, sel, tt := skipTree("Alpha", "Lima")
	tt.Click("Alpha")
	if err := tt.Click("Wait"); err != nil {
		t.Fatal(err)
	}
	if got := s.Item(*sel); got != "Alpha" {
		t.Fatalf("a click on Wait chose %q, want the choice to stay on Alpha", got)
	}
	tt.Key(0, KeyW)
	tt.Key(0, KeyA)
	if got := s.Item(*sel); got != "Alpha" {
		t.Fatalf("typing wa chose %q: Wait matches, but cannot be chosen", got)
	}
}
