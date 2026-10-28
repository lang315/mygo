package ui

import "testing"

// A frame resolves a line's glyphs in one call of the text system, not one
// per glyph.
func TestTextPaintsALineInOneCall(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Text(c, "one line of nine words to paint at once")
	}, 600, 100)
	tt.Frame()
	before := tt.rt.text.GlyphsCalls()
	tt.Frame()
	if n := tt.rt.text.GlyphsCalls() - before; n != 1 {
		t.Fatalf("a frame of one line asked the text system for its glyphs %d times, want 1", n)
	}
}

// The glyphs of a line resolved at once keep their span's color, come
// from a fallback font where the text needs one, and leave out what a
// clip culls.
func TestTextGlyphsKeepSpanColorsAndClip(t *testing.T) {
	red, blue := RGB(220, 20, 20), RGB(20, 20, 220)
	count := func(width float32) (reds, blues, all int) {
		tt := NewTester(func(c *Context) {
			Row(c).Width(width).Clip().Children(func() {
				RichText(c, Span{Text: "red red red ", Color: red}, Span{Text: "blue 😀 blue", Color: blue}).NoWrap()
			})
		}, 600, 100)
		tt.Frame()
		for _, g := range tt.rt.scene.Glyphs {
			switch g.Color {
			case red.scene():
				reds++
			case blue.scene():
				blues++
			}
			all++
		}
		return reds, blues, all
	}
	r, b, n := count(600)
	if r == 0 || b == 0 {
		t.Fatalf("unclipped: %d red and %d blue glyphs of %d", r, b, n)
	}
	cr, cb, cn := count(30) // the clip cuts the first span
	if cn == 0 || cn >= n || cb != 0 || cr >= r {
		t.Fatalf("clipped to 30: %d red and %d blue glyphs of %d, unclipped %d, %d of %d", cr, cb, cn, r, b, n)
	}
}
