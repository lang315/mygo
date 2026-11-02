package ui

import "time"

// MeasureFrames makes the frames of tt measure their parts, for FrameLaps.
func MeasureFrames(tt *Tester) { tt.rt.stats = newFrameStats(time.Hour) }

// TextCounts returns how many glyphs the text stack drew, layouts it made and
// calls of Glyphs it got, over the life of tt.
func TextCounts(tt *Tester) (rastered, layouts, glyphCalls uint64) {
	return tt.rt.text.Rastered(), tt.rt.text.LayoutsMade(), tt.rt.text.GlyphsCalls()
}

// SceneGlyphs returns, for the last frame of tt, how many glyphs it draws,
// how many of them are subpixel (ClearType) ones, and the pixels of the
// subpixel ones and of all.
func SceneGlyphs(tt *Tester) (glyphs, subpixel, subpixelPixels, pixels int) {
	for _, g := range tt.rt.scene.Glyphs {
		glyphs++
		n := int(g.UW) * int(g.VH)
		pixels += n
		if g.Subpixel {
			subpixel++
			subpixelPixels += n
		}
	}
	return
}

// FrameLaps returns how long the last frame of tt spent building, laying
// out, painting and presenting. It is zero unless MeasureFrames was called.
func FrameLaps(tt *Tester) [4]time.Duration {
	if tt.rt.stats == nil {
		return [4]time.Duration{}
	}
	l := tt.rt.stats.laps
	return [4]time.Duration{l[phaseBuild], l[phaseLayout], l[phasePaint], l[phasePresent]}
}
