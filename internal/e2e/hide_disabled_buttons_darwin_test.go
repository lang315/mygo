//go:build darwin

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/ui"
)

// With HideDisabledButtons a miniaturize or zoom button is hidden exactly
// when AppKit has it disabled (its isEnabled), and without it nothing is
// hidden. Each step logs what AppKit says, at once after a change and a
// pass of the run loop later, so the log tells what AppKit does with the
// zoom button of a window that is not resizable but can go full screen.

// expect is what a step wants of a button beyond the rule.
type expect int

const (
	byRule  expect = iota // hidden == !isEnabled, whichever AppKit gives
	visible               // not hidden (and so enabled)
	hidden                // hidden (and so disabled)
)

type buttonStep struct {
	name       string
	do         func(*mygo.Window) // nil: the window as created
	mini, zoom expect
}

type standardButtons struct{ miniHidden, miniEnabled, zoomHidden, zoomEnabled bool }

// readStandardButtons runs on the main thread.
func readStandardButtons(w *mygo.Window) (b standardButtons) {
	b.miniHidden, b.miniEnabled, b.zoomHidden, b.zoomEnabled = darwin.TestStandardButtons(w.NativeHandle())
	return b
}

// settledButtons reads the buttons of a window on screen, a pass of the run
// loop after what changed them, and logs them with the window's collection
// behavior.
func settledButtons(t *testing.T, w *mygo.Window, what string) (b standardButtons) {
	t.Helper()
	waitOnScreen(w)
	mygo.RunOnMain(func() {}) // a pass of the run loop
	var behavior uint
	mygo.RunOnMain(func() {
		b = readStandardButtons(w)
		behavior = darwin.TestCollectionBehavior(w.NativeHandle())
	})
	t.Logf("%s: zoom isEnabled=%v isHidden=%v; miniaturize isEnabled=%v isHidden=%v; collectionBehavior=%#x",
		what, b.zoomEnabled, b.zoomHidden, b.miniEnabled, b.miniHidden, behavior)
	return b
}

func buttonsWindow(t *testing.T, o mygo.WindowOptions, on bool) *mygo.Window {
	o.Title = "Buttons"
	o.X, o.Y, o.Width, o.Height = 80, 80, 320, 200
	o.HideDisabledButtons = on
	o.Content = ui.View(func(c *ui.Context) { ui.Box(c).Fill() })
	return newWindow(t, o)
}

func TestHideDisabledButtonsFollowsAppKit(t *testing.T) {
	type tc struct {
		name string
		opts mygo.WindowOptions
		on   bool // HideDisabledButtons
		// allHidden: the window hides every button itself (Frameless, or a
		// hidden title bar with HideWindowButtons), the option changes nothing.
		allHidden bool
		// twin also opens a window with the same options but without
		// HideDisabledButtons: the option must not change the zoom button's
		// isEnabled.
		twin  bool
		steps []buttonStep
	}
	created := func(mini, zoom expect) []buttonStep {
		return []buttonStep{{name: "created", mini: mini, zoom: zoom}}
	}
	cases := []tc{
		// 1
		{name: "off, not resizable", opts: mygo.WindowOptions{DisableResize: true},
			steps: created(visible, visible)},
		// 2
		{name: "on, resizable, maximizable, minimizable", on: true,
			steps: created(visible, visible)},
		// 3
		{name: "on, not minimizable", on: true, opts: mygo.WindowOptions{DisableMinimize: true},
			steps: created(hidden, visible)},
		// 4
		{name: "on, resizable, not maximizable", on: true, opts: mygo.WindowOptions{DisableMaximize: true},
			steps: created(visible, hidden)},
		// 5
		{name: "on, not resizable, full screen", on: true, twin: true, opts: mygo.WindowOptions{DisableResize: true},
			steps: created(visible, byRule)},
		// 6
		{name: "on, not resizable, no full screen", on: true, twin: true,
			opts:  mygo.WindowOptions{DisableResize: true, DisableFullScreen: true},
			steps: created(visible, byRule)},
		// 7
		{name: "on, not resizable then resizable", on: true, opts: mygo.WindowOptions{DisableResize: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: byRule},
				{name: "SetResizable(true)", do: func(w *mygo.Window) { w.SetResizable(true) }, mini: visible, zoom: visible},
			}},
		// 8
		{name: "on, maximizable off then on", on: true,
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: visible},
				{name: "SetMaximizable(false)", do: func(w *mygo.Window) { w.SetMaximizable(false) }, mini: visible, zoom: hidden},
				{name: "SetMaximizable(true)", do: func(w *mygo.Window) { w.SetMaximizable(true) }, mini: visible, zoom: visible},
			}},
		// 9
		{name: "on, frameless", on: true, allHidden: true, opts: mygo.WindowOptions{Frameless: true, DisableResize: true},
			steps: created(hidden, hidden)},
		{name: "on, hidden title bar without buttons", on: true, allHidden: true,
			opts:  mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden, HideWindowButtons: true, DisableResize: true},
			steps: created(hidden, hidden)},
		// Added: the other way round after the window shows.
		{name: "on, resizable then not resizable", on: true,
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: visible},
				{name: "SetResizable(false)", do: func(w *mygo.Window) { w.SetResizable(false) }, mini: visible, zoom: byRule},
			}},
		{name: "on, not resizable no full screen, then resizable", on: true,
			opts: mygo.WindowOptions{DisableResize: true, DisableFullScreen: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: byRule},
				{name: "SetResizable(true)", do: func(w *mygo.Window) { w.SetResizable(true) }, mini: visible, zoom: visible},
			}},
		{name: "on, minimizable off then on", on: true,
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: visible},
				{name: "SetMinimizable(false)", do: func(w *mygo.Window) { w.SetMinimizable(false) }, mini: hidden, zoom: visible},
				{name: "SetMinimizable(true)", do: func(w *mygo.Window) { w.SetMinimizable(true) }, mini: visible, zoom: visible},
			}},
		{name: "on, not resizable, not maximizable", on: true, opts: mygo.WindowOptions{DisableResize: true, DisableMaximize: true},
			steps: created(visible, hidden)},
		{name: "on, not resizable, maximizable off then on", on: true, opts: mygo.WindowOptions{DisableResize: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: byRule},
				{name: "SetMaximizable(false)", do: func(w *mygo.Window) { w.SetMaximizable(false) }, mini: visible, zoom: hidden},
				{name: "SetMaximizable(true)", do: func(w *mygo.Window) { w.SetMaximizable(true) }, mini: visible, zoom: byRule},
			}},
		// The collection behavior changes after creation: FullScreenNone
		// becomes FullScreenAuxiliary on all workspaces.
		{name: "on, not resizable no full screen, on all workspaces", on: true,
			opts: mygo.WindowOptions{DisableResize: true, DisableFullScreen: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: byRule},
				{name: "SetVisibleOnAllWorkspaces(true)", do: func(w *mygo.Window) { w.SetVisibleOnAllWorkspaces(true) }, mini: visible, zoom: byRule},
				{name: "SetVisibleOnAllWorkspaces(false)", do: func(w *mygo.Window) { w.SetVisibleOnAllWorkspaces(false) }, mini: visible, zoom: byRule},
			}},
		// AppKit disables zoom on a window with no full screen when it
		// stops being resizable: is isEnabled false at once?
		{name: "on, resizable no full screen, then not resizable", on: true,
			opts: mygo.WindowOptions{DisableFullScreen: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: visible},
				{name: "SetResizable(false)", do: func(w *mygo.Window) { w.SetResizable(false) }, mini: visible, zoom: hidden},
			}},
		// setCollectionBehavior: enables zoom again on a window that is not
		// resizable and can go full screen.
		{name: "on, not resizable, not maximizable, on all workspaces", on: true,
			opts: mygo.WindowOptions{DisableResize: true, DisableMaximize: true},
			steps: []buttonStep{
				{name: "created", mini: visible, zoom: hidden},
				{name: "SetVisibleOnAllWorkspaces(true)", do: func(w *mygo.Window) { w.SetVisibleOnAllWorkspaces(true) }, mini: visible, zoom: byRule},
			}},
		{name: "on, hidden title bar, not resizable", on: true,
			opts:  mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden, DisableResize: true},
			steps: created(visible, byRule)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := buttonsWindow(t, c.opts, c.on)
			var first standardButtons
			for i, s := range c.steps {
				if s.do != nil {
					var now standardButtons
					mygo.RunOnMain(func() {
						s.do(w)
						now = readStandardButtons(w)
					})
					t.Logf("%s, at once: zoom isEnabled=%v isHidden=%v; miniaturize isEnabled=%v isHidden=%v",
						s.name, now.zoomEnabled, now.zoomHidden, now.miniEnabled, now.miniHidden)
				}
				b := settledButtons(t, w, s.name)
				if i == 0 {
					first = b
				}
				checkButton(t, s.name, "miniaturize", c.on && !c.allHidden, c.on, s.mini, b.miniHidden, b.miniEnabled)
				checkButton(t, s.name, "zoom", c.on && !c.allHidden, c.on, s.zoom, b.zoomHidden, b.zoomEnabled)
			}
			if c.twin {
				twin := buttonsWindow(t, c.opts, false)
				b := settledButtons(t, twin, "twin without HideDisabledButtons")
				if b.zoomEnabled != first.zoomEnabled {
					t.Errorf("zoom isEnabled is %v with HideDisabledButtons and %v without", first.zoomEnabled, b.zoomEnabled)
				}
			}
		})
	}
}

// checkButton checks a button against the rule, when it applies, and
// against what the step wants; without the option no button is hidden.
func checkButton(t *testing.T, step, button string, rule, on bool, want expect, isHidden, enabled bool) {
	t.Helper()
	if !on && isHidden {
		t.Errorf("%s: the %s button is hidden without HideDisabledButtons", step, button)
	}
	if rule && isHidden == enabled {
		t.Errorf("%s: the %s button isHidden=%v while AppKit has isEnabled=%v", step, button, isHidden, enabled)
	}
	if want == visible && isHidden || want == hidden && !isHidden {
		t.Errorf("%s: the %s button isHidden=%v, want %v", step, button, isHidden, want == hidden)
	}
}
