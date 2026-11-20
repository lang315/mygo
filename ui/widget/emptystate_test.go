package widget

import (
	"math"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

var emptyIcon = lucide(`<circle cx="12" cy="12" r="9"/>`)

const emptyMessage = "Thêm một tài khoản để bắt đầu đồng bộ thư, lịch và danh bạ của bạn trên mọi thiết bị đang dùng."

// Case 9: centered both ways in a 600×400 box, the message at most 360
// wide, the action a Suggested button that runs f; with SetIcon(nil) no room
// is kept for an icon.
func TestEmptyState(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	tk := skinhook.Get(false)
	line := lineOf(t, 1)
	var n atomic.Int32
	e := EmptyState("Chưa có tài khoản", emptyMessage).SetIcon(emptyIcon).SetAction("Thêm", func() { n.Add(1) })
	box := VBox(0).Add(e).SetSize(600, 400)
	tt := NewTester(t, box, 800, 600)
	ut := tt.Tester
	centered := func(withIcon bool) {
		t.Helper()
		title, msg := find(t, ut, "Chưa có tài khoản"), find(t, ut, emptyMessage)
		btn := nodeOf(t, ut, ui.RoleButton, "Thêm").Bounds
		top := title.Y
		if withIcon {
			top = title.Y - 48 - space(2)
		}
		for name, mid := range map[string]float32{"title": title.X + title.W/2, "message": msg.X + msg.W/2, "action": btn.X + btn.W/2} {
			if math.Abs(float64(mid-300)) > 1.5 {
				t.Errorf("icon %v: the %s's middle is at %v, want 300", withIcon, name, mid)
			}
		}
		if mid := (top + btn.Y + btn.H) / 2; math.Abs(float64(mid-200)) > 1.5 {
			t.Errorf("icon %v: the content runs %v to %v, its middle %v, want 200", withIcon, top, btn.Y+btn.H, mid)
		}
		if msg.W > 360+0.5 || msg.H < 1.8*line {
			t.Errorf("the message %v is not wrapped at 360", msg)
		}
		if c := ut.Image().RGBAAt(int(btn.X+4), int(btn.Y+btn.H/2)); !near(c, tk.Accent.Base) {
			t.Errorf("the action is %v, want the Suggested fill, Accent %v", c, tk.Accent.Base)
		}
	}
	centered(true)
	// The icon is 48 high, in TextMuted, above the title.
	title := find(t, ut, "Chưa có tài khoản")
	if !inkNear(ut, ui.Rect{X: 300 - 24, Y: title.Y - 48 - space(2), W: 48, H: 48}, tk.TextMuted, 24) {
		t.Errorf("no icon in TextMuted %v above the title", tk.TextMuted)
	}
	tt.Find(ByLabel("Thêm")).Click()
	drained(t)
	if got := n.Load(); got != 1 {
		t.Errorf("the action ran %d times, want 1", got)
	}
	e.SetIcon(nil)
	tt.Frame()
	centered(false)
	e.SetAction("", nil)
	tt.Frame()
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 0 {
		t.Errorf("SetAction(\"\", nil) left %d buttons", n)
	}
	e.SetTitle("Không có gì").SetMessage("Trống.")
	tt.Frame()
	if !tt.Has(ByLabel("Không có gì")) || !tt.Has(ByLabel("Trống.")) {
		t.Errorf("SetTitle and SetMessage show %q", ut.Texts())
	}
}

// With no title, or no message, the one left is centered alone: no room is
// kept for the other.
func TestEmptyStateNoTitleOrMessage(t *testing.T) {
	calmOff()
	for _, c := range []struct{ title, message, shown string }{
		{"", "Chỉ có nội dung", "Chỉ có nội dung"},
		{"Chỉ có tiêu đề", "", "Chỉ có tiêu đề"},
	} {
		tt := tester(t, VBox(0).Add(EmptyState(c.title, c.message)).SetSize(600, 400))
		r := find(t, tt, c.shown)
		if mid := r.Y + r.H/2; math.Abs(float64(mid-200)) > 0.5 {
			t.Errorf("%q alone has its middle at %v, want 200", c.shown, mid)
		}
	}
}

// Case 10, its empty state part: a look of bg-chrome fills the empty state
// with Chrome.
func TestEmptyStateLook(t *testing.T) {
	useLooks(t, map[string]string{"EmptyState": "bg-chrome"})
	tk := skinhook.Get(false)
	tt := tester(t, VBox(0).Add(EmptyState("Trống", "")).SetSize(600, 400))
	if c := tt.Image().RGBAAt(20, 20); !near(c, tk.Chrome) || tk.Chrome == tk.Background {
		t.Errorf("the empty state's corner is %v, want Chrome %v", c, tk.Chrome)
	}
}

// A form of badges, info bars and empty states allocates in a frame only
// what ui allocates for the buttons in it: as many allocations as a form of
// those buttons alone, with and without a skin.
func TestStatusControlsFrameAllocs(t *testing.T) {
	status := func(i int) Control {
		return VBox(4).Add(
			HBox(0).Add(Badge("Live").SetStatus(StatusRunning).SetDetail("Bước 2/5")),
			InfoBar(StatusWarning, "Cần xác minh", "Tài khoản cần xác minh.").SetAction("Xác minh", func() {}).SetClosable(true),
			EmptyState("Chưa có tài khoản", "Thêm một tài khoản.").SetIcon(emptyIcon).SetAction("Thêm", func() {}),
		)
	}
	// The buttons the three hold: the info bar's action and close (an icon
	// button), and the empty state's Suggested action.
	buttons := func(i int) Control {
		return VBox(4).Add(Button("Xác minh").OnClick(func() {}), Button("").SetIcon(closeIcon).OnClick(func() {}),
			Button("Thêm").Suggested().OnClick(func() {}))
	}
	for _, skin := range []bool{false, true} {
		if skin {
			useCalm(t, true)
			t.Cleanup(calmOff)
		}
		got := (frameAllocs(status, 40) - frameAllocs(status, 20)) / 20
		want := (frameAllocs(buttons, 40) - frameAllocs(buttons, 20)) / 20
		if got > want+0.2 {
			t.Errorf("skin %v: a form of the three allocates %.2f a frame, its buttons alone %.2f", skin, got, want)
		}
	}
}
