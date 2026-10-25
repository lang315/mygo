//go:build !js

package mygo

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/internal/wait"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// testDocs makes documents of plain text, .txt, in their own user data,
// and closes their windows when the test ends.
func testDocs(t *testing.T, opts DocumentsOptions[string]) *Documents[string] {
	t.Helper()
	App.SetPath(PathUserData, t.TempDir())
	t.Cleanup(func() { App.SetPath(PathUserData, "") })
	if opts.Types == nil {
		opts.Types = []DocumentType{{Name: "Text " + t.Name(), Extensions: []string{"txt"}}}
	}
	opts.NoUntitled = true
	opts.Read = func(r io.Reader) (string, error) { b, err := io.ReadAll(r); return string(b), err }
	if opts.Write == nil {
		opts.Write = func(w io.Writer, s string) error { _, err := io.WriteString(w, s); return err }
	}
	opts.Window = func(doc *Document[string]) *Window { return NewWindow(WindowOptions{}) }
	ds := NewDocuments(opts)
	t.Cleanup(func() {
		for _, d := range ds.Documents() {
			d.SetEdited(false)
			if w := d.Window(); w != nil {
				w.Destroy()
			}
		}
		docWork.Wait()
		// Windows deletes no file a process holds open: the session's lock.
		ds.kind.recovery.close()
		onMain(func() { fb.OnMessageBox, fb.OnSaveDialog, fb.OnOpenDialog = nil, nil, nil })
	})
	return ds
}

// testDocsSettled is testDocs, once the restores the app begins with
// (registerDocKind, startDocuments) are over: a test that makes a session on
// disk and claims it would have one of them take it first.
func testDocsSettled(t *testing.T, opts DocumentsOptions[string]) *Documents[string] {
	t.Helper()
	ds := testDocs(t, opts)
	waitFor(t, "the documents to launch", func() bool {
		docRegistry.Lock()
		defer docRegistry.Unlock()
		return docRegistry.launched
	})
	docWork.Wait()
	return ds
}

// fakeWindow returns the fake native window of w.
func fakeWindow(w *Window) *fake.Window {
	return onMainValue(func() *fake.Window { return w.native.(*fake.Window) })
}

// waitFor waits until cond holds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	wait.Until(t, what, cond)
}

// asked records the messages of the message boxes shown.
type asked struct {
	mu   sync.Mutex
	list []string
}

func (a *asked) all() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.list)
}

// answer makes message boxes answer with the button labeled label, and
// records their messages.
func answer(label string) *asked {
	a := &asked{}
	onMain(func() {
		fb.OnMessageBox = func(o *platform.MessageBoxOptions) platform.MessageBoxResult {
			a.mu.Lock()
			a.list = append(a.list, o.Message)
			a.mu.Unlock()
			return platform.MessageBoxResult{Response: slices.Index(o.Buttons, label)}
		}
	})
	return a
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readString(t *testing.T, path string) string {
	t.Helper()
	b, err := readAppFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// fileIs reports whether the file at path holds want.
func fileIs(path, want string) bool {
	b, err := readAppFile(path)
	return err == nil && string(b) == want
}

func TestDocumentOpen(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff, Equal: func(a, b string) bool { return a == b }})
	path := writeTemp(t, "notes.txt", "hello")
	d, err := ds.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Content() != "hello" || d.Path() != path || d.Name() != "notes.txt" || d.IsEdited() {
		t.Fatalf("opened %q %q %q edited=%v", d.Content(), d.Path(), d.Name(), d.IsEdited())
	}
	w := d.Window()
	if ds.ForWindow(w) != d {
		t.Error("ForWindow")
	}
	fw := fakeWindow(w)
	want := "notes.txt"
	if runtime.GOOS != "darwin" {
		want += " - MyGoTest"
	}
	waitFor(t, "the window's title", func() bool { p, _ := fw.Document(); return w.Title() == want && p == path })

	// Opening it again focuses it.
	again, err := ds.Open(path)
	if err != nil || again != d || len(ds.Documents()) != 1 {
		t.Fatalf("opened again: %v, %d documents", err, len(ds.Documents()))
	}

	d.Set("hello!")
	waitFor(t, "the edited dot", func() bool { _, e := fw.Document(); return e })
	if !d.IsEdited() || !w.IsDocumentEdited() {
		t.Fatal("not edited")
	}
	if runtime.GOOS != "darwin" {
		if got := fw.Title(); got != "*"+want {
			t.Errorf("native title %q, want the mark", got)
		}
		if w.Title() != want {
			t.Errorf("Title() = %q, want it without the mark", w.Title())
		}
	}
	// Equal: back to what the file has is unedited.
	d.Set("hello")
	waitFor(t, "the dot to go", func() bool { _, e := fw.Document(); return !e })
	if d.IsEdited() {
		t.Error("edited after setting the saved content again")
	}
	if !slices.Contains(App.RecentDocuments(), path) {
		t.Errorf("recent documents %q lack %s", App.RecentDocuments(), path)
	}
}

func TestDocumentSave(t *testing.T) {
	release := make(chan struct{})
	writing := make(chan struct{}, 1)
	var block bool
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Write: func(w io.Writer, s string) error {
			if block {
				writing <- struct{}{}
				<-release
			}
			_, err := io.WriteString(w, s)
			return err
		},
	})
	path := writeTemp(t, "a.txt", "one")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	d, _ := ds.Open(path)
	d.Set("two")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "two" || d.IsEdited() {
		t.Fatalf("saved %q, edited %v", readString(t, path), d.IsEdited())
	}
	if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Errorf("permissions %v after saving", info.Mode().Perm())
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("files left next to the document: %v", entries)
	}

	// Edits made while it is written stay unsaved.
	block = true
	d.Set("three")
	done := make(chan error)
	go func() { done <- d.Save() }()
	<-writing
	d.Set("four")
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "three" || !d.IsEdited() {
		t.Fatalf("file %q, edited %v: the edit made while writing was lost", readString(t, path), d.IsEdited())
	}
}

// A system that removes the file and can't put the new one in its place
// (ReplaceFileW's ERROR_UNABLE_TO_MOVE_REPLACEMENT) leaves the new contents
// only in the sibling: the save keeps it and names it in its error. When the
// file is still there, the sibling goes as before.
func TestDocumentSaveKeepsTheNewContentsWhenReplacingFails(t *testing.T) {
	for _, removed := range []bool{true, false} {
		t.Run(fmt.Sprintf("removed=%v", removed), func(t *testing.T) {
			ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
			path := writeTemp(t, "a.txt", "one")
			d, _ := ds.Open(path)
			d.Set("two")
			onMain(func() {
				fb.ReplaceFileHook = func(target, replacement string) error {
					if removed {
						if err := os.Remove(target); err != nil {
							return err
						}
					}
					return errors.New("cannot move the replacement")
				}
			})
			err := d.Save()
			onMain(func() { fb.ReplaceFileHook = nil })
			entries, _ := os.ReadDir(filepath.Dir(path))
			if err == nil || len(entries) != 1 {
				t.Fatalf("save error %v, files %v", err, entries)
			}
			left := filepath.Join(filepath.Dir(path), entries[0].Name())
			// The error names the sibling by the file's real path, which on
			// Windows may spell a temporary directory's 8.3 name in full.
			if removed && (readString(t, left) != "two" || !strings.Contains(err.Error(), entries[0].Name())) {
				t.Errorf("kept %s with %q; error %v", left, readString(t, left), err)
			}
			if !removed && (left != path || readString(t, path) != "one") {
				t.Errorf("left %s with %q; error %v", left, readString(t, left), err)
			}
		})
	}
}

// A sibling that could not be written goes, even when the file went
// meanwhile: it holds no new contents to keep.
func TestDocumentSaveDropsASiblingNotWritten(t *testing.T) {
	var path string
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Write: func(w io.Writer, s string) error {
			if err := os.Remove(path); err != nil {
				return err
			}
			return errors.New("cannot write")
		},
	})
	path = writeTemp(t, "a.txt", "one")
	d, _ := ds.Open(path)
	d.Set("two")
	err := d.Save()
	entries, _ := os.ReadDir(filepath.Dir(path))
	if err == nil || len(entries) != 0 {
		t.Fatalf("save error %v, files %v", err, entries)
	}
}

func TestDocumentSaveKeepsLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hard links and symlinks need privileges")
	}
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	dir := t.TempDir()
	target := writeTemp(t, "target.txt", "x")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(dir, "hard.txt")
	if err := os.Link(target, hard); err != nil {
		t.Fatal(err)
	}
	d, _ := ds.Open(link)
	d.Set("y")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced by a file")
	}
	if readString(t, target) != "y" || readString(t, hard) != "y" {
		t.Errorf("target %q, hard link %q", readString(t, target), readString(t, hard))
	}
}

func TestDocumentAutosave(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{AutosaveDelay: 20 * time.Millisecond})
	path := writeTemp(t, "auto.txt", "a")
	d, _ := ds.Open(path)
	d.Set("b")
	waitFor(t, "the autosave", func() bool { return fileIs(path, "b") && !d.IsEdited() })

	// A saved document closes without asking, saving first.
	asked := answer("Cancel")
	d.Set("c")
	d.Close()
	waitFor(t, "the window to close", func() bool { return d.Window() == nil })
	if readString(t, path) != "c" || len(asked.all()) != 0 {
		t.Errorf("file %q, asked %q", readString(t, path), asked.all())
	}
}

func TestDocumentUntitledClose(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{})
	d1, d2 := ds.New(), ds.New()
	if d1.Name() != "Untitled" || d2.Name() != "Untitled 2" || d1.Path() != "" {
		t.Fatalf("names %q %q", d1.Name(), d2.Name())
	}
	d1.Set("draft")
	asked := answer("Cancel")
	d1.Close()
	waitFor(t, "the question", func() bool { return len(asked.all()) == 1 })
	if asked.all()[0] != "Do you want to keep this new document “Untitled”?" {
		t.Errorf("asked %q", asked.all()[0])
	}
	time.Sleep(20 * time.Millisecond)
	if d1.Window() == nil {
		t.Fatal("Cancel closed the window")
	}
	answer("Delete")
	d1.Close()
	waitFor(t, "the window to close", func() bool { return d1.Window() == nil })
	// An unedited one closes at once.
	d2.Close()
	waitFor(t, "the other window to close", func() bool { return d2.Window() == nil })
}

func TestDocumentCloseAsks(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "ask.txt", "1")
	d, _ := ds.Open(path)
	d.Set("2")
	asked := answer("Save")
	d.Close()
	waitFor(t, "the window to close", func() bool { return d.Window() == nil })
	if readString(t, path) != "2" || len(asked.all()) != 1 || !strings.Contains(asked.all()[0], "“ask.txt”") {
		t.Errorf("file %q, asked %q", readString(t, path), asked.all())
	}

	// Saving an untitled document asks where.
	u := ds.New()
	u.Set("new")
	target := filepath.Join(t.TempDir(), "named")
	onMain(func() { fb.OnSaveDialog = func(*platform.SaveDialogOptions) string { return target } })
	answer("Save…")
	u.Close()
	waitFor(t, "the untitled window to close", func() bool { return u.Window() == nil })
	if readString(t, target+".txt") != "new" {
		t.Error("the untitled document was not saved with its extension")
	}
}

// TestDocumentEditedWhileTheDiskIsChecked: a look at the file that began
// before the user edited the document must not replace the edit with the
// file's content, which is what the watch's look does when the file changes
// as the user types.
func TestDocumentEditedWhileTheDiskIsChecked(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "typing.txt", "mine")
	d, _ := ds.Open(path)
	// Every look at the file waits for the edit, those of the watch too.
	looked, edited := make(chan struct{}, 16), make(chan struct{})
	hold := func() {
		looked <- struct{}{}
		<-edited
	}
	afterDiskLook.Store(&hold)
	t.Cleanup(func() { afterDiskLook.Store(nil) })
	future := time.Now().Add(time.Hour)
	os.WriteFile(path, []byte("theirs"), 0o644)
	os.Chtimes(path, future, future)
	done := make(chan struct{})
	go func() { d.checkDisk(); close(done) }()
	wait.For(t, "the look at the file", looked)
	d.Set("mine again") // the document was not edited when the looks began
	close(edited)
	wait.For(t, "the look to end", done)
	if d.Content() != "mine again" || !d.IsEdited() {
		t.Errorf("content %q, edited %v: the user's edit was replaced", d.Content(), d.IsEdited())
	}
	if !d.HasConflict() {
		t.Error("the file changed under the edit and the document is not in conflict")
	}
}

func TestDocumentOutsideChanges(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "out.txt", "mine")
	d, _ := ds.Open(path)
	reloaded := make(chan struct{}, 1)
	d.OnReload(func() { reloaded <- struct{}{} })

	// Unedited: read again.
	future := time.Now().Add(time.Hour)
	os.WriteFile(path, []byte("theirs"), 0o644)
	os.Chtimes(path, future, future)
	d.checkDisk()
	<-reloaded
	if d.Content() != "theirs" || d.IsEdited() {
		t.Fatalf("content %q edited %v", d.Content(), d.IsEdited())
	}

	// Edited: in conflict, which saving asks about.
	d.Set("mine again")
	os.WriteFile(path, []byte("theirs again"), 0o644)
	os.Chtimes(path, future.Add(time.Hour), future.Add(time.Hour))
	d.checkDisk()
	if !d.HasConflict() || d.Content() != "mine again" {
		t.Fatalf("conflict %v, content %q", d.HasConflict(), d.Content())
	}
	asked := answer("Save Anyway")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "mine again" || d.HasConflict() || len(asked.all()) != 1 {
		t.Errorf("file %q, conflict %v, asked %q", readString(t, path), d.HasConflict(), asked.all())
	}

	// Without asking, Save reports the change.
	os.Chtimes(path, future.Add(2*time.Hour), future.Add(2*time.Hour))
	d.Set("x")
	if _, err := d.write(path, writeSave); !errors.Is(err, ErrDocumentChanged) {
		t.Errorf("write over a changed file: %v", err)
	}

	// Deleted: saving writes it again.
	os.Remove(path)
	d.checkDisk()
	if !d.State().Deleted {
		t.Fatal("not deleted")
	}
	if err := d.Save(); err != nil || readString(t, path) != "x" {
		t.Errorf("saving a deleted document: %v", err)
	}
}

// TestDocumentRecoveryRetriesAFailedWrite is a regression test: a recovery
// copy that could not be written (a file in use on Windows, a full disk)
// was not tried again until the next edit.
func TestDocumentRecoveryRetriesAFailedWrite(t *testing.T) {
	retry := backupRetry
	backupRetry = 10 * time.Millisecond
	t.Cleanup(func() { backupRetry = retry })
	var calls atomic.Int32
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff, Write: func(w io.Writer, s string) error {
		if calls.Add(1) == 1 {
			return errors.New("the file is in use")
		}
		_, err := io.WriteString(w, s)
		return err
	}})
	d := ds.New()
	d.Set("unsaved")
	root, _ := ds.kind.recovery.root()
	waitFor(t, "the recovery copy after a failed write", func() bool {
		m, _ := filepath.Glob(filepath.Join(root, "*", d.id))
		return len(m) == 1 && fileIs(m[0], "unsaved")
	})
}

// TestDocumentRecoveryWritesNothingAfterItIsClosed: a recovery copy written
// after quit (a timer that fired late) made a session in the user data of
// whatever the app, or the next test, had by then: in a test it was the
// directory the next test removed, "directory not empty".
func TestDocumentRecoveryWritesNothingAfterItIsClosed(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	root, _ := ds.kind.recovery.root()
	ds.kind.recovery.close() // quit
	d.Set("too late")
	d.backup() // what the timer does, a second later
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("the recovery copies are written after quit: %v", err)
	}
}

// A restore after quit would take for a crashed instance's the copies of this
// one (close forgot its session), write none (the store is closed) and remove
// them: the changes of the user, lost.
func TestDocumentRecoveryRestoreAfterQuitKeepsTheCopies(t *testing.T) {
	ds := testDocsSettled(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("unsaved")
	waitFor(t, "the recovery copy", func() bool { return d.backedUpAt() != 0 })
	root, _ := ds.kind.recovery.root()
	copies := func() []string { m, _ := filepath.Glob(filepath.Join(root, "*", d.id)); return m }
	if len(copies()) != 1 {
		t.Fatalf("recovery copies %q", copies())
	}
	ds.kind.recovery.close() // quit
	if n := ds.kind.restore(); n != 0 {
		t.Fatalf("restored %d documents after quit", n)
	}
	if len(copies()) != 1 {
		t.Fatalf("the restore after quit removed the copy: %q", copies())
	}
}

// Quit in the middle of a restore: the documents it opened write no copy of
// their own, so the session it claimed stays for the next launch.
func TestDocumentRecoveryQuitDuringARestoreKeepsTheSession(t *testing.T) {
	ds := testDocsSettled(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	root, _ := ds.kind.recovery.root()
	session := filepath.Join(root, "crashed")
	if err := os.MkdirAll(session, 0o700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(session, "index.json"), []byte(`[{"id":"a","name":"Untitled 3"}]`), 0o600)
	os.WriteFile(filepath.Join(session, "a"), []byte("draft"), 0o600)
	orphans, done := ds.kind.recovery.claim()
	if len(orphans) != 1 {
		t.Fatalf("claimed %d copies", len(orphans))
	}
	ds.kind.recovery.close() // quit
	done()                   // the restore ends
	if _, err := os.Stat(filepath.Join(session, "a")); err != nil {
		t.Fatalf("the session claimed was removed at quit: %v", err)
	}
}

// Quit in the middle of a restore leaves the same copy in two sessions, the
// one claimed and the one of the instance that restored it: the next launch
// restores the document once, with the later changes.
func TestDocumentRecoveryRestoresOneCopyOfADocumentInTwoSessions(t *testing.T) {
	for _, tie := range []bool{false, true} {
		ds := testDocsSettled(t, DocumentsOptions[string]{Autosave: AutosaveOff})
		root, _ := ds.kind.recovery.root()
		base := time.Now().Add(-time.Hour)
		for i, content := range []string{"older", "newer"} {
			session := filepath.Join(root, fmt.Sprintf("session%d", i))
			if err := os.MkdirAll(session, 0o700); err != nil {
				t.Fatal(err)
			}
			index := filepath.Join(session, "index.json")
			os.WriteFile(index, []byte(`[{"id":"a","name":"Untitled 3"}]`), 0o600)
			file := filepath.Join(session, "a")
			os.WriteFile(file, []byte(content), 0o600)
			at := base.Add(time.Duration(i) * time.Minute) // session1 is the later
			if tie {
				at = base // the same time to the file system's resolution: the later index tells
			}
			os.Chtimes(file, at, at)
			os.Chtimes(index, base.Add(time.Duration(i)*time.Minute), base.Add(time.Duration(i)*time.Minute))
		}
		if n := ds.kind.restore(); n != 1 {
			t.Fatalf("tie %v: restored %d documents, want 1", tie, n)
		}
		docs := ds.Documents()
		if len(docs) != 1 || docs[0].Content() != "newer" {
			for _, d := range docs {
				t.Logf("doc %q %q", d.Name(), d.Content())
			}
			t.Fatalf("tie %v: restored %d documents", tie, len(docs))
		}
		for _, d := range docs {
			d.SetEdited(false)
			d.Close()
		}
	}
}

func TestDocumentRecovery(t *testing.T) {
	ds := testDocsSettled(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("unsaved")
	root, _ := ds.kind.recovery.root()
	// Backed up once the copy and the index are written.
	waitFor(t, "the recovery copy", func() bool { return d.backedUpAt() != 0 })
	m, _ := filepath.Glob(filepath.Join(root, "*", d.id))
	if len(m) != 1 || readString(t, m[0]) != "unsaved" {
		t.Fatalf("recovery copies %q", m)
	}
	copyPath := m[0]
	var entries []recoveryEntry
	if err := json.Unmarshal([]byte(readString(t, filepath.Join(filepath.Dir(copyPath), "index.json"))), &entries); err != nil || len(entries) != 1 || entries[0].Name != "Untitled" {
		t.Fatalf("index %v: %v", entries, err)
	}
	// Discarded: gone.
	answer("Don’t Save")
	d.Close()
	waitFor(t, "the copy to go", func() bool { _, err := os.Stat(copyPath); return os.IsNotExist(err) })

	// A session no instance holds, left by a crash, comes back edited, and
	// its file, changed meanwhile, in conflict.
	path := writeTemp(t, "crashed.txt", "on disk")
	orphan := filepath.Join(root, "orphan")
	os.MkdirAll(orphan, 0o700)
	stamp, _, _ := statStamp(path)
	stamp.ModTime--
	index, _ := json.Marshal([]recoveryEntry{{ID: "a", Name: "Untitled 3"}, {ID: "b", Path: path, Stamp: stamp}})
	os.WriteFile(filepath.Join(orphan, "index.json"), index, 0o600)
	os.WriteFile(filepath.Join(orphan, "a"), []byte("draft"), 0o600)
	os.WriteFile(filepath.Join(orphan, "b"), []byte("changes"), 0o600)
	if n := ds.kind.restore(); n != 2 {
		t.Fatalf("restored %d documents", n)
	}
	var untitled, file *Document[string]
	for _, d := range ds.Documents() {
		if d.Path() == path {
			file = d
		} else if d.Name() == "Untitled 3" {
			untitled = d
		}
	}
	if untitled == nil || file == nil {
		t.Fatalf("restored %v", ds.Documents())
	}
	if untitled.Content() != "draft" || !untitled.IsEdited() || file.Content() != "changes" || !file.IsEdited() || !file.HasConflict() {
		t.Errorf("restored %q %v, %q %v conflict %v", untitled.Content(), untitled.IsEdited(), file.Content(), file.IsEdited(), file.HasConflict())
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("the claimed session is still there")
	}
}

func TestDocumentQuit(t *testing.T) {
	quits := make(chan struct{}, 1)
	quitAgain = func() {
		docQuit.resolved = false
		quits <- struct{}{}
	}
	t.Cleanup(func() { quitAgain = func() { App.Quit() } })

	// Kept for the next launch, without asking.
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("keep me")
	asked := answer("Cancel")
	if onMainValue(fb.QuitRequested) {
		t.Fatal("quit with an unsaved document not kept yet")
	}
	<-quits
	if len(asked.all()) != 0 || d.backedUpAt() == 0 {
		t.Errorf("asked %q, backed up %v", asked.all(), d.backedUpAt())
	}

	// Asked about.
	ask := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff, AskOnQuit: true})
	a := ask.New()
	a.Set("ask")
	asked = answer("Cancel")
	if onMainValue(fb.QuitRequested) {
		t.Fatal("quit without asking")
	}
	waitFor(t, "the question", func() bool { return len(asked.all()) == 1 })
	select {
	case <-quits:
		t.Fatal("quit after Cancel")
	case <-time.After(20 * time.Millisecond):
	}
	answer("Don’t Save")
	onMainValue(fb.QuitRequested)
	<-quits
}

func (b *docBase) backedUpAt() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.backedUp
}

func TestDocumentRenameMove(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "old.txt", "x")
	d, _ := ds.Open(path)
	if err := d.Rename("new.txt"); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(filepath.Dir(path), "new.txt")
	if d.Path() != renamed || readString(t, renamed) != "x" {
		t.Fatalf("renamed to %s", d.Path())
	}
	dir := t.TempDir()
	if err := d.MoveTo(dir); err != nil {
		t.Fatal(err)
	}
	if d.Path() != filepath.Join(dir, "new.txt") {
		t.Fatalf("moved to %s", d.Path())
	}
	fw := fakeWindow(d.Window())
	waitFor(t, "the represented file to follow", func() bool { p, _ := fw.Document(); return p == d.Path() })

	dup := d.Duplicate()
	if dup.Name() != "new copy" || dup.Content() != "x" || !dup.IsEdited() || dup.Path() != "" {
		t.Errorf("duplicate %q %q %v", dup.Name(), dup.Content(), dup.IsEdited())
	}
}

// TestDocumentFollowsReplacedRename renames a file that another app
// replaced, after the document read the new file but before its watch
// looked again: the watch knows only the file replaced, so the document
// finds the new one by what it read.
func TestDocumentFollowsReplacedRename(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "doc.txt", "one")
	path, _ = filepath.EvalSymlinks(path)
	d, _ := ds.Open(path)
	// The watch saw the first file once it enrolled the file itself.
	var watch *fake.FileWatch
	waitFor(t, "the watch to see the file", func() bool {
		watch = fb.LastFileWatch()
		return slices.ContainsFunc(watch.Targets(), func(t platform.FileWatchTarget) bool { return t.Path == path && !t.Directory })
	})
	if err := os.WriteFile(path+".tmp", []byte("two, longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		t.Fatal(err)
	}
	// Read by the look when the watch started or the window was focused.
	d.checkDisk()
	waitFor(t, "the document to read the new file", func() bool { return d.Content() == "two, longer" })
	renamed := filepath.Join(filepath.Dir(path), "renamed.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	for _, target := range watch.Targets() {
		if target.Path == filepath.Dir(path) {
			watch.Notify(platform.FileWatchNotice{Target: target.ID, Name: "doc.txt", Kind: platform.FileWatchRemove})
		}
	}
	// Following it records it as recent, in the user data the test removes.
	waitFor(t, "the document to follow the rename", func() bool {
		return d.Path() == renamed && slices.Contains(App.RecentDocuments(), renamed)
	})
	if d.State().Deleted {
		t.Error("deleted")
	}
}

func TestDocumentMenus(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	file := NewMenu([]*MenuItem{{Role: RoleFileMenu}}).Items()[0]
	var roles []MenuRole
	for _, it := range file.Submenu {
		roles = append(roles, it.Role)
	}
	for _, r := range []MenuRole{RoleNew, RoleOpen, RoleOpenRecent, RoleSave, RoleRevert, RoleClose} {
		if !slices.Contains(roles, r) {
			t.Errorf("the File menu lacks %s: %v", r, roles)
		}
	}
	path := writeTemp(t, "menu.txt", "1")
	d, _ := ds.Open(path)
	d.Set("2")
	documentRole(RoleSave, d.Window())
	waitFor(t, "Save", func() bool { return fileIs(path, "2") })
	n := len(allDocuments())
	documentRole(RoleNew, nil)
	waitFor(t, "New", func() bool { return len(allDocuments()) == n+1 })
	for _, b := range allDocuments() {
		if b.kind != ds.kind && b.parent() != nil {
			b.parent().Destroy() // the first kind's
		}
	}

	recent := NewMenu([]*MenuItem{{Role: RoleOpenRecent}}).Items()[0]
	if len(recent.Submenu) < 3 || recent.Submenu[0].Label != "menu.txt" {
		t.Errorf("Open Recent: %v", recent.Submenu)
	}
}

var serviceRuns int

func TestDocumentService(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	serviceRuns++
	note := fmt.Sprintf("Note%d", serviceRuns)
	if err := bind(note, ds.Service(), false); err != nil {
		t.Fatal(err)
	}
	path := writeTemp(t, "page.txt", "from Go")
	d, _ := ds.Open(path)
	fw := fakeWindow(d.Window())
	onMain(func() { fw.H.NavigationCommitted("about:blank") })
	page(fw, `{"t":"dom-ready"}`)
	if r := call(t, fw, 1, note+".Content"); r["v"] != "from Go" {
		t.Fatalf("content: %v", r)
	}
	call(t, fw, 2, note+".SetContent", "from the page")
	if d.Content() != "from the page" || !d.IsEdited() {
		t.Fatalf("content %q edited %v", d.Content(), d.IsEdited())
	}
	received(t, fw, func(m map[string]any) bool {
		p, _ := m["p"].(map[string]any)
		return m["n"] == documentStateEvent && p["edited"] == true && p["name"] == "page.txt"
	})
	if r := call(t, fw, 3, note+".State"); r["v"].(map[string]any)["path"] != path {
		t.Errorf("state: %v", r)
	}
}

func TestUntitledNumber(t *testing.T) {
	for name, want := range map[string]int{
		"Untitled": 1, "Untitled 2": 2, "Untitled 12": 12, "Untitled x": 0, "Notes": 0, "Untitled 1": 0,
	} {
		if got := untitledNumber(name); got != want {
			t.Errorf("untitledNumber(%q) = %d, want %d", name, got, want)
		}
	}
	// A language whose digits are others: the number reads back.
	if got := l10n.In("fa").UntitledNumbered(12); got != "Untitled ۱۲" || untitledNumberIn(got, "fa", "en") != 12 {
		t.Errorf("Persian: %q", got)
	}
	if n, ok := decimalValue("٣٤"); !ok || n != 34 {
		t.Errorf("Arabic-Indic 34: %d %v", n, ok)
	}
	if got := untitledName(3); got != "Untitled 3" || untitledNumber(got) != 3 {
		t.Errorf("untitledName(3) = %q", got)
	}
}

// openDocumentInTempDir opens a document of a file in a directory of its own.
func openDocumentInTempDir(t *testing.T) (d *Document[string], dir, path string) {
	t.Helper()
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	dir, _ = filepath.EvalSymlinks(t.TempDir())
	path = filepath.Join(dir, "other.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := ds.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return d, dir, path
}

// otherAppSaves has another app save d's file as editors do (a new file
// renamed over it), which d reads, as the first look of a watch that started
// late does.
func otherAppSaves(t *testing.T, d *Document[string], path string) {
	t.Helper()
	if err := os.WriteFile(path+".tmp", []byte("two, longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		t.Fatal(err)
	}
	d.checkDisk()
	if got := d.Content(); got != "two, longer" {
		t.Fatalf("the document reads %q", got)
	}
}

// waitMoved waits for d to follow its file to renamed. The move records the
// file as a recent document after it set the path, and the test's directories
// go when the test ends.
func waitMoved(t *testing.T, d *Document[string], renamed string) {
	t.Helper()
	wait.Until(t, "the document to follow the rename", func() bool { return d.Path() == renamed })
	wait.Until(t, "the moved document to be recorded", func() bool { return slices.Contains(App.RecentDocuments(), renamed) })
}

// documentRenamedAfter opens a document, has change replace its file (as
// another app saving as editors do, or the document itself) and the
// document know it, then renames the file at once, within one debounce of the
// watch. The watch looks once, and last saw the file before the one that
// replaced it, which is gone by then, so it names no file for the document
// to follow. The document knows the file it read or wrote, though, and finds
// that one where it was renamed.
func documentRenamedAfter(t *testing.T, change func(t *testing.T, d *Document[string], path string)) {
	t.Helper()
	d, dir, path := openDocumentInTempDir(t)
	// The watch the document starts is not fed here: this one is, by notices
	// the test sends once both changes are made.
	ctx := wait.Context(t)
	w, err := WatchFiles(ctx, path, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	go followDocumentFile(ctx, w, path)
	change(t, d, path)
	renamed := filepath.Join(dir, "renamed.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "other.txt", Kind: platform.FileWatchCreate})
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "other.txt", Kind: platform.FileWatchRemove})
	waitMoved(t, d, renamed)
}

// TestDocumentFollowsAFileReplacedThenRenamedInOneLook: another app saved
// the file, and the document read it.
func TestDocumentFollowsAFileReplacedThenRenamedInOneLook(t *testing.T) {
	documentRenamedAfter(t, otherAppSaves)
}

// TestDocumentFollowsAFileItSavedThenRenamedInOneLook: the document saved
// the file, which writes a new file and renames it over the old.
func TestDocumentFollowsAFileItSavedThenRenamedInOneLook(t *testing.T) {
	documentRenamedAfter(t, func(t *testing.T, d *Document[string], path string) {
		d.Set("saved, longer")
		if err := d.Save(); err != nil {
			t.Fatal(err)
		}
		if got := readString(t, path); got != "saved, longer" {
			t.Fatalf("the file has %q", got)
		}
	})
}

// documentRenamedUnwatched opens a document and has another app save its
// file, which the document reads, then rename it away, all before any watch
// of the file started or looked, as on a machine so busy that the watch
// starts after.
func documentRenamedUnwatched(t *testing.T) (d *Document[string], renamed string) {
	t.Helper()
	d, dir, path := openDocumentInTempDir(t)
	otherAppSaves(t, d, path)
	renamed = filepath.Join(dir, "renamed.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	return d, renamed
}

// TestDocumentCheckFollowsARenamedFile: a look at a document whose file is
// not at its path finds the file in the directory by its identity, as a
// look at a window focused may after the watch missed the rename, or
// started after it.
func TestDocumentCheckFollowsARenamedFile(t *testing.T) {
	d, renamed := documentRenamedUnwatched(t)
	d.checkDisk()
	if got := d.Path(); got != renamed {
		t.Fatalf("the document is at %q, want %q", got, renamed)
	}
	if d.State().Deleted {
		t.Error("the document is taken for deleted")
	}
}

// TestDocumentCheckFollowsAFileRelocatedByCopy: a document moved across
// devices is a copy of its file, with an identity of its own, which a look
// finds where another app renamed it.
func TestDocumentCheckFollowsAFileRelocatedByCopy(t *testing.T) {
	d, dir, _ := openDocumentInTempDir(t)
	rename := renameFile
	renameFile = func(from, to string) error {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { renameFile = rename })
	if err := d.Rename("copied.txt"); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(dir, "renamed.txt")
	if err := os.Rename(d.Path(), renamed); err != nil {
		t.Fatal(err)
	}
	d.checkDisk()
	if got := d.Path(); got != renamed {
		t.Fatalf("the document is at %q, want %q", got, renamed)
	}
	if d.State().Deleted {
		t.Error("the document is taken for deleted")
	}
}

// TestDocumentWatchStartedAfterTheRename: a watch that starts when the file
// is gone, which fails, looks at the document, which finds its file.
func TestDocumentWatchStartedAfterTheRename(t *testing.T) {
	d, renamed := documentRenamedUnwatched(t)
	go watchDocumentFile(wait.Context(t), d.Path())
	waitMoved(t, d, renamed)
}

// TestDocumentCheckDoesNotFollowAFileThatReusedTheInode: the identity of a
// file deleted may be given to a file made later, which is not the
// document's file: its size and modification time say.
func TestDocumentCheckDoesNotFollowAFileThatReusedTheInode(t *testing.T) {
	d, renamed := documentRenamedUnwatched(t)
	path := d.Path()
	// The same file, as it was, with another content: what the identity
	// of a deleted file would find when a new file took its number.
	if err := os.WriteFile(renamed, []byte("changed since, and longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	d.checkDisk()
	if got := d.Path(); got != path {
		t.Fatalf("the document followed %q", got)
	}
	if !d.State().Deleted {
		t.Error("the document whose file is gone is not taken for deleted")
	}
}

// TestDocumentFoundAgainIsNotDeleted: a look found the file gone, and took
// the document for deleted; the file is back in the directory under another
// name, which the next look finds, and the document is not deleted then.
func TestDocumentFoundAgainIsNotDeleted(t *testing.T) {
	d, renamed := documentRenamedUnwatched(t)
	path := d.Path()
	away := filepath.Join(t.TempDir(), "away.txt")
	if err := os.Rename(renamed, away); err != nil {
		t.Fatal(err)
	}
	d.checkDisk()
	if !d.State().Deleted {
		t.Fatal("the document whose file went to another directory is not taken for deleted")
	}
	back := filepath.Join(filepath.Dir(path), "back.txt")
	if err := os.Rename(away, back); err != nil {
		t.Fatal(err)
	}
	d.checkDisk()
	if got := d.Path(); got != back {
		t.Fatalf("the document is at %q, want %q", got, back)
	}
	if d.State().Deleted {
		t.Error("the document is at the file it found again and still deleted")
	}
}

// TestDocWorkWaitsForAPendingTimer: the timer of an edited document is
// counted in docWork while it is armed, so that a test that waits for
// docWork has the write of the recovery copy done, and does not race with
// the timer firing (docWork.Go in its callback, adding from zero, with Wait:
// "race detected during execution of test").
func TestDocWorkWaitsForAPendingTimer(t *testing.T) {
	ds := testDocsSettled(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("unsaved") // arms the timer, a second
	docWork.Wait()
	if d.backedUpAt() == 0 {
		t.Fatal("docWork.Wait returned before the timer fired and wrote the recovery copy")
	}
}
