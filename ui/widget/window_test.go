package widget

import (
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
)

// lastFake returns the fake of the window shown last.
func lastFake(t *testing.T) *fake.Window {
	t.Helper()
	ws := fb.Windows()
	if len(ws) == 0 {
		t.Fatal("no window")
	}
	return ws[len(ws)-1]
}

func frames(fw *fake.Window) int {
	n := 0
	mygo.RunOnMain(func() { n = fw.FakeSurface().Frames() })
	return n
}

func TestShowMakesTheWindow(t *testing.T) {
	w := NewWindow("Hello", 640, 480)
	w.SetContent(Label("content"))
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	if fw.Title() != "Hello" {
		t.Fatalf("title %q", fw.Title())
	}
	if w.Native() == nil {
		t.Fatal("Native() nil after Show")
	}
}

func TestSetterRedrawsAShownWindow(t *testing.T) {
	l := Label("a")
	w := NewWindow("Redraw", 300, 200)
	w.SetContent(l)
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
	before := frames(fw)
	l.SetText("b")
	deadline := time.Now().Add(2 * time.Second)
	for {
		mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
		if frames(fw) > before {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("SetText did not redraw the window")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHeaderBarHidesTheTitleBar(t *testing.T) {
	w := NewWindow("Bar", 400, 300)
	w.UseHeaderBar("subtitle").PackEnd(Button("Act"))
	w.Show()
	defer w.Close()
	if lastFake(t).Opts.TitleBarStyle != string(mygo.TitleBarHidden) {
		t.Fatalf("title bar style %q", lastFake(t).Opts.TitleBarStyle)
	}
}

func TestHeaderBarClientDecorations(t *testing.T) {
	plain := NewWindow("Bar", 400, 300)
	plain.UseHeaderBar("")
	plain.Show()
	defer plain.Close()
	if lastFake(t).Opts.ClientDecorations {
		t.Fatal("ClientDecorations set by default")
	}
	w := NewWindow("Own", 400, 300)
	w.UseHeaderBar("").ClientDecorations(true)
	w.Show()
	defer w.Close()
	if !lastFake(t).Opts.ClientDecorations {
		t.Fatal("ClientDecorations(true) did not reach the window")
	}
}

func TestOnCloseCanKeepTheWindow(t *testing.T) {
	w := NewWindow("Keep", 300, 200)
	asked := make(chan struct{}, 1)
	w.OnClose(func() bool { asked <- struct{}{}; return false })
	w.Show()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	wait(t, asked)
	if fw.IsClosed() {
		t.Fatal("window closed though OnClose said no")
	}
	w.OnClose(nil)
	w.Close()
}

// Review Focus 5: setters after the window closed do not touch it.
func TestSettersAfterCloseDoNothingToTheWindow(t *testing.T) {
	l := Label("a")
	w := NewWindow("Gone", 300, 200)
	w.SetContent(l)
	w.Show()
	n := w.Native()
	w.Close()
	waitClosed(t, w)
	l.SetText("b") // must not panic or redraw a dead window
	mu.Lock()
	_, still := shown[n]
	mu.Unlock()
	if still {
		t.Fatal("closed window still redrawn by changes")
	}
}

// Review Focus 2 for the header bar: a widget packed twice shows once.
func TestHeaderBarPackingAWidgetTwiceKeepsOne(t *testing.T) {
	b := Button("Twice")
	h := NewWindow("Dup", 300, 200).UseHeaderBar("")
	h.PackStart(b).PackStart(b).PackEnd(b)
	tt := tester(t, h)
	n := 0
	for _, s := range tt.Texts() {
		if s == "Twice" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("Twice shown %d times: %q", n, tt.Texts())
	}
}

// Quitting asks OnClose; a yes closes the window and quits again.
func TestOnCloseDuringQuitQuitsAgainOnYes(t *testing.T) {
	requitted := make(chan struct{}, 1)
	old := quit
	quit = func() { requitted <- struct{}{} }
	defer func() { quit = old }()
	w := NewWindow("Quit", 300, 200)
	w.OnClose(func() bool { return true })
	w.Show()
	mygo.App.Quit()
	wait(t, requitted)
	if !lastFakeClosed(w) {
		t.Fatal("window not closed after OnClose said yes")
	}
}

// A plain close after a quit attempt does not quit.
func TestOnCloseAfterQuitAttemptDoesNotQuit(t *testing.T) {
	quits := make(chan struct{}, 4)
	old := quit
	quit = func() { quits <- struct{}{} }
	defer func() { quit = old }()
	asked := make(chan struct{}, 4)
	w := NewWindow("NoQuit", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return false })
	w.Show()
	mygo.App.Quit() // asked, says no
	wait(t, asked)
	w.OnClose(func() bool { asked <- struct{}{}; return true })
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	wait(t, asked)
	Update(func() {})
	select {
	case <-quits:
		t.Fatal("closing one window quit the app")
	default:
	}
	w.Close()
}

func lastFakeClosed(w *Window) bool {
	for _, fw := range fb.Windows() {
		if fw.Title() == "Quit" {
			return fw.IsClosed()
		}
	}
	return false
}

// waitClosed waits until w's native window has closed.
func waitClosed(t *testing.T, w *Window) {
	t.Helper()
	mu.Lock()
	done := w.done
	mu.Unlock()
	wait(t, done)
}

func TestShowAfterCloseShowsAgain(t *testing.T) {
	w := NewWindow("Again", 300, 200)
	w.Show()
	first := w.Native()
	w.Close()
	waitClosed(t, w)
	if w.Native() != nil {
		t.Fatal("Native() not nil after the window closed")
	}
	w.Show()
	defer w.Close()
	if n := w.Native(); n == nil || n == first {
		t.Fatalf("Native() %p after a second Show, first %p", n, first)
	}
}

func TestConcurrentShowMakesOneWindow(t *testing.T) {
	n := len(fb.Windows())
	w := NewWindow("Once", 300, 200)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(w.Show)
	}
	wg.Wait()
	defer w.Close()
	if got := len(fb.Windows()); got != n+1 {
		t.Fatalf("%d windows made, want 1", got-n)
	}
}

func TestCloseRequestsWhileAskingAskOnce(t *testing.T) {
	asked := make(chan struct{}, 4)
	answer := make(chan bool)
	w := NewWindow("AskOnce", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return <-answer })
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	wait(t, asked)
	mygo.RunOnMain(func() { fw.H.ShouldClose() }) // while the first asks
	answer <- false
	Update(func() {})
	select {
	case <-asked:
		t.Fatal("a second request asked again while the first was asking")
	default:
	}
	mygo.RunOnMain(func() { fw.H.ShouldClose() }) // after the answer: asks
	wait(t, asked)
	answer <- false
}

func TestMaximizeBeforeShow(t *testing.T) {
	w := NewWindow("Max", 300, 200).Maximize()
	w.Show()
	defer w.Close()
	if !lastFake(t).Opts.Maximized {
		t.Fatal("not maximized when shown")
	}
}

func TestOnCloseSurvivesClose(t *testing.T) {
	asked := make(chan struct{}, 1)
	w := NewWindow("Keeps", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return true })
	w.Show()
	w.Close()
	waitClosed(t, w)
	w.Show()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	wait(t, asked)
	waitClosed(t, w)
}

// Close while Show is still making the window: the window closes.
func TestCloseWhileShowMakesTheWindow(t *testing.T) {
	w := NewWindow("Racing", 300, 200)
	off := mygo.App.OnWindowCreated(func(n *mygo.Window) {
		if n.Title() == "Racing" {
			w.Close() // Show has made n but not stored it yet
		}
	})
	defer off()
	w.Show()
	waitClosed(t, w)
}
