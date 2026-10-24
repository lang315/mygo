//go:build js

package web

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A destructive item's row has the class the page's CSS draws in the danger
// color, readable on the highlight and greyed out when disabled.
func TestMenuItemClassOfADestructiveItem(t *testing.T) {
	if got := menuItemClass(&platform.MenuItem{}); got != "mygo-menu-item" {
		t.Errorf("a plain item's class is %q", got)
	}
	if got := menuItemClass(&platform.MenuItem{Destructive: true}); got != "mygo-menu-item destructive" {
		t.Errorf("a destructive item's class is %q", got)
	}
	for _, rule := range []string{
		".mygo-menu-item.destructive{color:#c42b1c;font-weight:600}",
		".mygo-menu-item.destructive.on{background:#c42b1c;color:#fff}",
		".mygo-menu-item.destructive[aria-disabled=true]{color:inherit}",
		".mygo-menu-item.destructive{color:#ff99a4}",
	} {
		if !strings.Contains(menuCSS, rule) {
			t.Errorf("the menus' CSS lacks %s", rule)
		}
	}
}
