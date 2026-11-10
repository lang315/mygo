package ui

import (
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// MeasureFrames makes the frames of tt measure their parts, for FrameLaps.
func MeasureFrames(tt *Tester) { tt.rt.stats = newFrameStats(time.Hour) }

// FrameLaps returns how long the last frame of tt spent building, laying
// out, painting and presenting. It is zero unless MeasureFrames was called.
func FrameLaps(tt *Tester) [4]time.Duration {
	if tt.rt.stats == nil {
		return [4]time.Duration{}
	}
	l := tt.rt.stats.laps
	return [4]time.Duration{l[phaseBuild], l[phaseLayout], l[phasePaint], l[phasePresent]}
}

// HostMove moves the pointer to (x, y) as a window's surface does: the
// frame the move asks for, if any, comes as the surface's next, which
// paints the last frame's elements again when that is all it needs, where
// Move builds a frame anew whatever the move asked for.
func (t *Tester) HostMove(x, y float32) {
	t.hostEvent(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
}

// Paints returns how many times the Tester's window painted.
func (t *Tester) Paints() int { return int(t.rt.paints) }
