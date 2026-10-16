package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// MenuIconPNG draws s as a menu icon: a 32×32 PNG, shown at 16×16 points,
// with c for currentColor. template reports whether s paints only with
// currentColor, for a template image on macOS that the system colors.
// MenuItem.Icon calls it with the color of the text of the theme.
func MenuIconPNG(s *SVG, c Color) (b []byte, template bool) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	s.doc.Draw(img, color.NRGBA{c.R, c.G, c.B, c.A}, false)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, false
	}
	return buf.Bytes(), s.doc.OnlyCurrentColor()
}
