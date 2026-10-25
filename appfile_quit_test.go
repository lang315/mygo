//go:build !js

package mygo

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// slowStates is a store written off the main thread, like the file: its
// first save waits for the test and fails, as a write that Windows keeps
// retrying does; the others succeed.
type slowStates struct {
	stateFile
	started chan struct{}
	proceed chan struct{}
	saves   chan map[string]WindowState
	calls   int // the writer's goroutine only
}

func (s *slowStates) SaveWindowStates(states map[string]WindowState) error {
	s.calls++
	if s.calls == 1 {
		close(s.started)
		<-s.proceed
		return errors.New("in use")
	}
	s.saves <- states
	return nil
}

// TestQuitWritesAgainTheStatesOfAWriteStillRunning: the write started when
// the last window closed is still running when the quit begins, and fails
// after: the quit must wait for it before it saves, or it finds nothing to
// save again.
func TestQuitWritesAgainTheStatesOfAWriteStillRunning(t *testing.T) {
	store := &slowStates{started: make(chan struct{}), proceed: make(chan struct{}), saves: make(chan map[string]WindowState, 2)}
	App.SetWindowStateStore(store)
	forget := func() { windowStates.loaded, windowStates.byKey, windowStates.dirty = false, nil, false }
	t.Cleanup(func() {
		select {
		case <-store.proceed:
		default:
			close(store.proceed)
		}
		flushAppWrites()
		App.SetWindowStateStore(nil)
		onMain(forget)
	})
	want := WindowState{X: 1, Y: 2, Width: 300, Height: 200}
	onMain(func() {
		windowStates.loaded = true
		windowStates.byKey = map[string]WindowState{"main": want}
		windowStates.dirty = true
		saveWindowStates() // the last window closed
	})
	wait.For(t, "the first write to start", store.started)
	quit := make(chan struct{})
	go func() { onMain(saveAppFiles); close(quit) }()
	// The write fails once the quit is waiting for it, in the first wait of
	// saveAppFiles, or, without it, in the last.
	wait.Until(t, "the quit to wait for the write", func() bool {
		buf := make([]byte, 1<<20)
		return strings.Contains(string(buf[:runtime.Stack(buf, true)]), "mygo.flushAppWrites")
	})
	close(store.proceed)
	wait.For(t, "the quit to finish", quit)
	select {
	case got := <-store.saves:
		if got["main"] != want {
			t.Errorf("saved %+v, want %+v", got["main"], want)
		}
	default:
		t.Error("the quit did not write the states again after the write it waited for failed")
	}
}

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
