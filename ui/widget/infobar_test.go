package widget

import (
	"math"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// drained waits until the handlers posted before it have run: the event
// goroutine runs them one at a time, in order.
func drained(t *testing.T) {
	t.Helper()
	done := make(chan struct{}, 1)
	Post(func() { done <- struct{}{} })
	received(t, done)
}

// barOf is the box of the info bar, the node of role r.
func barOf(t *testing.T, tt *ui.Tester, r ui.Role) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r {
			return n.Bounds
		}
	}
	t.Fatalf("no node of role %v", r)
	return ui.Rect{}
}

const longMessage = "Tài khoản này cần được xác minh trước khi tiếp tục: mở email đã gửi và làm theo hướng dẫn trong đó để hoàn tất."

// Case 6: a Warning info bar is an alert, fills with Warning's Subtle in a
// 1 DIP Warning.Border, shows Warning's glyph in Warning.Base, its title in
// semibold and its message wrapped at the bar's width. The other statuses
// are a status.
func TestInfoBarWarning(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	bar := InfoBar(StatusWarning, "Cần xác minh", longMessage)
	tt := tester(t, VBox(0).Add(bar).SetSize(400, 0))
	r := barOf(t, tt, ui.RoleAlert)
	if !sameSize(r.W, 400) {
		t.Errorf("the bar is %v wide, want its box's 400", r.W)
	}
	if c := tt.Image().RGBAAt(int(r.X+4), int(r.Y+r.H-4)); !near(c, tk.Warning.Subtle) {
		t.Errorf("the fill is %v, want Warning.Subtle %v", c, tk.Warning.Subtle)
	}
	if c := tt.Image().RGBAAt(int(r.X), int(r.Y+r.H/2)); !near(c, tk.Warning.Border) {
		t.Errorf("the left edge is %v, want Warning.Border %v", c, tk.Warning.Border)
	}
	if c := tt.Image().RGBAAt(int(r.X+1), int(r.Y+r.H/2)); near(c, tk.Warning.Border) {
		t.Errorf("1 DIP in, the edge is still the border, %v: want 1 DIP", c)
	}
	title, msg := find(t, tt, "Cần xác minh"), find(t, tt, longMessage)
	icon := ui.Rect{X: r.X, Y: r.Y, W: title.X - r.X, H: title.Y + title.H - r.Y + 6}
	if !inkNear(tt, icon, tk.Warning.Base, 6) {
		t.Errorf("no icon in Warning.Base %v left of the title", tk.Warning.Base)
	}
	if msg.H < 2*title.H-1 || msg.X+msg.W > r.X+r.W {
		t.Errorf("the message %v does not wrap inside the bar %v (a line is %v)", msg, r, title.H)
	}
	for _, s := range []Status{StatusNone, StatusInfo, StatusRunning, StatusSuccess} {
		bar.SetStatus(s)
		tt.Frame()
		barOf(t, tt, ui.RoleStatus)
		for _, n := range tt.Nodes() {
			if n.Role == ui.RoleAlert {
				t.Errorf("%v: the bar is an alert", s)
			}
		}
	}
	bar.SetStatus(StatusDanger)
	tt.Frame()
	barOf(t, tt, ui.RoleAlert)
	if bar.Status() != StatusDanger || bar.Title() != "Cần xác minh" || bar.Message() != longMessage {
		t.Errorf("the getters give %v %q %q", bar.Status(), bar.Title(), bar.Message())
	}
	// The title in semibold, read from the inspector.
	wt := NewTester(t, InfoBar(StatusWarning, "Tiêu đề", "Nội dung"), 600, 400)
	if _, w := inspected(t, wt, "Tiêu đề"); w != 600 {
		t.Errorf("the title's weight is %d, want 600", w)
	}
}

// Case 2 for the bar: without a skin, the theme's color at 0.14, no border.
func TestInfoBarPlain(t *testing.T) {
	calmOff()
	th := ui.LightTheme()
	tt := tester(t, VBox(0).Add(InfoBar(StatusSuccess, "Đã lưu", "")).SetSize(400, 0))
	r := barOf(t, tt, ui.RoleStatus)
	if !sameSize(r.W, 400) {
		t.Errorf("a bar of a short title is %v wide, want its box's 400", r.W)
	}
	want := th.Success.Alpha(0.14).Over(th.Background)
	if c := tt.Image().RGBAAt(int(r.X+4), int(r.Y+r.H/2)); !near(c, want) {
		t.Errorf("the fill is %v, want Success at 0.14, %v", c, want)
	}
	if c := tt.Image().RGBAAt(int(r.X), int(r.Y+r.H/2)); !near(c, want) {
		t.Errorf("the left edge is %v, want the fill, no border", c)
	}
	if !inkNear(tt, ui.Rect{X: r.X, Y: r.Y, W: 40, H: r.H}, th.Success, 6) {
		t.Errorf("no check icon in Success %v", th.Success)
	}
	// With no border of its own, a border width the app sets takes the
	// theme's Border, as on any control without one.
	tt = tester(t, VBox(0).Add(InfoBar(StatusSuccess, "Đã lưu", "").Apply(NewStyle().BorderWidth(2))).SetSize(400, 0))
	r = barOf(t, tt, ui.RoleStatus)
	if c := tt.Image().RGBAAt(int(r.X), int(r.Y+r.H/2)); !near(c, th.Border) {
		t.Errorf("with the app's border width the left edge is %v, want the theme's Border %v", c, th.Border)
	}
}

// iconBox is the 20 DIP icon at the left of the bar r, inside its padding.
func iconBox(r ui.Rect) ui.Rect {
	pad := space(3)
	return ui.Rect{X: r.X + pad, Y: r.Y + pad, W: 20, H: 20}
}

// pixelsNear is how many pixels of r are within tol of c.
func pixelsNear(tt *ui.Tester, r ui.Rect, c ui.Color, tol int) int {
	n, img := 0, tt.Image() // a copy of the window's pixels: once, not per pixel
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if nearTol(img.RGBAAt(x, y), c, tol) {
				n++
			}
		}
	}
	return n
}

// The icon's glyph: under Calm, Info's On (white) inside Info's Base; with no
// skin, black on the amber warning sign, where white is under 3:1, and white
// on the red error circle.
func TestInfoBarIconGlyph(t *testing.T) {
	white, black := ui.RGB(255, 255, 255), ui.RGB(0, 0, 0)
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	if tk.Info.On != white {
		t.Fatalf("Calm light's Info.On is %v: the test wants white", tk.Info.On)
	}
	tt := tester(t, VBox(0).Add(InfoBar(StatusInfo, "Có bản cập nhật", "")).SetSize(400, 0))
	icon := iconBox(barOf(t, tt, ui.RoleStatus))
	if n := pixelsNear(tt, icon, tk.Info.Base, 6); n < 100 {
		t.Errorf("Calm's Info icon has %d pixels of Info.Base %v, want its circle", n, tk.Info.Base)
	}
	if n := pixelsNear(tt, icon, white, 6); n < 10 {
		t.Errorf("Calm's Info icon has %d white pixels, want its glyph in Info.On", n)
	}

	calmOff()
	// The dark theme's amber, #f59e0b, is 2.1:1 under white; its red, #ef4444,
	// 3.8:1.
	if w, d := ui.DarkTheme().Warning, ui.DarkTheme().Danger; w != ui.Hex("#f59e0b") || d != ui.Hex("#ef4444") {
		t.Fatalf("the dark theme's Warning and Danger are %v and %v: the test wants #f59e0b and #ef4444", w, d)
	}
	for _, c := range []struct {
		s                  Status
		glyph, other, base ui.Color
	}{
		{StatusWarning, black, white, ui.Hex("#f59e0b")},
		{StatusDanger, white, black, ui.Hex("#ef4444")},
	} {
		tt := tester(t, VBox(0).Add(InfoBar(c.s, "Tiêu đề", "")).SetSize(400, 0))
		tt.SetDark(true)
		tt.Frame()
		icon := iconBox(barOf(t, tt, ui.RoleAlert))
		if n := pixelsNear(tt, icon, c.base, 6); n < 60 {
			t.Errorf("%v: %d pixels of the base %v, want its shape", c.s, n, c.base)
		}
		// The thin mark is anti-aliased: its core within 30 of the glyph color.
		if n := pixelsNear(tt, icon, c.glyph, 30); n < 8 {
			t.Errorf("%v: %d pixels of the glyph %v, want its mark", c.s, n, c.glyph)
		}
		if n := pixelsNear(tt, icon, c.other, 30); n != 0 {
			t.Errorf("%v: %d pixels of %v, a glyph in the wrong color", c.s, n, c.other)
		}
	}
}

// A bar with no title starts its message at the top of the text column, and
// one with no message is as high as its title and padding: neither keeps
// room for the part it has not.
func TestInfoBarNoTitleOrMessage(t *testing.T) {
	calmOff()
	tt := tester(t, VBox(8).Add(InfoBar(StatusInfo, "", "Chỉ có nội dung"), InfoBar(StatusSuccess, "Chỉ có tiêu đề", "")).SetSize(400, 0))
	var bars []ui.Rect
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleStatus {
			bars = append(bars, n.Bounds)
		}
	}
	if len(bars) != 2 {
		t.Fatalf("%d bars, want 2", len(bars))
	}
	pad := space(3)
	if m := find(t, tt, "Chỉ có nội dung"); math.Abs(float64(m.Y-(bars[0].Y+pad))) > 0.5 {
		t.Errorf("with no title the message starts at %v, want the padding's %v", m.Y, bars[0].Y+pad)
	}
	icon := float32(20) // the icon is taller than a line: it sets the bar's height
	ti := find(t, tt, "Chỉ có tiêu đề")
	if want := 2*pad + max(icon, ti.H); math.Abs(float64(bars[1].H-want)) > 0.5 {
		t.Errorf("with no message the bar is %v high, want %v", bars[1].H, want)
	}
}

// Case 7: SetAction puts a button at the end that runs f once; "" removes
// it.
func TestInfoBarAction(t *testing.T) {
	var n atomic.Int32
	bar := InfoBar(StatusWarning, "Cần xác minh", "Mở email.").SetAction("Xác minh", func() { n.Add(1) })
	tt := NewTester(t, VBox(0).Add(bar).SetSize(500, 0), 600, 400)
	b := tt.Find(ByLabel("Xác minh"))
	if title := tt.Find(ByLabel("Mở email.")).Bounds(); b.Bounds().X < title.X+title.W {
		t.Errorf("the action at %v is not after the text %v", b.Bounds(), title)
	}
	b.Click()
	drained(t)
	if got := n.Load(); got != 1 {
		t.Errorf("the action ran %d times, want 1", got)
	}
	bar.SetAction("", nil)
	tt.Frame()
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 0 {
		t.Errorf("SetAction(\"\", nil) left %d buttons", n)
	}
}

// Case 8 and review focus 5: closing hides the bar, then runs OnClose once,
// after it is hidden; SetVisible(true) shows it again; SetVisible(false)
// from code never runs OnClose.
func TestInfoBarCloseOnlyByUser(t *testing.T) {
	closes := make(chan bool, 4)
	var bar *InfoBarWidget
	bar = InfoBar(StatusInfo, "Đã cập nhật", "").SetClosable(true).OnClose(func() { closes <- bar.Visible() })
	tt := NewTester(t, VBox(0).Add(bar).SetSize(500, 0), 600, 400)
	tt.Find(ByLabel("Close")).Click()
	if visible := received(t, closes); visible {
		t.Error("OnClose ran while the bar still showed")
	}
	tt.Frame()
	if bar.Visible() || tt.Has(ByLabel("Đã cập nhật")) {
		t.Error("the bar shows after close")
	}
	bar.SetVisible(true)
	tt.Frame()
	if !tt.Has(ByLabel("Đã cập nhật")) {
		t.Fatal("SetVisible(true) did not show the bar again")
	}
	bar.SetVisible(false)
	tt.Frame()
	bar.SetVisible(true)
	tt.Frame()
	tt.Find(ByLabel("Close")).Click()
	received(t, closes)
	drained(t)
	if len(closes) != 0 {
		t.Errorf("OnClose ran %d more times: once per close, never for SetVisible(false)", len(closes))
	}
	bar.SetClosable(false)
	bar.SetVisible(true)
	tt.Frame()
	if tt.Has(ByLabel("Close")) {
		t.Error("SetClosable(false) left the close button")
	}
}

// Case 10, its info bar part: a look of border-2 border-status gives the
// bar a 2 DIP border in its family's Border.
func TestInfoBarLook(t *testing.T) {
	useLooks(t, map[string]string{"InfoBar": "border-2 border-status"})
	tk := skinhook.Get(false)
	// With an action: the bar's look applies after its button is built, and
	// still takes the bar's status.
	tt := tester(t, VBox(0).Add(InfoBar(StatusDanger, "Sai mật khẩu", "").SetAction("Thử lại", func() {})).SetSize(400, 0))
	r := barOf(t, tt, ui.RoleAlert)
	for _, dx := range []float32{0, 1} {
		if c := tt.Image().RGBAAt(int(r.X+dx), int(r.Y+r.H/2)); !near(c, tk.Danger.Border) {
			t.Errorf("%v in from the left the bar is %v, want Danger.Border %v", dx, c, tk.Danger.Border)
		}
	}
	if c := tt.Image().RGBAAt(int(r.X+2), int(r.Y+r.H/2)); near(c, tk.Danger.Border) {
		t.Errorf("2 in, the bar is still its border, %v: want 2 DIPs", c)
	}
}

// Review focus 1: a long Vietnamese title at 280 DIPs wraps, and the action
// and close stay whole inside the bar.
func TestInfoBarNarrow(t *testing.T) {
	calmOff()
	title := "Tài khoản của bạn cần được xác minh lại trước khi đồng bộ"
	bar := InfoBar(StatusWarning, title, "Mở email.").SetAction("Xác minh", func() {}).SetClosable(true)
	tt := tester(t, VBox(0).Add(bar).SetSize(280, 0))
	r := barOf(t, tt, ui.RoleAlert)
	ti := find(t, tt, title)
	one := find(t, tt, "Mở email.").H
	if ti.H < 2*one-1 {
		t.Errorf("the title %v is on one line (a line is %v): want it wrapped", ti, one)
	}
	for _, name := range []string{"Xác minh", "Close"} {
		b := nodeOf(t, tt, ui.RoleButton, name).Bounds
		if b.W < 20 || b.X < r.X || b.X+b.W > r.X+r.W+0.5 {
			t.Errorf("%s %v is clipped in the bar %v", name, b, r)
		}
		if strings.Contains(name, "Xác") && b.W < find(t, tt, "Xác minh").W {
			t.Errorf("the action %v is narrower than its text", b)
		}
	}
}
