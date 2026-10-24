//go:build !js

package mygo

import "testing"

// Destructive marks an item that destroys: the platform item has it, at
// the start and after SetDestructive, which updates the native menu.
func TestMenuItemDestructiveReachesTheBackend(t *testing.T) {
	del := &MenuItem{Label: "Delete 3 accounts", Destructive: true}
	m := NewMenu([]*MenuItem{{Label: "Open"}, del, {Label: "More", Submenu: []*MenuItem{{Label: "Erase", Destructive: true}}}})
	snap := m.snapshot()
	if snap.Items[0].Destructive || !snap.Items[1].Destructive {
		t.Fatalf("Open %v, Delete %v: want false and true", snap.Items[0].Destructive, snap.Items[1].Destructive)
	}
	if !snap.Items[2].Submenu.Items[0].Destructive {
		t.Error("an item of a submenu lost the flag")
	}
	del.SetDestructive(false)
	if del.IsDestructive() {
		t.Error("IsDestructive after SetDestructive(false)")
	}
	onMain(func() {}) // the update, posted
	var last bool
	found := false
	for _, u := range fb.MenuUpdates() {
		if u.Label == "Delete 3 accounts" {
			last, found = u.Destructive, true
		}
	}
	if !found || last {
		t.Fatalf("SetDestructive(false) reached the backend as found %v, destructive %v", found, last)
	}
}
