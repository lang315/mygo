//go:build darwin && !ios

package macmenu

import (
	"testing"

	"github.com/egoist/mygo/internal/darwin"
)

// TestLinked checks that importing the package links the menu extras.
func TestLinked(t *testing.T) {
	if !darwin.MenuExtrasLinked() {
		t.Fatal("importing macmenu did not link the menu extras")
	}
}
