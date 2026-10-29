package ui

import "github.com/egoist/mygo/internal/scene"

// Replaying a row that moved (P4b). A row records what it painted, and replays
// it where it stands; when it moved by whole device pixels in Y, the recorded
// operations are shifted by as many, which gives what painting at the new place
// would: painting snaps boxes, edges and text baselines to device pixels, so
// their coordinates are integers, and an integer added to an integer is exact
// in float32 (below 2^24). A row is shifted only if every coordinate that is
// shifted is an integer, which shiftable checks once, when it records, and only
// operations that translate knows: another kind (or a paint other than a solid
// fill) makes the row not shiftable, never wrong.

// integral reports whether v is a whole number a float32 adds exactly.
func integral(v float32) bool { return v == float32(int32(v)) && v > -1<<23 && v < 1<<23 }

// shiftable reports whether the operations and glyphs of a row can be shifted
// in Y by a whole number of pixels with the result of painting there.
func shiftable(ops []scene.Op, glyphs []scene.Glyph) bool {
	for i := range ops {
		o := &ops[i]
		switch o.Kind {
		case scene.OpFill:
			// Only a solid fill: a row with a gradient or stripes is
			// volatile (paintVolatile) and never records, and a paint
			// whose coordinates translate does not know is not shifted.
			if o.Paint != scene.PaintSolid || !integral(o.Rect.Y) {
				return false
			}
		case scene.OpShadow:
			if !integral(o.Rect.Y) || !integral(o.Cast.Y) {
				return false
			}
		case scene.OpPushClip:
			if !integral(o.Rect.Y) {
				return false
			}
		case scene.OpGlyphs, scene.OpPopClip:
		default:
			return false
		}
	}
	for i := range glyphs {
		if !integral(glyphs[i].Y) {
			return false
		}
	}
	return true
}

// translate shifts the operations and glyphs of a shiftable row by d pixels
// in Y.
func translate(ops []scene.Op, glyphs []scene.Glyph, d float32) {
	for i := range ops {
		o := &ops[i]
		switch o.Kind {
		case scene.OpFill:
			o.Rect.Y += d
		case scene.OpShadow:
			o.Rect.Y += d
			o.Cast.Y += d
		case scene.OpPushClip:
			o.Rect.Y += d
		}
	}
	for i := range glyphs {
		glyphs[i].Y += d
	}
}

// insideClip reports whether everything the element paints (its ink) is within
// clip in Y: a row shifted in Y keeps what culling did in X, but the operations
// of a row cut at the top or the bottom are those culling kept at its place.
func insideClip(c *node, clip Rect) bool {
	ink := c.ink
	return ink[1] >= clip.Y && ink[3] <= clip.Y+clip.H
}
