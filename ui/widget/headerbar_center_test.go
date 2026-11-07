package widget

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// barTester shows a form with a header bar in a window width wide whose
// title bar leaves the room bar to the platform's controls.
func barTester(t *testing.T, width int, bar ui.TitleBar, title, subtitle string, start, end []Control) *ui.Tester {
	t.Helper()
	f := NewForm(title, width, 300)
	f.UseHeaderBar(subtitle).PackStart(start...).PackEnd(end...)
	tt := ui.NewTester(f.view, width, 300)
	tt.SetTitleBar(bar)
	tt.Frame()
	return tt
}

func centreX(r ui.Rect) float32 { return r.X + r.W/2 }

var (
	windowsBar = ui.TitleBar{Height: 32, Right: 138}
	macBar     = ui.TitleBar{Height: 32, Left: 70}
)

func TestHeaderBarTitleIsCentredOnTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bar        ui.TitleBar
		start, end []Control
	}{
		{"windows, nothing packed", windowsBar, nil, nil},
		{"mac, nothing packed", macBar, nil, nil},
		{"windows, unequal items", windowsBar, []Control{Label("s")}, []Control{Label("end one"), Label("end two")}},
		{"mac, unequal items", macBar, []Control{Label("start one"), Label("start two")}, []Control{Label("e")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := barTester(t, 800, tc.bar, "Title", "Subtitle", tc.start, tc.end)
			for _, s := range []string{"Title", "Subtitle"} {
				if c := centreX(find(t, tt, s)); c < 399 || c > 401 {
					t.Errorf("%s centre %v, window centre 400", s, c)
				}
			}
		})
	}
}

func overlaps(a, b ui.Rect) bool { return a.X < b.X+b.W && b.X < a.X+a.W }

// A title that would run under the items or the buttons if centred moves
// aside, ellipsized, and keeps clear of both.
func TestHeaderBarLongTitleFallsBackWithoutOverlap(t *testing.T) {
	long := strings.Repeat("a very long title ", 12)
	for _, tc := range []struct {
		name  string
		bar   ui.TitleBar
		width int
	}{
		{"windows", windowsBar, 800},
		{"mac", macBar, 800},
		{"narrow window", windowsBar, 260},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := barTester(t, tc.width, tc.bar, long, "", []Control{Label("start")}, []Control{Label("end")})
			title := ui.Rect{}
			for _, s := range tt.Texts() {
				if strings.HasPrefix(s, "a very long") {
					title = find(t, tt, s)
				}
			}
			if title.W == 0 {
				t.Fatalf("no title in %q", tt.Texts())
			}
			start, end := find(t, tt, "start"), find(t, tt, "end")
			if overlaps(title, start) || overlaps(title, end) {
				t.Errorf("title %v overlaps start %v or end %v", title, start, end)
			}
			if title.X < tc.bar.Left || title.X+title.W > float32(tc.width)-tc.bar.Right {
				t.Errorf("title %v runs under the window's buttons (insets %v)", title, tc.bar)
			}
		})
	}
}

// nodeBox returns the box of the control labelled s.
func nodeBox(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Label == s {
			return n.Bounds
		}
	}
	t.Fatalf("no node %q", s)
	return ui.Rect{}
}

// Buttons wider than a Label's minimum must not end up under the title, at
// any width.
func TestHeaderBarTitleNeverOverlapsButtons(t *testing.T) {
	names := []string{"Alpha", "Bravo", "Charlie", "Z"}
	for _, bar := range []ui.TitleBar{windowsBar, macBar, {Height: 32}} {
		for w := 900; w >= 200; w -= 4 {
			tt := barTester(t, w, bar, "Document Title Here.txt", "",
				[]Control{Button("Alpha"), Button("Bravo"), Button("Charlie")}, []Control{Button("Z")})
			title := find(t, tt, "Document Title Here.txt")
			for _, s := range names {
				if b := nodeBox(t, tt, s); b.W > 0 && overlaps(title, b) {
					t.Fatalf("insets %v, width %d: title %v overlaps %s %v", bar, w, title, s, b)
				}
			}
		}
	}
}

// The end items keep their order and sit against the right edge; the
// start items against the left; in a right-to-left window the other way.
func TestHeaderBarItemsKeepTheirSidesAndOrder(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		for _, bar := range []ui.TitleBar{windowsBar, macBar} {
			f := NewForm("Title", 800, 300)
			f.UseHeaderBar("").PackStart(Button("S1"), Button("S2")).PackEnd(Button("E1"), Button("E2"))
			tt := ui.NewTester(f.view, 800, 300)
			tt.SetTitleBar(bar)
			if rtl {
				tt.SetDirection(ui.RightToLeft)
			}
			tt.Frame()
			s1, s2, e1, e2 := nodeBox(t, tt, "S1"), nodeBox(t, tt, "S2"), nodeBox(t, tt, "E1"), nodeBox(t, tt, "E2")
			// The first packed item is at the outer edge, on each side.
			if rtl {
				if !(s1.X > s2.X && e1.X < e2.X) {
					t.Errorf("rtl %v: order S1 %v S2 %v, E1 %v E2 %v", bar, s1, s2, e1, e2)
				}
				if right := s1.X + s1.W; right > 800-bar.Right || right < 800-bar.Right-20 {
					t.Errorf("rtl %v: start items begin at %v, not at the right edge %v", bar, right, 800-bar.Right)
				}
				if e1.X < bar.Left || e1.X > bar.Left+20 {
					t.Errorf("rtl %v: end items end at %v, not at the left edge", bar, e1.X)
				}
				continue
			}
			if !(s1.X < s2.X && e2.X < e1.X) {
				t.Errorf("%v: order S1 %v S2 %v, E1 %v E2 %v", bar, s1, s2, e1, e2)
			}
			if right := e1.X + e1.W; right > 800-bar.Right || right < 800-bar.Right-20 {
				t.Errorf("%v: end items end at %v, not at the right edge %v", bar, right, 800-bar.Right)
			}
			if s1.X < bar.Left || s1.X > bar.Left+20 {
				t.Errorf("%v: start items begin at %v", bar, s1.X)
			}
		}
	}
}

func TestHeaderBarTitleIsCentredInARightToLeftWindow(t *testing.T) {
	for _, bar := range []ui.TitleBar{windowsBar, macBar} {
		f := NewForm("Title", 800, 300)
		f.UseHeaderBar("Subtitle").PackStart(Button("S1"), Button("S2")).PackEnd(Button("E1"), Button("E2"))
		tt := ui.NewTester(f.view, 800, 300)
		tt.SetTitleBar(bar)
		tt.SetDirection(ui.RightToLeft)
		tt.Frame()
		if c := centreX(find(t, tt, "Title")); c < 399 || c > 401 {
			t.Errorf("insets %v: title centre %v, window centre 400", bar, c)
		}
	}
}
