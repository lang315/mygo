package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// oscColor is the terminal's answer to a color query of OSC code, as
// "10", "11" or "4;1".
func oscColor(code string, c ui.Color) string {
	return fmt.Sprintf("\x1b]%s;rgb:%04x/%04x/%04x\x1b\\", code, uint16(c.R)*257, uint16(c.G)*257, uint16(c.B)*257)
}

// A terminal whose app set no Theme takes the text and background of its
// view's theme, with the 16 colors of the light or dark default; a Theme the
// app sets wins, and taking it back follows the view again.
func TestTerminalFollowsViewTheme(t *testing.T) {
	loadLib(t)
	custom := *ui.LightTheme()
	custom.Background, custom.Text = ui.Hex("#123456"), ui.Hex("#fedcba")
	onlyBg := *ui.LightTheme()
	onlyBg.Background = ui.Hex("#123456")
	onlyText := *ui.LightTheme()
	onlyText.Text = ui.Hex("#fedcba")
	customDark := *ui.DarkTheme()
	customDark.Background, customDark.Text = ui.Hex("#0a0b0c"), ui.Hex("#c0ffee")
	for _, c := range []struct {
		name   string
		view   *ui.Theme // nil: the window's own
		dark   bool
		opts   Options
		fg, bg ui.Color
		green  ui.Color // the palette's color 2, which light and dark differ in
	}{
		{"plain light", nil, false, Options{}, ui.LightTheme().Text, ui.LightTheme().Background, LightTheme().Palette[2]},
		{"plain dark", nil, true, Options{}, ui.DarkTheme().Text, ui.DarkTheme().Background, DarkTheme().Palette[2]},
		{"view's light theme", &custom, false, Options{}, custom.Text, custom.Background, LightTheme().Palette[2]},
		{"view's background only", &onlyBg, false, Options{}, ui.LightTheme().Text, onlyBg.Background, LightTheme().Palette[2]},
		{"view's text only", &onlyText, false, Options{}, onlyText.Text, ui.LightTheme().Background, LightTheme().Palette[2]},
		{"view's dark theme", &customDark, true, Options{}, customDark.Text, customDark.Background, DarkTheme().Palette[2]},
		{"app's Theme wins", &custom, false, Options{Theme: DarkTheme()}, DarkTheme().Foreground, DarkTheme().Background, DarkTheme().Palette[2]},
		{"app's DarkTheme in a light view", &custom, false, Options{DarkTheme: DarkTheme()}, custom.Text, custom.Background, LightTheme().Palette[2]},
		{"app's DarkTheme in a dark view", &customDark, true, Options{DarkTheme: LightTheme()}, LightTheme().Foreground, LightTheme().Background, LightTheme().Palette[2]},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newPipe()
			c.opts.Conn = p
			term, err := New(c.opts)
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			view := c.view
			tt := ui.NewTester(func(ctx *ui.Context) {
				if view != nil {
					ctx.SetTheme(view)
				}
				View(ctx, term).Fill()
			}, 400, 200)
			tt.SetDark(c.dark)
			tt.Frame()
			p.take(term)
			term.Feed([]byte("\x1b]10;?\x1b\\\x1b]11;?\x1b\\\x1b]4;2;?\x1b\\"))
			got := p.take(term)
			for _, want := range []string{oscColor("10", c.fg), oscColor("11", c.bg), oscColor("4;2", c.green)} {
				if !strings.Contains(got, want) {
					t.Errorf("got %q, want it to hold %q", got, want)
				}
			}
			// The theme is the same from frame to frame: it is applied once.
			term.mu.Lock()
			first := term.curTheme
			term.mu.Unlock()
			tt.Frame()
			term.mu.Lock()
			second := term.curTheme
			term.mu.Unlock()
			if first != second {
				t.Error("a new theme each frame")
			}
			// On the plain theme, every color is today's.
			if c.view == nil {
				want := LightTheme()
				if c.dark {
					want = DarkTheme()
				}
				if *first != *want {
					t.Errorf("the plain theme gives %v, want %v", first, want)
				}
			}
		})
	}

	// SetTheme(nil, nil) gives the view's colors back.
	p := newPipe()
	term, err := New(Options{Conn: p, Theme: DarkTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(ctx *ui.Context) { ctx.SetTheme(&custom); View(ctx, term).Fill() }, 400, 200)
	tt.Frame()
	term.SetTheme(nil, nil)
	tt.Frame()
	p.take(term)
	term.Feed([]byte("\x1b]11;?\x1b\\"))
	if got, want := p.take(term), oscColor("11", custom.Background); !strings.Contains(got, want) {
		t.Errorf("after SetTheme(nil, nil): got %q, want %q", got, want)
	}
}
