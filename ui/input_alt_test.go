package ui

import "testing"

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

// F10 held answers its shortcut at each repeat, on every system, unless the
// view has a menu bar's keys (a shortcut asks for KeyMenuBar): held, F10
// would take the bar in and out of its keyboard mode, so there it answers
// once. On Windows each repeat is taken either way, as a shortcut answers
// F10, so that the system does not take it back.
func TestF10RepeatsAnswerNoShortcutOnlyWithAMenuBar(t *testing.T) {
	for _, swallow := range []bool{false, true} {
		old := swallowAltShortcuts
		swallowAltShortcuts = swallow
		for _, bar := range []bool{false, true} {
			n := 0
			tt := NewTester(func(c *Context) {
				if c.Shortcut(0, KeyF10) {
					n++
				}
				if bar {
					c.Shortcut(0, KeyMenuBar)
				}
			}, 200, 100)
			tt.Frame()
			tt.rt.keyDown(0, KeyF10, false)
			tt.Frame()
			for range 3 {
				if taken := tt.rt.keyDown(0, KeyF10, true); taken != swallow {
					t.Errorf("Windows %v, menu bar %v: a repeat taken %v", swallow, bar, taken)
				}
				tt.Frame()
			}
			want := 4
			if bar {
				want = 1
			}
			if n != want {
				t.Errorf("Windows %v, menu bar %v: F10 held answered %d times, want %d", swallow, bar, n, want)
			}
			tt.rt.close()
		}
		swallowAltShortcuts = old
	}
}

// After a menu bar's key, the keys typed before the frame that answers it
// wait for that frame, and come in their order after it: the bar, which
// took the keyboard in it, gets them, as native menus do keys typed ahead.
func TestKeysAfterAMenuBarKeyWaitForItsFrame(t *testing.T) {
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
