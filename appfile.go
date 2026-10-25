package mygo

import (
	"os"
	"sync"

	"github.com/egoist/mygo/internal/update"
)

// Windows refuses to replace or delete a file that another program has
// open without sharing its deletion, as Go opens files, and antivirus
// software and indexers do while they scan new files; and it refuses to
// open a file while it is being replaced. The app's own files (window
// states, recovery copies, recent documents) are replaced in place, so
// their renames and reads retry for a few seconds while Windows reports
// them in use, rather than lose what they hold.

// rename renames a file of the app, retrying while Windows reports it in
// use.
func rename(from, to string) error {
	return update.Rename(from, to)
}

// readAppFile reads a file of the app, retrying while Windows reports it
// in use.
func readAppFile(path string) (data []byte, err error) {
	err = update.RetryInUse(func() error {
		data, err = os.ReadFile(path)
		return err
	})
	return data, err
}

// appWrites runs the writes of the app's files that the main thread asks for,
// off it: a write that Windows reports in use is retried for seconds
// (rename), which stood the whole interface for as long. The writes run one
// at a time on a goroutine, in the order of their keys' first request; a
// request for a key whose write has not started replaces it, as it is
// the newer state of the same file. flushAppWrites waits for them, which
// quitting does before the process ends.
var appWrites struct {
	mu      sync.Mutex
	cond    *sync.Cond
	pending map[string]func()
	order   []string
	running bool
}

// writeAppFileLater has fn, which writes the file of key, run off the main
// thread, soon.
func writeAppFileLater(key string, fn func()) {
	w := &appWrites
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pending == nil {
		w.pending, w.cond = map[string]func(){}, sync.NewCond(&w.mu)
	}
	if _, queued := w.pending[key]; !queued {
		w.order = append(w.order, key)
	}
	w.pending[key] = fn
	if !w.running {
		w.running = true
		docWork.Add(1) // tests wait for it: a late write would land in the next test's files
		go runAppWrites()
	}
}

func runAppWrites() {
	w := &appWrites
	w.mu.Lock()
	for len(w.order) > 0 {
		key := w.order[0]
		w.order = w.order[1:]
		fn := w.pending[key]
		delete(w.pending, key)
		w.mu.Unlock()
		fn()
		w.mu.Lock()
	}
	w.running = false
	w.cond.Broadcast()
	w.mu.Unlock()
	docWork.Done()
}

// saveAppFiles writes the window states and waits for every write asked for
// to be done: what quitting does, before OnQuit, which may end the process.
// Main thread only.
func saveAppFiles() {
	saveWindowStates()
	flushAppWrites()
}

// flushAppWrites waits for the writes asked for to be done.
func flushAppWrites() {
	w := &appWrites
	w.mu.Lock()
	defer w.mu.Unlock()
	for w.running {
		w.cond.Wait()
	}
}
