package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// formTester shows f's view alone in a headless window, as the form docks
// what it holds.
func formTester(t *testing.T, f *Form) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(f.view, 800, 600)
	tt.Frame()
	return tt
}

func TestStatusStripDocksAtTheBottom(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content")).StatusStrip(StatusStrip(StatusLabel("Ready")))
	tt := formTester(t, f)
	content, status := find(t, tt, "content"), find(t, tt, "Ready")
	if status.Y <= content.Y+content.H {
		t.Fatalf("status at %v is not below the content at %v", status, content)
	}
	if bottom := status.Y + status.H; bottom > 600 || bottom < 560 {
		t.Fatalf("the status text ends at y %v of a 600 high window: not at the bottom", bottom)
	}
}

// A line above the strip sets it apart from the content.
func TestStatusStripHasALineAbove(t *testing.T) {
	f := NewForm("Shell", 800, 600).SetContent(Label("content")).StatusStrip(StatusStrip(StatusLabel("Ready")))
	tt := formTester(t, f)
	status, border, found := find(t, tt, "Ready"), ui.LightTheme().Border, false
	for y := int(status.Y) - 12; y < int(status.Y); y++ {
		if near(tt.Image().RGBAAt(700, y), border) {
			found = true
		}
	}
	if !found {
		t.Fatal("no line above the status strip")
	}
}

func TestStatusStripSpringPushesTheRestToTheEnd(t *testing.T) {
	f := NewForm("Shell", 800, 600).StatusStrip(StatusStrip(StatusLabel("left"), Spring(), StatusLabel("right")))
	tt := formTester(t, f)
	left, right := find(t, tt, "left"), find(t, tt, "right")
	if left.X > 20 {
		t.Fatalf("left at x %v", left.X)
	}
	if end := right.X + right.W; end < 780 {
		t.Fatalf("right ends at x %v of 800: the spring did not take the free width", end)
	}
}

func TestStatusStripSeparatorIsALineBetweenItems(t *testing.T) {
	f := NewForm("Shell", 800, 600).StatusStrip(StatusStrip(StatusLabel("one"), Separator(), StatusLabel("two")))
	tt := formTester(t, f)
	one, two := find(t, tt, "one"), find(t, tt, "two")
	border := ui.LightTheme().Border
	y, found := int(one.Y+one.H/2), false
	for x := int(one.X + one.W); x < int(two.X); x++ {
		if near(tt.Image().RGBAAt(x, y), border) {
			found = true
		}
	}
	if !found {
		t.Fatal("no separator line between the two labels")
	}
}

func TestStatusLabelChangesAtRunTime(t *testing.T) {
	l := StatusLabel("Ready")
	f := NewForm("Shell", 800, 600).StatusStrip(StatusStrip(l))
	tt := formTester(t, f)
	plain := find(t, tt, "Ready").X
	l.SetText("Saved")
	l.SetIcon(chevronDown)
	tt.Frame()
	if tt.HasText("Ready") || !tt.HasText("Saved") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if with := find(t, tt, "Saved").X; with <= plain+8 {
		t.Fatalf("with an icon the text is at x %v, without at %v: the icon takes room before it", with, plain)
	}
	l.SetIcon(nil)
	tt.Frame()
	if find(t, tt, "Saved").X != plain {
		t.Fatal("removing the icon did not give its room back")
	}
}

func TestStatusStripUpdateRedrawsTheFormsWindow(t *testing.T) {
	l := StatusLabel("Ready")
	f := NewForm("Shell", 300, 200).SetContent(Label("content")).StatusStrip(StatusStrip(l))
	w := showFormFrame(t, f)
	redrawn := recordInvalidations(t)
	l.SetText("Busy")
	wantRedrawn(t, redrawn(), w)
}

func TestStatusProgressShowsItsFraction(t *testing.T) {
	p := StatusProgress().SetFraction(0.5)
	f := NewForm("Shell", 800, 600).StatusStrip(StatusStrip(p))
	tt := formTester(t, f)
	accent := ui.LightTheme().Accent
	filled := func(x int) bool {
		for y := 600 - 40; y < 600; y++ {
			if near(tt.Image().RGBAAt(x, y), accent) {
				return true
			}
		}
		return false
	}
	if !filled(12 + 20) {
		t.Fatal("the first part of a half full bar is not filled")
	}
	if filled(12 + 80) {
		t.Fatal("the end of a half full bar is filled")
	}
}

func TestStatusStripAddsItemsAndHoldsHiddenOnes(t *testing.T) {
	s := StatusStrip(StatusLabel("one"))
	two := StatusLabel("two").SetVisible(false)
	s.Add(two)
	f := NewForm("Shell", 300, 200).StatusStrip(s)
	if tt := formTester(t, f); tt.HasText("two") || !tt.HasText("one") {
		t.Fatalf("texts %q: a hidden item shows, or one is gone", tt.Texts())
	}
	w := showFormFrame(t, f) // the shown window builds with the hidden item held
	redrawn := recordInvalidations(t)
	two.SetVisible(true)
	wantRedrawn(t, redrawn(), w)
}
