package ui

import "testing"

// A submenu with no room on either side of its item, of a menu as wide as
// the window, stays in the window, over the menu, as macOS's do; left to
// right and right to left.
func TestSubmenuWithoutRoomStaysInTheWindow(t *testing.T) {
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		open := false
		tt := coreNewTester(func(c *context) {
			m := coreMenuBase(c, &open)
			m.Trigger.Label("Wide").Width(400).Children(func() { coreText(c, "Wide") })
			m.Popup(func(panel *node) {
				panel.MinWidth(400)
				sub := m.SubmenuItem().Padding(4).Children(func() { coreText(c, "More") })
				m.Submenu(sub, func(panel *node) {
					panel.Padding(4)
					m.Item().Padding(4).Children(func() { coreText(c, "Inner") })
				})
			})
		}, 400, 300)
		tt.SetDirection(dir)
		tt.Frame()
		tt.Click("Wide")
		tt.Click("More")
		r, ok := tt.Find("Inner")
		if !ok {
			t.Fatalf("%v: the submenu did not open: %q", dir, tt.Texts())
		}
		if r.X < 0 || r.X+r.W > 400 {
			t.Fatalf("%v: the submenu's item at %+v, out of the window 400 wide", dir, r)
		}
	}
}
