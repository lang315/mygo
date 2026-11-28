package widget

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// flyoutRig is a flyout over a 400 by 300 owner button, whose content is a
// label and a button.
type flyoutRig struct {
	f              *FlyoutPanelWidget
	ownerClicks    int
	closed         int
	insideClicked  int
	tt             *Tester
	ownerX, ownerY float32 // a point of the owner far from every edge's panel: its centre's left
}

func newFlyoutRig(t *testing.T, edge FlyoutEdge, rtl bool) *flyoutRig {
	t.Helper()
	r := &flyoutRig{}
	owner := Button("Owner").SetSize(400, 300).OnClick(func() { r.ownerClicks++ })
	content := VBox(4).Add(Label("Panel text"), Button("Inside").OnClick(func() { r.insideClicked++ }))
	r.f = FlyoutPanel(owner, content).OnClosed(func() { r.closed++ })
	r.f.SetEdge(edge)
	r.tt = NewTester(t, r.f, 600, 400)
	if rtl {
		r.tt.SetDirection(ui.RightToLeft)
	}
	r.tt.Settle()
	return r
}

// settled draws the frames of the next second: the slide is over.
func (r *flyoutRig) settled() {
	r.tt.Settle()
	r.tt.Advance(time.Second)
	r.tt.Settle()
}

func (r *flyoutRig) owner() ui.Rect { return r.tt.Find(ByRole(ui.RoleButton).Label("Owner")).Bounds() }
func (r *flyoutRig) text() ui.Rect  { return r.tt.Find(ByLabel("Panel text")).Bounds() }

// content is the box around the content's label and button.
func (r *flyoutRig) content() ui.Rect {
	a, b := r.text(), r.tt.Find(ByRole(ui.RoleButton).Label("Inside")).Bounds()
	x, y := min(a.X, b.X), min(a.Y, b.Y)
	return ui.Rect{X: x, Y: y, W: max(a.X+a.W, b.X+b.W) - x, H: max(a.Y+a.H, b.Y+b.H) - y}
}

// clickOwnerAway clicks the centre of the owner, which no edge's panel
// reaches: the content is far less than half the owner wide or tall.
func (r *flyoutRig) clickOwnerAway() {
	o := r.owner()
	r.tt.ClickAt(o.X+o.W/2, o.Y+o.H/2)
	r.tt.Settle()
}

// Case 1: for each edge, the shown content sits at that edge of the owner's
// box, and start and end mirror in RTL: FlyoutRight is the end, the left
// right to left.
func TestFlyoutPanelEdges(t *testing.T) {
	const near = 40 // the panel's padding and border, well under the half of the owner
	for _, tc := range []struct {
		edge FlyoutEdge
		rtl  bool
		side string // where the content shows: left, right, top or bottom
	}{
		{FlyoutRight, false, "right"},
		{FlyoutLeft, false, "left"},
		{FlyoutTop, false, "top"},
		{FlyoutBottom, false, "bottom"},
		{FlyoutRight, true, "left"},
		{FlyoutLeft, true, "right"},
		{FlyoutTop, true, "top"},
		{FlyoutBottom, true, "bottom"},
	} {
		t.Run(fmt.Sprintf("%d rtl %v", tc.edge, tc.rtl), func(t *testing.T) {
			r := newFlyoutRig(t, tc.edge, tc.rtl)
			if r.tt.Has(ByLabel("Panel text")) {
				t.Fatalf("the closed flyout shows its content")
			}
			r.f.ShowPopup()
			r.settled()
			o, l := r.owner(), r.content()
			if l.W <= 0 || l.H <= 0 {
				t.Fatalf("the content is not visible: %+v", l)
			}
			if l.X < o.X-0.5 || l.Y < o.Y-0.5 || l.X+l.W > o.X+o.W+0.5 || l.Y+l.H > o.Y+o.H+0.5 {
				t.Fatalf("the content %+v is outside the owner's box %+v", l, o)
			}
			var ok bool
			switch tc.side {
			case "right":
				ok = o.X+o.W-(l.X+l.W) < near && l.X > o.X+o.W/2
			case "left":
				ok = l.X-o.X < near && l.X+l.W < o.X+o.W/2
			case "top":
				ok = l.Y-o.Y < near && l.Y+l.H < o.Y+o.H/2
			case "bottom":
				ok = o.Y+o.H-(l.Y+l.H) < near && l.Y > o.Y+o.H/2
			}
			if !ok {
				t.Fatalf("the content %+v is not at the %s of the owner %+v", l, tc.side, o)
			}
		})
	}
}

// Case 2: after ShowPopup, advancing the clock moves the content in; with
// reduced motion it is in place at once.
func TestFlyoutPanelSlides(t *testing.T) {
	r := newFlyoutRig(t, FlyoutRight, false)
	r.f.ShowPopup()
	r.tt.Settle()
	r.tt.Advance(60 * time.Millisecond)
	mid := r.text().X
	if m, o := r.text(), r.owner(); m.X+m.W > o.X+o.W+0.5 {
		t.Fatalf("on its way the content %+v shows past the owner's box %+v", m, o)
	}
	r.tt.Advance(time.Second)
	r.tt.Settle()
	end := r.text().X
	if mid-end < 5 {
		t.Fatalf("the content at 60 ms is at x %v, at rest at %v: it did not slide in from the right", mid, end)
	}
	// It slides out the way it came, the content still drawn on the way.
	r.f.HidePopup()
	r.tt.Settle()
	r.tt.Advance(60 * time.Millisecond)
	if !r.tt.Has(ByLabel("Panel text")) || r.text().X-end < 5 {
		t.Fatalf("60 ms after HidePopup the content is gone or did not move: has %v", r.tt.Has(ByLabel("Panel text")))
	}
	r.tt.Advance(time.Second)
	r.tt.Settle()
	if r.tt.Has(ByLabel("Panel text")) {
		t.Fatalf("a second after HidePopup the content still shows")
	}

	// Right to left, FlyoutRight is the left: it slides in from there.
	rr := newFlyoutRig(t, FlyoutRight, true)
	rr.f.ShowPopup()
	rr.tt.Settle()
	rr.tt.Advance(60 * time.Millisecond)
	mid = rr.text().X
	rr.settled()
	if end := rr.text().X; end-mid < 5 {
		t.Fatalf("right to left, the content at 60 ms is at x %v, at rest at %v: it did not slide in from the left", mid, end)
	}

	rm := newFlyoutRig(t, FlyoutRight, false)
	rm.tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	rm.tt.Settle()
	rm.f.ShowPopup()
	rm.tt.Settle()
	if got := rm.text().X; got != end {
		t.Fatalf("with reduced motion the content is at x %v at once, want %v", got, end)
	}
	rm.f.HidePopup()
	rm.tt.Settle()
	if rm.tt.Has(ByLabel("Panel text")) {
		t.Fatalf("with reduced motion the content still shows after HidePopup")
	}
}

// Case 3: Escape closes it, and so does a click outside when that is on
// (not when off). OnClosed fires once a closing by the user, never for
// HidePopup.
func TestFlyoutPanelClose(t *testing.T) {
	r := newFlyoutRig(t, FlyoutRight, false)
	r.f.ShowPopup()
	r.settled()
	if !r.f.IsOpen() || r.closed != 0 {
		t.Fatalf("after ShowPopup: open %v, closed %d", r.f.IsOpen(), r.closed)
	}
	r.tt.Tester.Key(0, ui.KeyEscape)
	r.tt.Settle()
	if r.f.IsOpen() || r.closed != 1 {
		t.Fatalf("after Escape: open %v, closed %d; want false and 1", r.f.IsOpen(), r.closed)
	}
	r.tt.Tester.Key(0, ui.KeyEscape)
	r.settled()
	if r.closed != 1 {
		t.Fatalf("a second Escape, closing: closed %d, want 1", r.closed)
	}
	// Escape closes it before the slide has moved it, too.
	r.f.ShowPopup()
	r.tt.Settle()
	r.tt.Tester.Key(0, ui.KeyEscape)
	r.settled()
	if r.f.IsOpen() || r.closed != 2 {
		t.Fatalf("Escape in the frame it opened: open %v, closed %d; want false and 2", r.f.IsOpen(), r.closed)
	}

	r.f.ShowPopup()
	r.settled()
	r.clickOwnerAway()
	if r.f.IsOpen() || r.closed != 3 {
		t.Fatalf("after a click outside: open %v, closed %d; want false and 3", r.f.IsOpen(), r.closed)
	}

	r.settled()
	r.f.SetCloseOnOutsideClick(false).ShowPopup()
	r.settled()
	r.clickOwnerAway()
	if !r.f.IsOpen() || r.closed != 3 {
		t.Fatalf("a click outside with SetCloseOnOutsideClick(false): open %v, closed %d; want true and 3", r.f.IsOpen(), r.closed)
	}
	r.tt.Find(ByRole(ui.RoleButton).Label("Inside")).Click()
	if !r.f.IsOpen() || r.insideClicked != 1 {
		t.Fatalf("a click inside: open %v, the button clicked %d times", r.f.IsOpen(), r.insideClicked)
	}
	r.f.HidePopup()
	r.settled()
	if r.f.IsOpen() || r.closed != 3 {
		t.Fatalf("HidePopup: open %v, closed %d; want false and 3 (a setter runs no handler)", r.f.IsOpen(), r.closed)
	}
}

// Case 9: the flyout lays out as its owner would: a box holding a multi-line
// text box stretches down a row, the box passing the height down, wrapped or
// not.
func TestFlyoutPanelPlacedAsOwner(t *testing.T) {
	box := func(wrap bool) ui.Rect {
		owner := MultilineTextBox()
		var c Control = owner
		if wrap {
			c = FlyoutPanel(owner, Label("Panel text"))
		}
		tt := NewTester(t, HBox(8).Add(Label("Side"), VBox(0).Add(c)).SetSize(500, 300), 600, 400)
		return tt.Find(ByRole(ui.RoleTextField)).Bounds()
	}
	if alone, wrapped := box(false), box(true); alone != wrapped {
		t.Fatalf("the owner in a flyout is at %+v, alone at %+v", wrapped, alone)
	}
}

// Case 10: a measure (a form's SetSize(w, 0)) before the frame leaves the
// panel to the frame: the focus still goes into it.
func TestFlyoutPanelSurvivesAMeasure(t *testing.T) {
	field := TextBox()
	f := FlyoutPanel(VBox(8).Add(field).SetSize(400, 300), VBox(4).Add(Button("Inside")))
	form := NewForm("Flyout", 600, 400).SetContent(f)
	tt := TestForm(t, form)
	tt.Find(ByRole(ui.RoleTextField)).Focus()
	f.ShowPopup()
	form.SetSize(500, 0)
	tt.Settle()
	if tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Fatalf("the focus stayed in the owner: the measure took the panel's focus")
	}
}

// Case 11: without OnClosed, closing by the user runs no handler, which
// would panic.
func TestFlyoutPanelNoOnClosed(t *testing.T) {
	panics := 0
	OnPanic(func(any, []byte) { panics++ })
	t.Cleanup(func() { OnPanic(nil) })
	f := FlyoutPanel(Button("Owner").SetSize(400, 300), Label("Panel text"))
	tt := NewTester(t, f, 600, 400)
	f.ShowPopup()
	tt.Settle()
	tt.Tester.Key(0, ui.KeyEscape)
	tt.Settle()
	if f.IsOpen() || panics != 0 {
		t.Fatalf("Escape without OnClosed: open %v, %d panics", f.IsOpen(), panics)
	}
}

// Case 8: an owner that is hidden shows no panel, open or not.
func TestFlyoutPanelHiddenOwner(t *testing.T) {
	r := newFlyoutRig(t, FlyoutRight, false)
	mu.Lock()
	owner := r.f.owner.(*ButtonWidget)
	mu.Unlock()
	owner.SetVisible(false)
	r.f.ShowPopup()
	r.settled()
	if r.tt.Has(ByLabel("Panel text")) {
		t.Fatalf("the panel of a hidden owner shows")
	}
}

// Case 4: the focus goes into the content on show, and back on hide.
func TestFlyoutPanelFocus(t *testing.T) {
	owner := VBox(8).Add(TextBox(), Button("Owner")).SetSize(400, 300)
	content := VBox(4).Add(Label("Panel text"), Button("Inside"))
	f := FlyoutPanel(owner, content)
	tt := NewTester(t, f, 600, 400)
	field := tt.Find(ByRole(ui.RoleTextField)).Focus()
	f.ShowPopup()
	tt.Settle()
	if field.Focused() {
		t.Fatalf("the focus stayed in the owner as the panel showed")
	}
	tt.Advance(time.Second)
	tt.Tester.Key(0, ui.KeyTab)
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleButton).Label("Inside")).Focused() {
		for _, n := range tt.Nodes() {
			if n.Focused {
				t.Logf("focused: %+v", n)
			}
		}
		t.Fatalf("Tab from where the panel put the focus did not reach the content's button")
	}
	f.HidePopup()
	tt.Settle()
	tt.Advance(time.Second)
	tt.Settle()
	if !field.Focused() {
		t.Fatalf("the focus did not come back to the owner's field on hide")
	}
	// Escape gives it back too.
	f.ShowPopup()
	tt.Settle()
	tt.Advance(time.Second)
	tt.Tester.Key(0, ui.KeyEscape)
	tt.Settle()
	tt.Advance(time.Second)
	tt.Settle()
	if !field.Focused() {
		t.Fatalf("the focus did not come back to the owner's field after Escape")
	}
}

// Case 5: while it is open, a click on the owner does not reach it, with
// close on outside click on or off; closed, the owner takes clicks again.
func TestFlyoutPanelOwnerInert(t *testing.T) {
	for _, closeOutside := range []bool{false, true} {
		r := newFlyoutRig(t, FlyoutRight, false)
		r.f.SetCloseOnOutsideClick(closeOutside).ShowPopup()
		r.settled()
		r.clickOwnerAway()
		if r.ownerClicks != 0 {
			t.Fatalf("close outside %v: the owner took %d clicks while the panel was open", closeOutside, r.ownerClicks)
		}
		if r.f.IsOpen() == closeOutside {
			t.Fatalf("close outside %v: open %v after the click", closeOutside, r.f.IsOpen())
		}
		r.f.HidePopup()
		r.settled()
		r.clickOwnerAway()
		if r.ownerClicks != 1 {
			t.Fatalf("close outside %v: closed, the owner took %d clicks, want 1", closeOutside, r.ownerClicks)
		}
	}
}

// Case 6: the setters from another goroutine while frames run show on a
// following frame, without a race.
func TestFlyoutPanelFromGoroutine(t *testing.T) {
	r := newFlyoutRig(t, FlyoutRight, false)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			r.f.SetEdge(FlyoutEdge(i % 4)).SetCloseOnOutsideClick(i%2 == 0).OnClosed(func() {})
			if i%3 == 0 {
				r.f.ShowPopup()
			} else {
				r.f.HidePopup()
			}
			_ = r.f.IsOpen()
		}
		r.f.SetEdge(FlyoutLeft).ShowPopup()
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			r.tt.Frame()
		}
	}
	r.settled()
	o, l := r.owner(), r.text()
	if !r.f.IsOpen() || l.X-o.X >= 40 {
		t.Fatalf("after the goroutine: open %v, content %+v in owner %+v; want open at the left", r.f.IsOpen(), l, o)
	}
}

// Case 7: a frame at rest allocates nothing more than the owner alone, closed,
// and than the owner beside the content, open.
func TestFlyoutPanelAllocNothingAtRest(t *testing.T) {
	frames := func(c Control, show func()) float64 {
		tt := ui.NewTester(content(c), 600, 400)
		tt.Frame()
		if show != nil {
			show()
		}
		for range 10 {
			tt.Frame()
		}
		tt.Advance(time.Second)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(30, tt.Frame)
	}
	mk := func() (Control, Control) {
		return Button("Owner").SetSize(400, 300), VBox(4).Add(Label("Panel text"), Button("Inside"))
	}
	o, _ := mk()
	ref := frames(o, nil)
	o, c := mk()
	closed := frames(FlyoutPanel(o, c), nil)
	if closed > ref {
		t.Fatalf("a closed flyout's frame at rest allocates %v times; its owner alone, %v", closed, ref)
	}
	o, c = mk()
	both := frames(VBox(0).Add(o, c), nil)
	o, c = mk()
	f := FlyoutPanel(o, c)
	open := frames(f, func() { f.ShowPopup() })
	if open > both {
		t.Fatalf("an open flyout's frame at rest allocates %v times; its owner and content in a box, %v", open, both)
	}
	t.Logf("closed %v, owner %v; open %v, both %v", closed, ref, open, both)
}
