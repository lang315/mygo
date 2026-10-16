package e2e

import (
	"strings"
	"testing"
)

// desktopUnavailable says why the frontmost app, by bundle identifier, means
// the GUI tests cannot work: while the screen is locked, or the screen saver
// runs, the windows are not on screen, the app is not active, and clicks and
// drags are lost. Empty when the desktop is usable.
func desktopUnavailable(frontmost string) string {
	switch frontmost {
	case "com.apple.loginwindow":
		return "the screen is locked (the login window is frontmost): the GUI tests need an unlocked desktop session"
	case "com.apple.ScreenSaver.Engine":
		return "the screen saver is running: the GUI tests need an unlocked desktop session"
	}
	return ""
}

func TestDesktopUnavailable(t *testing.T) {
	for _, tt := range []struct{ frontmost, want string }{
		{"com.apple.loginwindow", "locked"},
		{"com.apple.ScreenSaver.Engine", "screen saver"},
		{"com.apple.finder", ""},
		{"", ""},
	} {
		got := desktopUnavailable(tt.frontmost)
		if (tt.want == "") != (got == "") || !strings.Contains(got, tt.want) {
			t.Errorf("desktopUnavailable(%q) = %q, want one with %q", tt.frontmost, got, tt.want)
		}
	}
}
