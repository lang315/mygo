package tw

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/twclass"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/twsource"
)

func TestSameAsTyped(t *testing.T) {
	n := widget.NewStyle
	blue := widget.Fixed(ui.RGB(0x25, 0x63, 0xeb))
	for _, c := range []struct {
		in   string
		want widget.Style
	}{
		{"p-2", n().Padding(2)},
		{"px-3 py-2", n().PaddingX(3).PaddingY(2)},
		{"mt-3 -ml-2", n().MarginTop(3).MarginLeft(-2)},
		{"rounded-l-none", n().RadiusTopLeft(0).RadiusBottomLeft(0)},
		{"rounded-lg", n().Radius(2)},
		{"border-l-0", n().BorderLeft(0)},
		{"border-2 border-dashed", n().BorderWidth(2).BorderStyle(ui.BorderDashed)},
		{"w-4 max-h-96", n().Width(16).MaxHeight(384)},
		{"bg-blue-600", n().Background(blue)},
		{"text-muted", n().TextColor(widget.RoleTextMuted)},
		{"border-default", n().BorderColor(widget.RoleBorder)},
		{"text-xl font-bold italic font-mono", n().TextSize(widget.XL).FontWeight(700).Italic().Mono()},
		{"text-5xl", n().FontSize(48)},
		{"italic not-italic", n()},
		{"opacity-50 cursor-pointer", n().Opacity(0.5).Cursor(ui.CursorPointer)},
		{"shadow-sm", n().NoShadow().Shadow(0, 1, 2, 0, widget.Fixed(ui.RGBA(0, 0, 0, 0.05)))},
		{"halign-start hexpand", n().Align(widget.Start).HExpand()},
		{"transition", n().Transition(150 * time.Millisecond)},
		{"hover:bg-blue-600", n().Hover(n().Background(blue))},
		{"active:p-1", n().Pressed(n().Padding(1))},
		{"dark:bg-surface", n().Dark(n().Background(widget.RoleSurface))},
		{"hover:p-1 dark:hover:p-2", n().Hover(n().Padding(1).Dark(n().Padding(2)))},
		{"p-2 p-4", n().Padding(4)},
		{"", n()},
	} {
		got, err := Parse(c.in)
		if err != nil || !reflect.DeepEqual(bare(got), c.want) {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
}

func TestEveryRoleMaps(t *testing.T) {
	for _, r := range twclass.Roles {
		if _, ok := roles[r]; !ok {
			t.Errorf("role %q has no widget color", r)
		}
	}
	if len(roles) != len(twclass.Roles) {
		t.Errorf("%d widget colors for %d twclass roles", len(roles), len(twclass.Roles))
	}
}

// Each role name draws the field of the theme it is named for: the corner
// of the card's padding shows its background, which is the color of the role.
func TestRolesDrawTheThemeField(t *testing.T) {
	th := ui.LightTheme()
	or := func(c, fallback ui.Color) ui.Color {
		if c == (ui.Color{}) {
			return fallback
		}
		return c
	}
	want := map[string]ui.Color{
		"background": th.Background, "surface": th.Surface, "surface-hover": th.SurfaceHover,
		"surface-pressed": th.SurfacePressed, "border": th.Border, "text": th.Text, "text-muted": th.TextMuted,
		"accent": th.Accent, "accent-hover": th.AccentHover, "accent-pressed": th.AccentPressed,
		"accent-text": th.AccentText, "danger": th.Danger, "warning": th.Warning, "success": th.Success,
		"selection": th.Selection, "focus": th.Focus, "inverse": or(th.Inverse, th.Text),
		"inverse-text": or(th.InverseText, th.Background),
	}
	for _, r := range twclass.Roles {
		w, ok := want[r]
		if !ok || w.A != 255 {
			continue // translucent roles blend with the page: not a pixel to compare
		}
		tt := ui.NewTester(widget.View(widget.VBox(0).Add(widget.Label("role").Align(widget.Start).
			Apply(widget.NewStyle().Padding(4).Background(roles[r])))), 200, 100)
		tt.Frame()
		b, ok := tt.Find("role")
		if !ok {
			t.Fatalf("%s: label not found", r)
		}
		got := tt.Image().RGBAAt(int(b.X)+1, int(b.Y)+1) // in the padding, a corner of the card
		if got.R != w.R || got.G != w.G || got.B != w.B {
			t.Errorf("role %q draws %v, the theme field is %v", r, got, w)
		}
	}
}

func TestParseError(t *testing.T) {
	s, err := Parse("p-2 text-green hover:hexpand")
	var e *Error
	if !errors.As(err, &e) || len(e.Bad) != 2 {
		t.Fatalf("err %v", err)
	}
	if !reflect.DeepEqual(bare(s), widget.NewStyle().Padding(2)) {
		t.Fatalf("the good class must still apply: %+v", s)
	}
	if want := `tw: unknown class "text-green"; layout in a variant "hover:hexpand"`; err.Error() != want {
		t.Fatalf("%q, want %q", err.Error(), want)
	}
}

// A Style cannot clear italic, so not-italic in a variant would do nothing:
// it is an error, not a silent no-op.
func TestNotItalicInAVariantIsAnError(t *testing.T) {
	s, err := Parse("italic dark:not-italic")
	var e *Error
	if !errors.As(err, &e) || len(e.Bad) != 1 || e.Bad[0] != (Bad{"dark:not-italic", twclass.NotItalicInVariant}) {
		t.Fatalf("err %v", err)
	}
	if !reflect.DeepEqual(bare(s), widget.NewStyle().Italic()) {
		t.Fatalf("the italic must still apply: %+v", s)
	}
}

func TestSIgnoresBadClasses(t *testing.T) {
	if got := S("p-2 bogus"); !reflect.DeepEqual(bare(got), widget.NewStyle().Padding(2)) {
		t.Fatalf("%+v", got)
	}
}

func TestCachedSDoesNotAllocate(t *testing.T) {
	const c = "p-2 mt-3 rounded-lg bg-blue-600 hover:bg-blue-700 transition"
	S(c)
	if n := testing.AllocsPerRun(100, func() { S(c) }); n != 0 {
		t.Fatalf("a cached S allocates %v times", n)
	}
}

// isolated gives the test a cache and a logged set of its own: the global
// ones, left full, would keep every later test's string out of them.
func isolated(t *testing.T) {
	cacheMu.Lock()
	savedCache, savedLogged := cache, logged
	cache, logged = map[string]widget.Style{}, map[Bad]bool{}
	cacheMu.Unlock()
	t.Cleanup(func() {
		cacheMu.Lock()
		cache, logged = savedCache, savedLogged
		cacheMu.Unlock()
	})
}

// blockingLog is a log output whose Write waits for release, after saying it
// is in it.
type blockingLog struct{ in, release chan struct{} }

func (w blockingLog) Write(p []byte) (int, error) {
	w.in <- struct{}{}
	<-w.release
	return len(p), nil
}

// A line being logged must not hold up the S calls that hit the cache.
func TestLoggingDoesNotBlockCachedCalls(t *testing.T) {
	isolated(t)
	const c = "p-2 mt-3"
	S(c)
	w := blockingLog{in: make(chan struct{}), release: make(chan struct{})}
	log.SetOutput(w)
	var once sync.Once
	release := func() { once.Do(func() { close(w.release) }) }
	// Registered after isolated's, so it runs first: a failing S holds the
	// lock while it waits here, and isolated's cleanup needs the lock.
	t.Cleanup(func() { release(); log.SetOutput(os.Stderr) })
	logged := make(chan struct{})
	go func() { S("p-2 not-a-class-for-the-log-test"); close(logged) }()
	wait.For(t, "S to be writing its line", w.in)
	hit := make(chan struct{})
	go func() { S(c); close(hit) }()
	wait.For(t, "a cached S while another logs", hit)
	release()
	wait.For(t, "the logging S to return", logged)
}

// With the cache full, a string that is not cached and has nothing to log
// stores and logs nothing, so it must not take the write lock.
func TestFullCacheMissTakesNoWriteLock(t *testing.T) {
	isolated(t)
	for i := range maxEntries {
		S("p-2" + strings.Repeat(" ", i))
	}
	cacheMu.RLock() // a write lock wanted now would wait for this
	defer cacheMu.RUnlock()
	done := make(chan struct{})
	go func() { S("p-3 mt-1"); close(done) }()
	wait.For(t, "S of an uncached string with a full cache", done)
}

func TestCacheIsBounded(t *testing.T) {
	isolated(t)
	want := widget.NewStyle().Padding(2)
	for i := range 5000 { // 5000 distinct strings, all p-2
		if got := S("p-2" + strings.Repeat(" ", i)); !reflect.DeepEqual(bare(got), want) {
			t.Fatalf("string %d: S = %+v after the cache filled", i, got)
		}
	}
	cacheMu.RLock()
	n := len(cache)
	cacheMu.RUnlock()
	if n > maxEntries {
		t.Fatalf("cache holds %d, more than %d", n, maxEntries)
	}
}

func TestSFromManyGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			for j := range 200 {
				S(fmt.Sprintf("p-%d hover:bg-red-%d00", (i+j)%8, 1+j%9))
			}
		})
	}
	wg.Wait()
}

// The same card styled with tw and written typed draws the same pixels.
func TestDrawsAsTyped(t *testing.T) {
	typed := widget.NewStyle().Padding(3).Radius(2).Background(widget.RoleSurface).
		BorderWidth(1).BorderColor(widget.RoleBorder).NoShadow().
		Shadow(0, 4, 6, -1, widget.Fixed(ui.RGBA(0, 0, 0, 0.1))).Shadow(0, 2, 4, -2, widget.Fixed(ui.RGBA(0, 0, 0, 0.1))).
		TextColor(widget.RoleTextMuted).FontWeight(600)
	img := func(s widget.Style) []byte {
		tt := ui.NewTester(widget.View(widget.VBox(8).Padding(8).Add(widget.Label("Card").Apply(s))), 200, 100)
		tt.Frame()
		return tt.Image().Pix
	}
	if !bytes.Equal(img(S("p-3 rounded-lg bg-surface border border-default shadow-md text-muted font-semibold")), img(typed)) {
		t.Fatal("tw and typed draw different pixels")
	}
}

func TestMaxWNoneLiftsTheCapOfAFixedWidth(t *testing.T) {
	for class, want := range map[string]float32{"w-96": 300, "w-96 max-w-none": 384} {
		b := widget.Button("W")
		b.Apply(S(class))
		tt := widget.NewTester(t, widget.VBox(0).Add(b), 300, 100)
		if w := tt.Find(widget.ByLabel("W")).Bounds().W; w != want {
			t.Errorf("%q: width %v, want %v", class, w, want)
		}
	}
	// A MaxWidth of the app's, before or after SetSize, overrides the cap.
	b := widget.Button("W")
	b.Apply(S("max-w-none"))
	b.SetSize(800, 0)
	if w := widget.NewTester(t, widget.VBox(0).Add(b), 300, 100).Find(widget.ByLabel("W")).Bounds().W; w != 800 {
		t.Errorf("MaxWidth set before SetSize: width %v, want 800", w)
	}
}

// bare takes off the classes S notes, to compare a Style with one of the typed
// methods.
func bare(s widget.Style) widget.Style {
	twsource.Note(&s, "")
	return s
}
