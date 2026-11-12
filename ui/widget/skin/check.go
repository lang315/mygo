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
// translucent layer over Content. The error names every failing pair.
func check(t *skinhook.Tokens) error {
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
	pair("OnAccent on Accent.Base", t.OnAccent, t.Accent.Base, 4.5)
	for _, f := range []struct {
		name string
		f    skinhook.Family
	}{{"Accent", t.Accent}, {"Success", t.Success}, {"Warning", t.Warning}, {"Danger", t.Danger}} {
		pair(f.name+" Text on Subtle", f.f.Text, f.f.Subtle, 4.5)
		pair(f.name+" On on Base", f.f.On, f.f.Base, 4.5)
	}
	pair("FieldLine on Content", t.FieldLine, t.Content, 3)
	pair("FieldLine on Chrome", t.FieldLine, t.Chrome, 3)
	pair("Accent.Base on Chrome", t.Accent.Base, t.Chrome, 3)
	return errors.Join(errs...)
}

// contrastOf is WCAG's contrast ratio of two opaque colors, as widget's
// (ui/widget/stylevalue.go), which this package does not reach.
func contrastOf(a, b ui.Color) float64 {
	l := func(c ui.Color) float64 {
		ch := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
	}
	x, y := l(a)+0.05, l(b)+0.05
	if x < y {
		x, y = y, x
	}
	return x / y
}
