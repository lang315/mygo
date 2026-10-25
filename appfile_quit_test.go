//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestQuitWritesAgainTheStatesAWriteFailedToSave: the write of the states
// that fails after the last window closed, while the app is quitting, is
// tried again by the quit: the main thread is not free for a message of the
// writer by then.
func TestQuitWritesAgainTheStatesAWriteFailedToSave(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, windowStateFile)
	App.SetPath(PathUserData, dir)
	forget := func() { windowStates.loaded, windowStates.byKey, windowStates.dirty = false, nil, false }
	t.Cleanup(func() {
		flushAppWrites()
		onMain(forget)
		App.SetPath(PathUserData, "")
	})
	if err := os.Mkdir(file+".tmp", 0o755); err != nil { // no temporary file can be written
		t.Fatal(err)
	}
	onMain(func() {
		windowStates.loaded = true
		windowStates.byKey = map[string]WindowState{"main": {X: 1, Y: 2, Width: 300, Height: 200}}
		windowStates.dirty = true
		saveWindowStates() // the last window closed
		flushAppWrites()   // the write failed, and the main thread has not run anything since
		if err := os.Remove(file + ".tmp"); err != nil {
			t.Error(err)
		}
		saveAppFiles() // the quit
	})
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the quit did not write the states again: %v", err)
	}
}

// TestQuitWritesWhatOnQuitListenersAsk: a listener of OnQuit may record a
// recent document or set the language, which is written off the main thread:
// the process must not end before.
func TestQuitWritesWhatOnQuitListenersAsk(t *testing.T) {
	testRecent(t)
	path := filepath.Join(t.TempDir(), "todo.txt")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	off := App.OnQuit(func() {
		if err := App.AddRecentDocument(path); err != nil {
			t.Error(err)
		}
	})
	defer off()
	onMain(App.saveAndFireQuit)
	file, err := recentFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the quit ended before what a listener asked to write was written: %v", err)
	}
}
