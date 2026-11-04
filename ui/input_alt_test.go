package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/keyhold"
	"github.com/egoist/mygo/internal/platform"
)

// On Windows an Alt+key a shortcut answers is taken, so that the system does
// not go into its menu mode; one nothing answers is left to it.
func TestAltShortcutsAreTakenWhereTheSystemWouldTakeThemForMenus(t *testing.T) {
	old := swallowAltShortcuts
	swallowAltShortcuts = true
	defer func() { swallowAltShortcuts = old }()
	tt := NewTester(func(c *Context) {
		c.Shortcut(Alt, KeyS)
		c.Shortcut(Cmd, KeyQ)
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	if !tt.rt.keyDown(Alt, KeyS, false) {
		t.Error("Alt+S, which a shortcut answers, was left to the system")
	}
	if tt.rt.keyDown(Alt, KeyQ, false) {
		t.Error("Alt+Q, which nothing answers, was taken")
	}
	if tt.rt.keyDown(Cmd, KeyQ, false) {
		t.Error("a key without Alt was taken by this rule")
	}
	swallowAltShortcuts = false
	if tt.rt.keyDown(Alt, KeyS, false) {
		t.Error("taken where the rule is off")
	}
}

// F10, which Windows takes to its menu mode as it does Alt alone (which the
// backend hands the view as F10), is taken when a shortcut answers it, and
// left to the system otherwise.
func TestF10IsTakenWhereAShortcutAnswersIt(t *testing.T) {
	old := swallowAltShortcuts
	swallowAltShortcuts = true
	defer func() { swallowAltShortcuts = old }()
	bar := false
	tt := NewTester(func(c *Context) {
		if bar {
			c.Shortcut(0, KeyF10)
		}
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	if tt.rt.keyDown(0, KeyF10, false) {
		t.Error("F10, which nothing answers, was taken")
	}
	bar = true
	tt.Frame()
	if !tt.rt.keyDown(0, KeyF10, false) {
		t.Error("F10, which a shortcut answers, was left to the system")
	}
	if tt.rt.keyDown(Shift, KeyF10, false) {
		t.Error("Shift+F10, the context menu's key, was taken by this rule")
	}
}

// KeyMenuBar, Alt alone as Windows' backend hands it over, goes to the
// view's shortcuts only: no handler of keys gets it, as a terminal would
// write it to its program, and it is taken only when a shortcut answers it.
func TestMenuBarKeyGoesToShortcutsOnly(t *testing.T) {
	var keys []Key
	bar, asked := false, 0
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).Focusable().Label("Field").HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown {
				keys = append(keys, ev.Key)
			}
			return true // takes every key, as a terminal does
		})
		if bar && c.Shortcut(0, KeyMenuBar) {
			asked++
		}
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	if tt.KeyTaken(0, KeyMenuBar) {
		t.Error("KeyMenuBar was taken with nothing asking for it")
	}
	bar = true
	tt.Frame()
	if !tt.KeyTaken(0, KeyMenuBar) || asked != 1 {
		t.Errorf("with a shortcut asking for it: answered %d times", asked)
	}
	if len(keys) != 0 {
		t.Errorf("the focused handler got %v", keys)
	}
}

// F10 held answers its shortcut, and reaches a handler of keys, at each
// repeat, unless the view asks for KeyMenuBar (a form with a MenuBar): held,
// F10 would take the bar in and out of its keyboard mode, so there it
// answers once, its repeats taken. A window without a bar keeps every
// repeat even while a MenuBar, somewhere in the app, enabled holding.
func TestF10RepeatsAnswerNoShortcutOnlyWithAMenuBar(t *testing.T) {
	enableKeyHold(t)
	// hold presses F10 and holds it for three repeats in a window built by
	// view, and counts what took them.
	hold := func(view func(c *Context), focus string) {
		tt := NewTester(view, 200, 100)
		defer tt.rt.close()
		tt.Frame()
		if focus != "" {
			if err := tt.Click(focus); err != nil {
				t.Fatal(err)
			}
		}
		press := platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyF10}
		tt.send(press)
		press.Repeat = true
		for range 3 {
			tt.send(press)
		}
	}
	// A terminal, a handler of every key, in a window without a bar.
	handled := 0
	hold(func(c *Context) {
		Box(c).Size(100, 100).Focusable().Label("Terminal").HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown && ev.Key == KeyF10 {
				handled++
			}
			return true
		})
	}, "Terminal")
	if handled != 4 {
		t.Errorf("without a menu bar: the terminal got %d of 4 F10 presses", handled)
	}
	// The window's own shortcut, without a bar.
	shortcut := 0
	hold(func(c *Context) {
		if c.Shortcut(0, KeyF10) {
			shortcut++
		}
	}, "")
	if shortcut != 4 {
		t.Errorf("without a menu bar: F10 held answered its shortcut %d times, want 4", shortcut)
	}
	// A view with a menu bar (it asks for KeyMenuBar): once.
	bar := 0
	hold(func(c *Context) {
		if c.Shortcut(0, KeyF10) {
			bar++
		}
		c.Shortcut(0, KeyMenuBar)
	}, "")
	if bar != 1 {
		t.Errorf("with a menu bar: F10 held answered %d times, want 1", bar)
	}
}

// needsKeyHold skips a test of holding where it is compiled out (macOS).
func needsKeyHold(t *testing.T) {
	if !holdsMenuKeys {
		t.Skip("no holding of menu bar keys on macOS (holdsMenuKeys)")
	}
}

// enableKeyHold enables holding the keys after a menu bar's key, as
// widget.MenuBar does, for the test.
func enableKeyHold(t *testing.T) {
	needsKeyHold(t)
	was := keyhold.Enabled()
	keyhold.Enable()
	t.Cleanup(func() {
		if !was {
			keyhold.Disable()
		}
	})
}

// After a menu bar's key, the keys typed before the frame that answers it
// wait for that frame, and come in their order after it: the bar, which
// took the keyboard in it, gets them, as native menus do keys typed ahead.
// Without a MenuBar holding is off, and they come at once.
func TestKeysAfterAMenuBarKeyWaitForItsFrame(t *testing.T) {
	enableKeyHold(t)
	for _, first := range []Key{KeyMenuBar, KeyF10} {
		var order []string
		bar := false
		tt := NewTester(func(c *Context) {
			Box(c).Size(100, 100).Focusable().Label("Field").HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputKeyDown {
					order = append(order, "field")
				}
				return true
			})
			if first != KeyMenuBar {
				c.Shortcut(0, KeyMenuBar) // the view has a menu bar's keys
			}
			if c.Shortcut(0, first) {
				bar = true
				order = append(order, "bar")
			}
		}, 200, 200)
		tt.Frame()
		if err := tt.Click("Field"); err != nil {
			t.Fatal(err)
		}
		tt.KeysAtOnce(first, KeyA, KeyB)
		if !bar || len(order) != 3 || order[0] != "bar" {
			t.Errorf("key %d then A and B at once: %v, want the bar first", first, order)
		}
		tt.rt.close()
	}
}

// keyOrderApp is a view with a field that takes every key and a menu bar's
// shortcut, and the order they answered in.
type keyOrderApp struct{ order []string }

func (a *keyOrderApp) view(c *Context) {
	Box(c).Size(100, 100).Focusable().Label("Field").HandleInput(func(ev InputEvent) bool {
		if ev.Kind == InputKeyDown {
			a.order = append(a.order, "field")
		}
		return true
	})
	if c.Shortcut(0, KeyMenuBar) {
		a.order = append(a.order, "bar")
	}
}

// Without a MenuBar, which enables holding, nothing is held: the keys after
// KeyMenuBar come at once, as before.
func TestKeysAreNotHeldWithoutHolding(t *testing.T) {
	if keyhold.Enabled() {
		keyhold.Disable()
		t.Cleanup(keyhold.Enable)
	}
	a := &keyOrderApp{}
	tt := NewTester(a.view, 200, 200)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	tt.KeysAtOnce(KeyMenuBar, KeyA)
	if len(a.order) != 2 || a.order[0] != "field" || tt.rt.hold != nil {
		t.Errorf("without holding: %v, hold %v", a.order, tt.rt.hold)
	}
}

// The keys held for a menu bar's key are dropped, not replayed, when the
// window goes inactive or out of sight before the frame that answers it:
// the mode they were typed into is over.
func TestHeldKeysAreDroppedWhenTheWindowGoesAway(t *testing.T) {
	enableKeyHold(t)
	for name, away := range map[string]platform.SurfaceEventKind{"inactive": platform.SurfaceBlur, "out of sight and back": platform.SurfaceShown} {
		a := &keyOrderApp{}
		tt := NewTester(a.view, 200, 200)
		tt.Frame()
		if err := tt.Click("Field"); err != nil {
			t.Fatal(err)
		}
		// KeyMenuBar and A, with no frame between: A is held.
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyMenuBar})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyA})
		tt.rt.event(platform.SurfaceEvent{Kind: away})
		if away == platform.SurfaceBlur {
			tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceFocus})
		}
		tt.Frame()
		for _, o := range a.order {
			if o == "field" {
				t.Errorf("%s: the key held reached the field: %v", name, a.order)
			}
		}
		// Holding is over: the next key reaches the field at once.
		tt.Key(0, KeyB)
		if n := len(a.order); n == 0 || a.order[n-1] != "field" {
			t.Errorf("%s: a key after did not reach the field: %v", name, a.order)
		}
		tt.rt.close()
	}
}

// focusSeenApp is a view whose element reads its own focus, then takes it
// the first time, as widget.render gives a control the focus once its build
// returned.
type focusSeenApp struct {
	take bool
	seen []bool // what each build of the frame read
}

func (a *focusSeenApp) view(c *Context) {
	e := Box(c).Size(100, 100).Focusable().Label("Field")
	a.seen = append(a.seen, e.Focused())
	if a.take {
		a.take = false
		e.Focus()
	}
}

// A build that moves the focus builds again in the same frame, as an event
// handled while building does: what read the focus before sees it moved.
// Without it the frame ended with the element reading false, and, with no
// other frame asked for, it never read true: on Windows, a control focused
// with widget's Focus() before the window showed never saw its focus.
func TestFocusMovedInABuildBuildsAgain(t *testing.T) {
	a := &focusSeenApp{}
	tt := NewTester(a.view, 200, 200)
	defer tt.rt.close()
	tt.Frame()
	a.take, a.seen = true, nil
	tt.rt.runFrame() // one frame, its passes, and no other
	if len(a.seen) < 2 || !a.seen[len(a.seen)-1] {
		t.Errorf("the builds of the frame that moved the focus read %v; the last must read true", a.seen)
	}
}

// modsApp is a view with Ctrl+S, a menu bar's key, and the modifiers it
// read last.
type modsApp struct {
	saves int
	mods  Modifiers
}

func (a *modsApp) view(c *Context) {
	c.Shortcut(0, KeyMenuBar)
	if c.Shortcut(Ctrl, KeyS) {
		a.saves++
	}
	a.mods = c.Modifiers()
}

// Ctrl+S typed ahead of the frame Alt alone asked for runs once, and leaves
// no modifier held: the changes of the modifiers wait with the keys, so
// that they come back in their order, Ctrl up last.
func TestModifiersTypedAheadWaitWithTheKeys(t *testing.T) {
	enableKeyHold(t)
	a := &modsApp{}
	tt := NewTester(a.view, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	for _, ev := range []platform.SurfaceEvent{
		{Kind: platform.KeyPressed, Key: platform.KeyMenuBar},
		{Kind: platform.ModifiersChanged, Mods: platform.ModCtrl},
		{Kind: platform.KeyPressed, Key: platform.KeyS, Mods: platform.ModCtrl},
		{Kind: platform.KeyReleased, Key: platform.KeyS, Mods: platform.ModCtrl},
		{Kind: platform.ModifiersChanged, Mods: 0},
	} {
		tt.rt.event(ev)
	}
	tt.Frame()
	tt.Frame()
	if a.saves != 1 || a.mods != 0 || tt.rt.mods != 0 {
		t.Errorf("Ctrl+S typed ahead: saved %d times, modifiers %v (engine %v), want once and none", a.saves, a.mods, tt.rt.mods)
	}
}
