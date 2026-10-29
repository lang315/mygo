//go:build !js

package mygo

import (
	"bytes"
	"runtime"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// The looks a watch starts at a document (checkDisk) are work docWork counts,
// as the watch is: a test waits for it with docWork.Wait before it resets
// what the look reads and writes (the paths of the app's files, the recent
// documents). A look no one waits for ran into the cleanups of the test, and
// into the next test.
func TestDocumentWatchWaitsForTheLookItStarts(t *testing.T) {
	var once, free sync.Once
	looked, release := make(chan struct{}), make(chan struct{})
	open := func() { free.Do(func() { close(release) }) }
	slow := func(bool) func() {
		once.Do(func() { close(looked) })
		<-release
		return nil
	}
	afterDiskLook.Store(&slow)
	t.Cleanup(func() { afterDiskLook.Store(nil) })
	t.Cleanup(open) // a failed test does not leave the look held
	openDocumentInTempDir(t)
	// The document's watch makes its look at start, and ends at once.
	docWatches.Lock()
	for _, w := range docWatches.files {
		w.cancel()
	}
	docWatches.Unlock()
	wait.For(t, "the look at the file", looked)
	waitFor(t, "the document's watch to end", func() bool {
		buf := make([]byte, 1<<20)
		return !bytes.Contains(buf[:runtime.Stack(buf, true)], []byte("mygo.followDocumentFile"))
	})
	waited := make(chan struct{})
	go func() { docWork.Wait(); close(waited) }()
	// Only the look can keep the wait now; a wait that is over returns within
	// these yields.
	for range 1000 {
		runtime.Gosched()
	}
	select {
	case <-waited:
		t.Error("the work is over while the look at the document goes on")
	default:
	}
	open()
	wait.For(t, "the work to end", waited)
}
