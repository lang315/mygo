package widget

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
)

// The event goroutine runs the handlers of every widget, one at a time.
// The main thread never waits on it; it may wait on the main thread (a
// dialog's window is made there), so its queue is unbounded and Post never
// blocks.
var (
	qmu     sync.Mutex
	queue   []func()
	wake    = make(chan struct{}, 1)
	eventID atomic.Uint64
	start   sync.Once
)

// Post runs fn on the event goroutine after what was posted before it,
// without waiting. It is safe from any goroutine, the main thread too.
func Post(fn func()) {
	start.Do(func() { go events() })
	qmu.Lock()
	queue = append(queue, fn)
	qmu.Unlock()
	select {
	case wake <- struct{}{}:
	default:
	}
}

// Update runs fn on the event goroutine and waits for it, so that a worker
// goroutine changes widgets in step with the handlers. In a handler it
// runs fn at once. It panics on the main thread, which must not wait.
func Update(fn func()) {
	if onEvents() {
		fn()
		return
	}
	notMain("Update")
	done := make(chan struct{})
	Post(func() { defer close(done); fn() })
	<-done
}

// block waits until done is closed. On the event goroutine it runs the
// queued handlers meanwhile, as GTK runs its main loop under a dialog, so
// the dialog's own buttons work. It panics on the main thread.
func block(name string, done <-chan struct{}) {
	notMain(name)
	if !onEvents() {
		<-done
		return
	}
	for {
		select {
		case <-done:
			return
		default:
		}
		if next() {
			continue
		}
		select {
		case <-done:
			return
		case <-wake:
		}
	}
}

func notMain(name string) {
	// Before the app is ready no thread runs it yet, and asking would
	// start the platform backend.
	if mygo.App.IsReady() && mygo.IsMainThread() {
		panic("widget: " + name + " called on the main thread; call it from a handler or another goroutine")
	}
}

func events() {
	eventID.Store(goid())
	for range wake {
		for next() {
		}
	}
}

// next runs the first queued function, if any.
func next() bool {
	qmu.Lock()
	if len(queue) == 0 {
		qmu.Unlock()
		return false
	}
	fn := queue[0]
	queue[0] = nil
	queue = queue[1:]
	qmu.Unlock()
	fn()
	return true
}

// onEvents reports whether the caller runs on the event goroutine.
func onEvents() bool {
	id := eventID.Load()
	return id != 0 && id == goid()
}

// goid returns the caller's goroutine id, parsed from its stack header
// ("goroutine 18 [running]:"), as internal/fake does. ~2µs: only Update
// and blocking calls pay it, never Post or frames.
func goid() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	id, _ := strconv.ParseUint(string(b), 10, 64)
	return id
}
