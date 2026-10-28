package text

import "testing"

// Glyphs returns, request by request, what Glyph returns: the same keys,
// the same entries, for every pen position within a pixel, both opaque
// values, shades and fonts mixed within one call.
func TestGlyphsMatchGlyph(t *testing.T) {
	s := Shared()
	// The emoji's glyphs fill the shared system's color atlas, which the
	// tests after this one must find with room.
	t.Cleanup(s.MakeRoom)
	l := s.Layout(Params{Text: "Ag fi 😀 Wy", Style: Style{Size: 15}})
	var reqs []GlyphRequest
	for _, line := range l.Lines {
		for i, g := range line.Glyphs {
			if g.Object {
				continue
			}
			for _, frac := range []float32{0, 0.13, 0.5, 0.74, 0.99} {
				shade := Flat
				if i%2 == 1 {
					shade = ShadeOf(250, 250, 250)
				}
				reqs = append(reqs, GlyphRequest{Font: g.Font, ID: g.ID, X: float32(i)*9 + frac, Shade: shade})
			}
		}
	}
	reqs = append(reqs, GlyphRequest{}) // a nil font, as Glyph takes it
	for _, opaque := range []bool{false, true} {
		for _, scale := range []float32{1, 2} {
			got := s.Glyphs(reqs, scale, opaque, nil)
			if len(got) != len(reqs) {
				t.Fatalf("%d images for %d requests", len(got), len(reqs))
			}
			for i, r := range reqs {
				if want := s.Glyph(r.Font, r.ID, scale, r.X, r.Shade, opaque); got[i] != want {
					t.Fatalf("opaque %v scale %v request %d %+v: Glyphs %+v, Glyph %+v", opaque, scale, i, r, got[i], want)
				}
			}
		}
	}
}

// Glyphs appends into the slice it is given, so that a caller reusing it
// allocates nothing once the glyphs are in the atlas.
func TestGlyphsAllocatesNothing(t *testing.T) {
	s := Shared()
	l := s.Layout(Params{Text: "acc000123", Style: Style{Size: 13}})
	var reqs []GlyphRequest
	for i, g := range l.Lines[0].Glyphs {
		reqs = append(reqs, GlyphRequest{Font: g.Font, ID: g.ID, X: float32(i) * 7.3, Shade: Flat})
	}
	out := s.Glyphs(reqs, 2, true, nil)
	if n := testing.AllocsPerRun(50, func() { out = s.Glyphs(reqs, 2, true, out[:0]) }); n != 0 {
		t.Fatalf("Glyphs allocates %.0f times a call", n)
	}
}

func BenchmarkGlyphsLine(b *testing.B) {
	s := Shared()
	l := s.Layout(Params{Text: "acc000123", Style: Style{Size: 13}})
	var reqs []GlyphRequest
	for i, g := range l.Lines[0].Glyphs {
		reqs = append(reqs, GlyphRequest{Font: g.Font, ID: g.ID, X: float32(i) * 7.3, Shade: Flat})
	}
	out := s.Glyphs(reqs, 2, true, nil)
	b.Run("Glyph", func(b *testing.B) {
		for b.Loop() {
			for _, r := range reqs {
				s.Glyph(r.Font, r.ID, 2, r.X, r.Shade, true)
			}
		}
	})
	b.Run("Glyphs", func(b *testing.B) {
		for b.Loop() {
			out = s.Glyphs(reqs, 2, true, out[:0])
		}
	})
}
