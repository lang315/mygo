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
