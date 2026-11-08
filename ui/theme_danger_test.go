package ui

import "testing"

// Danger is the color of error text and borders (forms, text inputs, menu
// items): small text, 4.5:1 on the backgrounds it shows on, in both themes.
func TestDangerAsTextContrast(t *testing.T) {
	for name, th := range map[string]*Theme{"light": LightTheme(), "dark": DarkTheme()} {
		if r := contrastRatio(th.Danger, th.Background); r < 4.5 {
			t.Errorf("%s theme: Danger on Background is %.2f:1, want at least 4.5:1", name, r)
		}
	}
}
