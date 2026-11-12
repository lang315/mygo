package skin

import (
	"errors"
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// check measures the tokens' contrast: text 4.5:1 or more, the outline of
// fields and the accent line on the chrome 3:1 or more. A translucent color
// is measured blended over the layer it sits on, the one after "on", and a
// translucent layer over Content. It also measures the state distance of the
// dark or the light appearance (stateDistance). The error names every
// failing pair.
func check(t *skinhook.Tokens, dark bool) error {
	var errs []error
	pair := func(name string, fg, bg ui.Color, floor float64) {
		bg = bg.Over(t.Content)
		if r := contrastOf(fg.Over(bg), bg); r < floor {
			errs = append(errs, fmt.Errorf("%s: %.2f:1, want %g:1", name, r, floor))
		}
	}
	pair("Text on Content", t.Text, t.Content, 4.5)
	pair("Text on Chrome", t.Text, t.Chrome, 4.5)
	pair("Text2 on Content", t.Text2, t.Content, 4.5)
	pair("Text2 on Chrome", t.Text2, t.Chrome, 4.5)
	pair("Text3 on Control", t.Text3, t.Control, 4.5)
	pair("Text2 on RowSelected", t.Text2, t.RowSelected, 4.5)
	pair("Text on RowSelected", t.Text, t.RowSelected, 4.5)
	pair("Text2 on RowHover", t.Text2, t.RowHover, 4.5)
	pair("OnAccent on Accent.Base", t.OnAccent, t.Accent.Base, 4.5)
	for _, f := range []struct {
		name string
		f    skinhook.Family
	}{{"Accent", t.Accent}, {"Success", t.Success}, {"Warning", t.Warning}, {"Danger", t.Danger}, {"Info", t.Info}} {
		pair(f.name+" Text on Subtle", f.f.Text, f.f.Subtle, 4.5)
		pair(f.name+" On on Base", f.f.On, f.f.Base, 4.5)
	}
	pair("FieldLine on Content", t.FieldLine, t.Content, 3)
	pair("FieldLine on Chrome", t.FieldLine, t.Chrome, 3)
	pair("Accent.Base on Chrome", t.Accent.Base, t.Chrome, 3)
	errs = append(errs, stateDistance(t, dark))
	return errors.Join(errs...)
}

// stateDistance keeps the running color apart from the status colors: the
// CIE76 ΔE of Info's text and each of Success's, Warning's and Danger's must
// be 45 or more in light and 35 or more in dark, for every eye of visions.
// The error names each failing pair once, with the eye that sees it nearest.
func stateDistance(t *skinhook.Tokens, dark bool) error {
	floor := 45.0
	if dark {
		floor = 35
	}
	var errs []error
	for _, s := range []struct {
		name string
		c    ui.Color
	}{{"Success", t.Success.Text}, {"Warning", t.Warning.Text}, {"Danger", t.Danger.Text}} {
		least, by := math.Inf(1), ""
		for _, v := range visions {
			if d := deltaE(t.Info.Text.Over(t.Content), s.c.Over(t.Content), v.m); d < least {
				least, by = d, v.name
			}
		}
		if least < floor {
			errs = append(errs, fmt.Errorf("Info Text near %s Text (%s): ΔE %.0f, want %g", s.name, by, least, floor))
		}
	}
	return errors.Join(errs...)
}

// visions are the eyes stateDistance measures for: normal vision, and
// Machado, Oliveira and Fernandes's (2009) simulation of the two red-green
// blindnesses at severity 1.0, a matrix over linear sRGB.
var visions = []struct {
	name string
	m    *[3][3]float64
}{
	{"normal", nil},
	{"deuteranopia", &[3][3]float64{{0.367322, 0.860646, -0.227968}, {0.280085, 0.672501, 0.047413}, {-0.011820, 0.042940, 0.968881}}},
	{"protanopia", &[3][3]float64{{0.152286, 1.052583, -0.204868}, {0.114503, 0.786281, 0.099216}, {-0.003882, -0.048116, 1.051998}}},
}

// deltaE is the CIE76 distance of two opaque colors in CIELAB (D65), as the
// eye of the matrix m sees them (nil for normal vision).
func deltaE(a, b ui.Color, m *[3][3]float64) float64 {
	la, lb := labOf(a, m), labOf(b, m)
	return math.Sqrt((la[0]-lb[0])*(la[0]-lb[0]) + (la[1]-lb[1])*(la[1]-lb[1]) + (la[2]-lb[2])*(la[2]-lb[2]))
}

// labOf converts c to CIELAB (D65) after the matrix m, if any.
func labOf(c ui.Color, m *[3][3]float64) [3]float64 {
	rgb := [3]float64{linear(c.R), linear(c.G), linear(c.B)}
	if m != nil {
		var o [3]float64
		for i := range o {
			o[i] = m[i][0]*rgb[0] + m[i][1]*rgb[1] + m[i][2]*rgb[2]
		}
		rgb = o
	}
	x := (0.4124564*rgb[0] + 0.3575761*rgb[1] + 0.1804375*rgb[2]) / 0.95047
	y := 0.2126729*rgb[0] + 0.7151522*rgb[1] + 0.0721750*rgb[2]
	z := (0.0193339*rgb[0] + 0.1191920*rgb[1] + 0.9503041*rgb[2]) / 1.08883
	f := func(v float64) float64 {
		if v > 216.0/24389 {
			return math.Cbrt(v)
		}
		return (24389.0/27*v + 16) / 116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return [3]float64{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

// linear is an sRGB channel in linear light.
func linear(v uint8) float64 {
	s := float64(v) / 255
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// contrastOf is WCAG's contrast ratio of two opaque colors, as widget's
// (ui/widget/stylevalue.go), which this package does not reach.
func contrastOf(a, b ui.Color) float64 {
	l := func(c ui.Color) float64 {
		return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
	}
	x, y := l(a)+0.05, l(b)+0.05
	if x < y {
		x, y = y, x
	}
	return x / y
}
