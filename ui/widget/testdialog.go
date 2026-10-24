package widget

import "github.com/egoist/mygo/ui"

// DialogTester drives a Dialog without a window, for tests: a ui.Tester
// over the dialog's content, with the dialog's Run, Close and Escape
// working on it. Make one with TestDialog.
type DialogTester struct {
	*ui.Tester
	d *Dialog
}

// TestDialog returns a tester of d's content, at the dialog's size, and
// has d.Run, from then on, run OnBeforeRun and block until the dialog
// closes, as it does with a window, but show none: the test clicks its
// buttons through the tester, and Close, by a button or the test, ends
// Run. Call it before Run, and only in a test.
//
//	d := NewMyDialog(nil)
//	dt := widget.TestDialog(d.Dialog)
//	go func() { result <- d.Run() }() // Run blocks until the dialog closes
//	dt.Type("Ada")
//	dt.Click("OK")
//	got := <-result
func TestDialog(d *Dialog) *DialogTester {
	mu.Lock()
	d.headless = make(chan struct{})
	root, w, h := d.root, d.width, d.height
	mu.Unlock()
	// A size of 0 is the dialog's natural one, which a window finds out as
	// it shows: the tester gives it room.
	if w <= 0 {
		w = 600
	}
	if h <= 0 {
		h = 800
	}
	tt := ui.NewTester(content(root), w, h)
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

// Escape presses Escape as in a dialog's window: the content handles it
// first, closing a drop-down that is open; an Escape it leaves closes the
// dialog, unless SetDeletable(false) forbids the user to. Whether the
// content handled the key is what its texts show: a change means it did.
func (t *DialogTester) Escape() {
	before := t.Texts()
	t.Key(0, ui.KeyEscape)
	t.Frame()
	if slicesEqual(before, t.Texts()) {
		mu.Lock()
		deletable := t.d.deletable
		mu.Unlock()
		if deletable {
			t.d.Close()
		}
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
