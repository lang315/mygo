package ui

import (
	"sync"

	"github.com/egoist/mygo/internal/raster"
)

// The engine holds the videos of the system's media engines it shows through
// these functions, which NewEngineVideoPlayer sets: an engine holds a
// videoEngine only after a player of one was made, in the same process, so
// every call below is made after linkVideo ran. Apps that play no such video
// never make one, and do not link the code that holds them, nor, through
// linkVideoRenderers, that which draws their frames.
//
// Each is read only by an engine whose videos or shownVideos are not empty,
// which advance wrote after a player existed, on that engine's goroutine: it
// waited for the write. An engine with no video does not read them, as one
// with no editor does not read the editor's functions (editorlink.go), so
// that a window building the first video of the process, on a goroutine of
// its own, does not race the others, which go on ending frames.
var (
	sweepVideosFn   func(rt *engine)
	releaseVideosFn func(rt *engine)
)

// videoOnce makes the first linkVideo the only one to write, as editorOnce
// does for the editor: the functions are shared by every engine of the
// process, and each is read for a video that engine's goroutine made.
var videoOnce sync.Once

// linkVideo sets the functions above, and those of the CPU rasterizer and of
// the renderers that draw frames of video. It is called by
// NewEngineVideoPlayer, the one way a video of a system engine is made.
func linkVideo() {
	videoOnce.Do(func() {
		sweepVideosFn = (*engine).sweepVideos
		releaseVideosFn = (*engine).releaseVideos
		raster.LinkVideo()
		linkVideoRenderers()
	})
}
