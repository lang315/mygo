//go:build !js

package mygo

import (
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// TestLoopInsideQueuedFunction checks that a function on the main thread
// running a loop of its own, as a context menu does, runs the functions
// queued behind it, which it may be waiting for.
func TestLoopInsideQueuedFunction(t *testing.T) {
	done := make(chan struct{})
	waited := make(chan bool, 1)
	// Both at once, as when a test asks for the menu just as it opens.
	loop.mu.Lock()
	first := len(loop.queue) == 0
	loop.queue = append(loop.queue, func() {
		waited <- wait.Poll(func() bool {
			select {
			case <-done:
				return true
			default:
			}
			backend().Step()
			return false
		})
	}, func() { close(done) })
	loop.mu.Unlock()
	if first {
		backend().Signal()
	}
	if !<-waited {
		t.Fatal("the function queued behind one running a loop did not run")
	}
}

func TestIsMainThread(t *testing.T) {
	if IsMainThread() {
		t.Fatal("the test goroutine reports the main thread")
	}
	var main bool
	RunOnMain(func() { main = IsMainThread() })
	if !main {
		t.Fatal("RunOnMain's function does not report the main thread")
	}
}
