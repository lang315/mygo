package skin

import (
	"sync"
	"sync/atomic"
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

// paletteName names the active skin's palette for a test's message.
func paletteName() string {
	if s := Active(); s != nil && s.Palette() != nil {
		return s.Palette().Name()
	}
	return "<nil>"
}

// Use of Calm with another palette gives a window that palette's accent,
// and the hook its Info; a second Use switches it, in both appearances.
func TestUsePalette(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	Use(Calm.WithPalette(Indigo))
	tt.Frame()
	if got.Accent != ui.Hex("#4f46e5") {
		t.Errorf("with Indigo, Accent %v, want #4f46e5", got.Accent)
	}
	if h := skinhook.Get(false); h == nil || h.Info.Text != ui.Hex("#3f37c9") {
		t.Errorf("with Indigo, the hook's light tokens are %p, want Info.Text #3f37c9", h)
	}
	if Active().Palette() != Indigo {
		t.Errorf("Active's palette is %v, want Indigo", paletteName())
	}
	Use(Calm.WithPalette(Ink))
	tt.Frame()
	if got.Accent != ui.Hex("#2c4f8c") {
		t.Errorf("with Ink, Accent %v, want #2c4f8c", got.Accent)
	}
	tt.SetDark(true)
	if got.Accent != ui.Hex("#7b9fdc") {
		t.Errorf("with Ink, dark, Accent %v, want #7b9fdc", got.Accent)
	}
	if h := skinhook.Get(true); h == nil || h.Info.Text != ui.Hex("#a9c1ea") {
		t.Errorf("with Ink, the hook's dark tokens are %p, want Info.Text #a9c1ea", h)
	}
}

// Switching palettes from another goroutine while the window draws does not
// race (go test -race), and the last palette wins.
func TestUsePaletteFromAnotherGoroutine(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var got ui.Theme
	tt := themeTester(t, &got)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			Use(Calm.WithPalette(Indigo))
			Use(Calm.WithPalette(Teal))
		}
		Use(Calm.WithPalette(Ink))
	}()
	for range 100 {
		tt.Frame()
		_ = skinhook.Get(got.Dark)
	}
	wg.Wait()
	tt.Frame()
	if got.Accent != ui.Hex("#2c4f8c") || Active().Palette() != Ink {
		t.Errorf("after the goroutine, Accent %v and palette %v, want #2c4f8c and Ink", got.Accent, paletteName())
	}
}

// gridHeights returns the heights of the rows of the one grid tt shows, and
// of its header Name.
func gridHeights(t *testing.T, tt *widget.Tester) (rows []float32, header float32) {
	t.Helper()
	for _, n := range tt.Nodes() {
		switch {
		case n.Role == ui.RoleRow && (n.Label == "Ada" || n.Label == "Grace" || n.Label == "Edsger"): // not the header's row
			rows = append(rows, n.Bounds.H)
		case n.Role == ui.RoleColumnHeader && n.Label == "Name":
			header = n.Bounds.H
		}
	}
	if len(rows) != 3 || header == 0 {
		t.Fatalf("%d rows and a header %v high, want 3 rows and a header", len(rows), header)
	}
	return rows, header
}

// wantRows fails the test unless every row is h high.
func wantRows(t *testing.T, when string, rows []float32, h float32) {
	t.Helper()
	for i, r := range rows {
		if !same(r, h) {
			t.Errorf("%s, row %d is %v high, want %v", when, i, r, h)
		}
	}
}

// The grid's own row height wins over the skin's, which still sizes its
// header.
func TestSetRowHeightWinsOverSkin(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	grid := widget.DataGridView(func(r *sizeRow) string { return r.ID },
		widget.Column[sizeRow]{Title: "Name", Text: func(r *sizeRow) string { return r.Name }},
	).SetRows([]sizeRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}).SetRowHeight(24).SetSize(0, 180)
	tt := widget.NewTester(t, widget.VBox(8).Add(grid), 600, 400)
	tt.Frame()
	rows, plainHeader := gridHeights(t, tt)
	wantRows(t, "without a skin, with SetRowHeight(24)", rows, 24)
	Use(Calm)
	tt.Frame()
	rows, header := gridHeights(t, tt)
	wantRows(t, "with Calm and SetRowHeight(24)", rows, 24)
	if !same(header, 30) {
		t.Errorf("with Calm and SetRowHeight(24), the header is %v high, want 30", header)
	}
	Use(nil)
	tt.Frame()
	if _, h := gridHeights(t, tt); !same(h, plainHeader) {
		t.Errorf("after Use(nil), the header is %v high, want its plain %v", h, plainHeader)
	}
}

// The rows a grid keeps from frame to frame (RowVersion) take the skin's
// row height in the frame after Use, and their plain one after Use(nil).
func TestKeptRowsResizeOnUse(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var builds atomic.Int32
	grid := widget.DataGridView(func(r *sizeRow) string { return r.ID },
		widget.Column[sizeRow]{Title: "Name", Text: func(r *sizeRow) string { builds.Add(1); return r.Name }},
	).SetRows([]sizeRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}).SetSize(0, 180)
	note := widget.Label("one")
	tt := widget.NewTester(t, widget.VBox(8).Add(note, grid), 600, 400)
	tt.Frame()
	builds.Store(0)
	note.SetText("two") // the window builds again, the rows need not
	tt.Frame()
	if !tt.HasText("two") {
		t.Fatal("the label did not change: the window did not build again")
	}
	if n := builds.Load(); n != 0 {
		t.Fatalf("a frame that changed only a label built %d cells: the grid keeps no rows, and the test sees nothing", n)
	}
	plain, _ := gridHeights(t, tt)
	Use(Calm)
	tt.Frame()
	rows, _ := gridHeights(t, tt)
	wantRows(t, "after Use(Calm)", rows, 30)
	Use(nil)
	tt.Frame()
	rows, _ = gridHeights(t, tt)
	wantRows(t, "one frame after Use(nil)", rows, plain[0])
	if same(plain[0], 30) {
		t.Errorf("the plain row is 30, as Calm's: the test does not see the skin")
	}
	Use(Calm.WithPalette(Teal))
	tt.Frame()
	rows, _ = gridHeights(t, tt)
	wantRows(t, "after Use(Calm.WithPalette(Teal))", rows, 30)
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
