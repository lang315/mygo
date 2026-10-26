package ui

import "testing"

// CanFocus applies Tab's rule: disabled, invisible, Hidden and inert elements do not
// take the focus, and an element holding one that does can.
func TestCanFocusIsTabsRule(t *testing.T) {
	var s string
	got := map[string]bool{}
	var ids = map[string]uint64{}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			plain := coreTextInput(c, &s)
			got["plain"] = plain.CanFocus()
			ids["plain"] = uint64(plain.id)
			inert := coreColumn(c)
			inert.flags |= flagInert
			inert.Children(func() {
				in := coreTextInput(c, &s)
				got["inert"] = in.CanFocus()
				ids["inert"] = uint64(in.id)
			})
			got["inertHolder"] = inert.CanFocus()
			hid := coreColumn(c).Invisible()
			hid.Children(func() {
				in := coreTextInput(c, &s)
				got["invisible"] = in.CanFocus()
				ids["invisible"] = uint64(in.id)
			})
			gone := coreColumn(c).Hidden(true)
			gone.Children(func() {
				in := coreTextInput(c, &s)
				got["hidden"] = in.CanFocus()
				ids["hidden"] = uint64(in.id)
			})
			got["hiddenHolder"] = gone.CanFocus()
			off := coreTextInput(c, &s).Disabled(true)
			got["disabled"] = off.CanFocus()
			ids["disabled"] = uint64(off.id)
			holder := coreColumn(c)
			holder.Children(func() { coreTextInput(c, &s) })
			got["holder"] = holder.CanFocus()
		})
	}, 300, 300)
	tt.Frame()
	want := map[string]bool{"plain": true, "inert": false, "inertHolder": false, "invisible": false, "hidden": false, "hiddenHolder": false, "disabled": false, "holder": true}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("CanFocus of %s = %v, want %v", k, got[k], w)
		}
	}
	inOrder := map[uint64]bool{}
	for _, id := range tt.rt.focusOrder {
		inOrder[uint64(id)] = true
	}
	for _, k := range []string{"plain", "inert", "invisible", "hidden", "disabled"} {
		if inOrder[ids[k]] != want[k] {
			t.Errorf("Tab reaches %s: %v, CanFocus says %v", k, inOrder[ids[k]], want[k])
		}
	}
}
