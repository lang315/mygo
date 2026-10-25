//go:build unix && !js

package mygo

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// blockedStates sets up window states to save, in a directory where the file
// cannot be written until the test reads its FIFO, as a file another program
// has open cannot be replaced on Windows: a named pipe in the place of the
// temporary file, which a write opens, and waits for a reader to open too.
func blockedStates(t *testing.T) (file string, release func()) {
	t.Helper()
	dir := t.TempDir()
	file = filepath.Join(dir, windowStateFile)
	App.SetPath(PathUserData, dir)
	forget := func() { windowStates.loaded, windowStates.byKey, windowStates.dirty = false, nil, false }
	t.Cleanup(func() {
		flushAppWrites()
		onMain(forget)
		App.SetPath(PathUserData, "")
	})
	if err := syscall.Mkfifo(file+".tmp", 0o600); err != nil {
		t.Skipf("no named pipe: %v", err)
	}
	onMain(func() {
		windowStates.loaded = true
		windowStates.byKey = map[string]WindowState{"main": {X: 1, Y: 2, Width: 300, Height: 200}}
		windowStates.dirty = true
	})
	release = func() {
		f, err := os.Open(file + ".tmp") // the write opens it, and goes on once this does
		if err != nil {
			t.Error(err)
			return
		}
		io.Copy(io.Discard, f)
		f.Close()
	}
	// Runs first of the cleanups: a write blocked on the pipe holds the
	// main thread, in code that does it there, and the others wait for it.
	t.Cleanup(func() {
		if _, err := os.Stat(file + ".tmp"); err == nil { // not renamed away by a write that went on
			release()
		}
	})
	return file, release
}

// TestWindowStatesAreWrittenOffTheMainThread: a window state that cannot be
// written for seconds (a file in use, a slow disk) stood the interface for
// as long: the write was done by the main thread.
func TestWindowStatesAreWrittenOffTheMainThread(t *testing.T) {
	file, release := blockedStates(t)
	returned := make(chan struct{})
	go func() { onMain(saveWindowStates); close(returned) }()
	wait.For(t, "the main thread to be free while the file is blocked", returned)
	release()
	flushAppWrites()
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the states were not written once the file was free: %v", err)
	}
}

// TestSaveAppFilesWaitsForTheWrites: quitting writes the last states before
// the process ends, off the main thread as they are.
func TestSaveAppFilesWaitsForTheWrites(t *testing.T) {
	file, release := blockedStates(t)
	saved := make(chan struct{})
	go func() { onMain(saveAppFiles); close(saved) }()
	release()
	wait.For(t, "the quit to save", saved)
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("saveAppFiles returned before the states were written: %v", err)
	}
}
