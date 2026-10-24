package widget

import (
	"sync"
	"testing"
	"time"
)

// fakeClock replaces the timer's scheduler: nothing fires until the test
// says so, and the test sees what is armed.
type fakeClock struct {
	mu      sync.Mutex
	armed   []*fakeTimer
	started int
}

type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (c *fakeClock) after(d time.Duration, f func()) func() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ft := &fakeTimer{d: d, f: f}
	c.armed = append(c.armed, ft)
	c.started++
	return func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		was := !ft.stopped
		ft.stopped = true
		return was
	}
}

// live returns the timers still armed.
func (c *fakeClock) live() []*fakeTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*fakeTimer
	for _, t := range c.armed {
		if !t.stopped {
			out = append(out, t)
		}
	}
	return out
}

// fire runs the callback of the one armed timer, as its time coming does,
// and marks it fired (a one-shot).
func (c *fakeClock) fire(t *testing.T) {
	t.Helper()
	l := c.live()
	if len(l) != 1 {
		t.Fatalf("%d timers armed, want 1", len(l))
	}
	c.mu.Lock()
	l[0].stopped = true
	f := l[0].f
	c.mu.Unlock()
	f()
}

func withFakeClock(t *testing.T) *fakeClock {
	t.Helper()
	c := &fakeClock{}
	old := timerAfter
	timerAfter = c.after
	t.Cleanup(func() { timerAfter = old })
	return c
}

func TestTimerTicksOnTheEventGoroutineOncePerFire(t *testing.T) {
	clock := withFakeClock(t)
	ticks := make(chan bool, 8)
	tm := Timer(time.Second).OnTick(func() { ticks <- onEvents() }).Start()
	if !tm.Enabled() {
		t.Fatal("not enabled after Start")
	}
	if l := clock.live(); len(l) != 1 || l[0].d != time.Second {
		t.Fatalf("armed %+v, want one of a second", l)
	}
	clock.fire(t)
	if !received(t, ticks) {
		t.Fatal("OnTick ran off the event goroutine")
	}
	clock.fire(t) // it re-arms itself for the next
	received(t, ticks)
}

func TestTimerDoesNotTickBeforeStartOrAfterStop(t *testing.T) {
	clock := withFakeClock(t)
	ticks := make(chan struct{}, 8)
	tm := Timer(time.Second).OnTick(func() { ticks <- struct{}{} })
	if len(clock.live()) != 0 || tm.Enabled() {
		t.Fatal("a timer that was not started is armed")
	}
	tm.Start()
	tm.Stop()
	if len(clock.live()) != 0 || tm.Enabled() {
		t.Fatalf("armed after Stop: %+v", clock.live())
	}
	<-posted()
	select {
	case <-ticks:
		t.Fatal("ticked")
	default:
	}
}

// A tick that was posted before Stop, and has not run yet, is dropped.
func TestTimerStopDropsATickAlreadyPosted(t *testing.T) {
	clock := withFakeClock(t)
	ran := false
	tm := Timer(time.Second).OnTick(func() { ran = true }).Start()
	gate := make(chan struct{})
	Post(func() { <-gate }) // the event goroutine is busy
	clock.fire(t)           // the tick waits behind the gate
	tm.Stop()
	close(gate)
	<-posted()
	if ran {
		t.Fatal("a tick ran after Stop returned")
	}
}

func TestTimerTicksThatPileUpRunOnce(t *testing.T) {
	clock := withFakeClock(t)
	n := 0
	tm := Timer(time.Second).OnTick(func() { n++ }).Start()
	gate := make(chan struct{})
	Post(func() { <-gate })
	clock.fire(t)
	clock.fire(t)
	clock.fire(t)
	close(gate)
	<-posted()
	if n != 1 {
		t.Fatalf("%d ticks ran for three fires while the handler could not run: a slow handler must not build a backlog", n)
	}
	tm.Stop()
}

func TestTimerStartRestartsAndSetIntervalRearms(t *testing.T) {
	clock := withFakeClock(t)
	tm := Timer(time.Second).Start()
	tm.Start()
	if l := clock.live(); len(l) != 1 {
		t.Fatalf("%d armed after Start twice, want 1", len(l))
	}
	tm.SetInterval(5 * time.Second)
	if l := clock.live(); len(l) != 1 || l[0].d != 5*time.Second || tm.Interval() != 5*time.Second {
		t.Fatalf("armed %+v after SetInterval", l)
	}
	tm.Stop()
	tm.SetInterval(time.Minute)
	if len(clock.live()) != 0 {
		t.Fatal("SetInterval started a stopped timer")
	}
}

func TestTimerCanStopAndStartItselfInItsHandler(t *testing.T) {
	clock := withFakeClock(t)
	var tm *TimerWidget
	n := make(chan int, 4)
	count := 0
	tm = Timer(time.Second).OnTick(func() {
		count++
		if count == 1 {
			tm.Stop()
			tm.Start() // from the handler, restarts
		}
		if count == 2 {
			tm.Stop()
		}
		n <- count
	})
	tm.Start()
	clock.fire(t)
	received(t, n)
	clock.fire(t)
	received(t, n)
	if tm.Enabled() || len(clock.live()) != 0 {
		t.Fatal("still running after Stop in the handler")
	}
}

func TestTimerIntervalMustBePositive(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Timer(%v) did not panic", d)
				}
			}()
			Timer(d)
		}()
	}
}

// With the real clock: only conditions, no durations, decide the test.
func TestTimerTicksWithTheRealClock(t *testing.T) {
	ticks := make(chan struct{}, 16)
	tm := Timer(time.Millisecond).OnTick(func() { ticks <- struct{}{} }).Start()
	for range 3 {
		received(t, ticks)
	}
	tm.Stop()
}

// Start and Stop from many goroutines at once, for the race detector; and a
// tick never runs once the last Stop returned.
func TestTimerStartAndStopFromManyGoroutines(t *testing.T) {
	var ticks int
	tm := Timer(time.Millisecond).OnTick(func() { ticks++ })
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				tm.Start()
				tm.Stop()
				tm.SetInterval(time.Duration(1+i%3) * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	tm.Stop()
	<-posted()
	at := ticks
	<-posted()
	if ticks != at || tm.Enabled() {
		t.Fatalf("ticks went from %d to %d after the last Stop", at, ticks)
	}
}

// The clock's callback may be running as Stop and Start go by (its stop
// returns false): it must not tick or arm a second timer for the new run.
func TestTimerIgnoresALateCallbackOfAnEarlierRun(t *testing.T) {
	clock := withFakeClock(t)
	ran := false
	tm := Timer(time.Second).OnTick(func() { ran = true }).Start()
	old := clock.live()[0].f
	tm.Stop()
	tm.Start()
	old()
	<-posted()
	if ran || len(clock.live()) != 1 {
		t.Fatalf("ticked %v, %d timers armed: a callback of the first run drove the second", ran, len(clock.live()))
	}
}
