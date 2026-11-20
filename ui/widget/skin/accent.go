package skin

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// WithAccent returns a new skin, s with a palette made from the one color
// c, named by c's hex ("#7c3aed"). s does not change.
//
// In light, c is the accent as it is when it passes the floors Check
// measures: white text on it reads (at s's text floor, 4.5:1 or 7:1), it
// stands out on the chrome (3:1), and text reads on the selected row mixed
// from it (at the text floor). A
// lighter color, as a yellow, darkens to the lightest of its hue that
// passes, losing chroma only if no lightness of it is enough; a near-black
// one lightens just enough (#0a0a0a to about #131313). In dark, the accent
// is c lightened, if it must be, until the dark color of its hue on it (at
// c's chroma, 0.04 at most, so a gray keeps a gray) reads. Hover, pressed
// and the text on the subtle fill follow from the base, in Oklab. Info is
// the accent, unless the accent sits near the status colors (a red, a green
// or an amber): then Info stays s's. c's alpha is ignored: the accent is
// opaque.
//
//	skin.Use(skin.Calm.WithAccent(ui.Hex("#7c3aed")))
func (s *Skin) WithAccent(c ui.Color) *Skin {
	s = s.get()
	c.A = 255
	lab := oklabOf(c)
	l, ch := float64(lab[0]), math.Hypot(float64(lab[1]), float64(lab[2]))
	h := math.Atan2(float64(lab[2]), float64(lab[1]))
	lt, dt := &s.light, &s.dark
	tf := s.textFloor()

	// Light: white on the base, the base on the chrome, and text on the
	// selected row NewPalette mixes from the base.
	base := c
	ok := func(b ui.Color) bool {
		row := b.Mix(white, 0.82)
		return ratio(lt, white, b) >= tf && ratio(lt, b, lt.Chrome) >= 3 &&
			ratio(lt, lt.TextMuted, row) >= tf && ratio(lt, lt.Text, row) >= tf
	}
	if !ok(base) {
		step := -0.005 // too light for white text: darker
		if ratio(lt, white, base) >= tf {
			step = 0.005 // too dark for the selected row: lighter
		}
		base = search(l, ch, h, step, ok)
	}
	bl, bc := lightChroma(base)
	text := base
	for tl := bl; tl >= 0 && ratio(lt, text, base.Mix(white, 0.92)) < tf; tl -= 0.01 {
		text = oklch(tl, bc, h)
	}
	light := LightFamily(base, oklch(bl-0.05, bc, h), oklch(bl-0.10, bc, h), text)

	// Dark: a dark color of the hue on the base, the base on the chrome.
	on := oklch(0.18, min(0.04, ch), h) // a gray keeps a gray On
	base = search(l, ch, h, 0.005, func(b ui.Color) bool { return ratio(dt, on, b) >= tf && ratio(dt, b, dt.Chrome) >= 3 })
	bl, bc = lightChroma(base)
	text = base
	for tl := bl; tl <= 1 && ratio(dt, text, base.Alpha(0.14)) < tf; tl += 0.01 {
		text = oklch(tl, bc, h)
	}
	dark := DarkFamily(base, oklch(bl+0.05, bc, h), oklch(bl-0.05, bc, h), text, on)

	p := NewPalette(hexOf(c), light, dark)
	l2, d2 := *lt, *dt
	p.light.apply(&l2)
	p.dark.apply(&d2)
	if stateDistance(&l2, false) != nil || stateDistance(&d2, true) != nil {
		p.withInfo(lt.Info, dt.Info)
	}
	return s.WithPalette(p)
}

// Palettes returns the palettes made for the skin, its own first: Calm's
// are Blue, Chàm, Indigo, Ink and Teal. It is nil for a skin made with New.
func (s *Skin) Palettes() []*Palette {
	return append([]*Palette(nil), s.get().palettes...)
}

// ratio is the contrast of fg on bg in the tokens t, as check measures it.
func ratio(t *skinhook.Tokens, fg, bg ui.Color) float64 {
	bg = bg.Over(t.Background)
	return contrastOf(fg.Over(bg), bg)
}

// search returns the first color of the hue h and the chroma ch, from the
// lightness l in steps of step, inside sRGB, that ok takes; if none, it
// tries again with 10% less chroma, down to a gray.
func search(l, ch, h, step float64, ok func(ui.Color) bool) ui.Color {
	for {
		if ch < 1e-3 {
			ch = 0
		}
		for x := l; x >= 0 && x <= 1; x += step {
			lab := [3]float32{float32(x), float32(ch * math.Cos(h)), float32(ch * math.Sin(h))}
			if c := fromOklab(lab); inGamut(lab) && ok(c) {
				return c
			}
		}
		if ch == 0 {
			// No gray passed: the end of the search, black or white.
			if step < 0 {
				return ui.RGB(0, 0, 0)
			}
			return white
		}
		ch *= 0.9
	}
}

// oklch is the color of the Oklab lightness l, chroma ch and hue h (in
// radians), clamped to sRGB.
func oklch(l, ch, h float64) ui.Color {
	return fromOklab([3]float32{float32(l), float32(ch * math.Cos(h)), float32(ch * math.Sin(h))})
}

// lightChroma is c's Oklab lightness and chroma.
func lightChroma(c ui.Color) (float64, float64) {
	lab := oklabOf(c)
	return float64(lab[0]), math.Hypot(float64(lab[1]), float64(lab[2]))
}

func oklabOf(c ui.Color) [3]float32 {
	return scene.Oklab(scene.Color{R: c.R, G: c.G, B: c.B, A: c.A})
}

// oklabLinear is the Oklab color lab in linear sRGB, which may lie outside
// 0 to 1.
func oklabLinear(lab [3]float32) [3]float64 {
	L, a, b := float64(lab[0]), float64(lab[1]), float64(lab[2])
	l := math.Pow(L+0.3963377774*a+0.2158037573*b, 3)
	m := math.Pow(L-0.1055613458*a-0.0638541728*b, 3)
	s := math.Pow(L-0.0894841775*a-1.2914855480*b, 3)
	return [3]float64{
		4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s,
	}
}

// inGamut reports whether the Oklab color lab is inside sRGB, and its sRGB
// color, 8 bits a channel, is still lab: near black, a chroma that rounds
// away would make a hue's search end on black.
func inGamut(lab [3]float32) bool {
	for _, v := range oklabLinear(lab) {
		if v < -1e-4 || v > 1+1e-4 {
			return false
		}
	}
	r := oklabOf(fromOklab(lab))
	for i := range r {
		if math.Abs(float64(r[i]-lab[i])) > 0.01 {
			return false
		}
	}
	return true
}

// fromOklab is the opaque sRGB color of the Oklab color lab, the inverse of
// scene.Oklab, each channel clamped to sRGB.
func fromOklab(lab [3]float32) ui.Color {
	var o [3]uint8
	for i, v := range oklabLinear(lab) {
		v = max(0, min(v, 1))
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		o[i] = uint8(v*255 + 0.5)
	}
	return ui.RGB(o[0], o[1], o[2])
}

// hexOf is c's "#rrggbb".
func hexOf(c ui.Color) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }
