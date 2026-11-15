//go:build windows && (amd64 || arm64)

package windows

import (
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// buttonsWindow returns a window of a hidden, never shown window of a
// predefined class with the styles given, for the caption buttons' rules,
// which read the styles from the window.
func buttonsWindow(t *testing.T, style uint32, opts platform.WindowOptions) *window {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	hwnd := createWindow(0, "STATIC", "", wsCaption|wsSysMenu|style, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("cannot create a window")
	}
	t.Cleanup(func() { procDestroyWindow.Call(hwnd) })
	return &window{hwnd: hwnd, opts: &opts}
}

func checkShown(t *testing.T, w *window, want ...captionButton) {
	t.Helper()
	if got := w.shownButtons(); !slices.Equal(got, want) {
		t.Fatalf("shownButtons = %v, want %v", got, want)
	}
}

// Without HideDisabledButtons a window that cannot be maximized keeps the
// maximize button, disabled, as Windows does.
func TestShownButtonsKeepDisabledByDefault(t *testing.T) {
	w := buttonsWindow(t, wsMinimizeBox, platform.WindowOptions{})
	checkShown(t, w, buttonMinimize, buttonMaximize, buttonClose)
	if w.buttonEnabled(buttonMaximize) {
		t.Fatal("the maximize button is enabled")
	}
}

func TestShownButtonsHideDisabledMaximize(t *testing.T) {
	w := buttonsWindow(t, wsMinimizeBox, platform.WindowOptions{HideDisabledButtons: true})
	checkShown(t, w, buttonMinimize, buttonClose)
}

func TestShownButtonsHideDisabledMinimize(t *testing.T) {
	w := buttonsWindow(t, wsMaximizeBox, platform.WindowOptions{HideDisabledButtons: true})
	checkShown(t, w, buttonMaximize, buttonClose)
}

func TestShownButtonsHideDisabledNeither(t *testing.T) {
	w := buttonsWindow(t, 0, platform.WindowOptions{HideDisabledButtons: true})
	checkShown(t, w, buttonClose)
}

func TestShownButtonsHideDisabledBothEnabled(t *testing.T) {
	w := buttonsWindow(t, wsMinimizeBox|wsMaximizeBox, platform.WindowOptions{HideDisabledButtons: true})
	checkShown(t, w, buttonMinimize, buttonMaximize, buttonClose)
}

func TestShownButtonsHideWindowButtonsWins(t *testing.T) {
	for _, hideDisabled := range []bool{false, true} {
		w := buttonsWindow(t, wsMinimizeBox, platform.WindowOptions{HideWindowButtons: true, HideDisabledButtons: hideDisabled})
		checkShown(t, w)
	}
}

// The buttons follow the styles as they change: a window made maximizable
// again gets its maximize button back, and the room the title bar keeps for
// the buttons follows their count.
func TestShownButtonsHideDisabledFollowsStyle(t *testing.T) {
	w := buttonsWindow(t, wsMinimizeBox, platform.WindowOptions{HideDisabledButtons: true})
	w.caption = &captionBar{w: w}
	if got := w.TitleBar().Right; got != 2*captionButtonWidth {
		t.Fatalf("TitleBar().Right = %d with two buttons, want %d", got, 2*captionButtonWidth)
	}
	w.setStyle(wsMaximizeBox, true)
	checkShown(t, w, buttonMinimize, buttonMaximize, buttonClose)
	if got := w.TitleBar().Right; got != 3*captionButtonWidth {
		t.Fatalf("TitleBar().Right = %d with three buttons, want %d", got, 3*captionButtonWidth)
	}
	w.setStyle(wsMaximizeBox, false)
	checkShown(t, w, buttonMinimize, buttonClose)
}
