package ui

import (
	"sync"
	"testing"
)

// An engine with no video reads no hook of the video path: the window that
// builds the first video of the process sets them (linkVideo) while the
// others, on goroutines of their own, go on ending frames. Under -race, the
// read of sweepVideosFn in endFrame by an engine that made no video, and the
// write of it by linkVideo, are two accesses nothing orders.
func TestLinkVideoWhileAnotherEngineEndsFrames(t *testing.T) {
	// As at the start of a process, so that linkVideo writes.
	videoOnce, sweepVideosFn, releaseVideosFn = sync.Once{}, nil, nil
	t.Cleanup(linkVideo) // the state other tests leave is linked, as a video makes it

	tt := coreNewTester(func(c *context) {
		coreText(c, "no video")
	}, 100, 40)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 3 {
			tt.Frame()
		}
	}()
	go func() {
		defer wg.Done()
		linkVideo()
	}()
	wg.Wait()
}
