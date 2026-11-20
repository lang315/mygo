package widget

import (
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

const nameDesc = "Dùng khi đăng nhập lần sau."

// mutedIn reports whether the rectangle r draws a pixel of the plain theme's
// TextMuted, the token's value, #71717a.
func mutedIn(tt *ui.Tester, r ui.Rect) bool {
	muted, img := ui.Hex("#71717a"), tt.Image()
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if near(img.RGBAAt(x, y), muted) {
				return true
			}
		}
	}
	return false
}

// textIn reports whether r draws a pixel of the plain theme's Text, #18181b.
func textIn(tt *ui.Tester, r ui.Rect) bool {
	text, img := ui.Hex("#18181b"), tt.Image()
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if near(img.RGBAAt(x, y), text) {
				return true
			}
		}
	}
	return false
}

// descColors reports whether the description nameDesc of form draws
// TextMuted, and whether it draws Text, at text scale 3: its stems are then
// several pixels wide, the ink itself under any anti-aliasing, grayscale or
// subpixel (ClearType).
func descColors(t *testing.T, f *FieldsWidget) (muted, text bool) {
	t.Helper()
	tt := NewTester(t, f, 1200, 700)
	tt.SetPreferences(ui.Preferences{TextScale: 3})
	tt.Frame()
	d := find(t, tt.Tester, nameDesc)
	return mutedIn(tt.Tester, d), textIn(tt.Tester, d)
}

// descriptionOf returns the accessible description of the text field.
func descriptionOf(t *testing.T, tt *Tester, name string) string {
	t.Helper()
	a, ok := tt.AccessByRole(ui.RoleTextField, name)
	if !ok {
		t.Fatalf("no text field %q", name)
	}
	return a.Description
}

// Case 8: Describe after Row draws a second line under the label in
// TextMuted, which wraps within the label's column; the row grows to hold
// it, and the control's accessible description is the text.
func TestDescribeDrawsUnderTheLabel(t *testing.T) {
	tb := TextBox()
	f := Fields().SettingsRows().Row("Tên", tb).Describe(nameDesc)
	tt := NewTester(t, f, 400, 300)
	l, d := find(t, tt.Tester, "Tên"), find(t, tt.Tester, nameDesc)
	if d.Y < l.Y+l.H-0.5 || !approx(d.X, l.X) {
		t.Errorf("the description %v is not under the label %v", d, l)
	}
	if muted, text := descColors(t, Fields().SettingsRows().Row("Tên", TextBox()).Describe(nameDesc)); !muted || text {
		t.Errorf("the description draws TextMuted %v and Text %v, want TextMuted alone", muted, text)
	}
	if got := descriptionOf(t, tt, "Tên"); got != nameDesc {
		t.Errorf("the control's description is %q, want %q", got, nameDesc)
	}
	plain := NewTester(t, Fields().SettingsRows().Row("Tên", TextBox()), 400, 300)
	k, p := cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border), cardsOf(t, plain.Tester, cardX, ui.LightTheme().Border)
	if k[0].bottom-k[0].top <= p[0].bottom-p[0].top {
		t.Errorf("the card is %d high with a description, %d without: the row did not grow", k[0].bottom-k[0].top, p[0].bottom-p[0].top)
	}

	// In a narrow window it wraps, and stops before the control.
	long := strings.TrimSpace(strings.Repeat(nameDesc+" ", 4))
	n := NewTester(t, Fields().SettingsRows().Row("Tên", TextBox()).Describe(long), 300, 400)
	d, c := find(t, n.Tester, long), n.Find(ByRole(ui.RoleTextField)).Bounds()
	if d.H < 2*find(t, n.Tester, "Tên").H {
		t.Errorf("the long description %v is one line", d)
	}
	if d.X+d.W > c.X+0.5 {
		t.Errorf("the description %v runs under the control %v", d, c)
	}
}

// Case 9: Describe before any Row changes nothing and does not panic;
// Describe("") takes a description away; a second Describe replaces it.
func TestDescribeEdges(t *testing.T) {
	f := Fields().SettingsRows().Describe("nowhere").Row("Tên", TextBox())
	tt := NewTester(t, f, 400, 300)
	if tt.HasText("nowhere") || descriptionOf(t, tt, "Tên") != "" {
		t.Errorf("Describe before any row shows %q, description %q", tt.Texts(), descriptionOf(t, tt, "Tên"))
	}
	f.Describe(nameDesc)
	tt.Frame()
	f.Describe("Other.")
	tt.Frame()
	if tt.HasText(nameDesc) || !tt.HasText("Other.") || descriptionOf(t, tt, "Tên") != "Other." {
		t.Errorf("a second Describe: texts %q, description %q", tt.Texts(), descriptionOf(t, tt, "Tên"))
	}
	f.Describe("")
	tt.Frame()
	if tt.HasText("Other.") || descriptionOf(t, tt, "Tên") != "" {
		t.Errorf("Describe(\"\"): texts %q, description %q", tt.Texts(), descriptionOf(t, tt, "Tên"))
	}
}

// Describe describes the last row added, not the one before it.
func TestDescribeTheLastRow(t *testing.T) {
	f := Fields().SettingsRows().Row("A", TextBox()).Row("B", TextBox()).Describe(nameDesc)
	tt := NewTester(t, f, 400, 300)
	if descriptionOf(t, tt, "A") != "" || descriptionOf(t, tt, "B") != nameDesc {
		t.Errorf("descriptions A %q, B %q: want only B's", descriptionOf(t, tt, "A"), descriptionOf(t, tt, "B"))
	}
	if d, b := find(t, tt.Tester, nameDesc), find(t, tt.Tester, "B"); d.Y < b.Y {
		t.Errorf("the description %v is above B %v", d, b)
	}
}

// Case 10: in the plain grid the description is under the label; stacked,
// under the label and above the control; both in TextMuted.
func TestDescribeInTheOtherLayouts(t *testing.T) {
	for _, stacked := range []bool{false, true} {
		f := Fields().Row("Tên", TextBox()).Describe(nameDesc)
		if stacked {
			f.Stacked()
		}
		tt := NewTester(t, f, 400, 300)
		l, d, c := find(t, tt.Tester, "Tên"), find(t, tt.Tester, nameDesc), tt.Find(ByRole(ui.RoleTextField)).Bounds()
		if d.Y < l.Y+l.H-0.5 || !approx(d.X, l.X) {
			t.Errorf("stacked %v: the description %v is not under the label %v", stacked, d, l)
		}
		if stacked && c.Y < d.Y+d.H-0.5 {
			t.Errorf("stacked: the control %v is not under the description %v", c, d)
		}
		if !stacked && c.X < l.X+l.W {
			t.Errorf("grid: the control %v is not beside the label %v", c, l)
		}
		g := Fields().Row("Tên", TextBox()).Describe(nameDesc)
		if stacked {
			g.Stacked()
		}
		if muted, text := descColors(t, g); !muted || text {
			t.Errorf("stacked %v: the description draws TextMuted %v and Text %v, want TextMuted alone", stacked, muted, text)
		}
		if got := descriptionOf(t, tt, "Tên"); got != nameDesc {
			t.Errorf("stacked %v: the control's description is %q", stacked, got)
		}
	}
}

// Case 14: a tooltip and a description: the description, then the
// tooltip, joined by ". ", for more than a text field; a tooltip set
// after Describe joins too. An error from an ErrorProvider comes first
// while it shows.
func TestDescribeWithATooltipAndAnError(t *testing.T) {
	tb := TextBox().SetTooltip("Họ và tên.")
	sw := Switch()
	f := Fields().SettingsRows().Row("Tên", tb).Describe(nameDesc).Row("Bật", sw).Describe("Tắt khi rời.")
	tt := NewTester(t, f, 400, 300)
	if got, want := descriptionOf(t, tt, "Tên"), nameDesc+". Họ và tên."; got != want {
		t.Errorf("description %q, want %q", got, want)
	}
	if a, ok := tt.AccessByRole(ui.RoleSwitch, "Bật"); !ok || a.Description != "Tắt khi rời." {
		t.Errorf("the switch's description is %q (found %v), want its row's", a.Description, ok)
	}
	sw.SetTooltip("Mở hay tắt.")
	tt.Frame()
	if a, _ := tt.AccessByRole(ui.RoleSwitch, "Bật"); a.Description != "Tắt khi rời.. Mở hay tắt." {
		t.Errorf("a tooltip set after Describe: description %q", a.Description)
	}
	ep := ErrorProvider()
	ep.SetError(tb, "Bắt buộc.")
	tt.Frame()
	got := descriptionOf(t, tt, "Tên")
	if !strings.HasPrefix(got, "Bắt buộc.") || !strings.Contains(got, nameDesc+". Họ và tên.") || strings.Count(got, "Họ và tên.") != 1 {
		t.Errorf("with an error the description is %q, want the error, then the row's description and the tooltip once", got)
	}
	ep.SetError(tb, "")
	tt.Frame()
	if got := descriptionOf(t, tt, "Tên"); got != nameDesc+". Họ và tên." {
		t.Errorf("the error gone, the description is %q", got)
	}
}

// Review Focus 1: a very long label beside a description: both wrap, and the
// control stays at the middle of the row, its height its own.
func TestDescribeNextToALongLabel(t *testing.T) {
	alone := NewTester(t, Fields().SettingsRows().Row("Name", TextBox()), 400, 300).Find(ByRole(ui.RoleTextField)).Bounds()
	long := strings.TrimSpace(strings.Repeat(nameDesc+" ", 3))
	tt := NewTester(t, Fields().SettingsRows().Row(longLine, TextBox()).Describe(long), 400, 400)
	l, d, c := find(t, tt.Tester, longLine), find(t, tt.Tester, long), tt.Find(ByRole(ui.RoleTextField)).Bounds()
	one := find(t, tester(t, Label("Name")), "Name").H
	if l.H < 2*one || d.H < 2*one {
		t.Errorf("label %v and description %v: want both wrapped", l, d)
	}
	if !approx(c.H, alone.H) {
		t.Errorf("the control is %v high, %v alone", c.H, alone.H)
	}
	if mid, cm := (l.Y+d.Y+d.H)/2, c.Y+c.H/2; !approx(mid, cm) {
		t.Errorf("the label and description's middle at %v, the control's at %v", mid, cm)
	}
	// Beside a control taller than both, the label and its description sit
	// at the control's middle together.
	tall := NewTester(t, Fields().SettingsRows().Row("Tên", wbar("c1", 60, 120)).Describe(nameDesc), 400, 400)
	l, d, c = find(t, tall.Tester, "Tên"), find(t, tall.Tester, nameDesc), find(t, tall.Tester, "c1")
	if mid, cm := (l.Y+d.Y+d.H)/2, c.Y+60; !approx(mid, cm) {
		t.Errorf("beside a tall control: the label and description's middle at %v, the control's at %v", mid, cm)
	}
}

// Review Focus 2: at text scale 1.5 the card grows, and the description is
// inside it, not clipped.
func TestDescribeAtTextScale(t *testing.T) {
	f := Fields().SettingsRows().Section("S").Row("Tên", TextBox()).Describe(nameDesc)
	tt := NewTester(t, f, 400, 400)
	before := cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)[0]
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	after := cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)[0]
	if after.bottom-after.top <= before.bottom-before.top {
		t.Errorf("the card is %d high at 1.5, %d at 1", after.bottom-after.top, before.bottom-before.top)
	}
	if d := find(t, tt.Tester, nameDesc); !after.inside(d) {
		t.Errorf("the description %v leaves the card %+v", d, after)
	}
}

// A row whose label is hidden draws no description either, in every layout,
// and its control keeps its column; screen readers still read it.
func TestDescribeWithAHiddenLabel(t *testing.T) {
	for _, mode := range []string{"default", "stacked", "settings"} {
		f := Fields().Row("Tên", TextBox()).Describe(nameDesc)
		switch mode {
		case "stacked":
			f.Stacked()
		case "settings":
			f.SettingsRows()
		}
		f.Label(0).SetVisible(false)
		tt := NewTester(t, f, 400, 300)
		if tt.HasText(nameDesc) {
			t.Errorf("%s: the description of a hidden label shows: %q", mode, tt.Texts())
		}
		if got := descriptionOf(t, tt, ""); got != nameDesc {
			t.Errorf("%s: the control's description is %q, want %q", mode, got, nameDesc)
		}
	}
}

// In the plain grid, whose label column fits its content, a long description
// widens that column as a label as long does: the control is placed as it is
// beside such a label (squeezed in a narrow form; SettingsRows or Stacked
// wrap instead).
func TestDescribeLongInThePlainGrid(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat(nameDesc+" ", 6))
	tt := NewTester(t, Fields().Row("Tên", TextBox()).Describe(long), 400, 400)
	c := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	tl := NewTester(t, Fields().Row(long, TextBox()), 400, 400)
	if l := tl.Find(ByRole(ui.RoleTextField)).Bounds(); !approx(c.X, l.X) || !approx(c.W, l.W) {
		t.Errorf("beside a long description the control is %v, beside a label as long %v", c, l)
	}
}

// Describe may be called from any goroutine while the form draws.
func TestDescribeFromGoroutines(t *testing.T) {
	f := Fields().SettingsRows().Row("Tên", TextBox())
	tt := NewTester(t, f, 400, 300)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() { f.Describe(nameDesc) })
	}
	tt.Frame()
	wg.Wait()
	tt.Frame()
	if got := descriptionOf(t, tt, "Tên"); got != nameDesc {
		t.Errorf("description %q", got)
	}
}
