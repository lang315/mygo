//go:build !js

package mygo

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// A write of the recent documents that fails (a file in use on Windows) is
// not lost: the list is written again when the app quits, though nothing
// changed it since.
func TestRecentDocumentsFailedWriteIsWrittenAtQuit(t *testing.T) {
	dir := t.TempDir()
	App.SetPath(PathUserData, dir)
	t.Cleanup(func() { flushAppWrites(); App.SetPath(PathUserData, "") })
	recentDocs.Lock()
	recentDocs.loaded, recentDocs.paths = true, nil
	recentDocs.Unlock()
	t.Cleanup(func() {
		recentDocs.Lock()
		recentDocs.loaded, recentDocs.paths = false, nil
		recentDocs.Unlock()
	})

	var calls atomic.Int32
	was := recentWrite
	recentWrite = func(path string, write func(io.Writer) error) error {
		if calls.Add(1) == 1 {
			return errors.New("the file is in use")
		}
		return was(path, write)
	}
	t.Cleanup(func() { recentWrite = was })

	doc := filepath.Join(dir, "a.txt")
	os.WriteFile(doc, []byte("x"), 0o644)
	if err := App.AddRecentDocument(doc); err != nil {
		t.Fatal(err)
	}
	flushAppWrites()
	file := filepath.Join(dir, "recent-documents.json")
	if _, err := os.Stat(file); err == nil {
		t.Fatal("the first write failed and the file exists")
	}
	onMain(saveAppFiles) // quit
	if b, err := os.ReadFile(file); err != nil || !strings.Contains(string(b), doc) {
		t.Fatalf("after quit the file has %q, %v: the failed write was lost", b, err)
	}
	if calls.Load() != 2 {
		t.Errorf("%d writes, want the failed one and one retry", calls.Load())
	}
	// Written, it is not written again at the next quit.
	onMain(saveAppFiles)
	if calls.Load() != 2 {
		t.Errorf("%d writes after a second quit, want 2", calls.Load())
	}
}
