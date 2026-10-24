package widget

import (
	"bytes"
	"log"
	"runtime"
	"runtime/debug"
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
	posts   atomic.Uint64 // how many Posts ran so far, for Tester.Settle
	start   sync.Once
)

// Post runs fn on the event goroutine after what was posted before it,
// without waiting. It is safe from any goroutine, the main thread too.
func Post(fn func()) {
	start.Do(func() { go events() })
	posts.Add(1)
	qmu.Lock()
	queue = append(queue, fn)
	qmu.Unlock()
	select {
	case wake <- struct{}{}:
	default:
	}
}

// BeginInvoke is Post under the name WinForms gives it: fn runs on the
// event goroutine after what was posted before it, and BeginInvoke does not
// wait.
func BeginInvoke(fn func()) { Post(fn) }

// appRunning reports whether the app is running, for Invoke; tests replace
// it. The app runs from when it is ready until Run returns.
var appRunning = func() bool { return mygo.App.IsReady() && !stopped.Load() }

// stopped is set when the app has quit: mygo calls OnQuit after its event
// loop stopped, for an app that started through widget.Run or mygo.App.Run.
var stopped atomic.Bool

var _ = mygo.App.OnQuit(markStopped)

func markStopped() { stopped.Store(true) }

// Invoke runs fn on the event goroutine and waits for it, as WinForms'
// Control.Invoke does, so that a worker goroutine changes widgets in step
// with the handlers. On the event goroutine, in a handler, it runs fn at
// once: it never waits for itself. It panics on the main thread, which must
// not wait. Before the app runs, or after it quit (the event loop stopped,
// whether Run or mygo.App.Run started it), it returns without running fn.
// Do not call it in a callback that runs under the package's lock
// (UpdateRow's fn, a RowSource's key func): it would wait for the lock.
//
// A panic in fn is recovered on the event goroutine and reported as any
// handler's is (see OnPanic), and Invoke panics again with the same value
// in the caller's goroutine, so the caller sees it.
func Invoke(fn func()) {
	if onEvents() {
		fn() // a panic goes up to the handler that called, which recovers it
		return
	}
	notMain("Invoke")
	if !appRunning() {
		return
	}
	var (
		done     = make(chan struct{})
		panicked bool
		value    any
	)
	Post(func() {
		defer close(done)
		defer func() {
			if v := recover(); v != nil {
				panicked, value = true, v
				report(v, debug.Stack())
			}
		}()
		fn()
	})
	<-done
	if panicked {
		panic(value)
	}
}

// Update is Invoke, as the package named it before.
func Update(fn func()) { Invoke(fn) }

var (
	panicMu      sync.Mutex
	panicHandler func(v any, stack []byte)
)

// OnPanic sets fn to be called, on the event goroutine, with the value and
// the stack of a panic in a handler: one run through Post, BeginInvoke or
// Invoke, or by a widget's event (OnClick, OnChange, a binding's set, a menu
// item's click...). The panic is recovered, so the app keeps running; with
// no fn, or nil, it is logged with its stack. A panic in fn itself is
// logged too.
func OnPanic(fn func(v any, stack []byte)) {
	panicMu.Lock()
	panicHandler = fn
	panicMu.Unlock()
}

// report hands a recovered panic to OnPanic's function, or the log.
func report(v any, stack []byte) {
	panicMu.Lock()
	h := panicHandler
	panicMu.Unlock()
	if h != nil {
		defer func() {
			if v2 := recover(); v2 != nil {
				log.Printf("widget: the OnPanic handler panicked: %v\n%s", v2, debug.Stack())
			}
		}()
		h(v, stack)
		return
	}
	log.Printf("widget: panic in a handler: %v\n%s", v, stack)
}

// run runs a handler, recovering and reporting its panic.
func run(fn func()) {
	defer func() {
		if v := recover(); v != nil {
			report(v, debug.Stack())
		}
	}()
	fn()
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
	run(fn)
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
