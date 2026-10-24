//go:build windows

package e2e

import (
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestMinimizedWindowDrawsNoFrames checks that a window out of sight asks for
// no frames of what moves, and draws again once it shows (OccludableSurface).
func TestMinimizedWindowDrawsNoFrames(t *testing.T) {
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Occluded", Width: 300, Height: 200, Content: ui.View(func(c *ui.Context) {
		c.AnimationFrame()
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) { frames.Add(1) })
	})})
	eventually(t, "frames while shown", func() bool { return frames.Load() >= 10 })
	shownCPU := cpuDuring(300 * time.Millisecond)
	w.Minimize()
	eventually(t, "the window minimized", w.IsMinimized)
	// Frames already asked for end; a negative can only be checked by
	// waiting, as the chart's idle test does.
	time.Sleep(300 * time.Millisecond)
	before := frames.Load()
	hiddenCPU := cpuDuring(300 * time.Millisecond)
	t.Logf("the process used %v of CPU in 300 ms while the window animated, %v while it was minimized", shownCPU, hiddenCPU)
	if n := frames.Load() - before; n > 1 {
		t.Fatalf("a minimized window drew %d frames", n)
	}
	w.Restore()
	eventually(t, "frames after the window shows again", func() bool { return frames.Load() >= before+10 })
}

// cpuDuring returns the CPU time (kernel and user) this process used in d.
func cpuDuring(d time.Duration) time.Duration {
	used := func() time.Duration {
		var c, e, k, u syscall.Filetime
		h, _ := syscall.GetCurrentProcess()
		syscall.GetProcessTimes(h, &c, &e, &k, &u)
		ticks := func(f syscall.Filetime) time.Duration { // 100 ns each
			return time.Duration(uint64(f.HighDateTime)<<32|uint64(f.LowDateTime)) * 100
		}
		return ticks(k) + ticks(u)
	}
	before := used()
	time.Sleep(d)
	return used() - before
}
