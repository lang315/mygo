//go:build darwin

package darwin

import (
	"strings"
	"testing"
)

// An app with no main menu (App.SetMenu(nil), or a test that did) has none to
// search: the hooks that find an item answer with an error, not a panic.
func TestMenuItemAtWithoutAMainMenuIsAnError(t *testing.T) {
	load()
	prev := theBackend
	theBackend = &Backend{} // no application: its main menu is nil
	defer func() { theBackend = prev }()
	for _, path := range [][]string{{"File"}, {"File", "New"}, {"File", "New", "Blank"}} {
		var err error
		withPool(func() { _, _, err = menuItemAt(path) })
		if err == nil || !strings.Contains(err.Error(), "no main menu") {
			t.Errorf("menuItemAt(%q) = %v, want an error saying there is no main menu", path, err)
		}
	}
	if err := TestPerformMenuItem("File", "New"); err == nil {
		t.Error("TestPerformMenuItem with no main menu returned no error")
	}
	if _, _, err := TestMenuItemImage("File", "New"); err == nil {
		t.Error("TestMenuItemImage with no main menu returned no error")
	}
}
