package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
)

// slopeOf is what each instance of build adds to the allocations of a frame.
func slopeOf(build func(c *context, n int)) float64 {
	frames := func(n int) float64 {
		tt := coreNewTester(func(c *context) { build(c, n) }, 600, n*60+400)
		for range 3 {
			tt.Frame()
		}
		return allocrun.Least(20, tt.Frame)
	}
	return (frames(60) - frames(20)) / 40
}

// TestHoverLooksMakeNoClosurePerFrame: the hover look of a tree item, a
// sidebar item, a breadcrumb, a collapsible's trigger, a stepper's
// arrows and a calendar's days is a static function that reads the node. The
// ceilings are what each instance made with a closure for its look, less
// those closures.
func TestHoverLooksMakeNoClosurePerFrame(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector makes allocations of its own")
	}
	for _, c := range []struct {
		name  string
		max   float64
		build func(c *context, n int)
	}{
		// 53 since upstream's calendar draws its month chevrons (two Draw
		// closures a month, #874) and keeps its settings with Browse (#903).
		{"calendar (42 days)", 53, func(c *context, n int) {
			d := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
			coreColumn(c).Children(func() {
				for range n {
					coreCalendar(c, &d)
				}
			})
		}},
		{"tree item", 5.5, func(c *context, n int) {
			coreTree(c, func() {
				for range n {
					open := false
					coreTreeItem(c, "x", &open, func() {})
				}
			})
		}},
		{"sidebar item", 3.5, func(c *context, n int) {
			sel := ""
			coreSidebar(c, &sel, func() {
				for i := range n {
					coreSidebarItem(c, fmt.Sprint("i", i), nil, "x")
				}
			})
		}},
		{"breadcrumbs (3)", 10.5, func(c *context, n int) {
			ch := 0
			coreColumn(c).Children(func() {
				for range n {
					coreBreadcrumbs(c, []string{"a", "b", "c"}, &ch)
				}
			})
		}},
		{"collapsible", 7.5, func(c *context, n int) {
			open := false
			coreColumn(c).Children(func() {
				for range n {
					coreCollapsible(c, "x", &open, func() {})
				}
			})
		}},
		{"stepper", 13.5, func(c *context, n int) {
			v := 1.0
			coreColumn(c).Children(func() {
				for range n {
					coreStepper(c, &v, 0, 10, 1)
				}
			})
		}},
	} {
		if got := slopeOf(c.build); got > c.max {
			t.Errorf("%s: %.2f allocations per instance in a frame, want at most %.2f", c.name, got, c.max)
		}
	}
}

// What these elements paint on hover follows the theme of the frame, light
// or dark: the static function reads the theme the element was built with.
func TestHoverLooksFollowTheTheme(t *testing.T) {
	light, dark := LightTheme(), DarkTheme()
	cur := light
	var item, side, trigger *node
	open, sel, ch := false, "b", 0
	tt := coreNewTester(func(c *context) {
		c.SetTheme(cur)
		coreColumn(c).Gap(4).AlignItems(Start).Children(func() {
			coreTree(c, func() {
				o := false
				item = coreTreeItem(c, "item", &o, func() {})
			})
			coreSidebar(c, &sel, func() {
				coreSidebarItem(c, "a", nil, "alpha")
				side = coreSidebarItem(c, "c", nil, "gamma")
			})
			coreBreadcrumbs(c, []string{"home", "docs", "x"}, &ch)
			trigger = coreCollapsible(c, "more", &open, func() {})
		})
	}, 400, 500)
	tt.Frame()
	for _, th := range []*Theme{light, dark, light} {
		cur = th
		tt.Frame()
		tt.Frame()
		for _, e := range []struct {
			name string
			n    *node
		}{{"tree item", item}, {"sidebar item", side}, {"collapsible trigger", trigger.first}} {
			r := e.n.Bounds()
			tt.Move(r.X+r.W/2, r.Y+r.H/2)
			tt.Frame()
			if got := painted(e.n).bg; got != th.SurfaceHover {
				t.Errorf("%s hovered: background %v, want %v", e.name, got, th.SurfaceHover)
			}
			tt.Move(399, 499)
			tt.Frame()
		}
	}
}

// A row or item that is chosen keeps the color of its choice under the
// pointer; one that is not takes the hover color. A stepper's arrow takes the
// pressed color while it is pressed, the hover color while it is not.
func TestHoverLooksKeepAChoiceAndPressAnArrow(t *testing.T) {
	light := LightTheme()
	names := []string{"a", "b", "c"}
	var chosen Selection[string]
	chosen.Add("b")
	sel := 1
	ls := ListState{Selection: &chosen, Selected: &sel, Key: func(i int) any { return names[i] }}
	val := 5.0
	var rows []*node
	var step *node
	tt := coreNewTester(func(c *context) {
		rows = rows[:0]
		coreColumn(c).Gap(4).AlignItems(Start).Children(func() {
			coreList(c, &ls, len(names), func(i int) { coreText(c, names[i]) }).Height(150)
			step = coreStepper(c, &val, 0, 10, 1)
		})
	}, 300, 400)
	tt.Frame()
	row := func(i int) *node {
		var out []*node
		for n := step.parent.first.first; n != nil; n = n.next {
			if n.styleFn != nil {
				out = append(out, n)
			}
		}
		if i >= len(out) {
			t.Fatalf("%d rows with a look, want %d", len(out), i+1)
		}
		return out[i]
	}
	at := func(n *node) (float32, float32) { r := n.Bounds(); return r.X + r.W/2, r.Y + r.H/2 }
	x, y := at(row(0))
	tt.Move(x, y)
	tt.Frame()
	if got := painted(row(0)).bg; got != light.SurfaceHover {
		t.Errorf("a row not chosen, hovered: %v, want %v", got, light.SurfaceHover)
	}
	x, y = at(row(1))
	tt.Move(x, y)
	tt.Frame()
	if got := painted(row(1)).bg; got == light.SurfaceHover {
		t.Errorf("a row chosen, hovered: %v, want the color of its choice", got)
	}
	var arrow *node
	for n := step.first; n != nil; n = n.next {
		if n.flags&flagClickable != 0 {
			arrow = n
			break
		}
	}
	if arrow == nil {
		t.Fatal("no arrow in the stepper")
	}
	x, y = at(arrow)
	tt.Move(x, y)
	tt.Frame()
	if got := painted(arrow).bg; got != light.SurfaceHover {
		t.Errorf("an arrow hovered: %v, want %v", got, light.SurfaceHover)
	}
	tt.Press(x, y)
	tt.Frame()
	if got := painted(arrow).bg; got != light.SurfacePressed {
		t.Errorf("an arrow pressed: %v, want %v", got, light.SurfacePressed)
	}
}
