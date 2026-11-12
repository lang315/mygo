package skin

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Calm is a calm light ground with the accent only on what acts or is
// chosen: the primary action, the selection, the open tab and checked
// controls; green, amber and red only in status. Controls are 36 DIPs
// high and table rows 30.
var Calm = &Skin{name: "Calm", light: calmLight, dark: calmDark}

// calmAccent is Calm's dark accent, which its translucent fills are made of.
var calmAccent = ui.Hex("#4a8bff")

var calmLight = skinhook.Tokens{
	Chrome:         ui.Hex("#f4f5f7"),
	Content:        ui.Hex("#ffffff"),
	Line:           ui.Hex("#e4e7eb"),
	LineStrong:     ui.Hex("#d5d9df"),
	Text:           ui.Hex("#1a1d21"),
	Text2:          ui.Hex("#565d68"),
	Text3:          ui.Hex("#6b7280"),
	Control:        ui.Hex("#ffffff"),
	ControlHover:   ui.Hex("#f3f5f8"),
	ControlPressed: ui.Hex("#e9ecf1"),
	ButtonLine:     ui.Hex("#d2d7de"),
	FieldLine:      ui.Hex("#838a94"),
	OnAccent:       ui.Hex("#ffffff"),
	Selection:      ui.Hex("#cfe0fb"),
	RowHover:       ui.Hex("#f5f7fa"),
	RowSelected:    ui.Hex("#e8f0fd"),
	Accent: skinhook.Family{Base: ui.Hex("#1a63db"), Hover: ui.Hex("#1555c2"), Pressed: ui.Hex("#1249a8"),
		Subtle: ui.Hex("#eaf1fd"), Border: ui.Hex("#b9d0f7"), Text: ui.Hex("#1450b8"), On: ui.Hex("#ffffff")},
	Success: skinhook.Family{Base: ui.Hex("#1e7b34"), Hover: ui.Hex("#196a2c"), Pressed: ui.Hex("#145a25"),
		Subtle: ui.Hex("#e6f5ea"), Border: ui.Hex("#b7e1c2"), Text: ui.Hex("#1d6b31"), On: ui.Hex("#ffffff")},
	Warning: skinhook.Family{Base: ui.Hex("#b45309"), Hover: ui.Hex("#9a4708"), Pressed: ui.Hex("#803b07"),
		Subtle: ui.Hex("#fdf1dc"), Border: ui.Hex("#f3d39a"), Text: ui.Hex("#7d4a00"), On: ui.Hex("#ffffff")},
	Danger: skinhook.Family{Base: ui.Hex("#c42b1c"), Hover: ui.Hex("#a82418"), Pressed: ui.Hex("#8c1e14"),
		Subtle: ui.Hex("#fcebea"), Border: ui.Hex("#f2b8b2"), Text: ui.Hex("#a8261b"), On: ui.Hex("#ffffff")},
	ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
	TabStripHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
	RadiusControl: 6, RadiusCard: 10, RadiusDialog: 12,
	ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
	Icon: 16, ToolbarIcon: 17, CheckBox: 15, SwitchWidth: 40, SwitchHeight: 22,
	Body:    skinhook.Type{Size: 13, Line: 20, Weight: 400},
	Caption: skinhook.Type{Size: 11, Line: 16, Weight: 400},
	Label:   skinhook.Type{Size: 12, Line: 16, Weight: 600},
	Heading: skinhook.Type{Size: 15, Line: 20, Weight: 600},
	Title:   skinhook.Type{Size: 20, Line: 28, Weight: 600},
	Elevation: [4]skinhook.Shadow{
		{},
		{Y: 1, Blur: 2, Color: ui.RGBA(0, 0, 0, 0.04)},
		{Y: 6, Blur: 20, Color: ui.RGBA(0, 0, 0, 0.18)},
		{Y: 12, Blur: 32, Spread: -8, Color: ui.RGBA(0, 0, 0, 0.18)},
	},
	Hover:  90 * time.Millisecond,
	Toggle: 120 * time.Millisecond,
}

var calmDark = skinhook.Tokens{
	Chrome:         ui.Hex("#1d1f23"),
	Content:        ui.Hex("#15171a"),
	Line:           ui.Hex("#2a2e34"),
	LineStrong:     ui.Hex("#353a42"),
	Text:           ui.Hex("#e7e9ec"),
	Text2:          ui.Hex("#a6adb7"),
	Text3:          ui.Hex("#8e96a1"),
	Control:        ui.Hex("#262a30"),
	ControlHover:   ui.Hex("#2d3239"),
	ControlPressed: ui.Hex("#343a42"),
	ButtonLine:     ui.Hex("#3a4048"),
	FieldLine:      ui.Hex("#79818c"),
	OnAccent:       ui.Hex("#0b1220"),
	Selection:      calmAccent.Alpha(0.35),
	RowHover:       ui.Hex("#1c1f23"),
	RowSelected:    calmAccent.Alpha(0.16),
	Accent: skinhook.Family{Base: calmAccent, Hover: ui.Hex("#6a9fff"), Pressed: ui.Hex("#3a78ea"),
		Subtle: calmAccent.Alpha(0.14), Border: calmAccent.Alpha(0.35), Text: ui.Hex("#8fb6ff"), On: ui.Hex("#0b1220")},
	Success: skinhook.Family{Base: ui.Hex("#3fb860"), Hover: ui.Hex("#52c771"), Pressed: ui.Hex("#2f9d4e"),
		Subtle: ui.Hex("#34c759").Alpha(0.14), Border: ui.Hex("#34c759").Alpha(0.35), Text: ui.Hex("#7fd99a"), On: ui.Hex("#0b1220")},
	Warning: skinhook.Family{Base: ui.Hex("#e8a33a"), Hover: ui.Hex("#f0b45a"), Pressed: ui.Hex("#cf8f2c"),
		Subtle: ui.Hex("#f5a623").Alpha(0.15), Border: ui.Hex("#f5a623").Alpha(0.35), Text: ui.Hex("#f5c271"), On: ui.Hex("#1a1205")},
	Danger: skinhook.Family{Base: ui.Hex("#ff6b5e"), Hover: ui.Hex("#ff8378"), Pressed: ui.Hex("#e85a4e"),
		Subtle: ui.Hex("#ff5c4d").Alpha(0.15), Border: ui.Hex("#ff5c4d").Alpha(0.35), Text: ui.Hex("#ff9a90"), On: ui.Hex("#1a0a08")},
	ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
	TabStripHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
	RadiusControl: 6, RadiusCard: 10, RadiusDialog: 12,
	ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
	Icon: 16, ToolbarIcon: 17, CheckBox: 15, SwitchWidth: 40, SwitchHeight: 22,
	Body:    skinhook.Type{Size: 13, Line: 20, Weight: 400},
	Caption: skinhook.Type{Size: 11, Line: 16, Weight: 400},
	Label:   skinhook.Type{Size: 12, Line: 16, Weight: 600},
	Heading: skinhook.Type{Size: 15, Line: 20, Weight: 600},
	Title:   skinhook.Type{Size: 20, Line: 28, Weight: 600},
	Elevation: [4]skinhook.Shadow{
		{},
		{},
		{Y: 6, Blur: 20, Color: ui.RGBA(0, 0, 0, 0.5)},
		{Y: 12, Blur: 32, Spread: -8, Color: ui.RGBA(0, 0, 0, 0.5)},
	},
	Hover:  90 * time.Millisecond,
	Toggle: 120 * time.Millisecond,
}
