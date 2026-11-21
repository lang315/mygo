package widget

import (
	"math"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// toastHookAtStart is whether the toast hook was set before the tests ran,
// which TestMain reads before m.Run.
var toastHookAtStart bool

// toastForm returns a Tester of a form w by h DIPs showing content (a label
// when nil), with the skins off; the test's end unsets the toast hook, so
// the forms of later tests build as an app's that shows no toast.
func toastForm(t *testing.T, w, h int, content Control) (*Form, *Tester) {
	t.Helper()
	calmOff()
	t.Cleanup(func() {
		calmOff()
		mu.Lock()
		toastHook = nil
		mu.Unlock()
	})
	if content == nil {
		content = Label("Nội dung")
	}
	f := NewForm("Toasts", w, h).SetContent(content)
	return f, TestForm(t, f)
}

// cards returns the toasts of the last frame: the nodes of RoleStatus and
// RoleAlert.
func cards(tt *Tester) []ui.TestNode {
	var out []ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleStatus || n.Role == ui.RoleAlert {
			out = append(out, n)
		}
	}
	return out
}

// toastCard returns the one toast of the last frame, failing without exactly one.
func toastCard(t *testing.T, tt *Tester) ui.TestNode {
	t.Helper()
	cs := cards(tt)
	if len(cs) != 1 {
		t.Fatalf("%d toasts, want 1: texts %q", len(cs), tt.Texts())
	}
	return cs[0]
}

// inside reports whether r is inside box, within half a DIP.
func inside(r, box ui.Rect) bool {
	return r.X >= box.X-0.5 && r.Y >= box.Y-0.5 && r.X+r.W <= box.X+box.W+0.5 && r.Y+r.H <= box.Y+box.H+0.5
}

// counter counts the runs of a callback, from any goroutine.
type counter struct{ n atomic.Int32 }

func (c *counter) fn() func() { return func() { c.n.Add(1) } }

// got returns the count once the handlers posted before have run.
func (c *counter) got(t *testing.T) int32 {
	t.Helper()
	drained(t)
	return c.n.Load()
}

// Case 1: a success toast on a shown form is a status holding its title and
// message, centred, its bottom less than Space(12) above the window's.
func TestToastShows(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	id := f.ShowToast(Toast{Status: StatusSuccess, Title: "Exported", Message: "12 accounts"})
	if id == "" {
		t.Error("ShowToast returned no ID")
	}
	tt.Frame()
	c := toastCard(t, tt)
	if c.Role != ui.RoleStatus {
		t.Errorf("the toast's role is %v, want RoleStatus", c.Role)
	}
	for _, s := range []string{"Exported", "12 accounts"} {
		if r := tt.Find(ByLabel(s)).Bounds(); !inside(r, c.Bounds) {
			t.Errorf("%q at %v is not inside the toast %v", s, r, c.Bounds)
		}
	}
	if mid := c.Bounds.X + c.Bounds.W/2; math.Abs(float64(mid-300)) > 1 {
		t.Errorf("the toast's centre is at %v, want the window's 300", mid)
	}
	if bottom := c.Bounds.Y + c.Bounds.H; bottom >= 400 || 400-bottom > space(12) {
		t.Errorf("the toast's bottom is at %v, want above 400 by less than %v", bottom, space(12))
	}
}

// Case 2: a toast closes 4 s after it showed, and runs OnClose once.
func TestToastTimesOut(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	f.ShowToast(Toast{Status: StatusSuccess, Title: "Exported", OnClose: closed.fn()})
	tt.Frame()
	tt.Advance(3900 * time.Millisecond)
	if !tt.Has(ByLabel("Exported")) {
		t.Fatal("the toast is gone at 3.9 s")
	}
	tt.Advance(100 * time.Millisecond)
	if tt.Has(ByLabel("Exported")) {
		t.Error("the toast shows at 4 s")
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("OnClose ran %d times, want 1", n)
	}
}

// Case 3: a negative Timeout shows until CloseToast, which runs OnClose once.
func TestToastUntilClosed(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	id := f.ShowToast(Toast{Title: "Uploading", Timeout: -1, OnClose: closed.fn()})
	tt.Frame()
	tt.Advance(60 * time.Second)
	if !tt.Has(ByLabel("Uploading")) {
		t.Fatal("a toast of negative Timeout is gone after 60 s")
	}
	f.CloseToast(id)
	tt.Frame()
	if tt.Has(ByLabel("Uploading")) {
		t.Error("the toast shows after CloseToast")
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("OnClose ran %d times, want 1", n)
	}
}

// Case 4: an Action is a button that runs OnAction once and closes the
// toast, OnClose once; not clicked, the toast shows for 8 s.
func TestToastAction(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var acted, closed counter
	f.ShowToast(Toast{Title: "Deleted", Action: "Undo", OnAction: acted.fn(), OnClose: closed.fn()})
	tt.Frame()
	b := tt.Find(ByRole(ui.RoleButton).Label("Undo"))
	if !inside(b.Bounds(), toastCard(t, tt).Bounds) {
		t.Errorf("the button %v is not in the toast", b.Bounds())
	}
	b.Click()
	if n := acted.got(t); n != 1 {
		t.Errorf("OnAction ran %d times, want 1", n)
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("OnClose ran %d times, want 1", n)
	}
	if tt.Has(ByLabel("Deleted")) {
		t.Error("the toast shows after its action")
	}
	tt.Move(5, 5) // off the toasts, whose time stops under the pointer
	f.ShowToast(Toast{Title: "Deleted again", Action: "Undo", OnAction: acted.fn()})
	tt.Frame()
	tt.Advance(7900 * time.Millisecond)
	if !tt.Has(ByLabel("Deleted again")) {
		t.Fatal("a toast with an action is gone at 7.9 s")
	}
	tt.Advance(100 * time.Millisecond)
	if tt.Has(ByLabel("Deleted again")) {
		t.Error("a toast with an action shows at 8 s")
	}
	if n := acted.got(t); n != 1 {
		t.Errorf("OnAction ran %d times without a click, want 1 in all", n)
	}
}

// Case 5: two toasts of one ID are one toast, the second's; queued in one
// frame, the first is never shown nor announced.
func TestToastSameID(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	tt.Announcements()
	a := f.ShowToast(Toast{ID: "export", Title: "First"})
	b := f.ShowToast(Toast{ID: "export", Title: "Second"})
	if a != "export" || b != "export" {
		t.Errorf("ShowToast returned %q and %q, want the ID", a, b)
	}
	tt.Frame()
	toastCard(t, tt)
	if !tt.Has(ByLabel("Second")) || tt.Has(ByLabel("First")) {
		t.Errorf("texts %q: want the second's title alone", tt.Texts())
	}
	if said := tt.Announcements(); slices.Contains(said, "First") || !slices.Contains(said, "Second") {
		t.Errorf("announced %q: want Second, never First", said)
	}
	// Showing: the same ID changes it.
	f.ShowToast(Toast{ID: "export", Title: "Third"})
	tt.Frame()
	toastCard(t, tt)
	if !tt.Has(ByLabel("Third")) {
		t.Errorf("texts %q: want Third", tt.Texts())
	}
}

// Case 6: of four toasts the three newest show; closing one shows the first.
func TestToastNewestThree(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	ids := map[string]string{}
	for _, s := range []string{"A", "B", "C", "D"} {
		ids[s] = f.ShowToast(Toast{Title: s, Timeout: -1})
	}
	tt.Frame()
	shows := func() (out []string) {
		for _, s := range []string{"A", "B", "C", "D"} {
			if tt.Has(ByLabel(s)) {
				out = append(out, s)
			}
		}
		return out
	}
	if got := shows(); !slices.Equal(got, []string{"B", "C", "D"}) {
		t.Errorf("four toasts show %q, want B C D", got)
	}
	if len(cards(tt)) != 3 {
		t.Errorf("%d toasts, want 3", len(cards(tt)))
	}
	f.CloseToast(ids["C"])
	tt.Frame()
	if got := shows(); !slices.Equal(got, []string{"A", "B", "D"}) {
		t.Errorf("after closing C %q show, want A B D", got)
	}
}

// Case 7: Warning and Danger are alerts, the others statuses; each toast's
// icon is the info bar's of its status.
func TestToastRolesAndIcons(t *testing.T) {
	for _, skinned := range []bool{false, true} {
		for s := StatusNone; s <= StatusDanger; s++ {
			f, tt := toastForm(t, 600, 400, nil)
			if skinned {
				useCalm(t, true)
			}
			f.ShowToast(Toast{Status: s, Title: "Exported", Message: "12 accounts", Timeout: -1})
			tt.Frame()
			c := toastCard(t, tt)
			want := ui.RoleStatus
			if s == StatusWarning || s == StatusDanger {
				want = ui.RoleAlert
			}
			if c.Role != want {
				t.Errorf("skin %v, %v: the role is %v, want %v", skinned, s, c.Role, want)
			}
			_, _, _, base := statusColorsOf(themeOf(false), s)
			toast := pixelsNear(tt.Tester, iconBox(c.Bounds), base, 6)
			bt := tester(t, VBox(0).Add(InfoBar(s, "Exported", "12 accounts")).SetSize(400, 0))
			bar := pixelsNear(bt, iconBox(barOf(t, bt, want)), base, 6)
			if toast < 40 || math.Abs(float64(toast-bar)) > max(4, float64(bar)/10) {
				t.Errorf("skin %v, %v: the icon has %d pixels of its base %v, the info bar's %d", skinned, s, toast, base, bar)
			}
		}
	}
}

// themeOf returns the theme a window draws with, dark or light, under the
// skin on.
func themeOf(dark bool) *ui.Theme {
	var th *ui.Theme
	probe := ui.NewTester(func(c *ui.Context) { th = c.Theme() }, 10, 10)
	probe.SetDark(dark)
	probe.Frame()
	return th
}

// toastRed is the content behind the toasts in the tests of their fill: a
// translucent fill shows it through.
var toastRed = Hex("#ff0000")

// toastFill is the toast's fill: inside its left padding, half way down.
func toastFill(tt *Tester, c ui.TestNode) ui.Color {
	p := tt.Image().RGBAAt(int(c.Bounds.X+4), int(c.Bounds.Y+c.Bounds.H/2))
	return ui.Color{R: p.R, G: p.G, B: p.B, A: p.A}
}

// Case 8: the fill is opaque, the status's fill over Surface, the title in
// the status's text; without a skin the plain colors; StatusNone neutral.
func TestToastColors(t *testing.T) {
	for _, look := range []struct {
		name string
		dark bool
	}{{"", false}, {"", true}, {"Calm", false}, {"Calm", true}} {
		for _, s := range []Status{StatusNone, StatusSuccess, StatusDanger} {
			f, tt := toastForm(t, 600, 400, VBox(0).Apply(NewStyle().Background(toastRed)))
			if look.name != "" {
				useSkinNamed(t, look.name, false)
			}
			tt.SetDark(look.dark)
			f.ShowToast(Toast{Status: s, Title: "Exported", Message: "12 accounts", Timeout: -1})
			tt.Frame()
			c := toastCard(t, tt)
			th := themeOf(look.dark)
			fill, text, _, _ := statusColorsOf(th, s)
			want := fill.Over(th.Surface)
			if want.A != 255 {
				t.Fatalf("%q dark %v, %v: Surface %v is not opaque", look.name, look.dark, s, th.Surface)
			}
			if got := toastFill(tt, c); !nearColor(got, want, 2) {
				t.Errorf("%q dark %v, %v: the fill is %v, want %v over Surface, %v", look.name, look.dark, s, got, fill, want)
			}
			if title := tt.Find(ByLabel("Exported")).Bounds(); !inkNear(tt.Tester, title, text, 12) {
				t.Errorf("%q dark %v, %v: no ink of the text %v in the title", look.name, look.dark, s, text)
			}
		}
	}
	// The neutral colors are the theme's own, with no skin and under Calm.
	calmOff()
	th := ui.LightTheme()
	if fill, text, _, _ := statusColorsOf(th, StatusNone); fill != th.Text.Alpha(0.1) || text != th.Text {
		t.Errorf("no skin: StatusNone is %v on %v", text, fill)
	}
	useCalm(t, true)
	tk := skinhook.Get(false)
	if fill, text, _, _ := statusColorsOf(th, StatusNone); fill != tk.SurfaceHover || text != tk.TextMuted {
		t.Errorf("Calm: StatusNone is %v on %v, want TextMuted on SurfaceHover", text, fill)
	}
	calmOff()
}

// nearColor reports whether a and b are within tol on each channel.
func nearColor(a, b ui.Color, tol int) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= tol && int(y)-int(x) <= tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

// Case 9: ShowToast from another goroutine while frames run shows on a
// following frame (run under -race).
func TestToastFromAGoroutine(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	done := make(chan string)
	go func() { done <- f.ShowToast(Toast{Title: "From a worker"}) }()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Frame()
	if !tt.Has(ByLabel("From a worker")) {
		t.Errorf("texts %q: the worker's toast does not show", tt.Texts())
	}
}

// Case 10: a toast shown before the form shows on its first frame, its time
// starting then.
func TestToastBeforeShow(t *testing.T) {
	calmOff()
	t.Cleanup(func() { mu.Lock(); toastHook = nil; mu.Unlock() })
	f := NewForm("Toasts", 600, 400).SetContent(Label("Nội dung"))
	f.ShowToast(Toast{Title: "Early"})
	tt := TestForm(t, f)
	if !tt.Has(ByLabel("Early")) {
		t.Fatalf("texts %q: the toast does not show on the first frame", tt.Texts())
	}
	tt.Advance(3900 * time.Millisecond)
	if !tt.Has(ByLabel("Early")) {
		t.Error("the toast is gone 3.9 s after the first frame")
	}
}

// Case 11: CloseToast("") closes every toast, each OnClose once.
func TestToastCloseAll(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	for _, s := range []string{"A", "B", "C"} {
		f.ShowToast(Toast{Title: s, Timeout: -1, OnClose: closed.fn()})
	}
	tt.Frame()
	if n := len(cards(tt)); n != 3 {
		t.Fatalf("%d toasts, want 3", n)
	}
	f.CloseToast("")
	tt.Frame()
	if n := len(cards(tt)); n != 0 {
		t.Errorf("%d toasts after CloseToast(\"\")", n)
	}
	if n := closed.got(t); n != 3 {
		t.Errorf("OnClose ran %d times, want 3", n)
	}
}

// Case 12: a dialog's toast shows in the dialog, not in its owner.
func TestToastInADialog(t *testing.T) {
	owner, ot := toastForm(t, 600, 400, nil)
	d := NewDialog(owner, "Hộp thoại", "", 400, 300, Label("Trong hộp thoại"))
	dt := TestDialog(t, d)
	d.ShowToast(Toast{Title: "In the dialog"})
	dt.Frame()
	ot.Frame()
	if !dt.Has(ByLabel("In the dialog")) {
		t.Errorf("the dialog's texts %q: no toast", dt.Texts())
	}
	if ot.Has(ByLabel("In the dialog")) || len(cards(ot)) != 0 {
		t.Error("the dialog's toast shows in its owner")
	}
}

// Case 13: the close button, and Escape with the focus in the toast, close
// it, each OnClose once.
func TestToastCloseButtonAndEscape(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	f.ShowToast(Toast{Title: "Saved", Timeout: -1, OnClose: closed.fn()})
	tt.Frame()
	tt.Find(ByRole(ui.RoleButton).Label("Close")).Click()
	if tt.Has(ByLabel("Saved")) {
		t.Error("the toast shows after its close button")
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("the close button: OnClose ran %d times, want 1", n)
	}
	f.ShowToast(Toast{Title: "Saved again", Timeout: -1, Action: "Undo", OnClose: closed.fn()})
	tt.Frame()
	tt.Find(ByRole(ui.RoleButton).Label("Undo")).Focus().Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.Has(ByLabel("Saved again")) {
		t.Error("the toast shows after Escape")
	}
	if n := closed.got(t); n != 2 {
		t.Errorf("Escape: OnClose ran %d times in all, want 2", n)
	}
}

// Escape in a toast closes only the toast: the form's cancel button is not
// clicked.
func TestToastEscapeLeavesTheCancelButton(t *testing.T) {
	var canceled counter
	cancel := Button("Cancel").OnClick(canceled.fn())
	f, tt := toastForm(t, 600, 400, VBox(0).Add(cancel))
	f.SetCancelButton(cancel)
	f.ShowToast(Toast{Title: "Saved", Timeout: -1, Action: "Undo"})
	tt.Frame()
	tt.Find(ByRole(ui.RoleButton).Label("Undo")).Focus().Key(0, ui.KeyEscape)
	tt.Settle()
	if tt.Has(ByLabel("Saved")) {
		t.Error("the toast shows after Escape")
	}
	if n := canceled.got(t); n != 0 {
		t.Errorf("Escape in the toast clicked the cancel button %d times", n)
	}
}

// Case 14: no toast hook before a ShowToast, and a form that never shows one
// leaves it unset.
func TestToastHookUnsetUntilShown(t *testing.T) {
	if toastHookAtStart {
		t.Error("the toast hook is set before any ShowToast")
	}
	mu.Lock()
	toastHook = nil
	mu.Unlock()
	f := NewForm("Plain", 600, 400).SetContent(Label("Nội dung"))
	tt := TestForm(t, f)
	tt.Frame()
	tt.Advance(5 * time.Second)
	mu.Lock()
	set := toastHook != nil
	mu.Unlock()
	if set {
		t.Error("a form that showed no toast set the toast hook")
	}
}

// CloseToast before any ShowToast queues nothing: an app that closes toasts
// it never showed does not grow the form's queue without a frame to drain it.
func TestToastCloseWithoutShowQueuesNothing(t *testing.T) {
	mu.Lock()
	toastHook = nil
	mu.Unlock()
	f := NewForm("Plain", 600, 400)
	for range 3 {
		f.CloseToast("")
	}
	mu.Lock()
	n := len(f.toasts)
	mu.Unlock()
	if n != 0 {
		t.Errorf("%d closes queued with no toast shown", n)
	}
}

// A toast queued before a refit (SetSize(w, 0)) shows after it: the measure
// leaves the queue to the frame.
func TestToastSurvivesAMeasure(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	f.ShowToast(Toast{Title: "Measured"})
	f.SetSize(500, 0)
	tt.Frame()
	if !tt.Has(ByLabel("Measured")) {
		t.Errorf("texts %q: the toast is lost in the measure", tt.Texts())
	}
}

// Show, close and show again of one ID before a frame: one toast, the last,
// and the first's OnClose once.
func TestToastQueueKeepsACloseBetween(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var closed counter
	f.ShowToast(Toast{ID: "x", Title: "One", Timeout: -1, OnClose: closed.fn()})
	f.CloseToast("x")
	f.ShowToast(Toast{ID: "x", Title: "Two", Timeout: -1})
	tt.Frame()
	toastCard(t, tt)
	if !tt.Has(ByLabel("Two")) || tt.Has(ByLabel("One")) {
		t.Errorf("texts %q: want Two alone", tt.Texts())
	}
	if n := closed.got(t); n != 1 {
		t.Errorf("the first's OnClose ran %d times, want 1", n)
	}
}

// IDs made up are distinct.
func TestToastMadeUpIDs(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	a, b := f.ShowToast(Toast{Title: "A"}), f.ShowToast(Toast{Title: "B"})
	if a == "" || a == b {
		t.Errorf("made-up IDs %q and %q", a, b)
	}
	tt.Frame()
	if n := len(cards(tt)); n != 2 {
		t.Errorf("%d toasts, want 2", n)
	}
}

// A toast ui code adds itself (c.Toast in a Custom control) shows in the
// same look, neutral, once the toasts of package widget are in use.
func TestToastOfUICode(t *testing.T) {
	added := false
	_, tt := toastForm(t, 600, 400, VBox(0).Apply(NewStyle().Background(toastRed)).Add(Custom(func(c *ui.Context) ui.Element {
		if !added {
			added = true
			c.AddToast(ui.Toast{Title: "Copied", Timeout: -1})
		}
		return ui.Text(c, "Custom")
	})))
	NewForm("Other", 400, 300).ShowToast(Toast{Title: "Elsewhere"})
	tt.Frame()
	c := toastCard(t, tt)
	th := themeOf(false)
	fill, _, _, _ := statusColorsOf(th, StatusNone)
	if got, want := toastFill(tt, c), fill.Over(th.Surface); !nearColor(got, want, 2) {
		t.Errorf("ui's toast is filled %v, want the neutral %v", got, want)
	}
}

// Review focus 1: at text scale 1.5 in a window 300 DIPs wide, a long
// message wraps inside the toast, which keeps Space(4) from each side.
func TestToastNarrowAndLarge(t *testing.T) {
	var th *ui.Theme
	f, tt := toastForm(t, 300, 500, Custom(func(c *ui.Context) ui.Element { th = c.Theme(); return ui.Text(c, "x") }))
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	f.ShowToast(Toast{Status: StatusWarning, Title: "Cần xác minh", Message: longMessage, Action: "Mở", Timeout: -1})
	tt.Frame()
	c := toastCard(t, tt)
	gap := th.Space(4)
	if c.Bounds.X < gap-0.5 || c.Bounds.X+c.Bounds.W > 300-gap+0.5 {
		t.Errorf("the toast %v does not keep %v from the sides of 300", c.Bounds, gap)
	}
	title, msg := tt.Find(ByLabel("Cần xác minh")).Bounds(), tt.Find(ByLabel(longMessage)).Bounds()
	if msg.H < 2*title.H-1 {
		t.Errorf("the message %v does not wrap (a line is %v)", msg, title.H)
	}
	for _, r := range []ui.Rect{title, msg, tt.Find(ByRole(ui.RoleButton).Label("Mở")).Bounds()} {
		if !inside(r, c.Bounds) {
			t.Errorf("%v is not inside the toast %v", r, c.Bounds)
		}
	}
}

// Review focus 2: in dark mode, under each built-in skin and none, the
// title and message reach 4.5:1 on the toast's fill as drawn.
func TestToastContrast(t *testing.T) {
	for _, name := range []string{"", "Calm", "Classic", "HighContrast"} {
		for s := StatusNone; s <= StatusDanger; s++ {
			f, tt := toastForm(t, 600, 400, nil)
			if name != "" {
				useSkinNamed(t, name, false)
			}
			tt.SetDark(true)
			f.ShowToast(Toast{Status: s, Title: "Exported", Message: "12 accounts", Timeout: -1})
			tt.Frame()
			_, text, _, _ := statusColorsOf(themeOf(true), s)
			if r := contrastOf(text, toastFill(tt, toastCard(t, tt))); r < 4.5 {
				t.Errorf("%q dark, %v: the text %v is %.2f:1 on the fill", name, s, text, r)
			}
		}
	}
}

// Review focus 3: the time stops while the pointer rests on the toast:
// hovered 10 s, then left, it closes 4 s after it showed, hovered time not
// counted.
func TestToastPausesWhileHovered(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	f.ShowToast(Toast{Title: "Exported"})
	tt.Frame()
	c := toastCard(t, tt).Bounds
	tt.Move(c.X+c.W/2, c.Y+c.H/2)
	tt.Advance(10 * time.Second)
	if !tt.Has(ByLabel("Exported")) {
		t.Fatal("the hovered toast closed")
	}
	tt.Move(5, 5)
	tt.Advance(3900 * time.Millisecond)
	if !tt.Has(ByLabel("Exported")) {
		t.Fatal("the toast closed 3.9 s after the pointer left")
	}
	tt.Advance(100 * time.Millisecond)
	if tt.Has(ByLabel("Exported")) {
		t.Error("the toast shows 4 s after the pointer left")
	}
}

// toastFrameAllocs returns the allocations of a frame of tt at rest.
func toastFrameAllocs(tt *ui.Tester) float64 {
	tt.Frame()
	return testing.AllocsPerRun(20, tt.Frame)
}

// Review focus 4: a form with the hook set and no toast allocates nothing
// more per frame; one toast at rest costs no more than ui's own.
func TestToastFrameAllocs(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	plain := toastFrameAllocs(tt.Tester)
	NewForm("Other", 400, 300).ShowToast(Toast{Title: "Elsewhere"})
	if hooked := toastFrameAllocs(tt.Tester); hooked != plain {
		t.Errorf("with the hook set and no toast a frame allocates %v, want %v", hooked, plain)
	}
	f.ShowToast(Toast{Status: StatusSuccess, Title: "Exported", Message: "12 accounts", Action: "Undo", Timeout: -1})
	toasted := toastFrameAllocs(tt.Tester)
	if len(cards(tt)) != 1 {
		t.Fatal("no toast")
	}
	add := false
	ut := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, "Nội dung")
		if add {
			add = false
			c.AddToast(ui.Toast{Title: "Exported", Description: "12 accounts", Action: "Undo", Timeout: -1})
		}
	}, 600, 400)
	uiPlain := toastFrameAllocs(ut)
	add = true
	ut.Frame()
	ut.Advance(time.Second) // past ui's fade in: at rest
	uiToasted := toastFrameAllocs(ut)
	if toasted-plain > uiToasted-uiPlain {
		t.Errorf("a toast at rest costs %v allocations a frame, ui's own %v", toasted-plain, uiToasted-uiPlain)
	}
	t.Logf("allocs a frame: form %v, with a toast %v; ui %v, with a toast %v", plain, toasted, uiPlain, uiToasted)
}

// Review focus 5: an OnAction that shows a toast (Undo, then "Restored")
// shows it, without a deadlock.
func TestToastActionShowsAToast(t *testing.T) {
	var f *Form
	f, tt := toastForm(t, 600, 400, nil)
	f.ShowToast(Toast{Title: "Deleted", Action: "Undo", OnAction: func() { f.ShowToast(Toast{Title: "Restored"}) }})
	tt.Frame()
	tt.Find(ByRole(ui.RoleButton).Label("Undo")).Click()
	tt.Settle()
	if !tt.Has(ByLabel("Restored")) || tt.Has(ByLabel("Deleted")) {
		t.Errorf("texts %q: want Restored alone", tt.Texts())
	}
}
