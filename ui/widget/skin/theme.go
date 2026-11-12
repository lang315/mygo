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
	th.Background = t.Content
	th.Surface = t.Control
	th.SurfaceHover = t.ControlHover
	th.SurfacePressed = t.ControlPressed
	th.Field = t.Control
	th.Border = t.Line
	th.ControlBorder = t.FieldLine
	th.Text = t.Text
	th.TextMuted = t.Text2
	th.Accent, th.AccentHover, th.AccentPressed = t.Accent.Base, t.Accent.Hover, t.Accent.Pressed
	th.AccentText = t.OnAccent
	th.Danger = t.Danger.Base
	th.Warning = t.Warning.Base
	th.Success = t.Success.Base
	th.Selection = t.Selection
	th.Focus = t.Accent.Base
	th.Radius = t.RadiusControl
	th.FontSize = t.Body.Size
	return th
}
