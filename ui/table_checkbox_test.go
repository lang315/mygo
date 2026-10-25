package ui

import (
	"fmt"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// checkTable is a table of n rows, all in view, with check boxes.
func checkTable(n int, checked *Selection[int], label func(i int) string) *Tester {
	cols := []TableColumn{{Title: "Name"}}
	s := ListState{Key: func(i int) any { return i }}
	if checked != nil {
		s.Checked = checked
	}
	if label != nil {
		s.Label = label
	}
	return coreNewTester(func(c *context) {
		coreTable(c, &s, cols, n, func(row, col int) { coreText(c, "x") }).Grow(1)
	}, 400, n*40+200)
}

// TestTableCheckboxAllocsPerRow: a row's check box makes nothing for each
// row in each frame (a closure for its look, for its input, a boxed state).
// The slope is what each row adds: the rest of the row is the table's own.
func TestTableCheckboxAllocsPerRow(t *testing.T) {
	slope := func(checked *Selection[int]) float64 {
		frames := func(n int) float64 {
			tt := checkTable(n, checked, nil)
			for range 3 {
				tt.Frame()
			}
			return testing.AllocsPerRun(20, tt.Frame)
		}
		return (frames(200) - frames(100)) / 100
	}
	var checked Selection[int]
	with := slope(&checked)
	without := slope(nil)
	t.Logf("allocations per row: %.2f with check boxes, %.2f without", with, without)
	if with > 0.5 {
		t.Errorf("a table with check boxes makes %.2f allocations per row in a frame, want none", with)
	}
	if extra := with - without; extra > 0.5 {
		t.Errorf("check boxes add %.2f allocations per row in a frame, want almost none", extra)
	}
}

// TestTableCheckboxFollowsTheStateOfTheFrame: what a row's check box says and
// paints is the state of the frame, not the one before: the check box reads it
// from its row as it paints.
func TestTableCheckboxFollowsTheStateOfTheFrame(t *testing.T) {
	var checked Selection[int]
	tt := checkTable(3, &checked, func(i int) string { return fmt.Sprint("row", i) })
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	state := func(i int) platform.AccessStates {
		for _, n := range tt.h.access.Nodes {
			if n.Role == platform.RoleCheckBox && n.Label == fmt.Sprintf("Check row%d", i) {
				return n.States
			}
		}
		t.Fatalf("no check box of row %d", i)
		return 0
	}
	if state(1)&platform.AccessChecked != 0 {
		t.Fatal("row 1 starts checked")
	}
	checked.Add(1) // the app checks it between two frames
	tt.Frame()
	if state(1)&platform.AccessChecked == 0 || state(0)&platform.AccessChecked != 0 {
		t.Errorf("after the app checked row 1: row 1 %v, row 0 %v", state(1), state(0))
	}
	checked.Clear()
	checked.Add(2)
	tt.Frame()
	if state(1)&platform.AccessChecked != 0 || state(2)&platform.AccessChecked == 0 {
		t.Errorf("after the app moved the check: row 1 %v, row 2 %v", state(1), state(2))
	}
}

// TestTableCheckAllPaintsTheStateOfTheFrame: the dash of the check-all box is
// painted from the state of the frame, which the app changed since the last.
func TestTableCheckAllPaintsTheStateOfTheFrame(t *testing.T) {
	var checked Selection[int]
	th := *LightTheme()
	th.AccentText = Hex("#ff0000") // the dash shows against the white of an unchecked box
	s := ListState{Checked: &checked, Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		c.SetTheme(&th)
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, 3, func(row, col int) { coreText(c, "x") }).Grow(1)
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	center := func() color.RGBA {
		n := accessNode(t, tt.h.access, platform.RoleCheckBox, "Check all")
		return tt.Image().RGBAAt(int(n.Bounds.X+n.Bounds.W/2), int(n.Bounds.Y+n.Bounds.H/2))
	}
	red := color.RGBA{255, 0, 0, 255}
	if got := center(); got == red {
		t.Fatalf("check-all box %v with nothing checked", got)
	}
	checked.Add(1) // some, not all: the box shows a dash
	tt.Frame()
	if got := center(); got != red {
		t.Fatalf("check-all box %v with one row of three checked, want the dash %v", got, red)
	}
	checked.Clear()
	tt.Frame()
	if got := center(); got == red {
		t.Errorf("check-all box %v after the rows were unchecked: it painted the frame before", got)
	}
}

// boxCenter is the pixel at the center of the check box named label.
func boxCenter(t *testing.T, tt *Tester, label string) color.RGBA {
	t.Helper()
	n := accessNode(t, tt.h.access, platform.RoleCheckBox, label)
	return tt.Image().RGBAAt(int(n.Bounds.X+n.Bounds.W/2), int(n.Bounds.Y+n.Bounds.H/2))
}

// TestCheckboxPaintsWithTheThemeItWasBuiltWith: SetTheme applies to the
// widgets made after it, so a check box made before another SetTheme keeps its
// dash, ring and look from the first theme, as it does its background.
func TestCheckboxPaintsWithTheThemeItWasBuiltWith(t *testing.T) {
	a, b := *LightTheme(), *LightTheme()
	a.AccentText = Hex("#ff0000")
	b.AccentText = Hex("#00ff00")
	var checked Selection[int]
	s := ListState{Checked: &checked, Key: func(i int) any { return i }}
	checked.Add(1) // some, not all: the check-all box shows a dash
	on := false
	tt := coreNewTester(func(c *context) {
		c.SetTheme(&a)
		coreColumn(c).AlignItems(Start).Children(func() {
			checkbox(c, &on, "", true).Label("Group")
			coreTable(c, &s, []TableColumn{{Title: "Name"}}, 3, func(row, col int) { coreText(c, "x") })
		})
		c.SetTheme(&b) // for what comes after
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	red := color.RGBA{255, 0, 0, 255}
	if got := boxCenter(t, tt, "Group"); got != red {
		t.Errorf("a mixed check box built with theme A paints %v, want its dash %v", got, red)
	}
	if got := boxCenter(t, tt, "Check all"); got != red {
		t.Errorf("a table's check-all box built with theme A paints %v, want its dash %v", got, red)
	}
}

// TestCheckboxSettersDoNotChangeTheLook: Checked, Mixed on a check box tell
// assistive technology; the box paints the state it was built with.
func TestCheckboxSettersDoNotChangeTheLook(t *testing.T) {
	th := *LightTheme()
	th.AccentText = Hex("#ff0000")
	off := false
	tt := coreNewTester(func(c *context) {
		c.SetTheme(&th)
		coreColumn(c).Gap(20).AlignItems(Start).Children(func() {
			checkbox(c, &off, "", false).Label("Mixed").Mixed()
			checkbox(c, &off, "", false).Label("Checked").Checked(true)
		})
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	red := color.RGBA{255, 0, 0, 255}
	for _, label := range []string{"Mixed", "Checked"} {
		if got := boxCenter(t, tt, label); got == red {
			t.Errorf("an unchecked box told %s paints its dash or check %v", label, got)
		}
	}
}

// TestTableRowHoverUsesTheThemeItWasBuiltWith: the row's hover color is that of
// the theme the table was built with, though SetTheme came after.
func TestTableRowHoverUsesTheThemeItWasBuiltWith(t *testing.T) {
	a, b := *LightTheme(), *LightTheme()
	a.SurfaceHover = Hex("#ff0000")
	b.SurfaceHover = Hex("#00ff00")
	s := ListState{Key: func(i int) any { return i }}
	tt := coreNewTester(func(c *context) {
		c.SetTheme(&a)
		coreTable(c, &s, []TableColumn{{Title: "Name"}}, 3, func(row, col int) { coreText(c, "x") })
		c.SetTheme(&b)
	}, 400, 400)
	tt.Frame()
	// Row 1 is below the header (the height of 8 spaces) and row 0.
	y := a.Space(8) + a.Space(8)*1.5
	tt.Move(300, y)
	tt.Frame()
	got := tt.Image().RGBAAt(300, int(y))
	if want := (color.RGBA{255, 0, 0, 255}); got != want {
		t.Errorf("a hovered row paints %v, want the hover of the theme it was built with %v", got, want)
	}
}

// TestCheckboxGoingKeepsTheThemeItWasBuiltWith: the copy of a check box going
// with its exit transition paints with the theme the box was built with, not
// the theme of the frame it goes in.
func TestCheckboxGoingKeepsTheThemeItWasBuiltWith(t *testing.T) {
	a, b := *LightTheme(), *LightTheme()
	a.AccentText = Hex("#ff0000")
	b.AccentText = Hex("#00ff00")
	on, shown := true, true
	tt, now := clockTester(func(c *context) {
		if shown {
			c.SetTheme(&a)
		} else {
			c.SetTheme(&b)
		}
		coreColumn(c).AlignItems(Start).Children(func() {
			if shown {
				checkbox(c, &on, "", false).Label("Gone").Transition(ElementTransition{Exit: &Motion{Opacity: 0}})
			}
		})
	}, 200, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	shown = false
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	var ghost *node
	for e := tt.rt.c.root.first.first; e != nil; e = e.next {
		if e.leaving != 0 {
			ghost = e
		}
	}
	if ghost == nil || ghost.first == nil {
		t.Fatal("the check box gone has no copy going")
	}
	if got := ghost.first.builtTheme(); got != &a {
		t.Errorf("the box going paints with a theme with AccentText %v, want its own %v", got.AccentText, a.AccentText)
	}
}

// TestGhostThemesFollowTheCopiesShowing: an app that makes a theme for each
// frame (SetTheme of a copy) and has a check box go in each of them does not
// make the themes the copies keep grow without end: they are those of the
// copies showing.
func TestGhostThemesFollowTheCopiesShowing(t *testing.T) {
	on := true
	frame := 0
	tt, now := clockTester(func(c *context) {
		th := *LightTheme()
		c.SetTheme(&th) // a new theme in each frame
		coreColumn(c).AlignItems(Start).Children(func() {
			coreRow(c).Key(frame).Transition(ElementTransition{Duration: time.Second, Exit: &Motion{Opacity: 0}}).Children(func() {
				checkbox(c, &on, "", false)
			})
		})
	}, 200, 100)
	for frame = 1; frame <= 400; frame++ {
		*now = now.Add(16 * time.Millisecond)
		tt.Frame()
	}
	live := 0
	for _, r := range tt.rt.trans {
		if r.ghost != nil {
			live++
		}
	}
	if live < 10 {
		t.Fatalf("%d copies showing: the test shows nothing", live)
	}
	if got := len(tt.rt.ghostThemes); got > live {
		t.Errorf("%d themes kept for %d copies showing", got, live)
	}
	// The last check box stays; the copies end, and their themes go.
	for range 5 {
		*now = now.Add(time.Second)
		tt.Frame()
	}
	if got := len(tt.rt.ghostThemes); got != 0 {
		t.Errorf("%d themes kept with no copy showing", got)
	}
}
