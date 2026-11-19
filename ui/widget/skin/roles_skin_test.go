package skin

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// skinRole is a role a skin decides: its widget color, the token it reads,
// and its color in the plain light theme, which sets no ButtonBorder,
// RowHover or RowSelected.
type skinRole struct {
	name  string
	color widget.Color
	token func(*Tokens) ui.Color
	plain ui.Color
}

var skinRoles = []skinRole{
	{"chrome", widget.RoleChrome, func(t *Tokens) ui.Color { return t.Chrome }, ui.Hex("#f4f4f5")},                    // Surface
	{"border-strong", widget.RoleBorderStrong, func(t *Tokens) ui.Color { return t.BorderStrong }, ui.Hex("#d9d9de")}, // Border
	{"control-border", widget.RoleControlBorder, func(t *Tokens) ui.Color { return t.ControlBorder }, ui.Hex("#8a8a93")},
	{"button-border", widget.RoleButtonBorder, func(t *Tokens) ui.Color { return t.ButtonBorder }, ui.Hex("#8a8a93")}, // ControlBorder
	{"row-hover", widget.RoleRowHover, func(t *Tokens) ui.Color { return t.RowHover }, ui.Hex("#e9e9ec")},             // SurfaceHover
	{"row-selected", widget.RoleRowSelected, func(t *Tokens) ui.Color { return t.RowSelected }, ui.Hex("#2563eb")},    // Accent
	{"info", widget.RoleInfo, func(t *Tokens) ui.Color { return t.Info.Base }, ui.Hex("#2563eb")},                     // Accent
}

// roleColorSentinel is Calm with its own color, light and dark apart, for
// each token a role reads.
func roleColorSentinel() *Skin {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	l.Chrome, d.Chrome = ui.Hex("#a10001"), ui.Hex("#b10001")
	l.BorderStrong, d.BorderStrong = ui.Hex("#a20002"), ui.Hex("#b20002")
	l.ControlBorder, d.ControlBorder = ui.Hex("#a30003"), ui.Hex("#b30003")
	l.ButtonBorder, d.ButtonBorder = ui.Hex("#a40004"), ui.Hex("#b40004")
	l.RowHover, d.RowHover = ui.Hex("#a50005"), ui.Hex("#b50005")
	l.RowSelected, d.RowSelected = ui.Hex("#a60006"), ui.Hex("#b60006")
	l.Info.Base, d.Info.Base = ui.Hex("#a70007"), ui.Hex("#b70007")
	return New("RoleColors", l, d)
}

// roleLabels is a label per role, its background the role, padded so that
// a pixel inside the padding is the background.
func roleLabels() widget.Control {
	box := widget.VBox(4)
	for _, r := range skinRoles {
		box.Add(widget.Label(r.name).Align(widget.Start).Apply(widget.NewStyle().Padding(4).Background(r.color)))
	}
	return box
}

// backgroundOf is the color in the padding of the label text: a corner of its
// background.
func backgroundOf(t *testing.T, tt *widget.Tester, text string) ui.Color {
	t.Helper()
	b := tt.Find(widget.ByLabel(text)).Bounds()
	return colorAt(tt.Image(), int(b.X)+1, int(b.Y)+1)
}

// Under a skin, each role paints its token, light and dark.
func TestSkinRolesResolve(t *testing.T) {
	s := roleColorSentinel()
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tt := showSkin(t, roleLabels(), 300, 400, s, dark, 1)
			tk := s.Tokens(dark)
			for _, r := range skinRoles {
				if got, want := backgroundOf(t, tt, r.name), r.token(&tk); got != want {
					t.Errorf("%s paints %v, want the token's %v", r.name, got, want)
				}
			}
		})
	}
}

// After Use(nil), each role paints the plain theme's nearest field, never
// nothing; a theme that sets ButtonBorder, RowHover and RowSelected, and no
// ControlBorder, gives those.
func TestSkinRolesWithoutSkin(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		tt := showSkin(t, roleLabels(), 300, 400, roleColorSentinel(), false, 1)
		Use(nil)
		tt.Frame()
		for _, r := range skinRoles {
			if got := backgroundOf(t, tt, r.name); got != r.plain {
				t.Errorf("%s paints %v, want the plain %v", r.name, got, r.plain)
			}
		}
	})
	t.Run("theme fields", func(t *testing.T) {
		tt := showSkin(t, roleLabels(), 300, 400, roleColorSentinel(), false, 1)
		Use(nil)
		th := ui.LightTheme()
		th.ControlBorder = ui.Color{}
		th.ButtonBorder, th.RowHover, th.RowSelected = ui.Hex("#c10011"), ui.Hex("#c20012"), ui.Hex("#c30013")
		ui.SetDefaultThemes(th, nil)
		t.Cleanup(func() { ui.SetDefaultThemes(nil, nil) })
		tt.Frame()
		want := map[string]ui.Color{
			"chrome": ui.Hex("#f4f4f5"), "border-strong": ui.Hex("#d9d9de"),
			"control-border": ui.Hex("#d9d9de"), // no ControlBorder: Border
			"button-border":  ui.Hex("#c10011"), "row-hover": ui.Hex("#c20012"), "row-selected": ui.Hex("#c30013"),
			"info": ui.Hex("#2563eb"),
		}
		for _, r := range skinRoles {
			if got := backgroundOf(t, tt, r.name); got != want[r.name] {
				t.Errorf("%s paints %v, want %v", r.name, got, want[r.name])
			}
		}
		// ButtonBorder unset, ControlBorder unset: Border.
		th.ButtonBorder = ui.Color{}
		ui.SetDefaultThemes(th, nil)
		tt.Frame()
		if got := backgroundOf(t, tt, "button-border"); got != ui.Hex("#d9d9de") {
			t.Errorf("button-border with neither border set paints %v, want Border #d9d9de", got)
		}
	})
}

type chromeRow struct{ ID, Name string }

// A grid styled with RoleChrome, whose rows are kept from frame to frame,
// paints the new skin's Chrome under every row in the frame after Use.
//
// It does not hold a role read inside a kept row: the grid's style resolves
// in its own render every frame, and no widget API styles a cell yet. That
// case comes with S2b's GridHeader look (ui.ListState.HeaderCell).
func TestSkinRoleSwitchRepaints(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	skinWith := func(name string, chrome ui.Color) *Skin {
		l, d := Calm.Tokens(false), Calm.Tokens(true)
		l.Chrome, d.Chrome = chrome, chrome
		return New(name, l, d)
	}
	a, b := skinWith("A", ui.Hex("#a10001")), skinWith("B", ui.Hex("#0001b1"))
	var builds atomic.Int32
	grid := widget.DataGridView(func(r *chromeRow) string { return r.ID },
		widget.Column[chromeRow]{Title: "Name", Text: func(r *chromeRow) string { builds.Add(1); return r.Name }},
	).SetRows([]chromeRow{{"1", "Ada"}, {"2", "Grace"}, {"3", "Edsger"}}).SetSize(0, 180)
	grid.Apply(widget.NewStyle().Background(widget.RoleChrome))
	note := widget.Label("one")
	tt := widget.NewTester(t, widget.VBox(8).Add(note, grid), 600, 400)
	Use(a)
	tt.Frame()
	builds.Store(0)
	note.SetText("two")
	tt.Frame()
	if n := builds.Load(); n != 0 {
		t.Fatalf("a frame that changed only a label built %d cells: the grid keeps no rows", n)
	}
	rowsPaint := func(when string, want ui.Color) {
		t.Helper()
		seen := 0
		for _, n := range tt.Nodes() {
			if n.Role != ui.RoleRow || (n.Label != "Ada" && n.Label != "Grace" && n.Label != "Edsger") {
				continue
			}
			seen++
			r := n.Bounds
			if got := colorAt(tt.Image(), int(r.X+r.W)-8, int(r.Y+r.H/2)); got != want {
				t.Errorf("%s, row %q paints %v, want %v", when, n.Label, got, want)
			}
		}
		if seen != 3 {
			t.Fatalf("%s: %d rows, want 3", when, seen)
		}
	}
	rowsPaint("under A", ui.Hex("#a10001"))
	Use(b)
	tt.Frame()
	rowsPaint("one frame after Use(B)", ui.Hex("#0001b1"))
}
