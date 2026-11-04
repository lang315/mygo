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

// keyRecorder is a control that takes the focus and every key, as a
// terminal does, and notes the keys it gets and whether it has the focus.
type keyRecorder struct {
	mu      sync.Mutex
	keys    []ui.Key
	focused bool
}

func (r *keyRecorder) control() *widget.CustomWidget {
	return widget.Custom(func(c *ui.Context) ui.Element {
		e := ui.Box(c).Size(120, 40).Focusable().Label("Recorder").HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputKeyDown {
				r.mu.Lock()
				r.keys = append(r.keys, ev.Key)
				r.mu.Unlock()
			}
			return true
		})
		r.mu.Lock()
		r.focused = e.Focused()
		r.mu.Unlock()
		return e
	})
}

func (r *keyRecorder) got() []ui.Key {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.keys)
}

func (r *keyRecorder) hasFocus() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.focused
}

// recorderForm shows a form with a menu bar of File › Open, when bar is
// set, and a recorder with the focus, under a header bar when header is.
func recorderForm(t *testing.T, bar, header bool, opened *atomic.Int32) (*widget.Form, *keyRecorder) {
	t.Helper()
	rec := &keyRecorder{}
	ctl := rec.control()
	f := widget.NewForm("MenuBar keys", 400, 200)
	if header {
		f.UseHeaderBar("")
	}
	content := widget.VBox(4)
	if bar {
		content.Add(widget.MenuBar(widget.MenuItem{Text: "&File", Children: []widget.MenuItem{{Text: "Open", OnClick: func() { opened.Add(1) }}}}))
	}
	f.SetContent(content.Add(ctl))
	ctl.Focus()
	f.Show()
	t.Cleanup(f.Close)
	eventually(t, "the form's window", func() bool { return f.Native() != nil })
	// An element has the focus only while its window has the keyboard: the
	// window comes to the front first, then the recorder reports it.
	mnFront(t, f)
	eventually(t, "the recorder's focus", rec.hasFocus)
	return f, rec
}

// Alt pressed and let go alone takes the keyboard to a MenuBar drawn in the
// content, in a window without a bar of Win32's: F then opens File, Enter
// chooses Open, the system's window menu loop never starts, and the focus
// goes back to the recorder, which got no key before the X typed after.
// Under a header bar too, the window then having no caption.
func TestAltAloneTakesTheKeyboardToAMenuBar(t *testing.T) {
	for _, header := range []bool{false, true} {
		var opened atomic.Int32
		f, rec := recorderForm(t, true, header, &opened)
		mnPress(mnVkAlt)
		eventually(t, "the bar taking the focus from the recorder", func() bool { return !rec.hasFocus() })
		mnPress('F')
		mnPress(0x0D) // Enter
		eventually(t, "Open chosen from the keyboard", func() bool { return opened.Load() == 1 })
		if mnMenuMode() {
			t.Errorf("header %v: Alt alone started the window menu's loop", header)
			mnPress(mnVkEscape)
		}
		eventually(t, "the focus back in the recorder", rec.hasFocus)
		mnPress('X')
		eventually(t, "X reaching the recorder", func() bool { return slices.Contains(rec.got(), ui.KeyX) })
		if got := rec.got(); !slices.Equal(got, []ui.Key{ui.KeyX}) {
			t.Errorf("header %v: the recorder got %v, want X alone", header, got)
		}
		f.Close()
	}
}

// Alt alone reaches no control as a key, with a MenuBar or without one:
// a terminal would write F10 to its program, which quits htop.
func TestAltAloneSendsNoKeyToTheFocusedControl(t *testing.T) {
	// With a MenuBar: Alt takes the keyboard to it, Escape gives it back,
	// and the recorder gets the X typed after, alone.
	var opened atomic.Int32
	f, rec := recorderForm(t, true, false, &opened)
	mnPress(mnVkAlt)
	eventually(t, "the bar taking the focus from the recorder", func() bool { return !rec.hasFocus() })
	if mnMenuMode() {
		t.Error("with a MenuBar, Alt alone started the window menu's loop")
		mnPress(mnVkEscape)
	}
	mnPress(mnVkEscape)
	eventually(t, "the focus back in the recorder", rec.hasFocus)
	mnPress('X')
	eventually(t, "X reaching the recorder", func() bool { return slices.Contains(rec.got(), ui.KeyX) })
	if got := rec.got(); !slices.Equal(got, []ui.Key{ui.KeyX}) {
		t.Errorf("with a MenuBar, the recorder got %v, want X alone", got)
	}
	f.Close()

	// Without one: Alt goes to the system, as before, which starts the
	// window menu's loop; the recorder gets nothing.
	_, rec = recorderForm(t, false, false, &opened)
	mnPress(mnVkAlt)
	eventually(t, "the window menu's loop, Alt alone being the system's", mnMenuMode)
	mnPress(mnVkEscape)
	if got := rec.got(); len(got) != 0 {
		t.Errorf("without a MenuBar, the recorder got %v", got)
	}
}
