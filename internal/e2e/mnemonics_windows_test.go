//go:build windows

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui/widget"
)

const (
	mnVkAlt    = 0x12 // Alt
	mnVkEscape = 0x1B
)

// mnForm shows a form with a focused text box and a button "&Save" counting its
// clicks.
func mnForm(t *testing.T) (f *widget.Form, tb *widget.TextBoxWidget, saves *atomic.Int32) {
	t.Helper()
	saves = new(atomic.Int32)
	tb = widget.TextBox().Focus()
	save := widget.Button("&Save").OnClick(func() { saves.Add(1) })
	f = widget.NewForm("Mnemonics", 400, 200)
	f.SetContent(widget.VBox(4).Add(tb, save))
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	return f, tb, saves
}

// front brings the form's window to the front, where the keyboard types.
func mnFront(t *testing.T, f *widget.Form) {
	t.Helper()
	var ok bool
	mygo.RunOnMain(func() { ok = win.TestForeground(f.Native().NativeHandle()) })
	if !ok {
		t.Fatal("the window did not come to the front")
	}
}

func mnPress(keys ...byte) { mygo.RunOnMain(func() { win.TestPressKeys(keys...) }) }

func mnMenuMode() (on bool) {
	mygo.RunOnMain(func() { on = win.TestInMenuMode() })
	return on
}

// Alt+S clicks the button "&Save" once, types nothing, and leaves the window
// out of menu mode: the next plain key still reaches the text box.
func TestMnemonicClicksOnceAndLeavesTheKeyboardAlone(t *testing.T) {
	f, tb, saves := mnForm(t)
	mnFront(t, f)
	mnPress(mnVkAlt, 'S')
	eventually(t, "Alt+S clicking Save", func() bool { return saves.Load() >= 1 })
	if mnMenuMode() {
		t.Error("the window went into menu mode after Alt+S")
		mnPress(mnVkEscape)
	}
	if got := tb.Text(); got != "" {
		t.Errorf("Alt+S typed %q into the text box", got)
	}
	mnPress('X')
	eventually(t, "the next plain key reaching the text box", func() bool { return tb.Text() == "x" })
	if n := saves.Load(); n != 1 {
		t.Errorf("Save was clicked %d times, want 1", n)
	}
}

// The menu bar's access key beats a control's: Alt+F opens the File menu and
// the button "&File" is not clicked.
func TestMenuBarAccessKeyBeatsAButtonMnemonic(t *testing.T) {
	var clicks atomic.Int32
	f := widget.NewForm("Mnemonics menu", 400, 200)
	f.SetMenuBar(widget.MenuItem{Text: "&File", Children: []widget.MenuItem{{Text: "New"}}})
	f.SetContent(widget.VBox(4).Add(widget.TextBox().Focus(), widget.Button("&File").OnClick(func() { clicks.Add(1) })))
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	mnFront(t, f)
	mnPress(mnVkAlt, 'F')
	eventually(t, "Alt+F opening the File menu", mnMenuMode)
	mnPress(mnVkEscape)
	eventually(t, "leaving menu mode", func() bool { return !mnMenuMode() })
	if n := clicks.Load(); n != 0 {
		t.Errorf("the button %q was clicked %d times by Alt+F", "&File", n)
	}
}

// An Alt+key nothing answers does nothing.
func TestUnmatchedAltKeyDoesNothing(t *testing.T) {
	f, tb, saves := mnForm(t)
	mnFront(t, f)
	mnPress(mnVkAlt, 'Q')
	if saves.Load() != 0 || tb.Text() != "" {
		t.Errorf("Alt+Q clicked %d times and typed %q", saves.Load(), tb.Text())
	}
	if mnMenuMode() {
		mnPress(mnVkEscape)
	}
}

// A mnemonic with an accent answers to its base letter: "&Đóng" is Alt+D.
func TestAccentedMnemonicAnswersToItsBaseLetter(t *testing.T) {
	var closes atomic.Int32
	f := widget.NewForm("Mnemonics vi", 400, 200)
	f.SetContent(widget.VBox(4).Add(widget.TextBox().Focus(), widget.Button("&Đóng").OnClick(func() { closes.Add(1) })))
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	mnFront(t, f)
	mnPress(mnVkAlt, 'D')
	eventually(t, "Alt+D clicking Đóng", func() bool { return closes.Load() == 1 })
}
