package widget

import (
	"cmp"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Splash is what ShowSplash shows: an image, a title, a status line and a
// progress bar of unknown length, in a window Width by Height DIPs. A zero
// side is the default's: 480 wide, 300 high.
type Splash struct {
	Image         *ui.Bitmap
	Title, Status string
	Width, Height int
}

// SplashScreen is a splash ShowSplash shows, until Close or CloseWhenShown.
type SplashScreen struct {
	s      Splash       // Status follows SetStatus
	win    *mygo.Window // nil until the window opens, and once closed
	closed bool
}

// ShowSplash shows s in a frameless window centered on the screen, above
// other windows and with no taskbar button, while the app starts, as
// DevExpress's SplashScreenManager. It may be called from any goroutine, and
// before Run, in main: it does not wait, and the window opens once the app is
// ready. Close it with CloseWhenShown(form), so the form shows before the
// splash goes: an app whose last window closes may quit.
func ShowSplash(s Splash) *SplashScreen {
	sp := &SplashScreen{s: s}
	mygo.App.WhenReady(sp.open)
	return sp
}

// open makes the window, on the main thread, unless Close came first.
func (sp *SplashScreen) open() {
	mu.Lock()
	if sp.closed {
		mu.Unlock()
		return
	}
	s := sp.s
	mu.Unlock()
	// No X and Y: the window is centered.
	n := mygo.NewWindow(mygo.WindowOptions{
		Title: s.Title, Width: cmp.Or(s.Width, 480), Height: cmp.Or(s.Height, 300),
		Frameless: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true,
		Content: ui.View(sp.view),
	})
	mu.Lock()
	closeNow := sp.closed // Close ran while the window was being made
	if !closeNow {
		sp.win = n
	}
	mu.Unlock()
	if closeNow {
		n.Close()
	}
}

// SetStatus sets the status line. It is safe from any goroutine.
func (sp *SplashScreen) SetStatus(text string) {
	mu.Lock()
	sp.s.Status = text
	n := sp.win
	mu.Unlock()
	if n != nil {
		invalidate(n)
	}
}

// Close closes the splash; called again, it does nothing. It is safe from
// any goroutine.
func (sp *SplashScreen) Close() {
	mu.Lock()
	n := sp.win
	sp.closed, sp.win = true, nil
	mu.Unlock()
	if n != nil {
		n.Close()
	}
}

// CloseWhenShown closes the splash once f has shown, at once if it shows
// already, so the form appears before the splash goes. It is safe from any
// goroutine.
func (sp *SplashScreen) CloseWhenShown(f *Form) {
	mu.Lock()
	now := f.win != nil
	if !now {
		prev := f.onShown
		f.onShown = func() {
			if prev != nil {
				prev()
			}
			sp.Close()
		}
	}
	mu.Unlock()
	if now {
		sp.Close()
	}
}

func (sp *SplashScreen) view(c *ui.Context) {
	mu.Lock()
	s := sp.s
	mu.Unlock()
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(3)).Background(t.Background).Border(1, t.Border).
		Children(func() {
			if s.Image != nil {
				// Scaled down to the room left, keeping its aspect ratio.
				ui.Image(c, s.Image).Grow(1).Shrink(1).AlignSelf(ui.Center)
			} else {
				ui.Spacer(c)
			}
			if s.Title != "" {
				ui.Text(c, s.Title).Bold().FontSize(t.FontSize * 1.5)
			}
			ui.Text(c, s.Status).TextColor(t.TextMuted).MaxLines(1).Role(ui.RoleStatus)
			ui.Progress(c, -1)
		})
}
