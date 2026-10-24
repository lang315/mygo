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
var buttons = map[string]string{}
