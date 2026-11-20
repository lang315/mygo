package widget

import (
	"image/color"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// inkNear reports whether a pixel of r is within tol of want on each
// channel: the core of a glyph drawn in that color.
func inkNear(tt *ui.Tester, r ui.Rect, want ui.Color, tol int) bool {
	img := tt.Image()
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if nearTol(img.RGBAAt(x, y), want, tol) {
				return true
			}
		}
	}
	return false
}

// darkest is the darkest pixel of r: the core of dark text on a light fill.
func darkest(tt *ui.Tester, r ui.Rect) color.RGBA {
	img := tt.Image()
	best, lum := color.RGBA{}, 1<<30
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := img.RGBAAt(x, y)
			if l := 2126*int(c.R) + 7152*int(c.G) + 722*int(c.B); l < lum {
				best, lum = c, l
			}
		}
	}
	return best
}

// pillEdges are the first and last x, on the row y, from x0 to x1, whose
// pixel differs from the page's: a pill's painted width.
func pillEdges(tt *ui.Tester, y, x0, x1 float32) (first, last int) {
	img, bg := tt.Image(), page(tt)
	first, last = -1, -1
	for x := int(x0); x < int(x1); x++ {
		if c := img.RGBAAt(x, int(y)); c != bg {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	return first, last
}

// badgeBox is the box of the badge read as name: its pill, and the detail
// after it when it shows.
func badgeBox(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	return nodeOf(t, tt, ui.RoleText, name).Bounds
}

// fillOf is the color in the padding of the badge read as name, 3 DIPs in
// from its left edge, before its dot and its text.
func fillOf(t *testing.T, tt *ui.Tester, name string) color.RGBA {
	t.Helper()
	r := badgeBox(t, tt, name)
	return tt.Image().RGBAAt(int(r.X+3), int(r.Y+r.H/2))
}

func badgeTester(t *testing.T, b *BadgeWidget) *ui.Tester {
	t.Helper()
	return tester(t, HBox(0).Add(b))
}

// Case 1: a Success badge under Calm light fills with Success's Subtle and
// writes in its Text, 20 high at text scale 1 and 30 at 1.5.
func TestBadgeCalmSuccess(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	tt := badgeTester(t, Badge("Live").SetStatus(StatusSuccess))
	if c := fillOf(t, tt, "Live"); !near(c, tk.Success.Subtle) {
		t.Errorf("the fill is %v, want Success.Subtle %v", c, tk.Success.Subtle)
	}
	if c := darkest(tt, badgeBox(t, tt, "Live")); !nearTol(c, tk.Success.Text, 12) {
		t.Errorf("the text's core is %v, want Success.Text %v", c, tk.Success.Text)
	}
	if h := badgeBox(t, tt, "Live").H; !sameSize(h, 20) {
		t.Errorf("the badge is %v high, want 20", h)
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	if h := badgeBox(t, tt, "Live").H; !sameSize(h, 30) {
		t.Errorf("at text scale 1.5 the badge is %v high, want 30", h)
	}
}

// Case 2: without a skin, theme.Success at alpha 0.14 over the page, and
// the theme's Text.
func TestBadgePlainSuccess(t *testing.T) {
	calmOff()
	th := ui.LightTheme()
	tt := badgeTester(t, Badge("Live").SetStatus(StatusSuccess))
	if c, want := fillOf(t, tt, "Live"), th.Success.Alpha(0.14).Over(th.Background); !near(c, want) {
		t.Errorf("the fill is %v, want Success at 0.14 over the page, %v", c, want)
	}
	if c := darkest(tt, badgeBox(t, tt, "Live")); !nearTol(c, th.Text, 12) {
		t.Errorf("the text's core is %v, want Text %v", c, th.Text)
	}
	if h := badgeBox(t, tt, "Live").H; !sameSize(h, 20) {
		t.Errorf("the plain badge is %v high, want 20", h)
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	if h := badgeBox(t, tt, "Live").H; !sameSize(h, 30) {
		t.Errorf("at text scale 1.5 the plain badge is %v high, want 30", h)
	}
}

// Case 3: a running badge with a detail: a 6 DIP dot in Info.Base before
// the text, the detail after the pill in TextMuted, read after the text as
// "Live, Bước 2/5". With Success, the detail is neither drawn nor read, and
// there is no dot.
func TestBadgeRunningDetail(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	b := Badge("Live").SetStatus(StatusRunning).SetDetail("Bước 2/5")
	tt := badgeTester(t, b)
	text, detail := find(t, tt, "Live"), find(t, tt, "Bước 2/5")
	box := badgeBox(t, tt, "Live, Bước 2/5")
	dot := ui.Rect{X: box.X, Y: text.Y, W: text.X - box.X, H: text.H}
	// 6 DIPs wide: on the dot's middle row, the pixels before the text that
	// are not the fill, its edges' blend included; its core in Info.Base.
	run := 0
	for x := int(dot.X); x < int(dot.X+dot.W)-1; x++ {
		if !near(tt.Image().RGBAAt(x, int(box.Y+box.H/2)), tk.Info.Subtle) {
			run++
		}
	}
	if run < 6 || run > 7 || !inkNear(tt, dot, tk.Info.Base, 8) {
		t.Errorf("the dot is %d pixels across, want 6 to 7 with a core of Info.Base %v", run, tk.Info.Base)
	}
	// The pill ends where the page shows again, after the text.
	pillRight := int(text.X + text.W)
	for bg := page(tt); tt.Image().RGBAAt(pillRight, int(box.Y+box.H/2)) != bg && pillRight < int(box.X+box.W); pillRight++ {
	}
	if detail.X <= float32(pillRight) || pillRight >= int(box.X+box.W) {
		t.Errorf("the detail starts at %v, inside the pill ending at %d", detail.X, pillRight)
	}
	if c := darkest(tt, detail); !nearTol(c, tk.TextMuted, 12) {
		t.Errorf("the detail's core is %v, want TextMuted %v", c, tk.TextMuted)
	}
	if b.Detail() != "Bước 2/5" || b.Text() != "Live" || b.Status() != StatusRunning {
		t.Errorf("the getters give %q %q %v", b.Text(), b.Detail(), b.Status())
	}
	b.SetStatus(StatusSuccess)
	tt.Frame()
	for _, s := range tt.Texts() {
		if s == "Bước 2/5" {
			t.Error("the detail is drawn for Success")
		}
	}
	for _, n := range tt.Nodes() {
		if n.Label == "Live, Bước 2/5" {
			t.Error("the detail is read for Success")
		}
	}
	// No dot: the pill is as wide as a StatusNone pill of the same text.
	plain := badgeBox(t, badgeTester(t, Badge("Live")), "Live")
	if w := badgeBox(t, tt, "Live").W; !sameSize(w, plain.W) {
		t.Errorf("the Success pill is %v wide, a StatusNone one %v: a dot is drawn for Success", w, plain.W)
	}
}

// A running badge with no detail is read as its text alone, draws no detail
// and ends where its pill ends.
func TestBadgeRunningNoDetail(t *testing.T) {
	calmOff()
	tt := badgeTester(t, Badge("Đang chạy").SetStatus(StatusRunning))
	box := badgeBox(t, tt, "Đang chạy")
	for _, n := range tt.Nodes() {
		if n.Label == "Đang chạy, " {
			t.Error("the badge is read as \"Đang chạy, \"")
		}
	}
	if texts := tt.Texts(); len(texts) != 2 || texts[0] != "Đang chạy" || texts[1] != "Đang chạy" {
		t.Errorf("the frame shows %q, want the badge's name and its text alone", texts)
	}
	// The pill's right end is the badge's: no gap, no detail after it.
	y := int(box.Y + box.H/2)
	if c := tt.Image().RGBAAt(int(box.X+box.W)-3, y); c == page(tt) {
		t.Errorf("3 DIPs inside the badge's right edge is the page %v: something after the pill", c)
	}
}

// Case 4: StatusNone is TextMuted on SurfaceHover with a skin, and as
// ui.Badge without one, Text at 0.1.
func TestBadgeNone(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	tt := badgeTester(t, Badge("Draft"))
	if c := fillOf(t, tt, "Draft"); !near(c, tk.SurfaceHover) {
		t.Errorf("the fill is %v, want SurfaceHover %v", c, tk.SurfaceHover)
	}
	if c := darkest(tt, badgeBox(t, tt, "Draft")); !nearTol(c, tk.TextMuted, 12) {
		t.Errorf("the text's core is %v, want TextMuted %v", c, tk.TextMuted)
	}
	calmOff()
	tt.Frame()
	th := ui.LightTheme()
	if c, want := fillOf(t, tt, "Draft"), th.Text.Alpha(0.1).Over(th.Background); !near(c, want) {
		t.Errorf("the plain fill is %v, want Text at 0.1, %v", c, want)
	}
}

// Case 5: a badge never stretches: its pill keeps its content's width in an
// HBox, HExpand or not.
func TestBadgeNoStretch(t *testing.T) {
	calmOff()
	width := func(b *BadgeWidget) int {
		tt := tester(t, HBox(0).Add(b).SetSize(600, 0))
		r := find(t, tt, "Live")
		first, last := pillEdges(tt, r.Y+r.H/2, 0, 600)
		return last - first
	}
	plain, expanded := width(Badge("Live").SetStatus(StatusSuccess)), width(Badge("Live").SetStatus(StatusSuccess).HExpand())
	if plain != expanded || plain > 100 {
		t.Errorf("the pill is %d wide, %d with HExpand: want its content's both times", plain, expanded)
	}
}

// Case 10, its badge and Button parts: a look of rounded-sm border
// border-status gives a Warning badge Warning's Border at radius 2; the same
// border-status in a Button look is StatusNone's border, the skin's Border.
func TestBadgeLook(t *testing.T) {
	useLooks(t, map[string]string{"Badge": "rounded-sm border border-status", "Button": "border-2 border-status"})
	tk := skinhook.Get(false)
	tt := tester(t, VBox(8).Add(HBox(0).Add(Badge("Cần xác minh").SetStatus(StatusWarning)), HBox(0).Add(Button("Go"))))
	box := badgeBox(t, tt, "Cần xác minh")
	if c := tt.Image().RGBAAt(int(box.X), int(box.Y+box.H/2)); !near(c, tk.Warning.Border) {
		t.Errorf("the badge's left edge is %v, want Warning.Border %v", c, tk.Warning.Border)
	}
	// 1 in from the corner: on the border at radius 2, the page at 10.
	if c := tt.Image().RGBAAt(int(box.X+1), int(box.Y+1)); near(c, ui.LightTheme().Background) {
		t.Errorf("the corner is the page, %v: the radius is not 2", c)
	}
	g := nodeOf(t, tt, ui.RoleButton, "Go").Bounds
	if c := tt.Image().RGBAAt(int(g.X+g.W/2), int(g.Y+0.5)); !near(c, tk.Border) || tk.Border == tk.Warning.Border {
		t.Errorf("the button's top line is %v, want StatusNone's border, Border %v", c, tk.Border)
	}
}

// Case 15: switching from Calm to Classic repaints a kept badge in Classic's
// look in the next frame.
func TestBadgeSkinSwitch(t *testing.T) {
	useSkinNamed(t, "Calm", false)
	tt := badgeTester(t, Badge("Live").SetStatus(StatusSuccess))
	edge := func() color.RGBA {
		box := badgeBox(t, tt, "Live")
		return tt.Image().RGBAAt(int(box.X), int(box.Y+box.H/2))
	}
	calm := skinhook.Get(false)
	if c := edge(); !near(c, calm.Success.Subtle) {
		t.Errorf("under Calm the pill's edge is %v, want its fill %v", c, calm.Success.Subtle)
	}
	useSkinNamed(t, "Classic", false)
	tt.Frame()
	classic := skinhook.Get(false)
	if c := edge(); !near(c, classic.Success.Border) {
		t.Errorf("under Classic the pill's edge is %v, want Classic's Success.Border %v", c, classic.Success.Border)
	}
}

// Review focus 2: in a Compact row of 26 the 20 DIP pill fits, unclipped.
func TestBadgeCompactFits(t *testing.T) {
	useSkinNamed(t, "Calm", true)
	row := HBox(0).Add(Badge("Đang chạy").SetStatus(StatusRunning).SetDetail("Bước 2/5")).SetSize(300, 26)
	row.SetAccessibleName("row")
	tt := tester(t, VBox(0).Add(row))
	r := nodeOf(t, tt, ui.RoleGroup, "row").Bounds
	p := badgeBox(t, tt, "Đang chạy, Bước 2/5")
	if !sameSize(r.H, 26) || !sameSize(p.H, 20) || p.Y < r.Y || p.Y+p.H > r.Y+r.H {
		t.Errorf("the pill %v in the row %v: want 20 high inside 26", p, r)
	}
}

// Review focus 3: Calm dark's translucent Subtle is blended over the page.
func TestBadgeDarkTranslucent(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(true)
	if tk.Success.Subtle.A == 255 {
		t.Fatal("Calm dark's Success.Subtle is opaque: the test needs a translucent one")
	}
	tt := badgeTester(t, Badge("Live").SetStatus(StatusSuccess))
	tt.SetDark(true)
	tt.Frame()
	bg := page(tt)
	want := tk.Success.Subtle.Over(ui.RGB(bg.R, bg.G, bg.B))
	if c := fillOf(t, tt, "Live"); !nearTol(c, want, 3) {
		t.Errorf("the dark fill is %v, want Subtle over the page %v", c, want)
	}
}

// Review focus 4: SetStatus from a goroutine while frames draw repaints in
// the next frame, with no race (go test -race).
func TestBadgeStatusFromGoroutine(t *testing.T) {
	calmOff()
	th := ui.LightTheme()
	b := Badge("Live")
	tt := badgeTester(t, b)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 50 {
			b.SetStatus(Status(i % 6))
		}
		b.SetStatus(StatusDanger)
	}()
	for range 20 {
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	if c, want := fillOf(t, tt, "Live"), th.Danger.Alpha(0.14).Over(th.Background); !near(c, want) {
		t.Errorf("after the last SetStatus the fill is %v, want Danger's %v", c, want)
	}
}

// A badge allocates nothing in a frame, running with a detail or not, with
// and without a skin: the slope of 200 badges over 100.
func TestBadgeFrameAllocs(t *testing.T) {
	for _, skin := range []bool{false, true} {
		if skin {
			useCalm(t, true)
			t.Cleanup(calmOff)
		}
		mk := func(i int) Control { // half running with their step, half of each status
			s := StatusRunning
			if i%2 == 1 {
				s = Status(i / 2 % 6)
			}
			return HBox(0).Add(Badge("Live").SetStatus(s).SetDetail("Bước 2/5"))
		}
		if slope := (frameAllocs(mk, 200) - frameAllocs(mk, 100)) / 100; slope > 0.2 {
			t.Errorf("skin %v: %.2f allocations per badge in a frame, want 0", skin, slope)
		}
	}
}
