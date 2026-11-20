package skin

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// themeOf returns the ui.Theme the tokens make in one appearance: the
// fields a theme has, from the tokens, and the rest from LightTheme or
// DarkTheme.
func themeOf(t *skinhook.Tokens, dark bool) *ui.Theme {
	th := ui.LightTheme()
	if dark {
		th = ui.DarkTheme()
	}
	th.Dark = dark
	th.Background = t.Background
	th.Surface = t.Surface
	th.SurfaceHover = t.SurfaceHover
	th.SurfacePressed = t.SurfacePressed
	th.Field = t.Surface
	th.Border = t.Border
	th.ControlBorder = t.ControlBorder
	th.Text = t.Text
	th.TextMuted = t.TextMuted
	th.Accent, th.AccentHover, th.AccentPressed = t.Accent.Base, t.Accent.Hover, t.Accent.Pressed
	th.AccentText = t.Accent.On
	th.Danger = t.Danger.Base
	th.Warning = t.Warning.Base
	th.Success = t.Success.Base
	th.Selection = t.Selection
	th.Focus = t.Accent.Base
	th.Radius = t.RadiusControl
	th.FontSize = t.FontSize
	th.Font = t.Font
	th.Metrics = ui.Metrics{ControlHeight: t.ControlHeight, RowHeight: t.RowHeight, HeaderHeight: t.HeaderHeight,
		TabHeight: t.TabHeight, CheckSize: t.CheckSize, SwitchWidth: t.SwitchWidth, SwitchHeight: t.SwitchHeight}
	th.ButtonBorder = t.ButtonBorder
	th.RowHover, th.RowSelected, th.RowSelectedText = t.RowHover, t.RowSelected, rowText(t)
	th.Scrollbar = t.Text.Alpha(0.32)
	if dark {
		th.Scrollbar = t.Text.Alpha(0.35)
	}
	th.Inverse, th.InverseText = t.Text, t.Background
	th.FocusRingWidth, th.FocusRingOffset = t.FocusRingWidth, t.FocusRingOffset
	th.ScrollbarWidth, th.ScrollbarStyle = t.ScrollbarWidth, t.ScrollbarStyle
	return th
}

// rowText is the text of a selected row: Text, or Background where that
// reads better on RowSelected, as on a skin whose selected rows are
// inverted (HighContrast's).
func rowText(t *skinhook.Tokens) ui.Color {
	if ratio(t, t.Background, t.RowSelected) > ratio(t, t.Text, t.RowSelected) {
		return t.Background
	}
	return t.Text
}
