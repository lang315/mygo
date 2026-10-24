package widget

import (
	"sync"
	"time"
)

// timerAfter runs f once after d and returns its stop; tests replace it.
var timerAfter = func(d time.Duration, f func()) func() bool {
	return time.AfterFunc(d, f).Stop
}

// TimerWidget calls a function at an interval, on the event goroutine, as
// WinForms' Timer does. It is not a control: it shows nothing.
type TimerWidget struct {
	mu       sync.Mutex // not the package's: Start and Stop work in a handler
	interval time.Duration
	enabled  bool
	gen      uint64 // changes at each (re)start, so a late tick of an earlier run is known
	stop     func() bool
	pending  bool   // a tick is posted and has not run
	pendGen  uint64 // the run (gen) that tick belongs to
	onTick   func()
}

// Timer makes a stopped timer ticking every interval, which must be
// positive; Start starts it.
func Timer(interval time.Duration) *TimerWidget {
	checkInterval(interval)
	return &TimerWidget{interval: interval}
}

func checkInterval(d time.Duration) {
	if d <= 0 {
		panic("widget: a Timer's interval must be positive")
	}
}

// OnTick sets the function a tick calls, on the event goroutine.
func (t *TimerWidget) OnTick(fn func()) *TimerWidget {
	t.mu.Lock()
	t.onTick = fn
	t.mu.Unlock()
	return t
}

// Interval returns the time between ticks.
func (t *TimerWidget) Interval() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.interval
}

// SetInterval changes the time between ticks, which must be positive. A
// running timer starts over with it; a stopped one stays stopped.
func (t *TimerWidget) SetInterval(d time.Duration) *TimerWidget {
	checkInterval(d)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.interval = d
	if t.enabled {
		t.restart()
	}
	return t
}

// Enabled reports whether the timer is running.
func (t *TimerWidget) Enabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// Start starts the timer, or starts a running one over: the first tick
// comes an interval from now. It is safe from any goroutine, a tick's
// handler too.
func (t *TimerWidget) Start() *TimerWidget {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
	t.restart()
	return t
}

// Stop stops the timer. Called on the event goroutine, as from a handler, no
// tick runs after it returns, not even one already posted. Called from
// another goroutine, no tick that has not begun runs after it returns, but a
// tick whose handler already started may finish, and one that just passed
// its check may still start. It is safe from any goroutine.
func (t *TimerWidget) Stop() *TimerWidget {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
	if t.stop != nil {
		t.stop()
		t.stop = nil
	}
	return t
}

// restart arms the timer anew. Under t.mu.
func (t *TimerWidget) restart() {
	t.gen++
	if t.stop != nil {
		t.stop()
	}
	t.arm()
}

// arm schedules the next tick of this generation. Under t.mu.
func (t *TimerWidget) arm() {
	gen := t.gen
	t.stop = timerAfter(t.interval, func() { t.fire(gen) })
}

// fire is the clock's: it arms the next tick and posts this one, unless one
// is still waiting, since a handler slower than the interval gets no backlog.
func (t *TimerWidget) fire(gen uint64) {
	t.mu.Lock()
	if !t.enabled || gen != t.gen {
		t.mu.Unlock()
		return
	}
	t.arm()
	if t.pending && t.pendGen == gen {
		t.mu.Unlock()
		return
	}
	t.pending, t.pendGen = true, gen
	t.mu.Unlock()
	Post(func() { t.tick(gen) })
}

// tick runs on the event goroutine.
func (t *TimerWidget) tick(gen uint64) {
	t.mu.Lock()
	if t.pendGen == gen {
		t.pending = false
	}
	h, live := t.onTick, t.enabled && gen == t.gen && !stopped.Load()
	t.mu.Unlock()
	if live && h != nil {
		h()
	}
}
