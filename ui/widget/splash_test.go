package widget

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// waitSplash waits for s's window, titled title, and returns its fake.
func waitSplash(t *testing.T, s *SplashScreen, title string) *fake.Window {
	t.Helper()
	var open *fake.Window
	wait.Until(t, "the splash window "+title, func() bool {
		mu.Lock()
		n := s.win
		mu.Unlock()
		if n == nil {
			return false
		}
		for _, fw := range fb.Windows() {
			if fw.Title() == title && !fw.IsClosed() {
				open = fw
				return true
			}
		}
		return false
	})
	return open
}

// splashFakes counts the fake windows titled title, closed ones too.
func splashFakes(title string) int {
	n := 0
	for _, fw := range fb.Windows() {
		if fw.Title() == title {
			n++
		}
	}
	return n
}

func TestSplashWindowOptions(t *testing.T) {
	s := ShowSplash(Splash{Title: "SplashOpts"})
	defer s.Close()
	o := waitSplash(t, s, "SplashOpts").Opts
	if !o.Frameless || !o.Center || !o.AlwaysOnTop || !o.SkipTaskbar {
		t.Errorf("frameless %v, centered %v, on top %v, skips the taskbar %v; want all true",
			o.Frameless, o.Center, o.AlwaysOnTop, o.SkipTaskbar)
	}
	if o.Width != 480 || o.Height != 300 {
		t.Errorf("size %vx%v, want 480x300", o.Width, o.Height)
	}
}

func TestSplashSize(t *testing.T) {
	for _, tc := range []struct {
		title         string
		width, height int
		wantW, wantH  int
	}{
		{"SplashSized", 600, 400, 600, 400},
		{"SplashWidthOnly", 600, 0, 600, 300},
		{"SplashHeightOnly", 0, 200, 480, 200},
	} {
		s := ShowSplash(Splash{Title: tc.title, Width: tc.width, Height: tc.height})
		o := waitSplash(t, s, tc.title).Opts
		s.Close()
		if o.Width != tc.wantW || o.Height != tc.wantH {
			t.Errorf("%s: size %vx%v, want %vx%v", tc.title, o.Width, o.Height, tc.wantW, tc.wantH)
		}
	}
}

func TestSplashShowsItsContent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(0, 0, color.White)
	s := ShowSplash(Splash{Image: ui.NewBitmap(img), Title: "SplashContent", Status: "Loading"})
	defer s.Close()
	fw := waitSplash(t, s, "SplashContent")
	got := labels(fw)
	for _, want := range []string{"SplashContent", "Loading"} {
		if !slices.Contains(got, want) {
			t.Errorf("no %q among %q", want, got)
		}
	}
	bar := false
	for _, n := range accessNodes(fw) {
		bar = bar || n.Role == platform.RoleProgress
	}
	if !bar {
		t.Errorf("no progress bar among %q", got)
	}
}

func TestSplashStatus(t *testing.T) {
	s := ShowSplash(Splash{Title: "SplashStatus", Status: "Loading"})
	defer s.Close()
	fw := waitSplash(t, s, "SplashStatus")
	frame(fw)
	mu.Lock()
	n := s.win
	mu.Unlock()
	redrawn := make(chan struct{}, 16)
	old := invalidate
	invalidate = func(w *mygo.Window) {
		if w == n {
			redrawn <- struct{}{}
		}
		old(w)
	}
	<-runAsync(func() bool { s.SetStatus("Reading settings"); return true })
	invalidate = old
	select {
	case <-redrawn:
	default:
		t.Fatal("SetStatus from another goroutine did not redraw the splash")
	}
	if got := labels(fw); !slices.Contains(got, "Reading settings") || slices.Contains(got, "Loading") {
		t.Fatalf("the next frame shows %q, want the new status", got)
	}
}

func TestSplashClose(t *testing.T) {
	s := ShowSplash(Splash{Title: "SplashClose"})
	fw := waitSplash(t, s, "SplashClose")
	s.Close()
	wait.Until(t, "the splash to close", fw.IsClosed)
	s.Close() // does nothing
	s.SetStatus("after")
	if n := splashFakes("SplashClose"); n != 1 {
		t.Fatalf("%d windows made, want 1", n)
	}
}

// A splash closed before its window opens never opens it.
func TestSplashCloseBeforeItOpens(t *testing.T) {
	mygo.RunOnMain(func() {
		s := ShowSplash(Splash{Title: "SplashEarlyClose"})
		s.Close()
	})
	mygo.RunOnMain(func() {}) // what ShowSplash queued has run
	if n := splashFakes("SplashEarlyClose"); n != 0 {
		t.Fatalf("%d windows made, want none", n)
	}
}

// Close while the window is being made: the window closes.
func TestSplashCloseWhileItOpens(t *testing.T) {
	var s *SplashScreen
	off := mygo.App.OnWindowCreated(func(n *mygo.Window) {
		if n.Title() == "SplashRacing" {
			s.Close() // open has made n but not stored it yet
		}
	})
	defer off()
	// On the main thread, s is set before the queued open runs there.
	mygo.RunOnMain(func() { s = ShowSplash(Splash{Title: "SplashRacing"}) })
	wait.Until(t, "the splash window to be made", func() bool { return splashFakes("SplashRacing") == 1 })
	for _, fw := range fb.Windows() {
		if fw.Title() == "SplashRacing" {
			wait.Until(t, "the splash to close", fw.IsClosed)
		}
	}
}

// A form closed while Show makes its window has not shown: the splash waits
// for its next Show.
func TestSplashCloseWhenShownSkipsAShowThatClosed(t *testing.T) {
	f := NewForm("SplashMain3", 300, 200)
	s := ShowSplash(Splash{Title: "SplashSkips"})
	defer s.Close()
	fw := waitSplash(t, s, "SplashSkips")
	s.CloseWhenShown(f)
	off := mygo.App.OnWindowCreated(func(n *mygo.Window) {
		if n.Title() == "SplashMain3" {
			f.Close()
		}
	})
	f.Show()
	off()
	waitClosed(t, f)
	mygo.RunOnMain(func() {})
	if fw.IsClosed() {
		t.Fatal("the splash closed though the form closed as it showed")
	}
	f.Show()
	defer f.Close()
	wait.Until(t, "the splash to close at the next Show", fw.IsClosed)
}

func TestSplashCloseWhenShown(t *testing.T) {
	f := NewForm("SplashMain", 300, 200)
	s := ShowSplash(Splash{Title: "SplashWaits"})
	fw := waitSplash(t, s, "SplashWaits")
	s.CloseWhenShown(f)
	mygo.RunOnMain(func() {})
	if fw.IsClosed() {
		t.Fatal("the splash closed before the form showed")
	}
	f.Show()
	defer f.Close()
	wait.Until(t, "the splash to close once the form showed", fw.IsClosed)
	if f.Native() == nil {
		t.Fatal("the form is not shown")
	}
}

// Two splashes waiting on one form both close; one waiting on a form that
// shows already closes at once.
func TestSplashCloseWhenShownTwiceAndAfter(t *testing.T) {
	f := NewForm("SplashMain2", 300, 200)
	a := ShowSplash(Splash{Title: "SplashA"})
	b := ShowSplash(Splash{Title: "SplashB"})
	fa, fbw := waitSplash(t, a, "SplashA"), waitSplash(t, b, "SplashB")
	a.CloseWhenShown(f)
	b.CloseWhenShown(f)
	f.Show()
	defer f.Close()
	wait.Until(t, "both splashes to close", func() bool { return fa.IsClosed() && fbw.IsClosed() })
	c := ShowSplash(Splash{Title: "SplashC"})
	fc := waitSplash(t, c, "SplashC")
	c.CloseWhenShown(f)
	if !fc.IsClosed() {
		t.Fatal("a splash waiting on a shown form did not close")
	}
}

// ShowSplash before the app is ready, on the main thread, as main does before
// Run: it returns without making the window there (mygo.NewWindow would panic
// before the app is ready), and the window opens after.
func TestSplashBeforeRun(t *testing.T) {
	var s *SplashScreen
	made := -1
	mygo.RunOnMain(func() {
		s = ShowSplash(Splash{Title: "SplashBeforeRun"})
		made = splashFakes("SplashBeforeRun")
	})
	defer s.Close()
	if made != 0 {
		t.Fatalf("ShowSplash made %d windows before returning, want none", made)
	}
	waitSplash(t, s, "SplashBeforeRun")
	// From a goroutine it returns as well.
	g := <-runAsync(func() *SplashScreen { return ShowSplash(Splash{Title: "SplashFromGoroutine"}) })
	defer g.Close()
	waitSplash(t, g, "SplashFromGoroutine")
}
