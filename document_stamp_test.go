//go:build !js

package mygo

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

// replaceKeepingStamp replaces the file at path by another of the same size
// and modification time, as cp -p and rsync -t do.
func replaceKeepingStamp(t *testing.T, path, content string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(content)) != info.Size() {
		t.Fatalf("%q is not %d bytes", content, info.Size())
	}
	other := path + ".other"
	if err := os.WriteFile(other, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(other, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	// Windows refuses a rename over a file for an instant while anything
	// looks at it, even for its attributes: the document's own check of its
	// file does, and so would an app syncing or indexing it. Apps that replace
	// files try again, as update.Rename does.
	if err := update.Rename(other, path); err != nil {
		t.Fatal(err)
	}
}

func TestStampTellsAFileReplacedWithTheSameTimeAndSize(t *testing.T) {
	path := writeTemp(t, "a.txt", "mine")
	before, ok, err := statStamp(path)
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	if !before.file.ok {
		t.Fatal("a file in the temporary directory has no identity")
	}
	again, _, _ := statStamp(path)
	if !before.same(again) {
		t.Error("a file nobody touched is not the same")
	}
	replaceKeepingStamp(t, path, "ours")
	after, _, _ := statStamp(path)
	if after.ModTime != before.ModTime || after.Size != before.Size {
		t.Fatalf("the test changed the stamp: %v, %v", before, after)
	}
	if before.same(after) {
		t.Error("a file replaced by another of the same time and size is the same")
	}
}

func TestStampWithoutIdentityComparesTimeAndSize(t *testing.T) {
	a := fileStamp{ModTime: 5, Size: 4, file: fileWatchGone{id: fileWatchID{volume: 1, index: 2}, ok: true}}
	b := fileStamp{ModTime: 5, Size: 4, file: fileWatchGone{id: fileWatchID{volume: 1, index: 3}, ok: true}}
	if a.same(b) {
		t.Error("two files of known, different identities are the same")
	}
	b.file.ok = false // a volume that doesn't keep them
	if !a.same(b) || !b.same(a) {
		t.Error("an identity one stamp lacks tells them apart")
	}
	a.file.ok = false
	if !a.same(b) {
		t.Error("stamps without identities of the same time and size are not the same")
	}
	if b.Size = 5; a.same(b) {
		t.Error("stamps of different sizes are the same")
	}
}

func TestLocalVolume(t *testing.T) {
	if !localVolume(t.TempDir()) {
		t.Error("the temporary directory is not on a local volume")
	}
	if runtime.GOOS != "windows" && localVolume(filepath.Join(t.TempDir(), "missing", "file")) {
		t.Error("a path the system cannot look at is on a local volume")
	}
}

// A document edited here, whose file another app replaced by one of the
// same time and size, is not written over it.
func TestDocumentSaveOverAFileReplacedWithTheSameTimeAndSize(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "same.txt", "mine")
	d, _ := ds.Open(path)
	d.Set("mine again")
	replaceKeepingStamp(t, path, "ours")
	if _, err := d.write(path, writeSave); !errors.Is(err, ErrDocumentChanged) {
		t.Errorf("write over a replaced file: %v", err)
	}
	if got := readString(t, path); got != "ours" {
		t.Errorf("the file has %q, want ours", got)
	}
	if !d.HasConflict() {
		t.Error("no conflict")
	}
}

// What the document wrote itself, replacing its file as saving does, is not
// a change by another app.
func TestDocumentSavesInTurnWithoutConflict(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "turn.txt", "mine")
	d, _ := ds.Open(path)
	for _, text := range []string{"one", "two", "three"} {
		d.Set(text)
		if err := d.Save(); err != nil {
			t.Fatalf("saving %q: %v", text, err)
		}
		d.checkDisk()
		if d.HasConflict() || d.State().Deleted {
			t.Fatalf("after saving %q: conflict %v, deleted %v", text, d.HasConflict(), d.State().Deleted)
		}
		if got := readString(t, path); got != text {
			t.Fatalf("the file has %q, want %q", got, text)
		}
	}
}
