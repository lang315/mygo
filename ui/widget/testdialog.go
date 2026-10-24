package widget

import "github.com/egoist/mygo/ui"

// DialogTester drives a Dialog without a window, for tests: a ui.Tester
// over the form's view, with the dialog's Run, Close and Escape working
// on it. Make one with TestDialog.
type DialogTester struct {
	*ui.Tester
	d *Dialog
}

// Cleaner is what TestDialog needs of a test: *testing.T and *testing.B.
type Cleaner interface{ Cleanup(func()) }

// TestDialog returns a tester of d's view, at the dialog's size (the
// form's header, tool strip, content and status strip), and has d.Run,
// from then on, run OnBeforeRun and block until the dialog closes, as it
// does with a window, but show none: the test clicks its buttons through
// the tester, and Close, by a button or the test, ends Run. A Run after a
// close is a new show, as with a window. Call it before Run, and only in a
// test: it closes d when the test ends, so no Run outlives it.
//
//	d := NewMyDialog(nil)
//	dt := widget.TestDialog(t, d.Dialog)
//	go func() { result <- d.Run() }() // Run blocks until the dialog closes
//	dt.Type("Ada")
//	dt.Click("OK")
//	got := <-result
func TestDialog(t Cleaner, d *Dialog) *DialogTester {
	mu.Lock()
	d.headless = make(chan struct{})
	w, h := d.width, d.height
	mu.Unlock()
	// A size of 0 is the dialog's natural one, which a window finds out as
	// it shows: the tester gives it room.
	if w <= 0 {
		w = 600
	}
	if h <= 0 {
		h = 800
	}
	t.Cleanup(d.Close)
	tt := ui.NewTester(d.view, w, h)
	tt.Frame()
	return &DialogTester{Tester: tt, d: d}
}

// Closed reports whether the dialog was closed, by Close or Escape.
func (t *DialogTester) Closed() bool {
	mu.Lock()
	defer mu.Unlock()
	select {
	case <-t.d.headless:
		return true
	default:
		return false
	}
}

// Escape presses Escape as in a dialog's window: the content takes it
// first (closing a drop-down that is open, or a shortcut for it); an
// Escape nothing took is a request to close, which asks OnClose and
// closes the dialog unless it says no or SetDeletable(false) forbids the
// user to. OnClose answers on the event goroutine, as with a window:
// wait for it before Closed.
func (t *DialogTester) Escape() {
	before := t.Escapes()
	t.Key(0, ui.KeyEscape)
	t.Frame()
	if t.Escapes() > before && !t.d.closeRequest() {
		t.d.Close()
	}
}
