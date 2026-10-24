package widget

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type panicReport struct {
	v     any
	stack []byte
}

// reports makes OnPanic send what it is given on the returned channel, until
// the test ends.
func reports(t *testing.T) chan panicReport {
	t.Helper()
	ch := make(chan panicReport, 8)
	OnPanic(func(v any, stack []byte) { ch <- panicReport{v, stack} })
	t.Cleanup(func() { OnPanic(nil) })
	return ch
}

// idle waits until the handlers posted so far have run.
func idle() { Update(func() {}) }

func TestInvokeFromAGoroutineWaitsAndSeesTheEffect(t *testing.T) {
	x := 0
	var seenOn atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		Invoke(func() { x = 1; seenOn.Store(onEvents()) })
		if x != 1 { // fn ran before Invoke returned
			t.Error("Invoke returned before fn ran")
		}
	}()
	received(t, done)
	if !seenOn.Load() {
		t.Fatal("fn did not run on the event goroutine")
	}
}

func TestInvokeInAHandlerRunsInline(t *testing.T) {
	ran := make(chan bool, 1)
	Post(func() {
		inline := false
		Invoke(func() { inline = true }) // queued behind this handler, it would deadlock
		ran <- inline
	})
	if !received(t, ran) {
		t.Fatal("Invoke in a handler did not run fn at once")
	}
}

func TestInvokeInAnInvokeRunsInline(t *testing.T) {
	x := 0
	Invoke(func() { Invoke(func() { x++ }); x++ })
	if x != 2 {
		t.Fatalf("x %d", x)
	}
}

func TestBeginInvokeDoesNotBlock(t *testing.T) {
	release := make(chan struct{})
	BeginInvoke(func() { <-release }) // holds the event goroutine
	ran := make(chan struct{})
	BeginInvoke(func() { close(ran) }) // returns at once, though the loop is busy
	select {
	case <-ran:
		t.Fatal("ran before the first handler returned")
	default:
	}
	close(release)
	received(t, ran)
}

func TestAPanickingOnClickIsReportedOnceAndLaterEventsRun(t *testing.T) {
	got := reports(t)
	tt := tester(t, Button("Boom").OnClick(func() { panic("boom") }))
	click(t, tt, "Boom")
	r := received(t, got)
	if r.v != "boom" || !bytes.Contains(r.stack, []byte("threading_test.go")) {
		t.Fatalf("reported %v with stack\n%s", r.v, r.stack)
	}
	after := make(chan struct{})
	Post(func() { close(after) }) // the event goroutine is alive
	received(t, after)
	idle()
	if len(got) != 0 {
		t.Fatalf("reported more than once: %d more", len(got))
	}
}

func TestAPanicInPostAndBeginInvokeIsReported(t *testing.T) {
	got := reports(t)
	Post(func() { panic("post") })
	BeginInvoke(func() { panic("begin") })
	idle()
	if a, b := received(t, got), received(t, got); a.v != "post" || b.v != "begin" {
		t.Fatalf("reported %v, %v", a.v, b.v)
	}
}

func TestAPanicInBindSetIsReported(t *testing.T) {
	got := reports(t)
	tb := Bind(TextBox(), func() string { return "" }, func(string) { panic("set") })
	tt := tester(t, tb)
	typeInto(t, tt, 1, "x")
	if r := received(t, got); r.v != "set" {
		t.Fatalf("reported %v", r.v)
	}
}

func TestInvokePropagatesThePanicToTheCaller(t *testing.T) {
	got := reports(t)
	var caught any
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { caught = recover() }()
		Invoke(func() { panic("from fn") })
		t.Error("Invoke returned after fn panicked")
	}()
	received(t, done)
	if caught != "from fn" {
		t.Fatalf("the caller saw %v", caught)
	}
	if r := received(t, got); r.v != "from fn" { // and the handler was reported, too
		t.Fatalf("reported %v", r.v)
	}
	after := make(chan struct{})
	Post(func() { close(after) })
	received(t, after)
}

func TestWithoutOnPanicTheStackGoesToTheLog(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	log.SetOutput(writerFunc(func(p []byte) { mu.Lock(); buf.Write(p); mu.Unlock() }))
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	OnPanic(nil)
	Post(func() { panic("to the log") })
	idle()
	mu.Lock()
	defer mu.Unlock()
	if s := buf.String(); !strings.Contains(s, "to the log") || !strings.Contains(s, "goroutine") {
		t.Fatalf("log %q has no panic value and stack", s)
	}
}

type writerFunc func([]byte)

func (f writerFunc) Write(p []byte) (int, error) { f(p); return len(p), nil }

func TestAPanickingOnPanicHandlerDoesNotKillTheLoop(t *testing.T) {
	OnPanic(func(any, []byte) { panic("handler of panics") })
	t.Cleanup(func() { OnPanic(nil) })
	Post(func() { panic("first") })
	idle()
}

func TestInvokeWhenTheAppIsNotRunningReturnsWithoutRunningFn(t *testing.T) {
	was := appRunning
	appRunning = func() bool { return false }
	t.Cleanup(func() { appRunning = was })
	ran := false
	Invoke(func() { ran = true })
	if ran {
		t.Fatal("Invoke ran fn with no app running")
	}
}

// unlocked fails the test if the package lock is held, and frees it so that
// the rest of the tests do not hang on it.
func unlocked(t *testing.T) {
	t.Helper()
	if !mu.TryLock() {
		mu.Unlock()
		t.Fatal("the package lock is still held after a panic in a callback that runs under it")
	}
	mu.Unlock()
}

type pubRow struct {
	k int
	p *int
}

// A callback that runs under the package lock and panics must not leave it
// held: the next frame and every setter would wait for it for ever.
func TestAPanicUnderTheLockLeavesItFree(t *testing.T) {
	for name, trigger := range map[string]func(){
		"UpdateRow's fn": func() {
			src := NewRowSource(func(r *pubRow) int { return r.k }).SetRows([]pubRow{{k: 1}})
			src.UpdateRow(1, func(r *pubRow) { *r.p = 1 }) // p is nil
		},
		"a RowSource's key func": func() {
			NewRowSource(func(r *pubRow) int {
				if r.k < 0 {
					panic("bad key")
				}
				return r.k
			}).AddRows([]pubRow{{k: -1}})
		},
		"a control's change":   func() { Label("x").change(func() { panic("in change") }) },
		"a form's change":      func() { NewForm("t", 10, 10).change(func() { panic("in change") }) },
		"the package's change": func() { change(func() { panic("in change") }) },
	} {
		func() {
			defer func() { recover() }()
			trigger()
		}()
		t.Run(name, func(t *testing.T) { unlocked(t) })
	}
}

// After such a panic, a setter still redraws its window.
func TestASetterStillRedrawsAfterAPanicUnderTheLock(t *testing.T) {
	got := reports(t)
	l := Label("x")
	_, w := showFrame(t, l)
	src := NewRowSource(func(r *pubRow) int { return r.k }).SetRows([]pubRow{{k: 1}})
	Post(func() { src.UpdateRow(1, func(r *pubRow) { *r.p = 1 }) })
	received(t, got)
	redrawn := recordInvalidations(t)
	l.SetText("after") // would hang on the lock
	wantRedrawn(t, redrawn(), w)
}

// A quit app is not running: Invoke does not run fn.
func TestInvokeAfterTheAppQuitReturnsWithoutRunningFn(t *testing.T) {
	t.Cleanup(func() { stopped.Store(false) })
	ran := false
	markStopped() // what mygo's OnQuit calls
	Invoke(func() { ran = true })
	if ran {
		t.Fatal("Invoke ran fn after the app quit")
	}
}

// Many goroutines invoke while handlers post and panic (go test -race).
func TestInvokeFromManyGoroutines(t *testing.T) {
	got := reports(t)
	var n atomic.Int64
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 50 {
				Invoke(func() { n.Add(1) })
				BeginInvoke(func() { n.Add(1) })
			}
		})
	}
	Post(func() { panic("amid") })
	wg.Wait()
	idle()
	if n.Load() != 16*50*2 {
		t.Fatalf("ran %d handlers, want %d", n.Load(), 16*50*2)
	}
	if r := received(t, got); r.v != "amid" {
		t.Fatalf("reported %v", r.v)
	}
}
