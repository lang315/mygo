package mygo

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Documents notice what other apps do to their files: one WatchFiles per
// open document's file, and a look when a document's window is focused,
// since watches miss changes (VS Code and Notepad++ look then too). An
// unedited document reads its file again, an edited one is in conflict
// and asks at its next save, a file renamed in its directory is followed,
// and a deleted one is written again by the next save. A watch of the
// file rather than of its directory costs the same whatever else the
// directory holds: a directory is one FSEvents stream on macOS, whatever
// is in it.

var docWatches struct {
	sync.Mutex
	files map[string]*docFileWatch
}

type docFileWatch struct {
	refs   int
	cancel context.CancelFunc
}

// watch watches b's file, after it opened or moved.
func (b *docBase) watch() {
	b.mu.Lock()
	path := ""
	if !b.closed {
		path = b.path
	}
	old := b.watched
	b.watched = path
	b.mu.Unlock()
	if path == old {
		return
	}
	docWatches.Lock()
	defer docWatches.Unlock()
	if old != "" {
		if w := docWatches.files[old]; w != nil {
			if w.refs--; w.refs == 0 {
				w.cancel()
				delete(docWatches.files, old)
			}
		}
	}
	if path == "" {
		return
	}
	if w := docWatches.files[path]; w != nil {
		w.refs++
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if docWatches.files == nil {
		docWatches.files = map[string]*docFileWatch{}
	}
	docWatches.files[path] = &docFileWatch{refs: 1, cancel: cancel}
	docWork.Go(func() { watchDocumentFile(ctx, path) }) // tests wait for it, so that no watch starts in the next test
}

// watchDocumentFile follows the changes of the file at path until ctx is
// done, or its directory goes away.
func watchDocumentFile(ctx context.Context, path string) {
	w, err := WatchFiles(ctx, path, FileWatchOptions{Debounce: 100 * time.Millisecond})
	if err != nil {
		// The file is gone, perhaps renamed before this watch started: a
		// look finds it, or tells it deleted. Focus checks remain.
		for _, b := range documentsAt(path) {
			go b.checkDisk()
		}
		return
	}
	defer w.Close()
	// What changed before the watch started, as after a rename.
	for _, b := range documentsAt(path) {
		go b.checkDisk()
	}
	followDocumentFile(ctx, w, path)
}

// followDocumentFile tells the documents of the file at path what w reports
// of it, until ctx is done or the file is renamed.
func followDocumentFile(ctx context.Context, w *FileWatcher, path string) {
	for {
		r := w.next(ctx)
		if r.err != nil {
			return
		}
		// The file that went away, by what it was when the watch last saw
		// it, as a look now may come after the next rename, or else as the
		// document last read or wrote it: the watch looks once changes
		// settle, so a file that replaced the one it saw, which the
		// document read (when the watch started, or its window was
		// focused), can be renamed before the watch ever saw it.
		docs := documentsAt(path)
		moved := ""
		if r.events[len(r.events)-1].Op == FileRemove {
			ids := []fileWatchGone{r.gone}
			for _, b := range docs {
				// A document in conflict read a file another replaced,
				// whose identity a new file may have taken since.
				b.mu.Lock()
				if !b.conflict && !b.deleted {
					ids = append(ids, b.stamp.file)
				}
				b.mu.Unlock()
			}
			moved = findRenamed(filepath.Dir(w.Root()), ids)
		}
		for _, b := range docs {
			if moved != "" {
				b.moved(moved)
			} else {
				go b.checkDisk()
			}
		}
		if moved != "" {
			return // moved watches the new path
		}
	}
}

// findMoved returns the path in dir of the file b last read or wrote, which
// another app renamed there, or "". Its identity finds it, and its size and
// modification time, which a rename keeps, make sure it is the same file: the
// identity of a file deleted may be given to one made later.
func (b *docBase) findMoved(dir string) string {
	b.mu.Lock()
	stamp := b.stamp
	b.mu.Unlock()
	if !stamp.file.ok {
		return ""
	}
	return findFile(dir, func(info os.FileInfo, id fileWatchID) bool {
		return id == stamp.file.id && stamp.same(stampOf(info, dir))
	})
}

// findFile returns the path of the first regular file in dir that match
// accepts, given its info and identity, or "".
func findFile(dir string, match func(info os.FileInfo, id fileWatchID) bool) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if other := identityOf(info); other.ok && match(info, other.id) {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

// findRenamed returns the path of the file in dir that one of ids
// identifies, the first of ids that one does, or "".
func findRenamed(dir string, ids []fileWatchGone) string {
	if !slices.ContainsFunc(ids, func(id fileWatchGone) bool { return id.ok }) {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	found, at := "", len(ids)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		id := fileWatchEntryOf(info).id
		for i, want := range ids[:at] {
			if want.ok && want.id == id {
				found, at = filepath.Join(dir, entry.Name()), i
				break
			}
		}
	}
	return found
}

// documentsAt returns the open documents of the file at path.
func documentsAt(path string) []*docBase {
	var at []*docBase
	for _, b := range allDocuments() {
		if b.Path() == path {
			at = append(at, b)
		}
	}
	return at
}

// afterDiskLook, set by tests, runs in checkDisk once it looked at the file
// and before it decides what to do: the time in which the document changes.
// It is told whether the file differs from what the document last read or
// wrote, and returns a function to call when the look ends, or nil.
var afterDiskLook atomic.Pointer[func(changed bool) func()]

// checkDisk compares b's file with what b last read or wrote.
func (b *docBase) checkDisk() {
	b.io.Lock()
	b.mu.Lock()
	path, known, edited, closed := b.path, b.stamp, b.edited, b.closed
	wasDeleted := b.deleted
	b.mu.Unlock()
	if path == "" || closed {
		b.io.Unlock()
		return
	}
	now, exists, err := statStamp(path)
	if err == nil && !exists && !wasDeleted {
		// Network shares report files gone that are not (VS Code).
		time.Sleep(100 * time.Millisecond)
		now, exists, err = statStamp(path)
	}
	if err != nil {
		b.io.Unlock()
		return
	}
	if f := afterDiskLook.Load(); f != nil {
		if done := (*f)(exists && !now.same(known)); done != nil {
			defer done()
		}
	}
	switch {
	case !exists:
		// Renamed in its directory by an app whose rename the watch missed or
		// started after: it is where the file it read or wrote went.
		if to := b.findMoved(filepath.Dir(path)); to != "" {
			b.io.Unlock()
			b.moved(to)
			return
		}
		b.mu.Lock()
		changed := !b.deleted
		b.deleted = true
		b.mu.Unlock()
		b.io.Unlock()
		if changed {
			b.notify()
		}
	case now.same(known) && !wasDeleted:
		readOnly := !fileWritable(path)
		b.mu.Lock()
		changed := b.readOnly != readOnly
		b.readOnly = readOnly
		b.mu.Unlock()
		b.io.Unlock()
		if changed {
			b.notify()
		}
	case !edited:
		b.io.Unlock()
		_ = b.reloadIf(path, true)
	default:
		b.mu.Lock()
		changed := !b.conflict || b.deleted
		b.conflict, b.deleted = true, false
		b.mu.Unlock()
		b.io.Unlock()
		if changed {
			b.notify()
		}
	}
}

// moved follows b's file, which another app renamed to path.
func (b *docBase) moved(path string) {
	b.mu.Lock()
	b.path = path
	// A look that found the file gone may have taken it for deleted before
	// one found it where it went.
	b.deleted = false
	b.mu.Unlock()
	addRecent(path)
	b.watch()
	b.notify()
}
