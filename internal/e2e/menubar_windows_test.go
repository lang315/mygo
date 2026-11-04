//go:build windows

package e2e

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// Alt pressed and let go alone takes the keyboard to a MenuBar drawn in the
// content, in a window without a bar of Win32's: F then opens File, Enter
// chooses Open, the system's window menu loop never starts, and the focus
// goes back to the text box. Under a header bar too, the window then having
// no caption.
func TestAltAloneTakesTheKeyboardToAMenuBar(t *testing.T) {
	for _, header := range []bool{false, true} {
		var opened atomic.Int32
		tb := widget.TextBox().Focus()
		f := widget.NewForm("MenuBar Alt", 400, 200)
		if header {
			f.UseHeaderBar("")
		}
		f.SetContent(widget.VBox(4).Add(widget.MenuBar(widget.MenuItem{Text: "&File", Children: []widget.MenuItem{{Text: "Open", OnClick: func() { opened.Add(1) }}}}), tb))
		f.Show()
		t.Cleanup(f.Close)
		eventually(t, "the form's window", func() bool { return f.Native() != nil })
		mnFront(t, f)
		mnPress(mnVkAlt)
		mnPress('F')
		mnPress(0x0D) // Enter
		eventually(t, "Open chosen from the keyboard", func() bool { return opened.Load() == 1 })
		if mnMenuMode() {
			t.Errorf("header %v: Alt alone started the window menu's loop", header)
			mnPress(mnVkEscape)
		}
		mnPress('X')
		eventually(t, "the focus back in the text box", func() bool { return tb.Text() == "x" })
		f.Close()
	}
}

// keyRecorder is a control that takes the focus and every key, as a
// terminal does, and notes the keys it gets.
type keyRecorder struct {
	mu   sync.Mutex
	keys []ui.Key
}

func (r *keyRecorder) control() *widget.CustomWidget {
	return widget.Custom(func(c *ui.Context) ui.Element {
		return ui.Box(c).Size(120, 40).Focusable().Label("Recorder").HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputKeyDown {
				r.mu.Lock()
				r.keys = append(r.keys, ev.Key)
				r.mu.Unlock()
			}
			return true
		})
	})
}

func (r *keyRecorder) got() []ui.Key {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.keys)
}

// Alt alone reaches no control as a key, with a MenuBar or without one:
// a terminal would write F10 to its program, which quits htop.
func TestAltAloneSendsNoKeyToTheFocusedControl(t *testing.T) {
	// With a MenuBar: Alt takes the keyboard to it, Escape gives it back, and
	// the recorder gets the X typed after, alone.
	rec := &keyRecorder{}
	ctl := rec.control()
	f := widget.NewForm("MenuBar recorder", 400, 200)
	f.SetContent(widget.VBox(4).Add(widget.MenuBar(widget.MenuItem{Text: "&File", Children: []widget.MenuItem{{Text: "Open"}}}), ctl))
	ctl.Focus()
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	mnFront(t, f)
	mnPress(mnVkAlt)
	mnPress(mnVkEscape)
	mnPress('X')
	eventually(t, "X reaching the recorder", func() bool { return slices.Contains(rec.got(), ui.KeyX) })
	if got := rec.got(); !slices.Equal(got, []ui.Key{ui.KeyX}) {
		t.Errorf("with a MenuBar, the recorder got %v, want X alone", got)
	}
	if mnMenuMode() {
		t.Error("with a MenuBar, the window menu's loop started")
		mnPress(mnVkEscape)
	}
	f.Close()

	// Without one: Alt goes to the system, as before, which starts the
	// window menu's loop; the recorder gets nothing.
	rec = &keyRecorder{}
	ctl = rec.control()
	f = widget.NewForm("Plain recorder", 400, 200)
	f.SetContent(widget.VBox(4).Add(ctl))
	ctl.Focus()
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	mnFront(t, f)
	mnPress(mnVkAlt)
	eventually(t, "the window menu's loop, Alt alone being the system's", mnMenuMode)
	mnPress(mnVkEscape)
	if got := rec.got(); len(got) != 0 {
		t.Errorf("without a MenuBar, the recorder got %v", got)
	}
}
