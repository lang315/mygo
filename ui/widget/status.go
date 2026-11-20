package widget

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Status is the state a badge or an info bar shows: neutral, information,
// work running, success, a warning or a danger. A skin colors each from its
// family (Info, Success, Warning, Danger); without one, the theme's colors
// tint the fill and the text stays the theme's Text.
type Status uint8

const (
	StatusNone    Status = iota // neutral: muted text on the hovered face; no skin: as ui.Badge
	StatusInfo                  // Info's colors
	StatusRunning               // Info's colors, with a dot before a badge's text
	StatusSuccess
	StatusWarning
	StatusDanger
)

var statusNames = [...]string{"none", "info", "running", "success", "warning", "danger"}

// String returns the status's name in lower case: "none", "info",
// "running", "success", "warning" or "danger".
func (s Status) String() string {
	if int(s) < len(statusNames) {
		return statusNames[s]
	}
	return "Status(" + strconv.Itoa(int(s)) + ")"
}

// styling is the status of the control being built or styled, which
// RoleStatus, RoleStatusText and RoleStatusBorder resolve to: render sets it
// from the control's core and restores the outer one after. Under mu.
var styling Status

// statusColors returns the colors of s under c's theme: its fill, its text,
// its border (zero for none) and its base, the color of a running dot or an
// info bar's icon.
func statusColors(c *ui.Context, s Status) (fill, text, border, base ui.Color) {
	return statusColorsOf(c.Theme(), s)
}

// statusColorsOf is statusColors under the theme t.
func statusColorsOf(t *ui.Theme, s Status) (fill, text, border, base ui.Color) {
	if tk := skinhook.Get(t.Dark); tk != nil {
		var f skinhook.Family
		switch s {
		case StatusInfo, StatusRunning:
			f = tk.Info
		case StatusSuccess:
			f = tk.Success
		case StatusWarning:
			f = tk.Warning
		case StatusDanger:
			f = tk.Danger
		default:
			return tk.SurfaceHover, tk.TextMuted, tk.Border, tk.TextMuted
		}
		return f.Subtle, f.Text, f.Border, f.Base
	}
	switch s {
	case StatusInfo, StatusRunning:
		base = t.Accent
	case StatusSuccess:
		base = t.Success
	case StatusWarning:
		base = t.Warning
	case StatusDanger:
		base = t.Danger
	default:
		return t.Text.Alpha(0.1), t.Text, ui.Color{}, t.TextMuted
	}
	return base.Alpha(0.14), t.Text, ui.Color{}, base
}
