package widget

import (
	"image/color"
	"testing"
)

func TestClipboardText(t *testing.T) {
	Clipboard.SetText("copied")
	if got := Clipboard.Text(); got != "copied" {
		t.Fatalf("Text() %q", got)
	}
}

func TestClipboardImage(t *testing.T) {
	if err := Clipboard.SetImage(testPNG(t, color.RGBA{9, 9, 9, 255})); err != nil {
		t.Fatal(err)
	}
	if Clipboard.Text() != "" {
		t.Fatal("an image on the clipboard still reads as text")
	}
}

func TestSetThemeAndIsDark(t *testing.T) {
	defer SetTheme(ThemeSystem)
	SetTheme(ThemeDark)
	if !IsDark() {
		t.Fatal("not dark after SetTheme(ThemeDark)")
	}
	SetTheme(ThemeLight)
	if IsDark() {
		t.Fatal("dark after SetTheme(ThemeLight)")
	}
}

func TestScreenSize(t *testing.T) {
	if w, h := ScreenSize(); w != 1440 || h != 900 { // the fake's display
		t.Fatalf("ScreenSize %v x %v", w, h)
	}
}
