package twclass

// spacing is Tailwind v3's scale, in its units (4 DIPs; Theme.Space(1)).
var spacing = map[string]float32{
	"0": 0, "px": 0.25, "0.5": 0.5, "1": 1, "1.5": 1.5, "2": 2, "2.5": 2.5, "3": 3, "3.5": 3.5,
	"4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9, "10": 10, "11": 11, "12": 12, "14": 14,
	"16": 16, "20": 20, "24": 24, "28": 28, "32": 32, "36": 36, "40": 40, "44": 44, "48": 48,
	"52": 52, "56": 56, "60": 60, "64": 64, "72": 72, "80": 80, "96": 96,
}

// radii are in theme units: Tailwind's rem values over 4 DIPs.
var radii = map[string]float32{"": 1, "none": 0, "sm": 0.5, "md": 1.5, "lg": 2, "xl": 3, "2xl": 4, "3xl": 6, "full": 9999}

var borderWidths = map[string]float32{"": 1, "0": 0, "2": 2, "4": 4, "8": 8}

var textSizes = map[string]Decl{
	"xs": {Kind: TextSize, Value: 1}, "sm": {Kind: TextSize, Value: 2}, "base": {Kind: TextSize, Value: 0},
	"lg": {Kind: TextSize, Value: 3}, "xl": {Kind: TextSize, Value: 4}, "2xl": {Kind: TextSize, Value: 5},
	"3xl": {Kind: TextSize, Value: 6},
	"4xl": {Kind: FontSize, Value: 36}, "5xl": {Kind: FontSize, Value: 48}, "6xl": {Kind: FontSize, Value: 60},
	"7xl": {Kind: FontSize, Value: 72}, "8xl": {Kind: FontSize, Value: 96}, "9xl": {Kind: FontSize, Value: 128},
}

var weights = map[string]float32{
	"thin": 100, "extralight": 200, "light": 300, "normal": 400, "medium": 500,
	"semibold": 600, "bold": 700, "extrabold": 800, "black": 900,
}

var opacities = map[string]float32{
	"0": 0, "5": 0.05, "10": 0.1, "15": 0.15, "20": 0.2, "25": 0.25, "30": 0.3, "35": 0.35,
	"40": 0.4, "45": 0.45, "50": 0.5, "55": 0.55, "60": 0.6, "65": 0.65, "70": 0.7, "75": 0.75,
	"80": 0.8, "85": 0.85, "90": 0.9, "95": 0.95, "100": 1,
}

var cursors = map[string]bool{"default": true, "pointer": true, "text": true, "move": true, "not-allowed": true, "grab": true, "grabbing": true, "crosshair": true}

var durations = map[string]float32{
	"duration-75": 75, "duration-100": 100, "duration-150": 150, "duration-200": 200,
	"duration-300": 300, "duration-500": 500, "duration-700": 700, "duration-1000": 1000,
}

// fixed are the utilities with no value to parse.
var fixed = map[string][]Decl{
	"shadow-sm":     {{Kind: Shadows, Shadows: []Shadow{{0, 1, 2, 0, 0.05, false}}}},
	"shadow":        {{Kind: Shadows, Shadows: []Shadow{{0, 1, 3, 0, 0.1, false}, {0, 1, 2, -1, 0.1, false}}}},
	"shadow-md":     {{Kind: Shadows, Shadows: []Shadow{{0, 4, 6, -1, 0.1, false}, {0, 2, 4, -2, 0.1, false}}}},
	"shadow-lg":     {{Kind: Shadows, Shadows: []Shadow{{0, 10, 15, -3, 0.1, false}, {0, 4, 6, -4, 0.1, false}}}},
	"shadow-xl":     {{Kind: Shadows, Shadows: []Shadow{{0, 20, 25, -5, 0.1, false}, {0, 8, 10, -6, 0.1, false}}}},
	"shadow-2xl":    {{Kind: Shadows, Shadows: []Shadow{{0, 25, 50, -12, 0.25, false}}}},
	"shadow-inner":  {{Kind: Shadows, Shadows: []Shadow{{0, 2, 4, 0, 0.05, true}}}},
	"shadow-none":   {{Kind: Shadows}},
	"max-w-none":    {{Kind: MaxWidth, Value: noMaxWidth}},
	"italic":        {{Kind: Italic, Value: 1}},
	"not-italic":    {{Kind: Italic, Value: 0}},
	"font-mono":     {{Kind: Mono}},
	"hexpand":       {{Kind: HExpand}},
	"vexpand":       {{Kind: VExpand}},
	"halign-start":  {{Kind: Align, Value: 1}},
	"halign-center": {{Kind: Align, Value: 2}},
	"halign-end":    {{Kind: Align, Value: 3}},
	"halign-fill":   {{Kind: Align, Value: 4}},
	"valign-start":  {{Kind: VAlign, Value: 1}},
	"valign-center": {{Kind: VAlign, Value: 2}},
	"valign-end":    {{Kind: VAlign, Value: 3}},
	"valign-fill":   {{Kind: VAlign, Value: 4}},
}

// buttons are gota's GTK button classes, as the classes they stand for
// (Task 4 fills it).
var buttons = map[string]string{
	"suggested-action":   "bg-accent text-on-accent hover:bg-accent-hover active:bg-accent-pressed",
	"destructive-action": "bg-danger text-on-accent hover:opacity-90 active:opacity-80",
}

// Roles are the theme roles a Color names.
var Roles = []string{
	"background", "surface", "surface-hover", "surface-pressed", "border", "text", "text-muted",
	"accent", "accent-hover", "accent-pressed", "accent-text", "danger", "warning", "success",
	"selection", "focus", "inverse", "inverse-text",
}

// roles are the theme tokens after bg-, text- and border-, by role.
var roles = map[string]string{
	"muted": "text-muted", "surface": "surface", "surface-hover": "surface-hover",
	"surface-pressed": "surface-pressed", "accent": "accent", "accent-hover": "accent-hover",
	"accent-pressed": "accent-pressed", "on-accent": "accent-text", "danger": "danger",
	"warning": "warning", "success": "success", "selection": "selection", "focus": "focus",
	"inverse": "inverse", "on-inverse": "inverse-text",
}

// defaults are the role of the token default, by prefix.
var defaults = map[string]string{"bg": "background", "text": "text", "border": "border"}

var shades = map[string]int{"50": 0, "100": 1, "200": 2, "300": 3, "400": 4, "500": 5, "600": 6, "700": 7, "800": 8, "900": 9, "950": 10}

// palette is Tailwind v3.4's (src/public/colors.js), 0xRRGGBB by shade
// 50, 100 … 900, 950.
var palette = map[string][11]uint32{
	"slate":   {0xf8fafc, 0xf1f5f9, 0xe2e8f0, 0xcbd5e1, 0x94a3b8, 0x64748b, 0x475569, 0x334155, 0x1e293b, 0x0f172a, 0x020617},
	"gray":    {0xf9fafb, 0xf3f4f6, 0xe5e7eb, 0xd1d5db, 0x9ca3af, 0x6b7280, 0x4b5563, 0x374151, 0x1f2937, 0x111827, 0x030712},
	"zinc":    {0xfafafa, 0xf4f4f5, 0xe4e4e7, 0xd4d4d8, 0xa1a1aa, 0x71717a, 0x52525b, 0x3f3f46, 0x27272a, 0x18181b, 0x09090b},
	"neutral": {0xfafafa, 0xf5f5f5, 0xe5e5e5, 0xd4d4d4, 0xa3a3a3, 0x737373, 0x525252, 0x404040, 0x262626, 0x171717, 0x0a0a0a},
	"stone":   {0xfafaf9, 0xf5f5f4, 0xe7e5e4, 0xd6d3d1, 0xa8a29e, 0x78716c, 0x57534e, 0x44403c, 0x292524, 0x1c1917, 0x0c0a09},
	"red":     {0xfef2f2, 0xfee2e2, 0xfecaca, 0xfca5a5, 0xf87171, 0xef4444, 0xdc2626, 0xb91c1c, 0x991b1b, 0x7f1d1d, 0x450a0a},
	"orange":  {0xfff7ed, 0xffedd5, 0xfed7aa, 0xfdba74, 0xfb923c, 0xf97316, 0xea580c, 0xc2410c, 0x9a3412, 0x7c2d12, 0x431407},
	"amber":   {0xfffbeb, 0xfef3c7, 0xfde68a, 0xfcd34d, 0xfbbf24, 0xf59e0b, 0xd97706, 0xb45309, 0x92400e, 0x78350f, 0x451a03},
	"yellow":  {0xfefce8, 0xfef9c3, 0xfef08a, 0xfde047, 0xfacc15, 0xeab308, 0xca8a04, 0xa16207, 0x854d0e, 0x713f12, 0x422006},
	"lime":    {0xf7fee7, 0xecfccb, 0xd9f99d, 0xbef264, 0xa3e635, 0x84cc16, 0x65a30d, 0x4d7c0f, 0x3f6212, 0x365314, 0x1a2e05},
	"green":   {0xf0fdf4, 0xdcfce7, 0xbbf7d0, 0x86efac, 0x4ade80, 0x22c55e, 0x16a34a, 0x15803d, 0x166534, 0x14532d, 0x052e16},
	"emerald": {0xecfdf5, 0xd1fae5, 0xa7f3d0, 0x6ee7b7, 0x34d399, 0x10b981, 0x059669, 0x047857, 0x065f46, 0x064e3b, 0x022c22},
	"teal":    {0xf0fdfa, 0xccfbf1, 0x99f6e4, 0x5eead4, 0x2dd4bf, 0x14b8a6, 0x0d9488, 0x0f766e, 0x115e59, 0x134e4a, 0x042f2e},
	"cyan":    {0xecfeff, 0xcffafe, 0xa5f3fc, 0x67e8f9, 0x22d3ee, 0x06b6d4, 0x0891b2, 0x0e7490, 0x155e75, 0x164e63, 0x083344},
	"sky":     {0xf0f9ff, 0xe0f2fe, 0xbae6fd, 0x7dd3fc, 0x38bdf8, 0x0ea5e9, 0x0284c7, 0x0369a1, 0x075985, 0x0c4a6e, 0x082f49},
	"blue":    {0xeff6ff, 0xdbeafe, 0xbfdbfe, 0x93c5fd, 0x60a5fa, 0x3b82f6, 0x2563eb, 0x1d4ed8, 0x1e40af, 0x1e3a8a, 0x172554},
	"indigo":  {0xeef2ff, 0xe0e7ff, 0xc7d2fe, 0xa5b4fc, 0x818cf8, 0x6366f1, 0x4f46e5, 0x4338ca, 0x3730a3, 0x312e81, 0x1e1b4b},
	"violet":  {0xf5f3ff, 0xede9fe, 0xddd6fe, 0xc4b5fd, 0xa78bfa, 0x8b5cf6, 0x7c3aed, 0x6d28d9, 0x5b21b6, 0x4c1d95, 0x2e1065},
	"purple":  {0xfaf5ff, 0xf3e8ff, 0xe9d5ff, 0xd8b4fe, 0xc084fc, 0xa855f7, 0x9333ea, 0x7e22ce, 0x6b21a8, 0x581c87, 0x3b0764},
	"fuchsia": {0xfdf4ff, 0xfae8ff, 0xf5d0fe, 0xf0abfc, 0xe879f9, 0xd946ef, 0xc026d3, 0xa21caf, 0x86198f, 0x701a75, 0x4a044e},
	"pink":    {0xfdf2f8, 0xfce7f3, 0xfbcfe8, 0xf9a8d4, 0xf472b6, 0xec4899, 0xdb2777, 0xbe185d, 0x9d174d, 0x831843, 0x500724},
	"rose":    {0xfff1f2, 0xffe4e6, 0xfecdd3, 0xfda4af, 0xfb7185, 0xf43f5e, 0xe11d48, 0xbe123c, 0x9f1239, 0x881337, 0x4c0519},
}

// noMaxWidth is the max-width max-w-none sets: more than any window, so a
// width is no longer capped by its container.
const noMaxWidth = 1e9
