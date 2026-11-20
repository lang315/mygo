package skin

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Classic is for screens full of data and users who know WinForms: small
// corners (2 on controls, 3 on cards), outlined buttons and group boxes,
// boxed tabs whose open one joins its page, grid headers filled with the
// chrome and divided by lines, check boxes of 14, flat tool buttons
// outlined on hover, and no shadows but under drop-downs. Its grays are a
// little cooler than Calm's. It is Compact by default: controls 30 DIPs
// high and rows 26. Its palette is ClassicIndigo; Palettes lists
// ClassicSlate too.
var Classic = (&Skin{name: "Classic", light: classicLight, dark: classicDark, density: Compact,
	palettes: []*Palette{ClassicIndigo, ClassicSlate}}).WithPalette(ClassicIndigo).WithLooks(classicLooks)

// Classic's palettes: Indigo, a deep indigo of its own, less violet than
// Calm's; Slate, a blue-gray, whose Info is Teal's blue, since a gray sits
// too near the status colors for a red-green color-blind eye.
var (
	ClassicIndigo = NewPalette("Indigo",
		LightFamily(ui.Hex("#3949ab"), ui.Hex("#303f9f"), ui.Hex("#283593"), ui.Hex("#303f9f")),
		DarkFamily(ui.Hex("#8c9eff"), ui.Hex("#a3b1ff"), ui.Hex("#7986f0"), ui.Hex("#b8c3ff"), ui.Hex("#0b0f2a")))
	ClassicSlate = NewPalette("Slate",
		LightFamily(ui.Hex("#44607a"), ui.Hex("#3a5268"), ui.Hex("#31465a"), ui.Hex("#3a5268")),
		DarkFamily(ui.Hex("#8fa8c0"), ui.Hex("#a3b8cc"), ui.Hex("#7c95ad"), ui.Hex("#b5c7d8"), ui.Hex("#0b1220"))).
		withInfo(Teal.light.info, Teal.dark.info)
)

var classicLooks = Looks{
	Button:     "border border-button rounded-sm",
	ToolButton: "border border-transparent hover:border-button",
	GroupBox:   "border border-strong rounded-sm",
	TabStrip:   "bg-chrome",
	TabHeader:  "border border-b-0 border-strong px-3 bg-chrome text-muted selected:bg-default selected:text-default",
	GridHeader: "bg-chrome border-r border-strong font-semibold",
}

var classicLight = skinhook.Tokens{
	Chrome:          ui.Hex("#eef0f3"),
	Background:      ui.Hex("#ffffff"),
	Border:          ui.Hex("#d0d4da"),
	BorderStrong:    ui.Hex("#aab0b9"),
	Text:            ui.Hex("#1a1d21"),
	TextMuted:       ui.Hex("#545b66"),
	TextPlaceholder: ui.Hex("#6b7280"),
	Surface:         ui.Hex("#ffffff"),
	SurfaceHover:    ui.Hex("#e2e6eb"),
	SurfacePressed:  ui.Hex("#d5dae1"),
	ButtonBorder:    ui.Hex("#aab0b9"),
	ControlBorder:   ui.Hex("#7d8590"),
	RowHover:        ui.Hex("#f3f5f8"),
	Success:         calmLight.Success,
	Warning:         calmLight.Warning,
	Danger:          calmLight.Danger,
	Chart:           calmLight.Chart,
	ControlHeight:   36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
	TabHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
	RadiusControl: 2, RadiusCard: 3,
	ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
	Icon: 16, ToolbarIcon: 17, CheckSize: 14, SwitchWidth: 40, SwitchHeight: 22,
	FontSize:       13,
	Caption:        skinhook.Type{Size: 11, Weight: 400},
	Label:          skinhook.Type{Size: 12, Weight: 600},
	Heading:        skinhook.Type{Size: 15, Weight: 600},
	Title:          skinhook.Type{Size: 20, Weight: 600},
	PopupShadow:    skinhook.Shadow{Y: 6, Blur: 20, Color: ui.RGBA(0, 0, 0, 0.18)},
	FocusRingWidth: 2, FocusRingOffset: 1,
}

var classicDark = skinhook.Tokens{
	Chrome:          ui.Hex("#1c1f24"),
	Background:      ui.Hex("#14171b"),
	Border:          ui.Hex("#2b3038"),
	BorderStrong:    ui.Hex("#4a515c"),
	Text:            ui.Hex("#e7e9ec"),
	TextMuted:       ui.Hex("#a6adb7"),
	TextPlaceholder: ui.Hex("#8e96a1"),
	Surface:         ui.Hex("#252a31"),
	SurfaceHover:    ui.Hex("#2c323a"),
	SurfacePressed:  ui.Hex("#343b44"),
	ButtonBorder:    ui.Hex("#4a515c"),
	ControlBorder:   ui.Hex("#7a8390"),
	RowHover:        ui.Hex("#1b1f24"),
	Success:         calmDark.Success,
	Warning:         calmDark.Warning,
	Danger:          calmDark.Danger,
	Chart:           calmDark.Chart,
	ControlHeight:   36, RowHeight: 30, HeaderHeight: 30, SettingsRowHeight: 56,
	TabHeight: 40, MenuBarStripHeight: 34, MenuBarItemHeight: 28,
	RadiusControl: 2, RadiusCard: 3,
	ToolbarPadTop: 10, ToolbarPadBottom: 12, ToolbarGap: 10, ToolbarGroupGap: 12,
	Icon: 16, ToolbarIcon: 17, CheckSize: 14, SwitchWidth: 40, SwitchHeight: 22,
	FontSize:       13,
	Caption:        skinhook.Type{Size: 11, Weight: 400},
	Label:          skinhook.Type{Size: 12, Weight: 600},
	Heading:        skinhook.Type{Size: 15, Weight: 600},
	Title:          skinhook.Type{Size: 20, Weight: 600},
	PopupShadow:    skinhook.Shadow{Y: 6, Blur: 20, Color: ui.RGBA(0, 0, 0, 0.18)},
	FocusRingWidth: 2, FocusRingOffset: 1,
}
