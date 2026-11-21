package video

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestSeekedOfTheSeekBefore checks that the position of a seek is its
// time while the engine seeks, though the eventSeeked of the seek before
// comes after it began, as it does once that seek's picture came first
// (TestSeek on Windows on Arm: "sought to 500ms: position 0s").
func TestSeekedOfTheSeekBefore(t *testing.T) {
	var e *mfEngine
	p, tt, shows := played(t, func() (*ui.VideoPlayer, error) {
		m, err := newEngine(source{data: rgbMP4})
		if err != nil {
			return nil, err
		}
		e = m.(*mfEngine)
		return ui.NewEngineVideoPlayer(m), nil
	})
	p.Pause()
	p.Seek(900 * time.Millisecond)
	wait.Until(t, "the picture at 900ms", func() bool {
		tt.Frame()
		return near(shows(), blue)
	})
	p.Seek(500 * time.Millisecond)
	e.event(eventSeeked, 0, 0) // the seek to 900 ms's, late
	if pos := p.Position(); pos < 400*time.Millisecond || pos > 600*time.Millisecond {
		t.Errorf("sought to 500ms, then the seek before's eventSeeked: position %v", pos)
	}
}
