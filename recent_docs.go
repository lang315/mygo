package mygo

import (
	"encoding/json/v2"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
)

// The recent documents: those the app opened lately, newest first, which
// RecentDocuments returns and the Open Recent menu lists, kept in
// recent-documents.json in PathUserData. AddRecentDocument records them
// with the system too (the Dock menu on macOS, the jump list on Windows,
// the recently used files on Linux), as NSDocumentController does.
var recentDocs struct {
	sync.Mutex
	loaded bool
	paths  []string
	// failed tells that the last write of the file failed: the list is
	// written again at quit.
	failed atomic.Bool
}

// recentWrite writes the file of the recent documents; tests replace it.
var recentWrite = writeAppFile

// maxRecentDocuments is how many documents the list keeps, as
// NSDocumentController's maximumRecentDocumentCount.
const maxRecentDocuments = 10

func recentFile() (string, error) {
	dir, err := App.Path(PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "recent-documents.json"), nil
}

// loadRecentLocked reads the list once. recentDocs is locked.
func loadRecentLocked() {
	if recentDocs.loaded {
		return
	}
	recentDocs.loaded = true
	if f, err := recentFile(); err == nil {
		if data, err := readAppFile(f); err == nil {
			_ = json.Unmarshal(data, &recentDocs.paths)
		}
	}
}

// AddRecentDocument records path as a document the user opened: it goes
// first in RecentDocuments and the Open Recent menu (RoleOpenRecent), and
// the system offers it again, in the app's Dock menu on macOS, in the
// recently used files of the user (the file chooser's and file manager's
// Recent) on Linux, and in the Recent category of the app's jump list on
// Windows. Documents (NewDocuments) record the files they open and save;
// other apps call it whenever they open a document or folder, including
// one the user chose with a file dialog:
//
//	if err := mygo.App.AddRecentDocument(path); err != nil { … }
//
// The list is written to its file off the calling thread, soon, and before
// the app quits; a failure of that write is logged, not returned, and the
// list is written again at quit. An app
// that ends the process itself (os.Exit, log.Fatal) right after may lose
// the last write: it should use App.Exit or App.Relaunch. A
// relative path is resolved against the working directory. Choosing an
// entry of the Dock menu, the jump list or the Open Recent menu opens the
// file with the app, which gets it through OnOpenFile, or opens it as a
// document. macOS 15 records nothing for an app outside an app bundle, as
// one started with go run.
func (a *Application) AddRecentDocument(path string) error {
	needsApp("App.AddRecentDocument")
	if path == "" {
		return errors.New("mygo: App.AddRecentDocument needs a path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return addRecentDocument(abs)
}

// addRecentDocument puts the absolute path first in the recent documents
// and records it with the system.
func addRecentDocument(path string) error {
	recentDocs.Lock()
	loadRecentLocked()
	paths := slices.DeleteFunc(slices.Clone(recentDocs.paths), func(p string) bool { return p == path })
	paths = slices.Insert(paths, 0, path)
	if len(paths) > maxRecentDocuments {
		paths = paths[:maxRecentDocuments]
	}
	recentDocs.paths = paths
	err := saveRecentLocked()
	recentDocs.Unlock()
	return errors.Join(err, onMainValue(func() error { return backend().App().AddRecentDocument(path) }))
}

// ClearRecentDocuments empties RecentDocuments and the Open Recent menu,
// and removes the documents the app recorded with the system: the app's
// recent documents on macOS, the files the app recorded on Linux (the
// user's other recent files stay), and the Recent category of the app's
// jump list on Windows. The Open Recent menu's Clear Menu calls it.
func (a *Application) ClearRecentDocuments() error {
	needsApp("App.ClearRecentDocuments")
	recentDocs.Lock()
	recentDocs.loaded = true
	recentDocs.paths = nil
	err := saveRecentLocked()
	recentDocs.Unlock()
	return errors.Join(err, onMainValue(func() error { return backend().App().ClearRecentDocuments() }))
}

// RecentDocuments returns the documents the app opened lately, newest
// first, which the Open Recent menu (RoleOpenRecent) lists: those of
// AddRecentDocument, also of earlier runs, up to ten. As in AppKit, files
// that no longer exist are left out, and come back if they do again.
func (a *Application) RecentDocuments() []string {
	recentDocs.Lock()
	loadRecentLocked()
	paths := slices.Clone(recentDocs.paths)
	recentDocs.Unlock()
	return slices.DeleteFunc(paths, func(p string) bool {
		_, err := os.Stat(p)
		return errors.Is(err, os.ErrNotExist)
	})
}

// saveRecentLocked writes the recent documents and updates the Open
// Recent menus. recentDocs is locked, so that writes keep the order of
// the changes.
func saveRecentLocked() error {
	postMain(updateRecentMenus)
	f, err := recentFile()
	if err != nil {
		return err
	}
	writeRecentLocked(f)
	return nil
}

// writeRecentLocked has the file f written off the main thread, in the order
// of the changes (the latest one replaces a write not begun); a failure is
// logged, and the list is written again at quit (retryRecent) or by the next
// change. recentDocs is locked.
func writeRecentLocked(f string) {
	paths := slices.Clone(recentDocs.paths)
	writeAppFileLater("recent-documents", func() {
		err := recentWrite(f, func(w io.Writer) error { return json.MarshalWrite(w, paths) })
		recentDocs.failed.Store(err != nil)
		if err != nil {
			log.Printf("mygo: recent documents: %v", err)
		}
	})
}

// retryRecent writes the recent documents again if the last write failed, and
// waits for it. Main thread only, at quit.
func retryRecent() {
	if !recentDocs.failed.Load() {
		return
	}
	f, err := recentFile()
	if err != nil {
		return
	}
	recentDocs.Lock()
	writeRecentLocked(f)
	recentDocs.Unlock()
	flushAppWrites()
}

// openRecentDocument opens a document of the Open Recent menu: as a
// document of the kind of its extension, or the first kind, or else
// through OnOpenFile, as the system opens those of the Dock menu and the
// jump list. Main thread only.
func openRecentDocument(path string) {
	_, open := linkedDocuments()
	if open == nil {
		fire1(&App.onOpenFile, path)
		return
	}
	go open(path)
}

// addRecent records a document's file, as AddRecentDocument does, and logs
// what fails.
func addRecent(path string) {
	if err := addRecentDocument(path); err != nil {
		log.Printf("mygo: documents: %v", err)
	}
}
