package skin

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// themeTester returns a Tester of a view that keeps the theme of its last
// frame in *got.
func themeTester(t *testing.T, got *ui.Theme) *widget.Tester {
	return widget.NewTester(t, widget.Custom(func(c *ui.Context) ui.Element {
		*got = *c.Theme()
		return ui.Text(c, "x")
	}), 200, 100)
}

// activeName names the active skin for a test's message: "<nil>" for none.
func activeName() string {
	if s := Active(); s != nil {
		return s.Name()
	}
	return "<nil>"
}

// Use(Calm) gives a window Calm's theme, light and dark, and its tokens to
// the hook.
func TestUseCalm(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	if Active() != nil {
		t.Fatalf("before Use, Active is %v, want nil", activeName())
	}
	Use(Calm)
	tt.Frame()
	if got.Accent != ui.Hex("#4e3bb0") || got.Background != ui.Hex("#ffffff") || got.ControlBorder != ui.Hex("#838a94") {
		t.Errorf("light, Accent %v, Background %v, ControlBorder %v; want #4e3bb0, #ffffff, #838a94", got.Accent, got.Background, got.ControlBorder)
	}
	if Active() != Calm {
		t.Errorf("Active is %v, want Calm", activeName())
	}
	if h := skinhook.Get(false); h == nil || h.ControlHeight != 36 {
		t.Errorf("the hook's light tokens are %p, want Calm's with ControlHeight 36", h)
	}
	if h := skinhook.Get(true); h == nil || h.Content != ui.Hex("#15171a") {
		t.Errorf("the hook's dark tokens are %p, want Calm's dark", h)
	}
	tt.SetDark(true)
	if got.Accent != ui.Hex("#9384f0") || got.Background != ui.Hex("#15171a") {
		t.Errorf("dark, Accent %v, Background %v; want #9384f0, #15171a", got.Accent, got.Background)
	}
}

// Use does not change the appearance: a dark window stays dark, a light
// one light.
func TestUseKeepsTheAppearance(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	tt.SetDark(true)
	Use(Calm)
	tt.Frame()
	if !got.Dark {
		t.Error("a dark window is light after Use(Calm)")
	}
	tt.SetDark(false)
	Use(nil)
	Use(Calm)
	tt.Frame()
	if got.Dark {
		t.Error("a light window is dark after Use(Calm)")
	}
}

// Use(nil) with no skin does nothing wrong; Use twice is harmless; Use(nil)
// brings the plain look back in both appearances.
func TestUseNilAndRepeat(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(nil)
	tt.Frame()
	if got.Accent != ui.LightTheme().Accent || Active() != nil {
		t.Errorf("Use(nil) with no skin gives Accent %v and Active %v, want LightTheme's and nil", got.Accent, activeName())
	}
	Use(Calm)
	Use(Calm)
	tt.Frame()
	if got.Accent != ui.Hex("#4e3bb0") || Active() != Calm {
		t.Errorf("Use(Calm) twice gives Accent %v and Active %v, want #4e3bb0 and Calm", got.Accent, activeName())
	}
	Use(nil)
	tt.Frame()
	if got.Accent != ui.LightTheme().Accent || got.Background != ui.LightTheme().Background {
		t.Errorf("after Use(nil), Accent %v and Background %v, want LightTheme's", got.Accent, got.Background)
	}
	if Active() != nil || skinhook.Get(false) != nil || skinhook.Get(true) != nil {
		t.Errorf("after Use(nil), Active %v and the hook %p, %p; want nil", activeName(), skinhook.Get(false), skinhook.Get(true))
	}
	tt.SetDark(true)
	if got.Accent != ui.DarkTheme().Accent {
		t.Errorf("after Use(nil), dark, Accent %v, want DarkTheme's %v", got.Accent, ui.DarkTheme().Accent)
	}
}

// Use from another goroutine while the window draws does not race (go test
// -race), and the last call wins.
func TestUseFromAnotherGoroutine(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			Use(Calm)
			Use(Calm) // over a skin, not only over none
			Use(nil)
		}
	}()
	for range 100 {
		tt.Frame()
		_ = skinhook.Get(got.Dark)
	}
	wg.Wait()
	Use(Calm)
	tt.Frame()
	if got.Accent != ui.Hex("#4e3bb0") || Active() != Calm {
		t.Errorf("after the goroutine and Use(Calm), Accent %v and Active %v, want #4e3bb0 and Calm", got.Accent, activeName())
	}
}

// The skin's accent wins over the desktop's accent color.
func TestUseKeepsTheSkinAccentOverTheDesktop(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	tt.SetPreferences(ui.Preferences{Accent: ui.Hex("#e11d48"), TextScale: 1})
	Use(Calm)
	tt.Frame()
	if got.Accent != ui.Hex("#4e3bb0") || got.Focus != ui.Hex("#4e3bb0") || got.Selection != ui.Hex("#dcd8ef") {
		t.Errorf("with a desktop accent, Accent %v, Focus %v, Selection %v; want Calm's #4e3bb0, #4e3bb0, #dcd8ef", got.Accent, got.Focus, got.Selection)
	}
	Use(nil)
	tt.Frame()
	if got.Accent != ui.Hex("#e11d48") {
		t.Errorf("after Use(nil), Accent %v, want the desktop's #e11d48 again", got.Accent)
	}
}
