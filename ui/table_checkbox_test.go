package ui

import (
	"fmt"
	"image/color"
	"testing"

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
