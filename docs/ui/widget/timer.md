# Timer

`Timer(interval)` calls a function at an interval, on the event goroutine, as
WinForms' `Timer`. It is not a control: it shows nothing, and a window need
not exist.

```go
var tick *widget.TimerWidget
n := 0
tick = widget.Timer(time.Second).OnTick(func() {
	n++
	status.SetText(fmt.Sprint("Ticks: ", n))
}).Start()
defer tick.Stop()
```

- `Start()` starts it, or starts a running one over: the first tick comes an
  interval from then. `Stop()` stops it. Both are safe from any goroutine and
  from the tick's own handler.
- On the event goroutine (in a handler), no tick runs after `Stop` returns,
  not even one already posted. From another goroutine, `Stop` does not wait
  for the event goroutine: a handler already running may finish, and a tick
  that was just about to start may still run once. A tick posted by an
  earlier run never replaces the first tick of a restarted one.
- A running timer lives until `Stop`, as in WinForms: dropping the reference
  does not stop it. Ticks stop when the app quits.
- A handler slower than the interval gets no backlog: ticks that come while
  one waits to run are one tick.
- `Interval()`, `SetInterval(d)` (a running timer starts over with it) and
  `Enabled()`. The interval must be positive; `Timer(0)` panics.
- `OnTick(fn)` sets the function. A panic in it is recovered as in any handler
  (see [Threads](../widget.md#threads)).

In a test, ticks come from the real clock; wait for the condition you care
about, not for a time. `widget.Tester.Settle` runs what a tick posted.
