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
