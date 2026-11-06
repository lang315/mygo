package widget

import (
	"image/color"
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
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
	w := NewForm("Hello", 640, 480)
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
	w := NewForm("Redraw", 300, 200)
	w.SetContent(l)
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
	before := frames(fw)
	l.SetText("b")
	wait.Until(t, "SetText to redraw the window", func() bool {
		mygo.RunOnMain(func() { fw.FakeSurface().Frame() })
		return frames(fw) > before
	})
}

func TestHeaderBarHidesTheTitleBar(t *testing.T) {
	w := NewForm("Bar", 400, 300)
	w.UseHeaderBar("subtitle").PackEnd(Button("Act"))
	w.Show()
	defer w.Close()
	if lastFake(t).Opts.TitleBarStyle != string(mygo.TitleBarHidden) {
		t.Fatalf("title bar style %q", lastFake(t).Opts.TitleBarStyle)
	}
}

func TestHeaderBarClientDecorations(t *testing.T) {
	plain := NewForm("Bar", 400, 300)
	plain.UseHeaderBar("")
	plain.Show()
	defer plain.Close()
	if lastFake(t).Opts.ClientDecorations {
		t.Fatal("ClientDecorations set by default")
	}
	w := NewForm("Own", 400, 300)
	w.UseHeaderBar("").ClientDecorations(true)
	w.Show()
	defer w.Close()
	if !lastFake(t).Opts.ClientDecorations {
		t.Fatal("ClientDecorations(true) did not reach the window")
	}
}

func TestOnCloseCanKeepTheWindow(t *testing.T) {
	w := NewForm("Keep", 300, 200)
	asked := make(chan struct{}, 1)
	w.OnClose(func() bool { asked <- struct{}{}; return false })
	w.Show()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	received(t, asked)
	if fw.IsClosed() {
		t.Fatal("window closed though OnClose said no")
	}
	w.OnClose(nil)
	w.Close()
}

// Review Focus 5: setters after the window closed do not touch it.
func TestSettersAfterCloseDoNothingToTheWindow(t *testing.T) {
	l := Label("a")
	w := NewForm("Gone", 300, 200)
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
	h := NewForm("Dup", 300, 200).UseHeaderBar("")
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
	w := NewForm("Quit", 300, 200)
	w.OnClose(func() bool { return true })
	w.Show()
	mygo.App.Quit()
	received(t, requitted)
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
	w := NewForm("NoQuit", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return false })
	w.Show()
	mygo.App.Quit() // asked, says no
	received(t, asked)
	// The handler has sent, not returned: a request while it asks is dropped.
	// What the event goroutine runs next comes after it has returned.
	Update(func() {})
	w.OnClose(func() bool { asked <- struct{}{}; return true })
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	received(t, asked)
	Update(func() {})
	select {
	case <-quits:
		t.Fatal("closing one window quit the app")
	default:
	}
	w.Close()
}

func lastFakeClosed(w *Form) bool {
	for _, fw := range fb.Windows() {
		if fw.Title() == "Quit" {
			return fw.IsClosed()
		}
	}
	return false
}

// waitClosed waits until w's native window has closed.
func waitClosed(t *testing.T, w *Form) {
	t.Helper()
	mu.Lock()
	done := w.done
	mu.Unlock()
	received(t, done)
}

func TestShowAfterCloseShowsAgain(t *testing.T) {
	w := NewForm("Again", 300, 200)
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
	w := NewForm("Once", 300, 200)
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
	w := NewForm("AskOnce", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return <-answer })
	w.Show()
	defer w.Close()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	received(t, asked)
	mygo.RunOnMain(func() { fw.H.ShouldClose() }) // while the first asks
	answer <- false
	Update(func() {})
	select {
	case <-asked:
		t.Fatal("a second request asked again while the first was asking")
	default:
	}
	mygo.RunOnMain(func() { fw.H.ShouldClose() }) // after the answer: asks
	received(t, asked)
	answer <- false
}

func TestMaximizeBeforeShow(t *testing.T) {
	w := NewForm("Max", 300, 200).Maximize()
	w.Show()
	defer w.Close()
	if !lastFake(t).Opts.Maximized {
		t.Fatal("not maximized when shown")
	}
}

func TestOnCloseSurvivesClose(t *testing.T) {
	asked := make(chan struct{}, 1)
	w := NewForm("Keeps", 300, 200)
	w.OnClose(func() bool { asked <- struct{}{}; return true })
	w.Show()
	w.Close()
	waitClosed(t, w)
	w.Show()
	fw := lastFake(t)
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	received(t, asked)
	waitClosed(t, w)
}

// Close while Show is still making the window: the window closes.
func TestCloseWhileShowMakesTheWindow(t *testing.T) {
	w := NewForm("Racing", 300, 200)
	off := mygo.App.OnWindowCreated(func(n *mygo.Window) {
		if n.Title() == "Racing" {
			w.Close() // Show has made n but not stored it yet
		}
	})
	defer off()
	w.Show()
	waitClosed(t, w)
}

// redBounds is the box of the pure red pixels of the frame (the picture).
func redBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	img := tt.Image()
	minX, minY, maxX, maxY := 1<<30, 1<<30, -1, -1
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if p := img.RGBAAt(x, y); p.R == 255 && p.G == 0 && p.B == 0 {
				minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			}
		}
	}
	if maxX < 0 {
		t.Fatal("no red picture in the frame")
	}
	return ui.Rect{X: float32(minX), Y: float32(minY), W: float32(maxX - minX + 1), H: float32(maxY - minY + 1)}
}

func TestHeaderBarGrowsToHoldTitleAndSubtitle(t *testing.T) {
	f := NewForm("Title", 800, 600).SetContent(Label("content"))
	f.UseHeaderBar("subtitle").PackStart(PictureBox(testPNG(t, color.RGBA{255, 0, 0, 255}), 32, 32))
	tt := formTester(t, f)
	title, sub, content := find(t, tt, "Title"), find(t, tt, "subtitle"), find(t, tt, "content")
	if title.Y <= 0 {
		t.Fatalf("the title at %v sits against the window's top edge", title)
	}
	if bottom := sub.Y + sub.H; bottom >= content.Y {
		t.Fatalf("the subtitle ends at %v, not above the content at %v", bottom, content.Y)
	}
	pic := redBounds(t, tt)
	barH := content.Y // the content starts where the bar ends (up to its padding)
	if pic.Y <= 0 || pic.Y+pic.H >= barH {
		t.Fatalf("the picture %v is not inside the bar of height ~%v", pic, barH)
	}
	// Centered on the text column, as on the bar's middle.
	if mid, textMid := pic.Y+pic.H/2, (title.Y+sub.Y+sub.H)/2; mid < textMid-1 || mid > textMid+1 {
		t.Fatalf("the picture's middle %v is not the text's middle %v", mid, textMid)
	}
}

func TestHeaderBarWithoutSubtitleKeepsItsHeight(t *testing.T) {
	f := NewForm("Title", 800, 600).SetContent(Label("content"))
	f.UseHeaderBar("").PackEnd(Button("x"))
	tt := formTester(t, f)
	if content := find(t, tt, "content"); content.Y != 32 {
		t.Fatalf("a bar with no subtitle grew: the content starts at %v", content.Y)
	}
}

func TestFitCountsAGrownHeaderBar(t *testing.T) {
	height := func(sub string) float32 {
		f := NewForm("Title", 800, 600).SetContent(Label("content"))
		f.UseHeaderBar(sub)
		return ui.View(f.view).MeasureHeight(800, 1)
	}
	if plain, tall := height(""), height("subtitle"); tall <= plain {
		t.Fatalf("a fitted window with a subtitle is %v high, one without %v", tall, plain)
	}
}

func TestHeaderBarLongTitleKeepsItsHeightAndTheFitCutsNothing(t *testing.T) {
	f := NewForm("A very long title that does not fit the narrow window at all", 800, 600).SetContent(Label("content"))
	f.UseHeaderBar("")
	tt := ui.NewTester(f.view, 200, 600)
	tt.SetTitleBar(ui.TitleBar{Height: 32, Right: 138})
	tt.Frame()
	if content := find(t, tt, "content"); content.Y != 32 {
		t.Fatalf("a long title grew the bar: the content starts at %v", content.Y)
	}
}
