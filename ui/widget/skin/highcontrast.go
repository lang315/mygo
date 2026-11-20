package skin

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// HighContrast is for users who need text to stand out, without the
// system's own High Contrast (which still applies over any skin): text at
// 7:1 or more, no muted text, every control outlined 2 DIPs wide, a 3 DIP
// focus ring and no shadows. Dark is black and white with a yellow accent,
// light white and black with a navy one, the sizes Calm's. A selected row is
// inverted (black text on cyan in dark, white on purple in light); selected
// text in an input is a tint of those colors the text reads on, not
// inverted. It has no palettes: WithPalette and WithAccent still work, and
// Check reports what falls under 7:1.
var HighContrast = &Skin{name: "HighContrast", lazy: &lazySkin{build: func() *Skin {
	return &Skin{name: "HighContrast", light: highContrastLight(), dark: highContrastDark(), floor: 7,
		looks: highContrastLooks, parsed: parseLooks(highContrastLooks)}
}}}

// highContrastLooks outline every control 2 DIPs wide. A check box's and a
// radio button's look would box their text, so they keep ui's outline.
var highContrastLooks = Looks{
	Button:     "border-2 border-button",
	ToolButton: "border-2 border-button",
	Field:      "border-2 border-control",
	ComboBox:   "border-2 border-control",
	GroupBox:   "border-2 border-strong",
	Card:       "border-2 border-strong",
}

// highContrastDark and highContrastLight are HighContrast's tokens, made
// when it is first used.
func highContrastDark() skinhook.Tokens {
	return skinhook.Tokens{
		Chrome:          ui.Hex("#000000"),
		Background:      ui.Hex("#000000"),
		Border:          ui.Hex("#ffffff"),
		BorderStrong:    ui.Hex("#ffffff"),
		Text:            ui.Hex("#ffffff"),
		TextMuted:       ui.Hex("#ffffff"),
		TextPlaceholder: ui.Hex("#c8c8c8"),
		Surface:         ui.Hex("#000000"),
		SurfaceHover:    ui.Hex("#262626"),
		SurfacePressed:  ui.Hex("#404040"),
		ButtonBorder:    ui.Hex("#ffffff"),
		ControlBorder:   ui.Hex("#ffffff"),
		Selection:       ui.Hex("#095259"), // #1aebff at 35% over black, opaque: it shows on the cyan row too
		RowHover:        ui.Hex("#1a1a1a"),
		RowSelected:     ui.Hex("#1aebff"),
		Accent:          DarkFamily(ui.Hex("#ffff00"), ui.Hex("#ffff66"), ui.Hex("#e6e600"), ui.Hex("#ffff00"), ui.Hex("#000000")),
		Info:            DarkFamily(ui.Hex("#1aebff"), ui.Hex("#66f2ff"), ui.Hex("#00d0e6"), ui.Hex("#1aebff"), ui.Hex("#000000")),
		Success:         DarkFamily(ui.Hex("#3ff23f"), ui.Hex("#6ff56f"), ui.Hex("#2fd02f"), ui.Hex("#3ff23f"), ui.Hex("#000000")),
		Warning:         DarkFamily(ui.Hex("#ffff00"), ui.Hex("#ffff66"), ui.Hex("#e6e600"), ui.Hex("#ffff00"), ui.Hex("#000000")),
		Danger:          DarkFamily(ui.Hex("#ff6b6b"), ui.Hex("#ff8f8f"), ui.Hex("#ff4d4d"), ui.Hex("#ff8f8f"), ui.Hex("#000000")),
		Chart: [8]ui.Color{ui.Hex("#60a5fa"), ui.Hex("#4ade80"), ui.Hex("#fb923c"), ui.Hex("#c084fc"),
			ui.Hex("#22d3ee"), ui.Hex("#f472b6"), ui.Hex("#facc15"), ui.Hex("#9ca3af")},
		ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
		TabHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
		RadiusControl: 6, RadiusCard: 10,
		ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
		Icon: 16, ToolbarIcon: 17, CheckSize: 15, SwitchWidth: 40, SwitchHeight: 22,
		FontSize:       13,
		Caption:        skinhook.Type{Size: 11, Weight: 400},
		Label:          skinhook.Type{Size: 12, Weight: 600},
		Heading:        skinhook.Type{Size: 15, Weight: 600},
		Title:          skinhook.Type{Size: 20, Weight: 600},
		FocusRingWidth: 3, FocusRingOffset: 1,
	}
}

func highContrastLight() skinhook.Tokens {
	return skinhook.Tokens{
		Chrome:          ui.Hex("#ffffff"),
		Background:      ui.Hex("#ffffff"),
		Border:          ui.Hex("#000000"),
		BorderStrong:    ui.Hex("#000000"),
		Text:            ui.Hex("#000000"),
		TextMuted:       ui.Hex("#000000"),
		TextPlaceholder: ui.Hex("#4d4d4d"),
		Surface:         ui.Hex("#ffffff"),
		SurfaceHover:    ui.Hex("#e6e6e6"),
		SurfacePressed:  ui.Hex("#cccccc"),
		ButtonBorder:    ui.Hex("#000000"),
		ControlBorder:   ui.Hex("#000000"),
		Selection:       ui.Hex("#b9a6cc"), // #37006e at 35% over white, opaque: it shows on the purple row too
		RowHover:        ui.Hex("#f0f0f0"),
		RowSelected:     ui.Hex("#37006e"),
		Accent:          LightFamily(ui.Hex("#00009f"), ui.Hex("#0000c8"), ui.Hex("#000080"), ui.Hex("#00009f")),
		Info:            LightFamily(ui.Hex("#00009f"), ui.Hex("#0000c8"), ui.Hex("#000080"), ui.Hex("#00009f")),
		Success:         LightFamily(ui.Hex("#005a00"), ui.Hex("#004a00"), ui.Hex("#003a00"), ui.Hex("#005a00")),
		Warning:         LightFamily(ui.Hex("#6b4100"), ui.Hex("#5a3700"), ui.Hex("#4a2d00"), ui.Hex("#6b4100")),
		Danger:          LightFamily(ui.Hex("#a00000"), ui.Hex("#880000"), ui.Hex("#700000"), ui.Hex("#a00000")),
		Chart: [8]ui.Color{ui.Hex("#2563eb"), ui.Hex("#16a34a"), ui.Hex("#ea580c"), ui.Hex("#9333ea"),
			ui.Hex("#0891b2"), ui.Hex("#db2777"), ui.Hex("#a16207"), ui.Hex("#4b5563")},
		ControlHeight: 36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
		TabHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
		RadiusControl: 6, RadiusCard: 10,
		ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
		Icon: 16, ToolbarIcon: 17, CheckSize: 15, SwitchWidth: 40, SwitchHeight: 22,
		FontSize:       13,
		Caption:        skinhook.Type{Size: 11, Weight: 400},
		Label:          skinhook.Type{Size: 12, Weight: 600},
		Heading:        skinhook.Type{Size: 15, Weight: 600},
		Title:          skinhook.Type{Size: 20, Weight: 600},
		FocusRingWidth: 3, FocusRingOffset: 1,
	}
}
