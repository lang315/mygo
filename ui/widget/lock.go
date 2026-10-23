package widget

import (
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
)

// mu guards the state of every widget of the package: views hold it while
// they build, setters while they change a widget.
//
// shortcut: one lock for all windows, as contention is UI-scale; split it
// per window if a profile shows waits.
var mu sync.Mutex

// viewing is set while a view builds, holding mu: ui calls a table back
// (keys, labels, cells) both then and, without mu, while it applies input
// or lays out rows it did not build. Every ui frame runs on the main
// thread (a Tester's on its goroutine), so a view building is the only
// one.
var viewing atomic.Bool

var lastID atomic.Int64

// newID returns a widget id, unique in the process, for the key of its
// element.
func newID() int { return int(lastID.Add(1)) }

// shown are the windows on screen, with the Form each shows, which a
// change redraws if it holds the widget. Guarded by mu.
var shown = map[*mygo.Window]*Form{}

// invalidate redraws a window; tests replace it to see which ones change
// redraws.
var invalidate = func(w *mygo.Window) { w.Invalidate() }

// change runs fn under mu, then redraws the windows on screen: for a change
// that is no widget's, as the state a Custom control's build reads.
func change(fn func()) {
	mu.Lock()
	fn()
	wins := make([]*mygo.Window, 0, len(shown))
	for w := range shown {
		wins = append(wins, w)
	}
	mu.Unlock()
	for _, w := range wins {
		invalidate(w)
	}
}

// change runs fn under mu, then redraws the windows that hold the widget:
// those whose view built it, or keeps its state while it is hidden or out
// of the view for a while.
func (b *core) change(fn func()) {
	mu.Lock()
	fn()
	var wins []*mygo.Window
	for w, f := range shown {
		if _, ok := f.keep.widgets[b.id]; ok {
			wins = append(wins, w)
		}
	}
	mu.Unlock()
	for _, w := range wins {
		invalidate(w)
	}
}

// change runs fn under mu, then redraws the window, if shown.
func (w *Form) change(fn func()) {
	mu.Lock()
	fn()
	n := w.win
	mu.Unlock()
	if n != nil {
		invalidate(n)
	}
}
