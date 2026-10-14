package widget

import (
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
	w.Close()
	l.SetText("b") // must not panic or redraw a dead window
	n := w.Native()
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
	if w.Native() == nil || !lastFakeClosed(w) {
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
