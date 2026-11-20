package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

func TestStatusString(t *testing.T) {
	for s, want := range map[Status]string{
		StatusNone: "none", StatusInfo: "info", StatusRunning: "running", StatusSuccess: "success",
		StatusWarning: "warning", StatusDanger: "danger", Status(6): "Status(6)", Status(42): "Status(42)",
	} {
		if got := s.String(); got != want {
			t.Errorf("Status(%d).String() = %q, want %q", uint8(s), got, want)
		}
	}
}

// Case 1, its colors: under Calm light a Success status fills with Success's
// Subtle and writes in Success's Text, and its border is Success's Border.
func TestStatusColorsCalm(t *testing.T) {
	useCalm(t, true)
	t.Cleanup(calmOff)
	for _, dark := range []bool{false, true} {
		tk := skinhook.Get(dark)
		th := ui.LightTheme()
		if dark {
			th = ui.DarkTheme()
		}
		for s, f := range map[Status]skinhook.Family{
			StatusInfo: tk.Info, StatusRunning: tk.Info, StatusSuccess: tk.Success,
			StatusWarning: tk.Warning, StatusDanger: tk.Danger,
		} {
			fill, text, border, base := statusColorsOf(th, s)
			if fill != f.Subtle || text != f.Text || border != f.Border || base != f.Base {
				t.Errorf("dark %v, %v: %v %v %v %v, want the family's Subtle %v, Text %v, Border %v, Base %v",
					dark, s, fill, text, border, base, f.Subtle, f.Text, f.Border, f.Base)
			}
		}
		// Case 4: StatusNone is TextMuted on SurfaceHover, in the skin's Border.
		fill, text, border, _ := statusColorsOf(th, StatusNone)
		if fill != tk.SurfaceHover || text != tk.TextMuted || border != tk.Border {
			t.Errorf("dark %v, none: %v %v %v, want SurfaceHover %v, TextMuted %v, Border %v",
				dark, fill, text, border, tk.SurfaceHover, tk.TextMuted, tk.Border)
		}
	}
}

// Cases 2 and 4 without a skin: the status's theme color at alpha 0.14 (the
// accent for Info and Running), the theme's Text, no border; StatusNone as
// ui.Badge, Text at alpha 0.1.
func TestStatusColorsPlain(t *testing.T) {
	calmOff()
	for _, th := range []*ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		for s, c := range map[Status]ui.Color{
			StatusInfo: th.Accent, StatusRunning: th.Accent, StatusSuccess: th.Success,
			StatusWarning: th.Warning, StatusDanger: th.Danger, StatusNone: th.Text,
		} {
			a := uint8(36) // 0.14 of 255, rounded
			if s == StatusNone {
				a = 26 // 0.1
			}
			fill, text, border, _ := statusColorsOf(th, s)
			if fill.R != c.R || fill.G != c.G || fill.B != c.B || fill.A != a {
				t.Errorf("dark %v, %v: fill %v, want %v at alpha %d", th.Dark, s, fill, c, a)
			}
			if text != th.Text || border != (ui.Color{}) {
				t.Errorf("dark %v, %v: text %v and border %v, want Text %v and none", th.Dark, s, text, border, th.Text)
			}
		}
	}
}

// The status roles resolve to the status of the control being styled, and
// to StatusNone's outside one.
func TestStatusRoles(t *testing.T) {
	tk := useSkin(t)
	tk.Warning = skinhook.Family{Subtle: ui.Hex("#fff3cd"), Text: ui.Hex("#6b4a00"), Border: ui.Hex("#e0b040"), Base: ui.Hex("#d08000")}
	tk.TextMuted = ui.Hex("#606060")
	th := ui.LightTheme()
	mu.Lock()
	defer mu.Unlock()
	if got := RoleStatus.resolve(th); got != tk.SurfaceHover {
		t.Errorf("RoleStatus outside a status control is %v, want SurfaceHover %v", got, tk.SurfaceHover)
	}
	if got := RoleStatusText.resolve(th); got != tk.TextMuted {
		t.Errorf("RoleStatusText outside a status control is %v, want TextMuted %v", got, tk.TextMuted)
	}
	if got := RoleStatusBorder.resolve(th); got != tk.Border {
		t.Errorf("RoleStatusBorder outside a status control is %v, want Border %v", got, tk.Border)
	}
	styling = StatusWarning
	defer func() { styling = StatusNone }()
	if got := RoleStatus.resolve(th); got != ui.Hex("#fff3cd") {
		t.Errorf("RoleStatus in a Warning control is %v, want #fff3cd", got)
	}
	if got := RoleStatusText.resolve(th); got != ui.Hex("#6b4a00") {
		t.Errorf("RoleStatusText in a Warning control is %v, want #6b4a00", got)
	}
	if got := RoleStatusBorder.resolve(th); got != ui.Hex("#e0b040") {
		t.Errorf("RoleStatusBorder in a Warning control is %v, want #e0b040", got)
	}
}
