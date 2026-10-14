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

// viewing is set while a view builds, holding mu: ui asks a table for the
// keys of its rows both then and while it applies input, without mu.
// Every ui frame runs on the main thread (a Tester's on its goroutine),
// so a view building is the only one.
var viewing atomic.Bool

var lastID atomic.Int64

// newID returns a widget id, unique in the process, for the key of its
// element.
func newID() int { return int(lastID.Add(1)) }

// shown are the windows on screen, which a change redraws. Guarded by mu.
var shown = map[*mygo.Window]struct{}{}

// change runs fn under mu, then redraws the windows on screen.
func change(fn func()) {
	mu.Lock()
	fn()
	wins := make([]*mygo.Window, 0, len(shown))
	for w := range shown {
		wins = append(wins, w)
	}
	mu.Unlock()
	for _, w := range wins {
		w.Invalidate()
	}
}
